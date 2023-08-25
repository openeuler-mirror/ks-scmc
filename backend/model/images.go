// images info
package model

import (
	"errors"
	"fmt"
	"os"

	log "github.com/sirupsen/logrus"

	"scmc/common"
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

func RemoveImageFile(images []ImageInfo) error {
	for _, image := range images {
		err := os.Remove(image.FilePath)
		signFileName := fmt.Sprintf("%s/%s_%s.sign", common.Config.Controller.ImageDir, image.Name, image.Version)
		errSign := os.Remove(signFileName)
		log.Debugf("db remove image file %v: %v, %v: %v", image.FilePath, err, signFileName, errSign)
	}

	return nil
}

func RemoveImage(ids []int64) error {
	db, err := getConn()
	if err != nil {
		return err
	}

	var images []ImageInfo
	db.Find(&images, ids)

	result := db.Where("id IN ?", ids).Delete(ImageInfo{})
	if result.Error != nil {
		log.Errorf("db remove image ids=%v: %v", ids, result.Error)
		return result.Error
	}

	RemoveImageFile(images)

	for _, info := range images {
		k := info.Name + ":" + info.Version
		RemoveRegistryImage(k)
	}

	log.Debugf("db remove image ids=%v OK", ids)
	return nil
}

func UpadteImage(image *ImageInfo) error {
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

func ApproveImage(id int64, approve bool, reason string) error {
	db, err := getConn()
	if err != nil {
		return err
	}
	var imageInfo ImageInfo
	result := db.First(&imageInfo, id)
	if result.Error != nil {
		log.Warnf("ApproveImage query image id=%v: %v", id, result.Error)
		return translateError(result.Error)
	}

	if approve && imageInfo.VerifyStatus != VerifyPass {
		log.Warnf("the signature does not pass")
		return errors.New("the signature does not pass")
	}

	if approve {
		imageInfo.ApprovalStatus = ApprovalPass
	} else {
		imageInfo.ApprovalStatus = ApprovalReject
	}

	imageInfo.RejectReason = reason

	err = db.Save(&imageInfo).Error
	if err != nil {
		log.Warnf("ApproveImage update %+v err=%v", imageInfo, err)
		return translateError(err)
	}

	chImage <- imageInfo
	return nil
}

func QueryImageByStatus() ([]ImageInfo, error) {
	db, err := getConn()
	if err != nil {
		return nil, err
	}

	var imageInfo []ImageInfo
	result := db.Where("approval_status = ?", ApprovalPass).Find(&imageInfo)

	if result.Error != nil {
		log.Warnf("query approved image failed: %v", result.Error)
		return nil, translateError(result.Error)
	} else if result.RowsAffected == 0 {
		log.Warnf("query approved image not found")
		return nil, ErrRecordNotFound
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
