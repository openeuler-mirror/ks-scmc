package litedb

import (
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"scmc/common"
)

var (
	db   *gorm.DB
	lock sync.Mutex
)

type gormLogger struct{}

func (*gormLogger) Printf(format string, v ...interface{}) {
	log.Infof(format, v...)
}

func dbFilePath() string {
	return common.Config.SQLite.Path
}

func getConn() (*gorm.DB, error) {
	lock.Lock()
	defer lock.Unlock()

	if db != nil {
		return db, nil
	}

	_db, err := gorm.Open(sqlite.Open(dbFilePath()), &gorm.Config{
		Logger: logger.New(&gormLogger{}, logger.Config{
			SlowThreshold: time.Millisecond * 100,
			LogLevel:      logger.Warn,
		}),
	})

	if err != nil {
		log.Warnf("Failed to inititlize database=%v err=%v", dbFilePath(), err)
		return nil, err
	}

	m := _db.Migrator()
	if !m.HasTable(&ContainerConfig{}) {
		_ = m.CreateTable(&ContainerConfig{})
	}
	if !m.HasTable(&ContainerBackup{}) {
		_ = m.CreateTable(&ContainerBackup{})
	}
	if !m.HasTable(&AppEntry{}) {
		_ = m.CreateTable(&AppEntry{})
	}
	if !m.HasTable(&NodeInfo{}) {
		_ = m.CreateTable(&NodeInfo{})
	}

	conn, err := _db.DB()
	if err != nil {
		log.Warnf("access to database/sql: %v", err)
		return nil, err
	}
	conn.SetMaxOpenConns(1)
	conn.SetMaxIdleConns(1)

	db = _db
	return db, nil
}
