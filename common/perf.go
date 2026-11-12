package common

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/pprof"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	rpprof "runtime/pprof"
	"sync"
	"syscall"
	"time"

	log "github.com/sirupsen/logrus"
)

var perfOnce sync.Once

func StartPerfTools(serviceMode string) {
	perfOnce.Do(func() {
		if !Config.Perf.Enable {
			return
		}

		if Config.Perf.BlockProfileRate > 0 {
			runtime.SetBlockProfileRate(Config.Perf.BlockProfileRate)
		}
		if Config.Perf.MutexProfileFraction > 0 {
			runtime.SetMutexProfileFraction(Config.Perf.MutexProfileFraction)
		}

		go startPprofHTTP(serviceMode)
		go startMemStatsLogger(serviceMode)
		go startHeapDumpSignalHandler(serviceMode)
	})
}

