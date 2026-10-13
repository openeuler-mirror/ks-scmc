package db

import (
	"sync"

	log "github.com/sirupsen/logrus"
)

type SecuritySwitch struct {
	ID        int64 `gorm:"primaryKey"`
	IsOn      bool
	CreatedAt int64 `gorm:"autoCreateTime"`
	UpdatedAt int64 `gorm:"autoUpdateTime"`
}

type SecuritySwitchCache struct {
	sync.Mutex

	V *SecuritySwitch
}

var (
	securitySwitchCache   SecuritySwitchCache
	DefaultSecuritySwitch = true
)

func SetSecuritySwitch(isOn bool) error {
	db, err := getConn()
	if err != nil {
		return err
	}

	if err := db.Where(map[string]interface{}{"id": 1}).Assign(map[string]interface{}{"is_on": isOn}).FirstOrCreate(&SecuritySwitch{}).Error; err != nil {
		log.Warnf("set security switch from db err=%v", err)
		return err
	}

	// invalidate cache
	securitySwitchCache.Lock()
	defer securitySwitchCache.Unlock()
	securitySwitchCache.V = nil

	return nil
}

func GetSecuritySwitch() (bool, error) {
	securitySwitchCache.Lock()
	defer securitySwitchCache.Unlock()

	// return cache value is cache exists
	if securitySwitchCache.V != nil {
		return securitySwitchCache.V.IsOn, nil
	}

	db, err := getConn()
	if err != nil {
		return DefaultSecuritySwitch, err
	}

	var data SecuritySwitch

	if err := db.First(&data).Error; err != nil {
		if err == ErrRecordNotFound {
			// database empty, use default value
			data.IsOn = DefaultSecuritySwitch
			goto finish
		}
		log.Warnf("get security switch from db err=%v", err)
		return DefaultSecuritySwitch, err

	}

finish:
	// sync DB data to cache
	securitySwitchCache.V = &SecuritySwitch{
		ID:   data.ID,
		IsOn: data.IsOn,
	}

	return data.IsOn, nil
}
