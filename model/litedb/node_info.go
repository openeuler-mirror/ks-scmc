package litedb

import (
	log "github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type NodeInfo struct {
	ID        int64  `gorm:"primaryKey;not null"`
	UUID      string `gorm:"uniqueIndex"`
	CreatedAt int64  `gorm:"autoCreateTime"`
	UpdatedAt int64  `gorm:"autoUpdateTime"`
}

func CreateNodeInfo(id int64, uuid string) error {
	db, err := getConn()
	if err != nil {
		return err
	}

	if err := db.Create(&NodeInfo{ID: id, UUID: uuid}).Error; err != nil {
		log.Errorf("db create node info: %v", err)
		return err
	}
	return nil
}

func RemoveNodeInfo(id int64, uuid string) error {
	db, err := getConn()
	if err != nil {
		return err
	}

	result := db.Delete(&NodeInfo{}, "id = ? AND uuid = ?", id, uuid)
	if result.Error != nil {
		log.Errorf("db remove node info: %v", err)
		return result.Error
	} else if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}
