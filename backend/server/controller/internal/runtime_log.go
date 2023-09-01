package internal

import (
	"scmc/model"

	log "github.com/sirupsen/logrus"
)

var runtimeLogChan = make(chan *model.RuntimeLog, 10)

func appendRuntimeLog(record *model.RuntimeLog) {
	runtimeLogChan <- record
}

func RuntimeLogWriter() {
	for {
		select {
		case r := <-runtimeLogChan:
			if err := model.CreateRuntimeLog([]*model.RuntimeLog{r}); err != nil {
				log.Warnf("CreateLog %+v err=%v", *r, err)
			}
		}
	}
}
