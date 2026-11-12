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
	db, err := getAuditMappingConn()
	if err != nil {
		log.Warnf("UpsertRuntimeAuditContainerName: audit sqlite unavailable: %v", err)
		return err
	}
	var existing RuntimeAuditContainerName
	q := db.Where("node_id = ? AND container_id = ?", nodeID, containerID).First(&existing)
	if q.Error != nil {
		if !errors.Is(q.Error, ErrRecordNotFound) {
			log.Warnf("UpsertRuntimeAuditContainerName query node=%d cid=%s err=%v", nodeID, containerID, q.Error)
			return translateError(q.Error)
		}
		row := RuntimeAuditContainerName{
			NodeId:        nodeID,
			ContainerId:   containerID,
			ContainerName: containerName,
			IsDeleted:     false,
			DeletedAt:     0,
		}
		if err := db.Create(&row).Error; err != nil {
			log.Warnf("UpsertRuntimeAuditContainerName create node=%d cid=%s err=%v", nodeID, containerID, err)
			return translateError(err)
		}
		return nil
	}
	if err := db.Model(&existing).Updates(map[string]interface{}{
		"container_name": containerName,
		"is_deleted":     false,
		"deleted_at":     0,
	}).Error; err != nil {
		log.Warnf("UpsertRuntimeAuditContainerName update node=%d cid=%s err=%v", nodeID, containerID, err)
		return translateError(err)
	}
	return nil
}

// Docker 列表/操作接口常用 12 位短 ID，Create 落库多为完整 ID；低于此长度不做前缀匹配以免误命中。
const minDockerContainerIDPrefixLen = 12

func normalizeDockerContainerID(id string) string {
	id = strings.TrimSpace(id)
	id = strings.TrimPrefix(strings.ToLower(id), "sha256:")
	return id
}

func findRuntimeAuditContainerRow(gdb *gorm.DB, nodeID int64, cid string) (*RuntimeAuditContainerName, bool) {
	n := normalizeDockerContainerID(cid)
	if n == "" {
		return nil, false
	}
	var row RuntimeAuditContainerName
	if err := gdb.Where("node_id = ? AND LOWER(container_id) = ?", nodeID, n).First(&row).Error; err == nil {
		return &row, true
	}
	if len(n) >= minDockerContainerIDPrefixLen {
		if err := gdb.Where("node_id = ? AND LOWER(container_id) LIKE ?", nodeID, n+"%").First(&row).Error; err == nil {
			return &row, true
		}
	}
	// SQLite：LIKE 优先级低于 ||，必须加括号，否则语义错误。
	if len(n) > minDockerContainerIDPrefixLen {
		if err := gdb.Where("node_id = ? AND LENGTH(container_id) >= ? AND LOWER(?) LIKE (LOWER(container_id) || ?)",
			nodeID, minDockerContainerIDPrefixLen, n, "%").First(&row).Error; err == nil {
			return &row, true
		}
	}
	return nil, false
}

func resolveRuntimeAuditContainerName(gdb *gorm.DB, nodeID int64, cid string) (string, bool) {
	row, ok := findRuntimeAuditContainerRow(gdb, nodeID, cid)
	if !ok || row == nil || row.ContainerName == "" {
		return "", false
	}
	return row.ContainerName, true
}

func MarkRuntimeAuditContainersDeleted(nodeID int64, containerIDs []string) error {
	if nodeID <= 0 || len(containerIDs) == 0 {
		return nil
	}
	db, err := getAuditMappingConn()
	if err != nil {
		return err
	}
	now := time.Now().Unix()
	seen := make(map[string]struct{})
	for _, cid := range containerIDs {
		if cid == "" {
			continue
		}
		row, ok := findRuntimeAuditContainerRow(db, nodeID, cid)
		if !ok || row == nil {
			continue
		}
		if _, dup := seen[row.ContainerId]; dup {
			continue
		}
		seen[row.ContainerId] = struct{}{}
		res := db.Model(&RuntimeAuditContainerName{}).
			Where("node_id = ? AND container_id = ?", nodeID, row.ContainerId).
			Updates(map[string]interface{}{
				"is_deleted": true,
				"deleted_at": now,
			})
		if res.Error != nil {
			log.Warnf("MarkRuntimeAuditContainersDeleted node=%d cid=%s err=%v", nodeID, row.ContainerId, res.Error)
			return translateError(res.Error)
		}
	}
	return nil
}

