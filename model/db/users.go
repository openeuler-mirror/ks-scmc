package db

import (
	"context"
	"errors"
	"time"

	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	// "gorm.io/gorm/clause"

	"scmc/common"
)

type UserInfo struct {
	ID         int64 `gorm:"primaryKey"`
	Username   string
	RealName   string
	PasswordEn string
	IsActive   bool
	IsEditable bool
	RoleID     int64
	FailCnt    int32
	LockUntil  int64
	CreatedAt  int64 `gorm:"autoCreateTime"`
	UpdatedAt  int64 `gorm:"autoUpdateTime"`
}

