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
	IsDeleted      bool   `gorm:"column:is_deleted"`
	DeletedAt      int64  `gorm:"column:deleted_at"`
	CreatedAt      int64  `gorm:"autoCreateTime"`
	UpdatedAt      int64  `gorm:"autoUpdateTime"`
}

func (RuntimeAuditContainerName) TableName() string {
	return "runtime_audit_container_names"
}

// RuntimeAuditImageTarget 审计用镜像 ID→展示串（name:version）。
type RuntimeAuditImageTarget struct {
	ImageID   int64  `gorm:"column:image_id;primaryKey"`
	Display   string `gorm:"column:display;size:512"`
	IsDeleted bool   `gorm:"column:is_deleted"`
	DeletedAt int64  `gorm:"column:deleted_at"`
	CreatedAt int64  `gorm:"autoCreateTime"`
	UpdatedAt int64  `gorm:"autoUpdateTime"`
}

func (RuntimeAuditImageTarget) TableName() string {
	return "runtime_audit_image_targets"
}

// ImageAuditDisplay 与审计 Target 展示规则一致。
func ImageAuditDisplay(name, version string) string {
	name = strings.TrimSpace(name)
	version = strings.TrimSpace(version)
	if name == "" {
		return ""
	}
	if version != "" {
		return name + ":" + version
	}
	return name
}

func UpsertRuntimeAuditContainerName(nodeID int64, containerID, containerName string) error {
	if nodeID <= 0 || containerID == "" {
		return nil
	}
