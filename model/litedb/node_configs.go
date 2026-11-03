package litedb

import (
	log "github.com/sirupsen/logrus"

	pb "scmc/rpc/pb/node"
	"scmc/rpc/pb/security"
)

type NodeConfig struct {
	ID             int64  `gorm:"primaryKey;not null"`
	UUID           string `gorm:"unique"`
	NodeID    string `gorm:"unique"`
	NodeName  string
	SecurityConfig *NodeSecurityConfigs `gorm:"serializer:json"`
	CreatedAt      int64            `gorm:"autoCreateTime"`
	UpdatedAt      int64            `gorm:"autoUpdateTime"`
}

type NodeSecurityConfigs = pb.SecurityConfig
type NodeProcProtection = security.ProcProtection
type NodeNetworkRule = security.NetworkRule

func CreateNodeConfig(data *NodeConfig) error {
	
	log.Debugf("CreateNodeConfig data=%+v ",data)
	if data == nil {
		return nil
	}

	db, err := getConn()
	if err != nil {
		return err
	}

	result := db.Create(data)
	if result.Error != nil {
		log.Errorf("db create Node configs %v", result.Error)
		return result.Error
	}

	return nil
}

func GetNodeConfig(NodeID string) (*NodeConfig, error) {
	db, err := getConn()
	if err != nil {
		return nil, err
	}

	var data NodeConfig
	result := db.First(&data, "Node_id = ?", NodeID)
	if result.Error != nil {
		log.Warnf("GetNodeConfig Node_id=%v, err:%v", NodeID, result.Error)
		return nil, result.Error
	}

	return &data, nil
}

func GetNodeConfigList(NodeID []string) ([]*NodeConfig, error) {
	db, err := getConn()
	if err != nil {
		return nil, err
	}

	var data []*NodeConfig
	result := db.Find(&data, "Node_id IN ?", NodeID)
	if result.Error != nil {
		log.Warnf("GetNodeConfigs Node_id=%v, err:%v", NodeID, result.Error)
		return nil, result.Error
	}

	return data, nil
}

func GetNodeConfigByUUID(uuid string) (*NodeConfig, error) {
	db, err := getConn()
	if err != nil {
		return nil, err
	}

	var data NodeConfig
	result := db.First(&data, "uuid = ?", uuid)
	if result.Error != nil {
		log.Warnf("GetNodeConfigs uuid=%v, err:%v", uuid, result.Error)
		return nil, result.Error
	}

	return &data, nil
}

func UpdateNodeConfig(data *NodeConfig) error {
	log.Debugf("UpdateNodeConfigs data=%+v ",data)
	db, err := getConn()
	if err != nil {
		return err
	}

	result := db.Save(data)
	if result.Error != nil {
		log.Warnf("UpdateNodeConfigs data=%+v err=%v", data, result.Error)
		return result.Error
	}

	return nil
}

func NodeConfigList() ([]*NodeConfig, error) {
	db, err := getConn()
	if err != nil {
		return nil, err
	}

	var data []*NodeConfig
	if r := db.Find(&data); r.Error != nil {
		log.Errorf("db query Node configs: %v", r.Error)
		return nil, r.Error
	}

	return data, nil
}

func RemoveNodeConfig(NodeID []string) error {
	db, err := getConn()
	if err != nil {
		return err
	}

	result := db.Where("Node_id IN ?", NodeID).Delete(NodeConfig{})
	if result.Error != nil {
		log.Errorf("db remove Node configs Node_id=%v: %v", NodeID, result.Error)
		return result.Error
	}

	return nil
}

func RemoveNodeConfigByID(id int64) error {
	db, err := getConn()
	if err != nil {
		return err
	}

	result := db.Where("id = ?", id).Delete(NodeConfig{})
	if result.Error != nil {
		log.Errorf("db remove Node configs id=%v, err=%v", id, result.Error)
		return result.Error
	}

	return nil
}

func RemoveNodeConfigByUUID(uuid string) error {
	db, err := getConn()
	if err != nil {
		return err
	}

	result := db.Where("uuid = ?", uuid).Delete(NodeConfig{})
	if result.Error != nil {
		log.Errorf("db remove Node uuid=%v: %v", uuid, result.Error)
		return result.Error
	}

	return nil
}