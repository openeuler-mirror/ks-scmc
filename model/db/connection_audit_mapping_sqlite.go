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
