package db

import (
	"errors"
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type RuntimeLog struct {
	ID          int64 `gorm:"primaryKey"`
	EventType   int64
	EventType_  string `gorm:"column:event_type_"`
	EventModule int64
	NodeId      int64
	NodeInfo    string
	UserID      int64
	Target      string
	Detail      string
	StatusCode  int64
	Error       string
	CreatedAt   int64 `gorm:"autoCreateTime"`
	UpdatedAt   int64 `gorm:"autoUpdateTime"`
}

type RuntimeLogs struct {
	RuntimeLog
	Username string
}

func (RuntimeLogs) TableName() string {
	return "runtime_logs"
}

type WarnLog struct {
	ID            int64 `gorm:"primaryKey"`
	EventType     int64
	EventModule   int64
	NodeId        int64
	NodeInfo      string
	ContainerID   string
	ContainerName string
	Detail        string
	HaveRead      bool
	CreatedAt     int64 `gorm:"autoCreateTime"`
	UpdatedAt     int64 `gorm:"autoUpdateTime"`
}

type LogSettings struct {
	ID           int64  `gorm:"primaryKey"`
	SettingName  string `gorm:"unique;not null"` // 添加唯一键约束
	SettingValue string
}

func CreateRuntimeLog(logs []*RuntimeLog) error {
	db, err := getConn()
	if err != nil {
		return err
	}

	result := db.Create(&logs)
	if result.Error != nil {
		log.Errorf("db runtime log %v", result.Error)
		return translateError(result.Error)
	}

	return nil
}

type LogFilter struct {
	Property string
	Query    string
	Fuzzy    bool
}

func ListRuntimeLog(pageSize, pageNo int64, startTime, endTime, nodeID, eventModule int64, f *LogFilter) (*Pager, []*RuntimeLogs, error) {
	qs := queries{
		Where:  &query{},
		Select: &query{"runtime_logs.*, user_infos.username AS username", nil},
		Join:   &query{"LEFT JOIN user_infos ON runtime_logs.user_id = user_infos.id", nil},
		Model:  &RuntimeLogs{},
		Order:  "runtime_logs.id ASC",
	}

	if startTime > 0 && startTime < endTime {
		qs.Where.And("(runtime_logs.created_at BETWEEN ? AND ?)", startTime, endTime)
	}
	if nodeID > 0 {
		qs.Where.And("node_id = ?", nodeID)
	}
	if eventModule > 0 {
		qs.Where.And("event_module = ?", eventModule)
	}
	if f != nil {
		if f.Fuzzy {
			propertyString := fmt.Sprintf("%s LIKE ?", f.Property)
			qs.Where.And(propertyString, fmt.Sprintf("%%%s%%", f.Query))
		} else {
			propertyString := fmt.Sprintf("%s = ?", f.Property)
			qs.Where.And(propertyString, f.Query)
		}
	}

	var data []*RuntimeLogs
	pager, err := PageQuery(pageSize, pageNo, qs, &data)
	if err != nil {
		return nil, nil, err
	}

	return pager, data, nil
}

func CreateWarnLog(logs []*WarnLog) error {
	if len(logs) == 0 {
		return nil
	}

	nodeCnt := make(map[int64]int)
	for _, l := range logs {
		nodeCnt[l.NodeId] += 1
	}

	db, err := getConn()
	if err != nil {
		return err
	}

	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&logs).Error; err != nil {
			log.Warnf("db create warn log %v", err)
			return translateError(err)
		}

		for nodeID, c := range nodeCnt {
			if err := tx.Model(&NodeInfo{}).Where("id = ?", nodeID).Update("unread_warn", gorm.Expr("unread_warn+?", c)).Error; err != nil {
				log.Warnf("db increase unread_warn %v", err)
				return translateError(err)
			}
		}

		return nil
	})
}

func ListWarnLog(pageSize, pageNo int64, nodeID, eventModule int64) (*Pager, []*WarnLog, error) {
	qs := queries{
		Where: &query{},
		Model: &WarnLog{},
		Order: "id ASC",
	}

	if nodeID > 0 {
		qs.Where.And("node_id = ?", nodeID)
	}
	if eventModule > 0 {
		qs.Where.And("event_module = ?", eventModule)
	}
	qs.Where.And("have_read = 0")

	var data []*WarnLog
	pager, err := PageQuery(pageSize, pageNo, qs, &data)
	if err != nil {
		return nil, nil, err
	}

	return pager, data, nil
}

