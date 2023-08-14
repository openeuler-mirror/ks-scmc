package model

import (
	"time"

	log "github.com/sirupsen/logrus"
)

type UserInfo struct {
	ID         int64 `gorm:"primaryKey"`
	Username   string
	PasswordEn string
	Role       string
	CreatedAt  int64 `gorm:"autoCreateTime"`
	UpdatedAt  int64 `gorm:"autoUpdateTime"`
}

func (UserInfo) TableName() string {
	return "user_infos"
}

type UserSession struct {
	ID         int64
	UserID     int64
	SessionKey string // UUID
	Source     string
	ExpiredAt  int64
	CreatedAt  int64 `gorm:"autoCreateTime"`
	UpdatedAt  int64 `gorm:"autoUpdateTime"`
}

func CreateUser(username, password, role string) error {
	db, err := getConn()
	if err != nil {
		return err
	}

	userInfo := UserInfo{Username: username, PasswordEn: password, Role: role}
	result := db.Create(&userInfo)

	if result.Error != nil {
		log.Warnf("db create user: %v", result.Error)
		return result.Error
	}

	log.Debugf("db create user id=%v", userInfo.ID)
	return nil
}

func QueryUser(username string) (*UserInfo, error) {
	db, err := getConn()
	if err != nil {
		return nil, err
	}

	var userInfo UserInfo
	result := db.Where("username = ?", username).First(&userInfo)
	if result.Error != nil {
		log.Warnf("db query user: %v", result.Error)
		return nil, result.Error
	}

	log.Debugf("db query user: RowsAffected=%v", result.RowsAffected)
	if result.RowsAffected == 0 {
		return nil, nil
	}
	return &userInfo, nil
}

func CreateSession(userID int64, sessionKey, source string) error {
	db, err := getConn()
	if err != nil {
		return err
	}

	sess := UserSession{
		UserID:     userID,
		SessionKey: sessionKey,
		Source:     source,
		ExpiredAt:  time.Now().Add(time.Minute * 20).Unix(),
	}

	// TODO create or update
	result := db.Create(&sess)

	if result.Error != nil {
		log.Warnf("db create session: %v", result.Error)
		return result.Error
	}

	log.Debugf("db create session %+v", sess)
	return nil
}

/*
func UpdatePassword(username, oldPass, newPass string) {
}

func UpdatePermission() {
}


func QuerySession(userID int64) (*UserSession, error) {
}
*/
