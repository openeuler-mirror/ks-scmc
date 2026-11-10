package litedb

import (
	//"encoding/json"
	//"database/sql/driver"
	//"errors"
	log "github.com/sirupsen/logrus"
)

// CREATE TABLE `ipc_configs` (`id` integer NOT NULL,`container_id` text UNIQUE,`container_name` text UNIQUE,`ipc` text,`created_at` integer,`update_at` integer,PRIMARY KEY (`id`));

type IpcConfig struct {
	ID            int64  `gorm:"primaryKey;not null"`
	ContainerID   string `gorm:"unique"`
	ContainerName string `gorm:"unique"`
	// IPC     *IPCData        `gorm:"column:ipc;type:text"` // IPC 映射关系 (JSON 存储)
	CreatedAt int64 `gorm:"autoCreateTime"`
	UpdateAt  int64 `gorm:"autoUpdateTime"`
}

/*
type IPCData map[string] string

// 实现 gorm 序列化接口（可选，若使用默认JSON序列化可省略）
func (i IPCData) Value() (driver.Value, error) {
        return json.Marshal(i)
}

func (i *IPCData) Scan(value interface{}) error {
        b, ok := value.([]byte)
        if !ok {
                return errors.New("type assertion to []byte failed")
        }
        return json.Unmarshal(b, i)
}
*/

// 新建一条记录
func CreateIpcConfig(data *IpcConfig) error {
	if data == nil {
		return nil
	}

	db, err := getConn()
	if err != nil {
		return err
	}

	result := db.Create(data)
	if result.Error != nil {
		log.Warnf("db create ipc configs %v", result.Error)
		return result.Error
	}
	return nil
}

// 通过 IPC 容器ID 返回记录
func GetIpcConfigByID(containerID string) (*IpcConfig, error) {
	db, err := getConn()
	if err != nil {
		return nil, err
	}

	var data IpcConfig
	result := db.First(&data, "container_id = ?", containerID)
	if result.Error != nil {
		return nil, result.Error
	}

	return &data, nil
}

// 通过 IPC 容器名称 返回记录
func GetIpcConfigByName(name string) (*IpcConfig, error) {
	db, err := getConn()
	if err != nil {
		return nil, err
	}

	var data IpcConfig
	result := db.First(&data, "container_name = ?", name)
	if result.Error != nil {
		return nil, result.Error
	}

	return &data, nil
}

// 更新一条记录
func UpdateIpcConfig(data *IpcConfig) error {
	db, err := getConn()
	if err != nil {
		return err
	}

	//result := db.Model(data).Select("IPC").Updates(data)
	result := db.Save(data)
	if result.Error != nil {
		return result.Error
	}
	return nil
}

// 返回所有的 IPC 记录信息
func IpcConfigList() ([]*IpcConfig, error) {
	db, err := getConn()
	if err != nil {
		return nil, err
	}

	var data []*IpcConfig
	if r := db.Find(&data); r.Error != nil {
		return nil, r.Error
	}

	return data, nil
}

// 依据 ID 删除记录
func IpcConfigRemoveByID(id int64) error {
	db, err := getConn()
	if err != nil {
		return err
	}

	result := db.Where("id = ?", id).Delete(IpcConfig{})
	if result.Error != nil {
		return result.Error
	}

	return nil
}