func SetWarnLogRead(ids []int64) error {
	var data []*WarnLog

	db, err := getConn()
	if err != nil {
		return err
	}

	if err := db.Find(&data, "id IN ?", ids).Error; err != nil {
		log.Warnf("SetWarnLogRead query data %v", err)
		return translateError(err)
	}

	nodeCnt := make(map[int64]int)
	for _, l := range data {
		if l.HaveRead {
			continue
		}

		nodeCnt[l.NodeId] += 1
	}

	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&WarnLog{}).Where("id IN ?", ids).Update("have_read", 1).Error; err != nil {
			log.Warnf("db create warn log %v", err)
			return translateError(err)
		}

		for nodeID, c := range nodeCnt {
			if err := tx.Model(&NodeInfo{}).Where("id = ?", nodeID).Update("unread_warn", gorm.Expr("unread_warn-?", c)).Error; err != nil {
				log.Warnf("db increase unread_warn %v", err)
				return translateError(err)
			}
		}

		return nil
	})
}

func ListWarnLogAll(pageSize, pageNo int64, nodeID, eventModule int64) (*Pager, []*WarnLog, error) {
	qs := queries{
		Where: &query{},
		Model: &WarnLog{},
		Order: "id ASC",
	}

	if nodeID > 0 {
		qs.Where.And("node_id = ?", nodeID)
	}
	if eventModule > 0 {
		qs.Where.And("event_module = ?", eventModule)
	}
	var data []*WarnLog
	pager, err := PageQuery(pageSize, pageNo, qs, &data)
	if err != nil {
		return nil, nil, err
	}
	return pager, data, nil
}

func SetLog(settingName, settingValue string) error {
	db, err := getConn()
	if err != nil {
		log.Warnf("SetLog failed, err=%v", err)
		return translateError(err)
	}

	// 使用 Upsert 方式（存在则更新，不存在则创建）
	result := db.Model(&LogSettings{}).
		Where("setting_name = ?", settingName).
		Updates(map[string]interface{}{
			"setting_value": settingValue,
		})

	// 如果没有记录被更新，则创建新记录
	if result.Error != nil {
		log.Warnf("failed to update log settings: %v", result.Error)
		return translateError(result.Error)
	}

	if result.RowsAffected == 0 {
		setting := LogSettings{
			SettingName:  settingName,
			SettingValue: settingValue,
		}
		if err := db.Create(&setting).Error; err != nil {
			log.Warnf("failed to create log setting: %v", err)
			return translateError(err)
		}
	}

	return nil
}

func GetLog(settingName string) (string, error) {
	var setting LogSettings

	db, err := getConn()
	if err != nil {
		log.Warnf("SetLog failed, err=%v", err)
		return "", translateError(err)
	}

	// 查询数据库
	result := db.Model(&LogSettings{}).
		Where("setting_name = ?", settingName).
		First(&setting)

	// 处理查询结果
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			// 记录不存在时返回空字符串而非错误
			return "180", nil
		}
		// 其他数据库错误
		log.Warnf("failed to get log setting: %v", result.Error)
		return "", translateError(result.Error)
	}

	return setting.SettingValue, nil
}

func cleanTable(db *gorm.DB, tableName string, threshold int64) error {
	result := db.Table(tableName).
		Where("created_at < ?", threshold).
		Delete(nil)

	if result.Error != nil {
		return result.Error
	}

	return nil
}

func CleanLogs(saveLogDays int64) error {
	if saveLogDays < 1 {
		return nil
	}

	// 计算保留的截止日期（当天00:00:00往前推saveLogDays天）
	now := time.Now()
	retentionDate := time.Date(now.Year(), now.Month(), now.Day()-int(saveLogDays)+1, 0, 0, 0, 0, now.Location())

	db, err := getConn()
	if err != nil {
		log.Warnf("SetLog failed, err=%v", err)
		return translateError(err)
	}

	// 清理 runtime_logs 表
	if err := cleanTable(db, "runtime_logs", retentionDate.Unix()); err != nil {
		log.Warnf("failed to clean runtime_logs: %v", err)
		return translateError(err)
	}

	// 清理 warn_logs 表
	if err := cleanTable(db, "warn_logs", retentionDate.Unix()); err != nil {
		log.Warnf("failed to clean warn_logs: %v", err)
		return translateError(err)
	}

	log.Infof("Logs cleaned successfully. Retained logs after %s (saveLogDays=%d)",
		retentionDate.Format("2006-01-02"),
		saveLogDays)
	return nil
}
