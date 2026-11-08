// 存储容器配置数据   
package litedb

import (
    log "github.com/sirupsen/logrus"
	pb "scmc/rpc/pb/container"
)

type ContainerMisc struct {
    ID              int64   `gorm:"primaryKey;not null"`
    UUID            string `gorm:"unique"`
    ContainerID     string `gorm:"unique"`
    ContainerName   string 
    MiscConfig      *ContainerMiscConfig `gorm:"serializer:json"`
    CreatedAt       int64   `gorm:"autoCreateTime"`
    UpdateAt        int64   `gorm:"autoUpdateTime"`
}

type ContainerMiscConfig = pb.ContainerMiscConfig

func CreateContainerMiscConfig(data *ContainerMisc) error {
    if data == nil {
        return nil
    }
    db, err := getConn()
    if err != nil {
        return err
    }
    result := db.Create(data)
    if result.Error != nil {
        return result.Error
        log.Warnf("db create container misc %v", result.Error)
    }
    return nil
}

func GetContainerMiscConfig(containerID string) (*ContainerMisc, error) {
    db, err := getConn()
    if err != nil {
        return nil, err
    }

    var data ContainerMisc
    result := db.First(&data, "container_id = ?", containerID)
    if result.Error != nil {
        return nil, result.Error
    }
    return &data, nil
}

func GetContainerMiscConfigList(containerID []string)([]*ContainerMisc, error) {
    db, err := getConn()
    if err != nil {
        return nil, err
    }

    var data []*ContainerMisc
    result := db.Find(&data, "container_id IN ?", containerID)
    if result.Error != nil {
        return nil, result.Error
    }
    return data, nil
}

func UpdateContainerMiscConfig(data *ContainerMisc) error {
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

func ContainerMiscList()([] *ContainerMisc, error) {
    db, err := getConn()
    if err != nil {
        return nil, err
    }
    
    var data []*ContainerMisc
    if r := db.Find(&data); r.Error != nil {
        return nil, r.Error
    }
    return data, nil
}

func RemoveContainerMisc(containerID []string) error {
    db, err := getConn()
    if err != nil {
        return err
    }
    
    result := db.Where("container_id IN ?", containerID).Delete(ContainerMisc{})
    if result.Error != nil {
        return result.Error
    }
    return nil
}

