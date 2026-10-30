package db

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	log "github.com/sirupsen/logrus"
)

type ContainerHosts struct {
	ID          int64 `gorm:"primaryKey"`
	NodeId      int64
	ContainerId string
	HostsConfig StringSlice `gorm:"type:json"`
	CreatedAt   int64       `gorm:"autoCreateTime"`
	UpdatedAt   int64       `gorm:"autoUpdateTime"`
}
type StringSlice []string

// 实现 driver.Valuer 接口（写入数据库时调用）
func (s StringSlice) Value() (driver.Value, error) {
	return json.Marshal(s)
}

// 实现 sql.Scanner 接口（从数据库读取时调用）
func (s *StringSlice) Scan(value interface{}) error {
	if value == nil {
		*s = nil
		return nil
	}
	b, ok := value.([]byte)
	if !ok {
		return errors.New("invalid type for StringSlice")
	}
	return json.Unmarshal(b, s)
}
func (ContainerHosts) TableName() string {
	return "container_hosts"
}
func CreateContainerHosts(data *ContainerHosts) error {
	db, err := getConn()
	if err != nil {
		return err
	}

	result := db.Create(&data)
	if result.Error != nil {
		log.Errorf("db create container hosts %v", result.Error)
		return translateError(result.Error)
	}

	log.Debugf("db create node id=%v", data.ID)
	return nil
}
func UpdateContainerHosts(n *ContainerHosts) error {
	db, err := getConn()
	if err != nil {
		return err
	}

	result := db.Save(n)
	if result.Error != nil {
		log.Warnf("save container hosts info=%+v err=%v", n, result.Error)
		return translateError(result.Error)
	}
	return nil
}
func QueryContainerHostsByContainerId(containerId string, nodeId int64) (*ContainerHosts, error) {
	db, err := getConn()
	if err != nil {
		return nil, err
	}

	var containerHosts ContainerHosts
	result := db.Where("node_id=?", nodeId).First(&containerHosts, containerId)
	if result.Error != nil {
		log.Warnf("query hosts container_id=%v: %v", containerId, err)
		return nil, translateError(result.Error)
	} else if result.RowsAffected == 0 {
		log.Warnf("query hosts container_id=%v not found", containerId)
		return nil, ErrRecordNotFound
	}
	return &containerHosts, nil
}
