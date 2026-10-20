package internal

import (
	log "github.com/sirupsen/logrus"

	"scmc/model/db"
)

var runtimeLogChan = make(chan *db.RuntimeLog, 10)

func RuntimeLogWriter() {
	for r := range runtimeLogChan {
		if err := db.CreateRuntimeLog([]*db.RuntimeLog{r}); err != nil {
			log.Warnf("CreateLog %+v err=%v", *r, err)
		}
	}
}
