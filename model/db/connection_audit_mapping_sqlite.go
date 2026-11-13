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
