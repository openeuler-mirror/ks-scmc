package model

import (
	log "github.com/sirupsen/logrus"
)

const (
	EventCreateNode       = "EVENT_CREATE_NODE"
	EventUpdateNode       = "EVENT_UPDATE_NODE"
	EventRemoveNode       = "EVENT_REMOVE_NODE"
	EventCreateContainer  = "EVENT_CREATE_CONTAINER"
	EventStartContainer   = "EVENT_START_CONTAINER"
	EventStopContainer    = "EVENT_STOP_CONTAINER"
	EventRemoveContainer  = "EVENT_REMOVE_CONTAINER"
	EventRestartContainer = "EVENT_RESTART_CONTAINER"
	EventUploadImage      = "EVENT_UPLOAD_IMAGE"
	EventDownloadImage    = "EVENT_DOWNLOAD_IMAGE"
	EventApproveImage     = "EVENT_APPROVE_IMAGE"
	EventRemoveImage      = "EVENT_REMOVE_IMAGE"
	EventUserLogin        = "EVENT_USER_LOGIN"
	EventUserLogout       = "EVENT_USER_LOGOUT"
	EventCreateUser       = "EVENT_CREATE_USER"
	EventUpdateUser       = "EVENT_UPDATE_USER"
	EventRemoveUser       = "EVENT_REMOVE_USER"
	EventCreateRole       = "EVENT_CREATE_ROLE"
	EventUpdateRole       = "EVENT_UPDATE_ROLE"
	EventRemoveRole       = "EVENT_REMOVE_ROLE"

	// node offline
	// node resource overload
	// illegal image
	// illegal container
)

type RuntimeLog struct {
	ID            int64 `gorm:"primaryKey"`
	Level         int8
	NodeId        int64
	NodeInfo      string
	ContainerName string
	EventType     string
	Username      string
	Detail        string
	HaveRead      int8
	CreatedAt     int64 `gorm:"autoCreateTime"`
	UpdatedAt     int64 `gorm:"autoUpdateTime"`
}

func CreateLog(logs []RuntimeLog) error {
	db, err := getConn()
	if err != nil {
		return err
	}

	result := db.Create(&logs)
	if result.Error != nil {
		log.Errorf("db create log %v", result.Error)
		return translateError(result.Error)
	}

	return nil
}

func ListLog(pageSize, pageNo int64, condition interface{}) (*Pager, []RuntimeLog, error) {

	var data []RuntimeLog
	pager, err := PageQuery(pageSize, pageNo, &RuntimeLog{}, condition, "id desc", &data)
	if err != nil {
		return nil, nil, err
	}

	return pager, data, nil
}
