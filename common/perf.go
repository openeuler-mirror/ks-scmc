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

func startPprofHTTP(serviceMode string) {
	addr := Config.Perf.Addr()

	mux := http.NewServeMux()
	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)

	// 内存泄漏排查辅助：落盘 heap/goroutine profile，便于离线对比
	mux.HandleFunc("/debug/leak/heap", func(w http.ResponseWriter, r *http.Request) {
		path, err := dumpProfileToFile("heap", ".pprof")
		writeJSON(w, dumpReply{Profile: "heap", Path: path, Error: errString(err)})
	})
	mux.HandleFunc("/debug/leak/goroutine", func(w http.ResponseWriter, r *http.Request) {
		path, err := dumpProfileToFile("goroutine", ".txt")
		writeJSON(w, dumpReply{Profile: "goroutine", Path: path, Error: errString(err)})
	})

	srv := &http.Server{
		Addr:              addr,
		Handler:           mux,
