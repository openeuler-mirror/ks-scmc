package model

import (
	"flag"
	"log"
	"sync"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var (
	db   *gorm.DB
	lock sync.Mutex
	dsn  string
)

func getConn() (*gorm.DB, error) {
	lock.Lock()
	defer lock.Unlock()

	if db != nil {
		return db, nil
	}

	_db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Printf("connect to database: %s, error: %v", dsn, err)
		return nil, err
	}

	conn, err := _db.DB()
	if err != nil {
		log.Printf("access to database/sql: %v", err)
		return nil, err
	}
	conn.SetMaxOpenConns(10)
	conn.SetMaxIdleConns(10)
	conn.SetConnMaxLifetime(time.Hour)

	db = _db
	return db, nil
}

func init() {
	flag.StringVar(&dsn, "mysql-dsn", "root:123456@tcp(127.0.0.1:13306)/ksc-mcube?charset=utf8mb4&timeout=10s", "mysql connection")
}
