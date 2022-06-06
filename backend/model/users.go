package model

import (
	"context"
	"time"

	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm/clause"
)

const sessionLivePeriod = time.Minute * 10

type UserInfo struct {
	ID         int64 `gorm:"primaryKey"`
	Username   string
	RealName   string
	PasswordEn string
	IsActive   bool
	IsEditable bool
	RoleID     int64
	CreatedAt  int64 `gorm:"autoCreateTime"`
	UpdatedAt  int64 `gorm:"autoUpdateTime"`
}

func (UserInfo) TableName() string {
	return "user_infos"
}

func (u *UserInfo) CheckPassword(input string) error {
	return bcrypt.CompareHashAndPassword([]byte(u.PasswordEn), []byte(input))
}

func (u *UserInfo) UpdatePassword(newPassword string) error {
	db, err := getConn()
	if err != nil {
		return err
	}

	u.PasswordEn = newPassword
	result := db.Save(u)
	return result.Error
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

type UserRoleInfo struct {
	UserInfo
	PermsJSON string
}

func (UserRoleInfo) TableName() string {
	return "user_infos"
}

func CreateUser(ctx context.Context, userInfo *UserInfo) error {
	db, err := getConn()
	if err != nil {
		return err
	}

	if result := db.WithContext(ctx).Create(&userInfo); result.Error != nil {
		return translateError(result.Error)
	}

	return nil
}

func UpdateUser(ctx context.Context, userInfo *UserInfo) error {
	db, err := getConn()
	if err != nil {
		return err
	}

	if result := db.WithContext(ctx).Save(userInfo); result.Error != nil {
		return result.Error
	}

	return nil
}

func RemoveUser(ctx context.Context, userId int64) error {
	db, err := getConn()
	if err != nil {
		return err
	}

	if result := db.WithContext(ctx).Delete(&UserInfo{}, userId); result.Error != nil {
		return result.Error
	}

	return nil
}

func RemoveUsers(users []*UserInfo) error {
	db, err := getConn()
	if err != nil {
		return err
	}

	result := db.Delete(&users)
	if result.Error != nil {
		log.Warnf("db delete users err=%v", err)
		return result.Error
	}

	return nil
}

func ListUser() ([]*UserInfo, error) {
	db, err := getConn()
	if err != nil {
		return nil, err
	}

	var users []*UserInfo
	result := db.Find(&users)
	if result.Error != nil {
		log.Warnf("db get users err=%v", err)
		return nil, translateError(result.Error)
	}

	return users, nil
}

func QueryUser(ctx context.Context, username string) (*UserInfo, error) {
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

func QueryUserByID(id interface{}) (*UserInfo, error) {
	db, err := getConn()
	if err != nil {
		return nil, err
	}

	var userInfo UserInfo
	result := db.First(&userInfo, id)
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

func QueryUsers(ids []int64) ([]*UserInfo, error) {
	db, err := getConn()
	if err != nil {
		return nil, err
	}

	var users []*UserInfo
	result := db.Find(&users, "id IN ?", ids)
	if result.Error != nil {
		log.Warnf("db query user: %v", result.Error)
		return nil, result.Error
	}

	return users, nil
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
		ExpiredAt:  time.Now().Add(sessionLivePeriod).Unix(),
	}

	result := db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"session_key", "source", "expired_at"}),
	}).Create(&sess)

	if result.Error != nil {
		log.Warnf("db create session: %v", result.Error)
		return result.Error
	}

	log.Debugf("db create session %+v", sess)
	return nil
}

func CheckUserSession(userID, sessionKey string) (bool, error) {
	db, err := getConn()
	if err != nil {
		return false, err
	}

	var (
		now      = time.Now()
		sessList []UserSession
	)
	result := db.Where("user_id = ? AND session_key = ? AND expired_at > ?", userID, sessionKey, now.Unix()).Find(&sessList)

	if result.Error != nil {
		log.Warnf("CheckUserSession user_id=%v error=%v", userID, result.Error)
		return false, result.Error
	} else if len(sessList) != 1 {
		log.Infof("CheckUserSession user_id=%v session=%s affected rows=%d", userID, sessionKey, result.RowsAffected)
		return false, nil
	}

	sess := sessList[0]
	if now.Unix() > sess.UpdatedAt {
		sess.ExpiredAt = now.Add(sessionLivePeriod).Unix()
		sess.UpdatedAt = now.Unix()
		if r := db.Save(&sess); r.Error != nil {
			log.Infof("CheckUserSession update session for user_id=%v error=%v", userID, r.Error)
		}
	}

	return true, nil
}

func RemoveSession(userID, sessionKey string) error {
	db, err := getConn()
	if err != nil {
		return err
	}

	now := time.Now().Unix()
	result := db.Model(UserSession{}).Where("user_id = ?", userID).
		Updates(map[string]interface{}{"session_key": "", "source": "", "expired_at": 0, "updated_at": now})

	if result.Error != nil {
		log.Warnf("RemoveSession user_id=%v error=%v", userID, result.Error)
		return result.Error
	}

	if result.RowsAffected < 1 {
		log.Infof("RemoveSession user_id=%v session=%s affected rows=%d", userID, sessionKey, result.RowsAffected)
		return ErrRecordNotFound
	}

	return nil
}

func QueryUserRole(userID interface{}) (*UserRoleInfo, error) {
	db, err := getConn()
	if err != nil {
		return nil, err
	}

	var r UserRoleInfo
	db.Model(&UserRoleInfo{}).Joins("LEFT JOIN user_roles ON user_infos.role_id = user_roles.id").
		Select("user_infos.*, user_roles.perms_json").First(&r, "user_infos.id = ?", userID)

	return &r, nil
}
