// node info
package model

import (
	log "github.com/sirupsen/logrus"
)

type NodeInfo struct {
	ID        int64 `gorm:"primaryKey"`
	Name      string
	Address   string
	Comment   string
	CreatedAt int64 `gorm:"autoCreateTime"`
	UpdatedAt int64 `gorm:"autoUpdateTime"`
}

func (NodeInfo) TableName() string {
	return "node_infos"
}

func ListNodes() ([]NodeInfo, error) {
	db, err := getConn()
	if err != nil {
		return nil, err
	}

	var nodes []NodeInfo
	result := db.Find(&nodes)
	if result.Error != nil {
		log.Errorf("db query nodes: %v", result.Error)
		return nil, result.Error
	}

	return nodes, nil
}

func CreateNode(name, address, comment string) error {
	db, err := getConn()
	if err != nil {
		return err
	}

	nodeInfo := NodeInfo{Name: name, Address: address, Comment: comment}
	result := db.Create(&nodeInfo)
	if result.Error != nil {
		log.Errorf("db create node %v", result.Error)
		return translateError(result.Error)
	}

	log.Debugf("db create node id=%v", nodeInfo.ID)
	return nil
}

func RemoveNode(ids []int64) error {
	db, err := getConn()
	if err != nil {
		return err
	}

	/*
		var nodeInfo NodeInfo
		result := db.First(&nodeInfo, id)

		if result.Error == gorm.ErrRecordNotFound {
			log.Printf("DeleteNode query id=%v not found", id)
			return ErrDBRecordNotFound
		} else if result.Error != nil {
			log.Printf("DeleteNode query: %v", result.Error)
			return result.Error
		}
	*/

	// result = db.Delete(&nodeInfo)
	result := db.Where("id IN ?", ids).Delete(NodeInfo{})
	if result.Error != nil {
		log.Errorf("db remove node ids=%v: %v", ids, result.Error)
		return result.Error
	}

	log.Debugf("db remove node ids=%v OK", ids)
	return nil
}
