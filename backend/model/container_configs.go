package model

import (
	log "github.com/sirupsen/logrus"
)

type ContainerConfigs struct {
	ID             int64 `gorm:"primaryKey"`
	NodeID         int64
	UUID           string
	ContainerID    string
	ContainerName  string
	SecurityConfig string
	CreatedAt      int64 `gorm:"autoCreateTime"`
	UpdatedAt      int64 `gorm:"autoUpdateTime"`
}

type SecurityConfigs struct {
	DisableExternalNetwork bool             `json:"disable_external_network,omitempty"` // 禁止访问外部网络
	DisableCmdOperation    bool             `json:"disable_cmd_operation,omitempty"`    // 禁止命令行控制容器(启停控制)
	ProcProtection         *ProcProtection  `json:"proc_protection,omitempty"`          // 进程保护
	NprocProtection        *ProcProtection  `json:"nproc_protection,omitempty"`         // 网络进程保护
	FileProtection         *FileProtection  `json:"file_protection,omitempty"`          // 文件防篡改保护
	NetworkRule            *NetworkRuleList `json:"network_rule,omitempty"`             // 网络访问规则
}

type ProcProtection struct {
	Type    int32    `json:"type,omitempty"`  // PROC_PROTECTION 必填
	IsOn    bool     `json:"is_on,omitempty"` // 开关
	ExeList []string `json:"exe_list,omitempty"`
}
type FileProtection struct {
	IsOn     bool     `json:"is_on,omitempty"`
	FileList []string `json:"file_list,omitempty"`
}
type NetworkRuleList struct {
	IsOn  bool           `json:"is_on,omitempty"` // 0:关闭 1:白名单
	Rules []*NetworkRule `json:"rules,omitempty"`
}

type NetworkRule struct {
	Protocols []string `json:"protocols,omitempty"` // tcp, udp, icmp
	Addr      string   `json:"addr,omitempty"`      // e.g. 192.168.10.3 192.168.120.0/24
	Port      uint32   `json:"port,omitempty"`      // 0 for all ports
}

func CreateContainerConfigs(data *ContainerConfigs) error {
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
		return translateError(result.Error)
	}

	return nil
}

func GetContainerConfigs(nodeID int64, containerID string) (*ContainerConfigs, error) {
	db, err := getConn()
	if err != nil {
		return nil, err
	}

	var data ContainerConfigs
	result := db.First(&data, "node_id = ? AND container_id = ?", nodeID, containerID)
	if result.Error != nil {
		log.Warnf("GetContainerConfigs node_id=%v container_id=%v, err:%v", nodeID, containerID, result.Error)
		return nil, translateError(result.Error)
	}

	return &data, nil
}

func GetContainerConfigsList(nodeID int64, containerID []string) ([]*ContainerConfigs, error) {
	db, err := getConn()
	if err != nil {
		return nil, err
	}

	var data []*ContainerConfigs
	result := db.Find(&data, "node_id = ? AND container_id IN ?", nodeID, containerID)
	if result.Error != nil {
		log.Warnf("GetContainerConfigs node_id=%v container_id=%v, err:%v", nodeID, containerID, result.Error)
		return nil, translateError(result.Error)
	}

	return data, nil
}

func GetContainerConfigsByUUID(uuid string) (*ContainerConfigs, error) {
	db, err := getConn()
	if err != nil {
		return nil, err
	}

	var data ContainerConfigs
	result := db.First(&data, "uuid = ?", uuid)
	if result.Error != nil {
		log.Warnf("GetContainerConfigs uuid=%v, err:%v", uuid, result.Error)
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

func ListContainerConfigs() ([]ContainerConfigs, error) {
	db, err := getConn()
	if err != nil {
		return nil, err
	}

	var datas []ContainerConfigs
	result := db.Find(&datas)
	if result.Error != nil {
		log.Errorf("db query container configs: %v", result.Error)
		return nil, result.Error
	}

	return datas, nil
}

func RemoveContainerConfigs(nodeID int64, containerID []string) error {
	db, err := getConn()
	if err != nil {
		return err
	}

	result := db.Where("node_id = ? AND container_id IN ?", nodeID, containerID).Delete(ContainerConfigs{})
	if result.Error != nil {
		log.Errorf("db remove container configs node_id=%v, container_id=%v: %v", nodeID, containerID, result.Error)
		return result.Error
	}

	return nil
}

func RemoveContainerConfigsByID(id int64) error {
	db, err := getConn()
	if err != nil {
		return err
	}

	result := db.Where("id = ?", id).Delete(ContainerConfigs{})
	if result.Error != nil {
		log.Errorf("db remove container configs id=%v, err=%v", id, result.Error)
		return result.Error
	}

	return nil
}

func RemoveContainerConfigsByUUID(uuid string) error {
	db, err := getConn()
	if err != nil {
		return err
	}

	result := db.Where("uuid = ?", uuid).Delete(ContainerConfigs{})
	if result.Error != nil {
		log.Errorf("db remove container uuid=%v: %v", uuid, result.Error)
		return result.Error
	}

	return nil
}
