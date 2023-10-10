// node info
package model

import (
	log "github.com/sirupsen/logrus"
)

type NodeInfo struct {
	ID          int64 `gorm:"primaryKey"`
	Name        string
	Address     string
	Comment     string
	UnreadWarn  int64
	CpuLimit    float64
	MemoryLimit float64
	DiskLimit   float64
	Deleted     bool
	CreatedAt   int64 `gorm:"autoCreateTime"`
	UpdatedAt   int64 `gorm:"autoUpdateTime"`
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
	result := db.Where("deleted = 0").Find(&nodes)
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

	result := db.Model(&NodeInfo{}).Where("id IN ?", ids).Update("deleted", 1)
	if result.Error != nil {
		log.Errorf("db remove node ids=%v: %v", ids, result.Error)
		return result.Error
	}

	log.Debugf("db remove node ids=%v OK", ids)
	return nil
}

func UpdateNode(n *NodeInfo) error {
	db, err := getConn()
	if err != nil {
		return err
	}

	result := db.Save(n)
	if result.Error != nil {
		log.Warnf("save node info=%+v err=%v", n, result.Error)
		return translateError(result.Error)
	}

	return nil
}

func QueryNodeByID(id int64) (*NodeInfo, error) {
	db, err := getConn()
	if err != nil {
		return nil, err
	}

	var nodeInfo NodeInfo
	result := db.Where("deleted = 0").First(&nodeInfo, id)
	if result.Error != nil {
		log.Warnf("query node id=%v: %v", id, err)
		return nil, translateError(result.Error)
	} else if result.RowsAffected == 0 {
		log.Warnf("query node id=%v not found", id)
		return nil, ErrRecordNotFound
	}

	return &nodeInfo, nil
}

func QueryNodeByAddr(addr string) (*NodeInfo, error) {
	db, err := getConn()
	if err != nil {
		return nil, err
	}

	var nodeInfo NodeInfo
	result := db.Where("address = ?", addr).First(&nodeInfo)
	if result.Error != nil {
		log.Warnf("query node addr=%v: %v", addr, err)
		return nil, translateError(result.Error)
	} else if result.RowsAffected == 0 {
		log.Infof("query node addr=%v not found", addr)
		return nil, ErrRecordNotFound
	}

	return &nodeInfo, nil
}
