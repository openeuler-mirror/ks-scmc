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

type ImageStat struct {
	ToApproveCount int64 // 待审批镜像数
	TotalCount     int64 // 镜像个数
	TotalSize      int64 // 镜像总大小 单位字节
}

type ImageInfo struct {
	ID          int64 `gorm:"primaryKey"`
	Name        string
	Version     string
	Description string
	FileSize    int64
	CheckSum    string
	ImageId     string
	FilePath    string
	// BinaryHashes 镜像签名 metadata 中携带的关键二进制文件 hash（JSON，path->md5Hex）。
	// 为空表示镜像未携带该信息或未启用该能力。
	BinaryHashes   string `gorm:"type:longtext"`
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
