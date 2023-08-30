package model

import log "github.com/sirupsen/logrus"

type ContainerConfigs struct {
	ID             int64 `gorm:"primaryKey"`
	NodeID         int64
	UUID           string
	ContainerID    string
	SecurityConfig string
	CreatedAt      int64 `gorm:"autoCreateTime"`
	UpdatedAt      int64 `gorm:"autoUpdateTime"`
}

func CreateContainerConfigs(nodeID int64, uuid, containerID, securityConfig string) (*ContainerConfigs, error) {
	db, err := getConn()
	if err != nil {
		return nil, err
	}

	data := ContainerConfigs{
		NodeID:         nodeID,
		UUID:           uuid,
		ContainerID:    containerID,
		SecurityConfig: securityConfig,
	}

	result := db.Create(&data)
	if result.Error != nil {
		log.Errorf("db create container configs %v", result.Error)
		return nil, translateError(result.Error)
	}

	return &data, nil
}

func GetContainerConfigs(nodeID, containerID string) (*ContainerConfigs, error) {
	db, err := getConn()
	if err != nil {
		return nil, err
	}

	var data ContainerConfigs
	result := db.First(&data, "node_id = ? AND container_id", nodeID, containerID)
	if result.Error != nil {
		log.Warnf("GetContainerConfigs node_id=%s container_id=%v", nodeID, containerID, result.Error)
		return nil, translateError(result.Error)
	}

	return &data, nil
}

func UpdateContainerConfigs(data *ContainerConfigs) error {
	db, err := getConn()
	if err != nil {
		return err
	}

	result := db.Save(data)
	if result.Error != nil {
		log.Warnf("UpdateContainerConfigs data=%+v err=%v", data, result.Error)
		return translateError(result.Error)
	}

	return nil
}
