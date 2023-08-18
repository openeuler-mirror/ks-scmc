package model

import (
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"ksc-mcube/common"
)

var (
	db   *gorm.DB
	lock sync.Mutex
)

func getConn() (*gorm.DB, error) {
	lock.Lock()
	defer lock.Unlock()

	if db != nil {
		return db, nil
	}

	dsn := common.Config.MySQL.DSN()
	_db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
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

	db = _db
	return db, nil
}
