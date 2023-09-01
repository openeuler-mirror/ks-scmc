package model

import (
	"context"

	log "github.com/sirupsen/logrus"
)

type UserRole struct {
	ID         int64 ` gorm:"primaryKey"`
	Name       string
	IsEditable bool // 是否可更新删除
	PermsJson  string
	CreatedAt  int64 `gorm:"autoCreateTime"`
	UpdatedAt  int64 `gorm:"autoCreateTime"`
}

func QueryRoleByName(ctx context.Context, roleName string) (*UserRole, error) {
	db, err := getConn()
	if err != nil {
		return nil, err
	}

	var userRole UserRole
	result := db.WithContext(ctx).Where("name = ?", roleName).First(&userRole)
	if result.Error != nil {
		return nil, result.Error
	}

	log.Debugf("db query role: RowsAffected=%v", result.RowsAffected)
	if result.RowsAffected == 0 {
		return nil, nil
	}
	return &userRole, nil
}

func QueryRoleById(ctx context.Context, roleId int64) (*UserRole, error) {
	db, err := getConn()
	if err != nil {
		return nil, err
	}
	var userRole UserRole
	result := db.WithContext(ctx).Where("id = ?", roleId).First(&userRole)
	if result.Error != nil {
		return nil, result.Error
	}

	log.Debugf("db query role: RowsAffected=%v", result.RowsAffected)
	if result.RowsAffected == 0 {
		return nil, nil
	}
	return &userRole, nil
}

func ListRole() ([]*UserRole, error) {
	db, err := getConn()
	if err != nil {
		return nil, err
	}

	var roles []*UserRole
	result := db.Find(&roles)
	if result.Error != nil {
		log.Warnf("db get roles err=%v", err)
		return nil, translateError(result.Error)
	}

	return roles, nil
}

func CreateRole(ctx context.Context, userRole *UserRole) error {
	db, err := getConn()
	if err != nil {
		return err
	}

	if result := db.WithContext(ctx).Create(&userRole); result.Error != nil {
		return translateError(result.Error)
	}

	return nil
}

func UpdateRole(ctx context.Context, userRole *UserRole) error {
	db, err := getConn()
	if err != nil {
		return err
	}

	if result := db.WithContext(ctx).Save(userRole); result.Error != nil {
		return result.Error
	}

	return nil
}

func RemoveRole(ctx context.Context, roleId int64) error {
	db, err := getConn()
	if err != nil {
		return err
	}

	if result := db.WithContext(ctx).Delete(&UserRole{}, roleId); result.Error != nil {
		return result.Error
	}

	return nil
}
