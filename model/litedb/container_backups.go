package litedb

import log "github.com/sirupsen/logrus"

type ContainerBackup struct {
	ID          int64 `gorm:"primaryKey;not null"`
	UUID        string
	ContainerID string
	BackupName  string `gorm:"uniqueIndex"`
	BackupDesc  string
	ImageRef    string
	ImageID     string
	ImageSize   int64
	Status      int8  // 0:备份中 1:成功 2:失败
	CreatedAt   int64 `gorm:"autoCreateTime"`
	UpdatedAt   int64 `gorm:"autoUpdateTime"`
}

func CreateContainerBackup(uuid string, backupName, backupDesc string) (*ContainerBackup, error) {
	db, err := getConn()
	if err != nil {
		return nil, err
	}

	data := ContainerBackup{
		UUID:       uuid,
		BackupName: backupName,
		BackupDesc: backupDesc,
	}

	if err := db.Create(&data).Error; err != nil {
		log.Errorf("db create container backup %v", err)
		return nil, err
	}

	return &data, nil
}

func UpdateContainerBackup(data interface{}) error {
	db, err := getConn()
	if err != nil {
		return err
	}

	if err := db.Save(data).Error; err != nil {
		log.Errorf("db update container backup %v", err)
		return err
	}

	return nil
}

func RemoveContainerBackup(data interface{}) error {
	db, err := getConn()
	if err != nil {
		return err
	}

	if err := db.Delete(data).Error; err != nil {
		log.Errorf("db remove container backup %v", err)
		return err
	}

	return nil
}

func RemoveContainerBackupByUUID(uuid []string) error {
	db, err := getConn()
	if err != nil {
		return err
	}

	result := db.Where("uuid IN ?", uuid).Delete(ContainerBackup{})
	if result.Error != nil {
		log.Errorf("db remove container backup, uuid=%v: %v", uuid, result.Error)
		return result.Error
	}

	return nil
}

func QueryContainerBackupByID(id int64) (*ContainerBackup, error) {
	db, err := getConn()
	if err != nil {
		return nil, err
	}

	var data ContainerBackup
	if err := db.First(&data, "id = ?", id).Error; err != nil {
		log.Warnf("QueryContainerBackupByID err=%v", err)
		return nil, err
	}

	return &data, nil
}

func QueryContainerBackupByUUID(uuid string) ([]*ContainerBackup, error) {
	db, err := getConn()
	if err != nil {
		return nil, err
	}

	var data []*ContainerBackup
	if err := db.Find(&data, "uuid = ?", uuid).Error; err != nil {
		log.Warnf("QueryContainerBackupByUUID err=%v", err)
		return nil, err
	}

	return data, nil
}

func QueryContainerBackupByContainer(containerIDs []string) ([]*ContainerBackup, error) {
	db, err := getConn()
	if err != nil {
		return nil, err
	}

	var data []*ContainerBackup
	if err := db.Find(&data, "container_id IN ?", containerIDs).Error; err != nil {
		log.Warnf("QueryContainerBackupByContainer err=%v", err)
		return nil, err
	}

	return data, nil
}

func ListContainerBackup() ([]*ContainerBackup, error) {
	db, err := getConn()
	if err != nil {
		return nil, err
	}

	var data []*ContainerBackup
	if err := db.Find(&data).Error; err != nil {
		log.Warnf("ListContainerBackup err=%v", err)
		return nil, err
	}

	return data, nil
}
