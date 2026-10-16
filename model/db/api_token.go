package db

import log "github.com/sirupsen/logrus"

type ApiToken struct {
	ID    int64 `gorm:"primaryKey"`
	Name  string
	Token string
}

func (ApiToken) TableName() string {
	return "api_tokens"
}

func FindApiTokenByName(name string) (*ApiToken, error) {
	db, err := getConn()
	if err != nil {
		return nil, err
	}

	var data ApiToken
	if err := db.First(&data, "name = ?", name).Error; err != nil {
		log.Warnf("ApiToken.FindByName: %v", err)
		if err == ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}

	return &data, nil
}
