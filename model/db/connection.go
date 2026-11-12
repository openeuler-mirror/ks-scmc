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
