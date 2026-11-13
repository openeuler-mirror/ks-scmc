package db

import (
	"context"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"scmc/common"
)

var (
	db           *gorm.DB
	lock         sync.Mutex
	healthOnce   sync.Once
	pingInterval = 15 * time.Second
	pingTimeout  = 3 * time.Second
	// 缩短存活与空闲时间，减轻 VIP 切换后坏连接在池内滞留
	connMaxLifetime = 5 * time.Minute
	connMaxIdleTime = 2 * time.Minute
)

type gormLogger struct{}

func (*gormLogger) Printf(format string, v ...interface{}) {
	log.Infof(format, v...)
}

// resetPool 应用层断连：关闭底层 sql.DB 并清空全局引用，后续 GetConn 会重建连接池。
func resetPool(reason error) {
	lock.Lock()
	old := db
	db = nil
	lock.Unlock()

	if old == nil {
		return
	}
	sqlDB, err := old.DB()
	if err != nil {
		log.Warnf("Database pool reset (get sql.DB): %v; reason: %v", err, reason)
		return
	}
	if err := sqlDB.Close(); err != nil {
		log.Warnf("Database pool close: %v; reason: %v", err, reason)
	} else {
		log.Warnf("Database pool reset and closed: %v", reason)
	}
}

// 检查数据库连接池健康状态，不健康重置，触发重连
func healthCheckLoop() {
	ticker := time.NewTicker(pingInterval)
	defer ticker.Stop()
	for range ticker.C {
		lock.Lock()
		g := db
		lock.Unlock()
		if g == nil {
			continue
		}
		sqlDB, err := g.DB()
		if err != nil {
			resetPool(err)
			continue
		}
		ctx, cancel := context.WithTimeout(context.Background(), pingTimeout)
		err = sqlDB.PingContext(ctx)
		cancel()
		if err != nil {
			resetPool(err)
		}
	}
}

func applyPoolSettings(conn *gorm.DB) error {
	sqlDB, err := conn.DB()
	if err != nil {
		return err
	}
	sqlDB.SetMaxOpenConns(10)
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetConnMaxLifetime(connMaxLifetime)
	sqlDB.SetConnMaxIdleTime(connMaxIdleTime)
	return nil
}

func getConn() (*gorm.DB, error) {
	lock.Lock()
	defer lock.Unlock()

	if db != nil {
		return db, nil
	}
