package tasks

import (
	"time"
	"strconv"

	log "github.com/sirupsen/logrus"
	"scmc/model/db"
)

type CleanLogs struct {
}

func (i *CleanLogs) Interval() time.Duration {
	return time.Minute
}

func (i *CleanLogs) RunAtInit() bool {
	return true
}

func (i *CleanLogs) Run() {

	value, err := db.GetLog("saveLogDays")
	if err != nil {
		log.Warnf("CleanLogs err=%v", err)
		return
	}

	num, err := strconv.ParseInt(value, 10, 64)
	if err != nil || num < 1 || num > 999 {
		log.Warnf("CleanLogs invalid value, err: %v value: %v", err, value)
		return
	}

	err = db.CleanLogs(num)
	if err != nil {
		log.Warnf("CleanLogs failed, err=%v", err)
		return
	}
}
