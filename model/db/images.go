// images info
package db

import (
	log "github.com/sirupsen/logrus"
)

const (
	ApprovalWait   = 0
	ApprovalReject = 1
	ApprovalPass   = 2
)

const (
	VerifyFail     = 0
	VerifyAbnormal = 1
	VerifyPass     = 2
)

type ImageInfo struct {
	ID             int64 `gorm:"primaryKey"`
	Name           string
	Version        string
	Description    string
	FileSize       int64
	CheckSum       string
	ImageId        string
	FilePath       string
	RejectReason   string
	ApprovalStatus int32
	VerifyStatus   int32
	Signature      []byte
	CreatedAt      int64 `gorm:"autoCreateTime"`
	UpdatedAt      int64 `gorm:"autoUpdateTime"`
}

func (ImageInfo) TableName() string {
	return "image_infos"
}

func FindImage(ids []int64) ([]*ImageInfo, error) {
	db, err := getConn()
	if err != nil {
		return nil, err
	}

	var data []*ImageInfo
	if err := db.First(&data, ids).Error; err != nil {
		log.Warnf("db query images: %v", err)
		return nil, err
	}

	return data, nil
}

func ListImages() ([]*ImageInfo, error) {
	db, err := getConn()
	if err != nil {
		return nil, err
	}

	var data []*ImageInfo
	if err := db.Find(&data).Error; err != nil {
		log.Warnf("db query images: %v", err)
		return nil, err
	}

	return data, nil
}

func CreateImage(i *ImageInfo) (int64, error) {
	db, err := getConn()
	if err != nil {
		return 0, err
	}

	if err := db.Create(i).Error; err != nil {
		log.Warnf("db create image %v", err)
		return 0, translateError(err)
	}

	log.Debugf("db create image id=%v", i.ID)
	return i.ID, nil
}

func RemoveImages(ids []int64) ([]*ImageInfo, error) {
	db, err := getConn()
	if err != nil {
		return nil, err
	}

	var data []*ImageInfo
	if err := db.Find(&data, ids).Error; err != nil {
		log.Warnf("RemoveImages query ids=%v err=%v", ids, err)
		return nil, err
	}

	if err := db.Where("id IN ?", ids).Delete(ImageInfo{}).Error; err != nil {
		log.Warnf("RemoveImages delete ids=%v err=%v", ids, err)
		return nil, err
	}

	return data, nil
}

func UpdateImage(image *ImageInfo) error {
	db, err := getConn()
	if err != nil {
		return err
	}

	if err := db.Save(image).Error; err != nil {
		log.Warnf("UpdateImage image=%+v err=%v", image, err)
		return translateError(err)
	}

	return nil
}

func QueryImageByStatus() ([]*ImageInfo, error) {
	db, err := getConn()
	if err != nil {
		return nil, err
	}

	var data []*ImageInfo
	err = db.Where("approval_status = ?", ApprovalPass).Find(&data).Error

	if err != nil {
		log.Warnf("query approved image failed: %v", err)
		return nil, translateError(err)
	}

	return data, nil
}

func QueryImageByID(id int64) (*ImageInfo, error) {
	db, err := getConn()
	if err != nil {
		return nil, err
	}

	var data ImageInfo
	result := db.First(&data, id)
	if result.Error != nil {
		log.Warnf("query image id=%v: %v", id, result.Error)
		return nil, translateError(result.Error)
	} else if result.RowsAffected == 0 {
		log.Infof("query image id=%v not found", id)
		return nil, nil
	}

	return &data, nil
}

func QueryImageByIDs(ids []int64) ([]*ImageInfo, error) {
	db, err := getConn()
	if err != nil {
		return nil, err
	}

	var data []*ImageInfo
	if err := db.Find(&data, "id IN ?", ids).Error; err != nil {
		log.Warnf("query image ids=%v: %v", ids, err)
		return nil, translateError(err)
	}

	return data, nil
}

func QueryImageByNameVersion(name, version string) (*ImageInfo, error) {
	db, err := getConn()
	if err != nil {
		return nil, err
	}

	var data ImageInfo
	result := db.First(&data, "name = ? AND version = ?", name, version)
	if result.Error != nil {
		if result.Error == ErrRecordNotFound {
			return nil, nil
		}

		log.Warnf("query imange name=%v version=%v err=%v", name, version, result.Error)
		return nil, result.Error
	} else if result.RowsAffected == 0 {
		return nil, nil
	}

	return &data, nil
}
