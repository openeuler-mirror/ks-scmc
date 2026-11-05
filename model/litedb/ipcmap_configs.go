package litedb

import (
	log "github.com/sirupsen/logrus"
)

/* 记录容器 与 IPC 容器的映射关系 
 * 容器关联到那个 IPC 容器上去了
 */

type IpcmapConfig struct {
        ID      int64   `gorm:"primaryKey;not null"`
        ContainerName   string `gorm:"unique"`
        ContainerID     string `gorm:"unique"`
	IpcName		string
	IpcID		string
        CreatedAt       int64   `gorm:"autoCreateTime"`
        UpdateAt        int64   `gorm:"autoUpdateTime"`
}


func CreateIpcmapConfig(data *IpcmapConfig) error {
	if data == nil {
		return nil
	}
	db, err := getConn()
	if err != nil {
		return err
	}
	result := db.Create(data)
	if result.Error != nil {
		log.Warnf("db create ipcmapconfig err: %v", result.Error)
		return result.Error
	}
	return nil
}

// 通过容器 ID 返回记录
func GetIpcmapConfigByCID(containerID string) (*IpcmapConfig, error) {
	db, err := getConn()
	if err != nil {
		return nil, err
	}

	var data IpcmapConfig
	result := db.First(&data, "container_id = ?", containerID)
	if result.Error != nil {
		return nil, result.Error
	}
	return &data, nil
}

// 通过容器名称 返回记录
func GetIpcmapConfigByName(name string) (*IpcmapConfig, error) {
	db, err := getConn()
	if err != nil {
		return nil, err
	}

	var data IpcmapConfig
	result := db.First(&data, "container_name = ?", name)
	if result.Error != nil {
		return nil, result.Error
	}
	return &data, nil
}

// 更新一条记录
func UpdateIpcmanConfig(data *IpcmapConfig) error {
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

// 通过ID 删除记录
func IpcmapConfigRemoveByID(id int64) error {
	db, err := getConn()
	if err != nil {
		return err
	}
	result := db.Where("id = ?", id).Delete(IpcmapConfig{})
	if result.Error != nil {
		return result.Error
	}

	return nil
}

// 通过容器ID删除记录
func IpcmapConfigRemoveByCID(cid []string) error {
	db, err := getConn()
	if err != nil {
		return err
	}
	result := db.Where("container_id IN (?)", cid).Delete(IpcmapConfig{})
	if result.Error != nil {
		return result.Error
	}

	return nil
}

// 通过 IPC 名称 返回记录
func IpcmapConfigByIPCName(name []string) ([]*IpcmapConfig ,error) {
	db, err := getConn()
	if err != nil {
		return nil, err
	}

	var data []*IpcmapConfig
	result := db.Find(&data, "ipc_name IN (?)", name)
	if result.Error != nil {
		return nil, result.Error
	}
	return data, nil
}
