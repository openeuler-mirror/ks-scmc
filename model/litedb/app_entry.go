package litedb

import log "github.com/sirupsen/logrus"

type AppEntry struct {
	ID          int64  `gorm:"primaryKey;not null"`
	ContainerID string `gorm:"uniqueIndex:idx_container_exe;"`
	ExePath     string `gorm:"uniqueIndex:idx_container_exe;"`
	Name        string
	IsGUI       bool   `gorm:"column:is_gui"`
	IDResponse  string `gorm:"column:id_response"`
	CreatedAt   int64  `gorm:"autoCreateTime"`
	UpdatedAt   int64  `gorm:"autoUpdateTime"`
}

func AddAppEntry(e *AppEntry) error {
	if e == nil {
		return nil
	}

	db, err := getConn()
	if err != nil {
		return err
	}
	if err := db.Create(&e).Error; err != nil {
		log.Errorf("db create container configs %v", err)
		return err
	}
	return nil
}

func RemoveAppEntry(ids []int64) error {
	if len(ids) == 0 {
		return nil
	}

	db, err := getConn()
	if err != nil {
		return err
	}
	if err := db.Where("id IN ?", ids).Delete(AppEntry{}).Error; err != nil {
		log.Errorf("db remove app entries ids=%v err=%v", ids, err)
		return err
	}
	return nil
}

func UpdateAppEntry(e *AppEntry) error {
	if e == nil {
		return nil
	}

	db, err := getConn()
	if err != nil {
		return err
	}
	if err := db.Save(e).Error; err != nil {
		log.Warnf("UpdateAppEntrys data=%+v err=%v", e, err)
		return err
	}
	return nil
}

func ListAppEntry(containerID string) ([]*AppEntry, error) {
	if containerID == "" {
		return nil, nil
	}

	db, err := getConn()
	if err != nil {
		return nil, err
	}
	var data []*AppEntry
	if err := db.Where("container_id = ?", containerID).Find(&data).Error; err != nil {
		log.Errorf("db query app entries: %v", err)
		return nil, err
	}
	return data, nil
}

func GetAppEntryByID(id int64) (*AppEntry, error) {
	if id == 0 {
		return nil, nil
	}

	db, err := getConn()
	if err != nil {
		return nil, err
	}
	var data *AppEntry
	if err := db.Where("id = ?", id).First(&data).Error; err != nil {
		log.Errorf("db query app entry: %v", err)
		return nil, err
	}
	return data, nil
}
