package litedb

import (
	log "github.com/sirupsen/logrus"

	pb "scmc/rpc/pb/container"
	"scmc/rpc/pb/security"
)

type ContainerConfig struct {
	ID             int64  `gorm:"primaryKey;not null"`
	UUID           string `gorm:"unique"`
	ContainerID    string `gorm:"unique"`
	ContainerName  string
	SecurityConfig *SecurityConfigs `gorm:"serializer:json"`
	CreatedAt      int64            `gorm:"autoCreateTime"`
	UpdatedAt      int64            `gorm:"autoUpdateTime"`
}

type SecurityConfigs = pb.SecurityConfig
type ProcProtection = security.ProcProtection
type NetworkRule = security.NetworkRule

func CreateContainerConfig(data *ContainerConfig) error {
	if data == nil {
		return nil
	}

	db, err := getConn()
	if err != nil {
		return err
	}

	result := db.Create(data)
	if result.Error != nil {
		log.Errorf("db create container configs %v", result.Error)
		return result.Error
	}

	return nil
}

func GetContainerConfig(containerID string) (*ContainerConfig, error) {
	db, err := getConn()
	if err != nil {
		return nil, err
	}

	var data ContainerConfig
	result := db.First(&data, "container_id = ?", containerID)
	if result.Error != nil {
		log.Warnf("GetContainerConfig container_id=%v, err:%v", containerID, result.Error)
		return nil, result.Error
	}

	return &data, nil
}

func GetContainerConfigList(containerID []string) ([]*ContainerConfig, error) {
	db, err := getConn()
	if err != nil {
		return nil, err
	}

	var data []*ContainerConfig
	result := db.Find(&data, "container_id IN ?", containerID)
	if result.Error != nil {
		log.Warnf("GetContainerConfigs container_id=%v, err:%v", containerID, result.Error)
		return nil, result.Error
	}

	return data, nil
}

func GetContainerConfigByUUID(uuid string) (*ContainerConfig, error) {
	db, err := getConn()
	if err != nil {
		return nil, err
	}

	var data ContainerConfig
	result := db.First(&data, "uuid = ?", uuid)
	if result.Error != nil {
		log.Warnf("GetContainerConfigs uuid=%v, err:%v", uuid, result.Error)
		return nil, result.Error
	}

	return &data, nil
}

func UpdateContainerConfig(data *ContainerConfig) error {
	db, err := getConn()
	if err != nil {
		return err
	}

	result := db.Save(data)
	if result.Error != nil {
		log.Warnf("UpdateContainerConfigs data=%+v err=%v", data, result.Error)
		return result.Error
	}

	return nil
}

func ContainerConfigList() ([]*ContainerConfig, error) {
	db, err := getConn()
	if err != nil {
		return nil, err
	}

	var data []*ContainerConfig
	if r := db.Find(&data); r.Error != nil {
		log.Errorf("db query container configs: %v", r.Error)
		return nil, r.Error
	}

	return data, nil
}

func RemoveContainerConfig(containerID []string) error {
	db, err := getConn()
	if err != nil {
		return err
	}

	result := db.Where("container_id IN ?", containerID).Delete(ContainerConfig{})
	if result.Error != nil {
		log.Errorf("db remove container configs container_id=%v: %v", containerID, result.Error)
		return result.Error
	}

	return nil
}

func RemoveContainerConfigByID(id int64) error {
	db, err := getConn()
	if err != nil {
		return err
	}

	result := db.Where("id = ?", id).Delete(ContainerConfig{})
	if result.Error != nil {
		log.Errorf("db remove container configs id=%v, err=%v", id, result.Error)
		return result.Error
	}

	return nil
}

func RemoveContainerConfigByUUID(uuid string) error {
	db, err := getConn()
	if err != nil {
		return err
	}

	result := db.Where("uuid = ?", uuid).Delete(ContainerConfig{})
	if result.Error != nil {
		log.Errorf("db remove container uuid=%v: %v", uuid, result.Error)
		return result.Error
	}

	return nil
}
