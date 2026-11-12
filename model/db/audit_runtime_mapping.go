package db

import (
	"errors"
	"fmt"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// RuntimeAuditContainerName 审计用容器 ID→展示名映射（与 ContainerIdList 对齐）。
type RuntimeAuditContainerName struct {
	NodeId         int64  `gorm:"column:node_id;primaryKey"`
	ContainerId    string `gorm:"column:container_id;primaryKey;size:255"`
	ContainerName  string `gorm:"column:container_name;size:255"`
