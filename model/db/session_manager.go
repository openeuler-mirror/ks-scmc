package db

import (
	"time"

	uuid "github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// 默认是五分钟检查一次
const _sessionCheckInterval = 5 * time.Second

// session 有效期为一天
const _sessionKeepAliveTime = 24 * time.Hour

// const _sessionKeepAliveTime = 10 * time.Second

type _Session struct {
	SessionID string `gorm:"primaryKey"`
	CreatedAt int64  `gorm:"autoCreateTime"`
	UpdatedAt int64  `gorm:"autoCreateTime"`
	LastUsed  int64
	NodeIP    string `gorm:"column:node_ip"`
}

func (_Session) TableName() string {
	return "session"
}

type _SessionManager struct {
	_sessionChecker *time.Ticker
}

var _sessionManagerInstance *_SessionManager

func init() {
	_NewSessionManager()
}

func _NewSessionManager() {
	if _sessionManagerInstance != nil {
		return
	}
	_sessionManagerInstance = &_SessionManager{
		_sessionChecker: time.NewTicker(_sessionCheckInterval),
	}

	go _sessionManagerInstance._SessionCheck()
}

func (s *_SessionManager) _SessionCheck() {
	if s == nil {
		log.Warn("Please init SessionManager first!")
		return
	}
	for {
		select {
		case <-s._sessionChecker.C:
			db, err := getConn()
			if err != nil {
				log.Errorf("Failed to get database connection, because %v", err)
				return
			}
			db.Transaction(func(tx *gorm.DB) error {
				result := db.Where("last_used < ?", time.Now().Add(-_sessionKeepAliveTime).Unix()).Delete(&_Session{})
				if result.Error != nil {
					log.Error("批量删除失败:", result.Error)
					return result.Error
				}
				return nil
			})
		}
	}
}

func SessionManagerGetNodeIP(sessionID string) *string {
	db, err := getConn()
	if err != nil {
		log.Errorf("Failed to get database connection, because %v", err)
		return nil
	}
	sessions := []_Session{}
	if err := db.Where("session_id = ?", sessionID).Find(&sessions).Error; err != nil {
		log.Errorf("Failed to get sessions! because: %v", err)
		return nil
	}
	if len(sessions) == 0 {
		return nil
	}
	sessions[0].LastUsed = time.Now().Unix()
	if err := db.Save(&sessions[0]).Error; err != nil {
		log.Error("Failed to update session last use time")
	}
	return &sessions[0].NodeIP
}

func SessionManagerCreateSession(nodeIP string) (*string, error) {
	db, err := getConn()
	if err != nil {
		log.Errorf("Failed to get database connection, because %s", err)
		return nil, err
	}
	tx := db.Begin()
	newSessionID := uuid.NewString()
	newSession := &_Session{
		SessionID: newSessionID,
		LastUsed:  time.Now().Unix(),
		NodeIP:    nodeIP,
	}
	if err := tx.Create(newSession).Error; err != nil {
		log.Error("Failed to Create Session data , because: ", err)
		return nil, err
	}
	if err := tx.Commit().Error; err != nil {
		log.Error("Failed to Commit Session data , because: ", err)
		return nil, err
	}
	return &newSessionID, nil
}