// ResolveContainerNamesForNode 按节点解析容器展示名（请求里常为短 ID，映射表多为 Create 返回的完整 ID）。
// 未命中映射时 map 中无该 key。不过滤 is_deleted：Remove 审计 defer 时可能已软删，仍展示历史名称。
func ResolveContainerNamesForNode(nodeID int64, containerIDs []string) map[string]string {
	out := make(map[string]string)
	if nodeID <= 0 || len(containerIDs) == 0 {
		return out
	}
	db, err := getAuditMappingConn()
	if err != nil {
		log.Warnf("ResolveContainerNamesForNode: audit sqlite unavailable: %v", err)
		return out
	}
	for _, cid := range containerIDs {
		if cid == "" {
			continue
		}
		if _, ok := out[cid]; ok {
			continue
		}
		if name, ok := resolveRuntimeAuditContainerName(db, nodeID, cid); ok {
			out[cid] = name
		}
	}
	return out
}

func UpsertRuntimeAuditImageTarget(imageID int64, display string) error {
	if imageID <= 0 {
		return nil
	}
	db, err := getAuditMappingConn()
	if err != nil {
		log.Warnf("UpsertRuntimeAuditImageTarget: audit sqlite unavailable: %v", err)
		return err
	}
	var existing RuntimeAuditImageTarget
	q := db.Where("image_id = ?", imageID).First(&existing)
	if q.Error != nil {
		if !errors.Is(q.Error, ErrRecordNotFound) {
			log.Warnf("UpsertRuntimeAuditImageTarget query id=%d err=%v", imageID, q.Error)
			return translateError(q.Error)
		}
		row := RuntimeAuditImageTarget{
			ImageID:   imageID,
			Display:   display,
			IsDeleted: false,
			DeletedAt: 0,
		}
		if err := db.Create(&row).Error; err != nil {
			log.Warnf("UpsertRuntimeAuditImageTarget create id=%d err=%v", imageID, err)
			return translateError(err)
		}
		return nil
	}
	if err := db.Model(&existing).Updates(map[string]interface{}{
		"display":    display,
		"is_deleted": false,
		"deleted_at": 0,
	}).Error; err != nil {
		log.Warnf("UpsertRuntimeAuditImageTarget update id=%d err=%v", imageID, err)
		return translateError(err)
	}
	return nil
}

func MarkRuntimeAuditImageDeleted(imageID int64) error {
	if imageID <= 0 {
		return nil
	}
	db, err := getAuditMappingConn()
	if err != nil {
		return err
	}
	now := time.Now().Unix()
	res := db.Model(&RuntimeAuditImageTarget{}).Where("image_id = ?", imageID).Updates(map[string]interface{}{
		"is_deleted": true,
		"deleted_at": now,
	})
	if res.Error != nil {
		log.Warnf("MarkRuntimeAuditImageDeleted id=%d err=%v", imageID, res.Error)
		return translateError(res.Error)
	}
	return nil
}

// ResolveImageAuditTarget 按 image_id 顺序解析展示串；含已软删映射行（Remove 审计 defer 仍可读 display）；否则回退 image_infos。
func ResolveImageAuditTarget(ids []int64) string {
	if len(ids) == 0 {
		return ""
	}
	db, err := getAuditMappingConn()
	if err != nil {
		log.Warnf("ResolveImageAuditTarget: audit sqlite unavailable: %v", err)
		return joinInt64s(ids)
	}
	var rows []RuntimeAuditImageTarget
	if err := db.Where("image_id IN ?", ids).Find(&rows).Error; err != nil {
		log.Warnf("ResolveImageAuditTarget query mapping err=%v", err)
	}
	fromMap := make(map[int64]string, len(rows))
