package model

import (
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	"gorm.io/driver/mysql"
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

func getConn() (*gorm.DB, error) {
	lock.Lock()
	defer lock.Unlock()

	if db != nil {
		return db, nil
	}

	dsn := common.Config.MySQL.DSN()
	_db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logger.New(&gormLogger{}, logger.Config{
			SlowThreshold: time.Millisecond * 50,
			LogLevel:      logger.Warn,
		}),
	})
	if err != nil {
		log.Warnf("connect to database: %s, error: %v", dsn, err)
		return nil, err
	}

	conn, err := _db.DB()
	if err != nil {
		log.Warnf("access to database/sql: %v", err)
		return nil, err
	}
	conn.SetMaxOpenConns(10)
	conn.SetMaxIdleConns(10)
	conn.SetConnMaxLifetime(time.Hour)

	if common.Config.MySQL.Debug {
		db = _db.Debug()
	} else {
		db = _db
	}

	return db, nil
}
