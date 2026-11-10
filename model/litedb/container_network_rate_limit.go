package litedb

import (
	log "github.com/sirupsen/logrus"
)

// NetworkLimit 定义一个结构体来映射数据库表
type NetworkLimit struct {
	ID                     int64 `gorm:"primaryKey;not null"`
	ContainerID            string
	DockerInterfaceName    string
	ContainerInterfaceName string
	NetWbw                 float64 // 上行带宽(Mbps)
	NetRbw                 float64 // 下行带宽(Mbps)
	CreatedAt              int64   `gorm:"autoCreateTime"`
	UpdatedAt              int64   `gorm:"autoUpdateTime"`
}

func (NetworkLimit) TableName() string {
	return "container_network_limits"
}

func CreateNetworkLimit(data *NetworkLimit) error {
	if data == nil {
		return nil
	}
	db, err := getConn()
	if err != nil {
		return err
	}
	result := db.Create(data)
	if result.Error != nil {
		log.Warnf("db create networklimit err: %v", result.Error)
		return result.Error
	}
	return nil
}

func UpdateNetworkLimit(data *NetworkLimit) error {
	db, err := getConn()
	if err != nil {
		return err
	}
	result := db.Save(data)
	if result.Error != nil {
		return result.Error
	}
	return nil
}

func GetNetworkLimitByNic(containerID string, nic string) (*NetworkLimit, error) {
	db, err := getConn()
	if err != nil {
		return nil, err
	}

	var data NetworkLimit
	result := db.Where("container_id = ? AND docker_interface_name = ?", containerID, nic).First(&data)
	if result.Error != nil {
		return nil, result.Error
	}
	return &data, nil
}

func DelNetworkLimit(containerID string) error {
	db, err := getConn()
	if err != nil {
		return err
	}

	result := db.Where("container_id = ?", containerID).Delete(NetworkLimit{})
	if result.Error != nil {
		return result.Error
	}

	return nil
}

func DelNetworkLimitByDockNic(containerID, nic string) error {
	db, err := getConn()
	if err != nil {
		return err
	}

	result := db.Where("container_id = ? AND docker_interface_name = ?", containerID, nic).Delete(NetworkLimit{})
	if result.Error != nil {
		return result.Error
	}

	return nil
}

func ListNetworkLimit(containerID string) ([]*NetworkLimit, error) {
	if containerID == "" {
		return nil, nil
	}

	db, err := getConn()
	if err != nil {
		return nil, err
	}
	var data []*NetworkLimit
	if err := db.Where("container_id = ?", containerID).Find(&data).Error; err != nil {
		log.Errorf("db query network limit entries err: %v", err)
		return nil, err
	}
	return data, nil
}
