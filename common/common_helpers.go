package common

import (
	"runtime/debug"

	log "github.com/sirupsen/logrus"
)

func RecoverAndPrintStack() {
	if err := recover(); err != nil {
		log.Printf("recover from [%v], stack trace:\n%s", err, debug.Stack())
	}
}
