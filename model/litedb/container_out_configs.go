/*
 * 保存一些 外部的配置信息
 */
package litedb

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"database/sql/driver"
	"encoding/json"
	"errors"

	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

// KyConfig 定义需要存储为JSON的配置结构
type KyConfig struct {
        DiskRbw         int64   `json:"disk_rbw,omitempty"`
        DiskWbw         int64   `json:"disk_wbw,omitempty"`
        AppRun          bool    `json:"app_run,omitempty"`
        IpcName         string  `json:"ipc_name,omitempty"`
        IpcID           string  `json:"ipc_id,omitempty"`
	// 可以根据需要添加更多字段
}

// 实现GORM的Scanner接口，用于从数据库读取时反序列化JSON
func (k *KyConfig) Scan(value interface{}) error {
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("类型断言失败，无法将value转换为[]byte")
	}
	return json.Unmarshal(bytes, k)
}

// 实现driver.Valuer接口，用于向数据库存储时序列化JSON
func (k KyConfig) Value() (driver.Value, error) {
	return json.Marshal(k)
}

// 实现GORM的SchemaInterface接口，指定字段类型
func (KyConfig) GormDBDataType(db *gorm.DB, field *schema.Field) string {
	// 根据数据库类型返回合适的字段类型，SQLite中使用TEXT
	switch db.Dialector.Name() {
	case "sqlite", "sqlite3":
		return "TEXT"
	default:
		return "JSONB" // 其他数据库如PostgreSQL可以使用JSONB
	}
}

// ContainerOutConfig 包含需要存储的容器配置
type ContainerOutConfig struct {
	ID            int64      `gorm:"primaryKey;not null"`
	ContainerID   string     `gorm:"unique"`
	ContainerName string     `gorm:"unique"`
	Config        KyConfig   `gorm:"column:config"` // 存储JSON的字段
	CreatedAt     int64	 `gorm:"autoCreateTime"`
        UpdateAt      int64	 `gorm:"autoUpdateTime"`
}

// 可选：设置表名
func (ContainerOutConfig) TableName() string {
	return "container_out_configs"
}

// 插入新的容器配置
func CreateContainerOutConfig(data *ContainerOutConfig) error {
        if data == nil {
                return nil
        }
        db, err := getConn()
        if err != nil {
                return err
        }
        result := db.Create(data)
        if result.Error != nil {
                log.Warnf("db create ContainerOutConfig err: %v", result.Error)
        }
        return nil
}

// 根据容器ID查询配置
func GetContainerOutConfig(containerID string) (*ContainerOutConfig, error) {
        db, err := getConn()
        if err != nil {
                return nil, err
        }

	var config ContainerOutConfig
	result := db.Where("container_id = ?", containerID).First(&config)
	if result.Error != nil {
		return nil, result.Error
	}

	return &config, nil
}

func GetAllContainerOutConfig()([]ContainerOutConfig, error) {
        db, err := getConn()
        if err != nil {
                return nil, err
        }

	var configs []ContainerOutConfig
	result := db.Find(&configs)
	if result.Error != nil {
		return nil, result.Error
	}
	return configs, nil
}

// 根据 JSON 字段条件查询
func GetContainerOutConfigByJsonField(field string, value interface{}) ([]ContainerOutConfig, error) {
        db, err := getConn()
        if err != nil {
                return nil, err
        }

	var configs []ContainerOutConfig
	// 使用SQLite的json_extract函数查询JSON字段
	query := fmt.Sprintf("json_extract(config, '$.%s') = ?", field)
	result := db.Where(query, value).Find(&configs)
	if result.Error != nil {
		return nil, result.Error
	}
	return configs, nil
}

// 更新记录
func UpdateContainerOutConfig(config *ContainerOutConfig) error {
        db, err := getConn()
        if err != nil {
                return err
        }
	result := db.Save(config)
	if result.Error != nil {
		return result.Error
	}
	return nil
}

// 删除记录
func DeleteContainerOutConfigByIDs(containerIDs []string) error {
	if len(containerIDs) == 0 {
		return nil
	}

        db, err := getConn()
        if err != nil {
                return err
        }

	result := db.Where("container_id IN (?)", containerIDs).Delete(&ContainerOutConfig{})
	if result.Error != nil {
		return result.Error
	}

	return nil // result.RowsAffected
}
