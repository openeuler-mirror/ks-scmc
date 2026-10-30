package db

import (
	log "github.com/sirupsen/logrus"
)

type ContainerLimits struct {
	ID          int64 `gorm:"primaryKey"`
	NodeId      int64
	ContainerId string
	UploadSpeed float64
	DownSpeed   float64
	CreatedAt   int64 `gorm:"autoCreateTime"`
	UpdatedAt   int64 `gorm:"autoUpdateTime"`
}

func (ContainerLimits) TableName() string {
	return "container_limits"
}
func CreateContainerLimits(data *ContainerHosts) error {
	db, err := getConn()
	if err != nil {
		return err
	}

	result := db.Create(&data)
	if result.Error != nil {
		log.Errorf("db create container limits %v", result.Error)
		return translateError(result.Error)
	}
	return nil
}
func UpdateContainerLimits(n *ContainerLimits) error {
	db, err := getConn()
	if err != nil {
		return err
	}

	result := db.Save(n)
	if result.Error != nil {
		log.Warnf("save container limits info=%+v err=%v", n, result.Error)
		return translateError(result.Error)
	}
	return nil
}
func QueryContainerLimitsByContainerId(containerId string, nodeId int64) (*ContainerLimits, error) {
	db, err := getConn()
	if err != nil {
		return nil, err
	}

	var containerLimits ContainerLimits
	result := db.Where("container_id=?", containerId).First(&containerLimits, "node_id=? AND container_id=?", nodeId, containerId)
	if result.Error != nil {
		log.Warnf("query container limits container_id=%v: %v", containerId, err)
		return nil, translateError(result.Error)
	} else if result.RowsAffected == 0 {
		log.Warnf("query container limits container_id=%v not found", containerId)
		return nil, ErrRecordNotFound
	}
	return &containerLimits, nil
}
