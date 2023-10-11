// images info
package model

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
	FileType       string
	CheckSum       string
	ImageId        string
	FilePath       string
	SignPath       string
	RejectReason   string
	ApprovalStatus int32
	VerifyStatus   int32
	CreatedAt      int64 `gorm:"autoCreateTime"`
	UpdatedAt      int64 `gorm:"autoUpdateTime"`
}

func (ImageInfo) TableName() string {
	return "image_infos"
}

func FindImage(ids []int64) ([]ImageInfo, error) {
	db, err := getConn()
	if err != nil {
		return nil, err
	}
	var images []ImageInfo
	result := db.First(&images, ids)
	if result.Error != nil {
		log.Errorf("db query images: %v", result.Error)
		return nil, result.Error
	}

	return images, nil
}

func ListImages() ([]ImageInfo, error) {
	db, err := getConn()
	if err != nil {
		return nil, err
	}

	var images []ImageInfo
	result := db.Find(&images)
	if result.Error != nil {
		log.Errorf("db query images: %v", result.Error)
		return nil, result.Error
	}

	return images, nil
}

func CreateImages(image ImageInfo) (int64, error) {
	db, err := getConn()
	if err != nil {
		return 0, err
	}

	result := db.Create(&image)
	if result.Error != nil {
		log.Errorf("db create image %v", result.Error)
		return 0, translateError(result.Error)
	}

	log.Debugf("db create image id=%v", image.ID)
	return image.ID, nil
}

func RemoveImages(ids []int64) error {
	db, err := getConn()
	if err != nil {
		return err
	}

	var images []ImageInfo
	result := db.Find(&images, ids)
	if result.Error != nil {
		log.Errorf("db remove image ids=%v: %v", ids, result.Error)
		return result.Error
	}

	result = db.Where("id IN ?", ids).Delete(ImageInfo{})
	if result.Error != nil {
		log.Errorf("db remove image ids=%v: %v", ids, result.Error)
		return result.Error
	}

	log.Debugf("db remove image ids=%v OK", ids)
	return nil
}

func UpdateImage(image *ImageInfo) error {
	db, err := getConn()
	if err != nil {
		return err
	}

	err = db.Save(image).Error
	if err != nil {
		log.Warnf("ApproveImage update %+v err=%v", image, err)
		return translateError(err)
	}

	return nil
}

func QueryImageByStatus() ([]*ImageInfo, error) {
	db, err := getConn()
	if err != nil {
		return nil, err
	}

	var imageInfo []*ImageInfo
	result := db.Where("approval_status = ?", ApprovalPass).Find(&imageInfo)

	if result.Error != nil {
		log.Warnf("query approved image failed: %v", result.Error)
		return nil, translateError(result.Error)
	}

	return imageInfo, nil
}

func QueryImageByID(id int64) (*ImageInfo, error) {
	db, err := getConn()
	if err != nil {
		return nil, err
	}

	var imageInfo ImageInfo
	result := db.First(&imageInfo, id)
	if result.Error != nil {
		log.Warnf("query image id=%v: %v", id, result.Error)
		return nil, translateError(result.Error)
	} else if result.RowsAffected == 0 {
		log.Warnf("query image id=%v not found", id)
		return nil, ErrRecordNotFound
	}

	return &imageInfo, nil
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

func GetRepositoriesImages() map[string][]string {
	hub, err := newRegistryClient()
	if err != nil {
		log.Warnf("connect to registry [%v][%v][%v], err=%v", registryUrl(), registryUsername(), registryPassword(), err)
		return nil
	}
	repositories, err := hub.Repositories()
	if err != nil {
		log.Warnf("get registry err=%v", err)
		return nil
	}
	repositoriesimagemap := make(map[string][]string)
	for _, repository := range repositories {
		tags, err := hub.Tags(repository)
		if err != nil {
			log.Warnf("tag %v ", err)
			return nil
		}
		repositoriesimagemap[repository] = tags
	}
	return repositoriesimagemap
}
