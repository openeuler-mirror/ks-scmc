package db

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"scmc/common"
)

var (
	auditMappingDB   *gorm.DB
	auditMappingLock sync.Mutex
)

// getAuditMappingConn 返回仅承载 runtime_audit_* 映射表的 SQLite 连接（与 MySQL 业务库分离）。
func getAuditMappingConn() (*gorm.DB, error) {
	auditMappingLock.Lock()
	defer auditMappingLock.Unlock()

	if auditMappingDB != nil {
		return auditMappingDB, nil
	}

	path := strings.TrimSpace(common.Config.SQLite.Path)
	if path == "" {
		return nil, fmt.Errorf("config sqlite.path is empty: required for audit runtime mapping tables")
	}
	path = filepath.Clean(path)

	if err := os.MkdirAll(filepath.Dir(path), 0750); err != nil {
		return nil, fmt.Errorf("create sqlite dir: %w", err)
	}

	_db, err := gorm.Open(sqlite.Open(path), &gorm.Config{
		Logger: logger.New(&gormLogger{}, logger.Config{
			SlowThreshold: time.Millisecond * 100,
			LogLevel:      logger.Warn,
		}),
	})
	if err != nil {
		return nil, fmt.Errorf("open sqlite %s: %w", path, err)
	}

	sqlDB, err := _db.DB()
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxOpenConns(1) // SQLite 单写；映射访问轻量
	sqlDB.SetMaxIdleConns(1)
	sqlDB.SetConnMaxLifetime(time.Hour)

	// 不用 GORM AutoMigrate：SQLite 驱动对复合主键等结构会生成 invalid DDL；与 scripts/etc/sqlite_audit_mapping.sql 一致。
	if err := ensureAuditMappingSQLiteSchema(_db); err != nil {
		return nil, err
	}

	auditMappingDB = _db
	log.Infof("audit mapping sqlite ready: %s", path)
	return auditMappingDB, nil
}

const (
	sqlCreateRuntimeAuditContainerNames = `
CREATE TABLE IF NOT EXISTS runtime_audit_container_names (
  node_id integer NOT NULL,
  container_id text NOT NULL,
  container_name text NOT NULL DEFAULT '',
  is_deleted integer NOT NULL DEFAULT 0,
  deleted_at integer NOT NULL DEFAULT 0,
  created_at integer NOT NULL DEFAULT 0,
  updated_at integer NOT NULL DEFAULT 0,
  PRIMARY KEY (node_id, container_id)
);`
	sqlCreateRuntimeAuditImageTargets = `
CREATE TABLE IF NOT EXISTS runtime_audit_image_targets (
  image_id integer NOT NULL,
  display text NOT NULL DEFAULT '',
