// 存储限制容器磁盘读写速率的相关数据   
package litedb

import (
	log "github.com/sirupsen/logrus"
)

type ContainerLimit struct {
        ID      int64   `gorm:"primaryKey;not null"`
        ContainerID     string `gorm:"unique"`
        ContainerName   string `gorm:"unique"`
	DiskRbw		int64
	DiskWbw		int64
        CreatedAt       int64   `gorm:"autoCreateTime"`
        UpdateAt        int64   `gorm:"autoUpdateTime"`
}

func CreateContainerLimit(data *ContainerLimit) error {
	if data == nil {
		return nil
	}
	db, err :=getConn()
	if err != nil {
		return err
	}
	result := db.Create(data)
	if result.Error != nil {
		log.Warnf("db create container limit %v", result.Error)
		return result.Error
	}
	return nil
}

// 通过容器 ID 返回结构体
func GetContainerLimitByCID(containerID string) (*ContainerLimit, error) {
	db, err := getConn()
	if err != nil {
		return nil, err
	}
	var data ContainerLimit
	result := db.First(&data, "container_id = ?", containerID)
	if result.Error != nil {
		return nil, result.Error
	}
	return &data, nil
}

// 更新一条记录
func UpdateContainerLimit(data *ContainerLimit) error {
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

// 通过 id 删除记录
func ContainerLimitRemoveByID (id int64) error {
	db, err := getConn()
	if err != nil {
		return err
	}
	result := db.Where("id = ?", id).Delete(ContainerLimit{})
	if result.Error != nil {
		return result.Error
	}
	return nil
}

// 通过容器ID 删除记录
func ContainerLimitRemoveByCID(cid []string) error {
	db, err := getConn()
	if err != nil {
		return err
	}
	result := db.Where("container_id IN (?)", cid).Delete(ContainerLimit{})
	if result.Error != nil {
		return result.Error
	}
	return nil
}

// 通过容器 ID[] 返回记录
func ContainerLimitByCID(cid []string) ([]*ContainerLimit, error) {
	db, err := getConn()
	if err != nil {
		return nil, err
	}
	var data []*ContainerLimit
	result := db.Find(&data, "container_id IN (?)", cid)
	if result.Error != nil {
		return nil, result.Error
	}
	return data, nil
}
