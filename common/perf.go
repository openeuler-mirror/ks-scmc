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
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Infof("perf enabled: mode=%s pprof_listen=%s", serviceMode, addr)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Errorf("pprof http server failed: %v", err)
	}
}

type dumpReply struct {
	Profile string `json:"profile"`
	Path    string `json:"path,omitempty"`
	Error   string `json:"error,omitempty"`
}

func writeJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(v)
}

func errString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func dumpProfileToFile(profileName, suffix string) (string, error) {
	dir := Config.Perf.HeapDumpDir
	if dir == "" {
		return "", fmt.Errorf("perf.heap_dump_dir is empty")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}

	ts := time.Now().Format("20060102-150405.000")
	path := filepath.Join(dir, fmt.Sprintf("%s-%s-%s%s", profileName, ts, hostnameShort(), suffix))

	f, err := os.Create(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	switch profileName {
	case "heap":
		// 为了更稳定的对比，尽量做一次 GC
		runtime.GC()
		if err := rpprof.WriteHeapProfile(f); err != nil {
			return "", err
		}
	case "goroutine":
		p := rpprof.Lookup("goroutine")
		if p == nil {
			return "", fmt.Errorf("profile not found: %s", profileName)
		}
		// 2: 带栈信息
		if err := p.WriteTo(f, 2); err != nil {
			return "", err
		}
	default:
		return "", fmt.Errorf("unsupported profile: %s", profileName)
