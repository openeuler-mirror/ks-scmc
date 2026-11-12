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
