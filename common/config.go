package common

import (
	"bufio"
	"bytes"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"

	"scmc/common/aes"
)

var defaultNetProList = []string{
	"/usr/bin/ks-scmc-server",
	"/usr/bin/ks-scmc-authz",
	"/usr/bin/mysql",
	"/usr/bin/sqlite3",
	"/usr/bin/cadvisor",
	"/usr/bin/influxd",
	"/usr/libexec/ks-scmc-gui",
	"/usr/sbin/keepalived",
	"/usr/bin/rsync",
	"/usr/bin/lsyncd",
	"/usr/bin/opensnitchd",
	"/usr/bin/ks-scmc-gui",
	"/usr/libexec/mysqld",
}

type AgentConfig struct {
	Host                       string  `mapstructure:"host"`
	Port                       uint    `mapstructure:"port"`
	ContainerExtraDataBasedir  string  `mapstructure:"container-extra-data-basedir"`
	ContainerBackupBasedir     string  `mapstructure:"container-backup-basedir"`
	OpensnitchRuleDir          string  `mapstructure:"opensnitch-rule-dir"`
	AuthzSock                  string  `mapstructure:"authz-sock"`
	BackupJob                  string  `mapstructure:"backup-job"`
	ContainerBackupTimeout     uint    `mapstructure:"container-backup-timeout"`
	AsyncDiskResizeThresholdMB float64 `mapstructure:"async-disk-resize-threshold-mb"`
}

func (m *AgentConfig) Addr() string {
	return fmt.Sprintf("%s:%d", m.Host, m.Port)
}

func (m *AgentConfig) BackupTimeoutDuration() time.Duration {
	return time.Second * time.Duration(m.ContainerBackupTimeout)
}

type ControllerConfig struct {
	Host           string `mapstructure:"host"`
	Port           uint   `mapstructure:"port"`
	VirtualIf      string `mapstructure:"virtual-if"`
	VirtualIP      string `mapstructure:"virtual-ip"`
	ImageDir       string `mapstructure:"image-dir"`
	ImageSigner    string `mapstructure:"image-signer"`   // 验签公钥
	ImageSignKey   string `mapstructure:"image-sign-key"` // 签名私钥
	ImageMaxSize   uint64 `mapstructure:"image-max-size"`
	SessionTimeout uint   `mapstructure:"session-timeout"`
	SuperUser      bool   `mapstructure:"super-user"`
	// cert
}

func (m *ControllerConfig) Addr() string {
	return fmt.Sprintf("%s:%d", m.Host, m.Port)
}

func (m *ControllerConfig) SessionLiveDuration() time.Duration {
	return time.Second * time.Duration(m.SessionTimeout)
}

type MySQLConfig struct {
	Addr     string `mapstructure:"addr"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	DB       string `mapstructure:"db"`
	Debug    bool   `mapstructure:"debug"`
}

func (m *MySQLConfig) DSN() (string, error) {
	passwordDe, err := aes.Decrypt(m.Password)
	if err != nil {
		return "", err
	}

	q := url.Values{
		"charset": []string{"utf8mb4"},
		"timeout": []string{"10s"},
	}
	// return fmt.Sprintf("%s:%s@tcp(%s)/%s?charset=utf8mb4&timeout=10s", m.User, m.Password, m.Addr, m.DB)
	return fmt.Sprintf("%s:%s@tcp(%s)/%s?%s", m.User, passwordDe, m.Addr, m.DB, q.Encode()), nil
}

type SQLiteConfig struct {
	Path string `mapstructure:"path"`
}

type LogConfig struct {
	Basedir string `mapstructure:"basedir"`
	Level   string `mapstructure:"level"`
	Stdout  bool   `mapstructure:"stdout"`
}

type ServiceConfig struct {
	Addr     string `mapstructure:"addr"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
}

// DecryptPassword 解密密码字段
func (s *ServiceConfig) DecryptPassword() (string, error) {
	passwordDe, err := aes.Decrypt(s.Password)
	if err != nil {
		return "", err
	}
	return string(passwordDe), nil
}

type TLSConfig struct {
	Enable     bool   `mapstructure:"enable"`
	CA         string `mapstructure:"ca"`
	ServerCert string `mapstructure:"server_cert"`
	ServerKey  string `mapstructure:"server_key"`
}

type VirtualNicConfig struct {
	Name         string `mapstructure:"name"`
	IpAddr       string `mapstructure:"ipaddr"`
	ContainerIfs string `mapstructure:"container_ifs"`
}

type NetworkConfig struct {
	IPtablesJsonFile string `mapstructure:"iptables_json_file"`
	IPtablesPath     string `mapstructure:"iptables_path"`
}

type RegistryConfig struct {
	Secure   bool   `mapstructure:"secure"`
	Addr     string `mapstructure:"addr"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
}

// KsimcConfig：与内核 ksimc 通过字符设备通信（默认 /dev/ksimc）。
type KsimcConfig struct {
	Device string `mapstructure:"device"`
}

type NodeProWhiteComListConfig struct {
	Path []string `mapstructure:"path"`
}

type NetworkProcessWhitelistConfig struct {
	Paths []string `mapstructure:"paths"`
}

type AlertConfig struct {
	ResourceUsageThreshold float64 `mapstructure:"resource_usage_threshold"`
}

type PerfConfig struct {
	Enable                 bool   `mapstructure:"enable"`
	Host                   string `mapstructure:"host"`
	Port                   uint   `mapstructure:"port"`
	BlockProfileRate       int    `mapstructure:"block_profile_rate"`
	MutexProfileFraction   int    `mapstructure:"mutex_profile_fraction"`
	MemStatsLogIntervalSec uint   `mapstructure:"memstats_log_interval_sec"`
	HeapDumpDir            string `mapstructure:"heap_dump_dir"`
}

func (m *PerfConfig) Addr() string {
	return fmt.Sprintf("%s:%d", m.Host, m.Port)
}

var Config struct {
	Log                  LogConfig                     `mapstructure:"log"`
	ApiTLS               TLSConfig                     `mapstructure:"apitls"`
	ApiController        ControllerConfig              `mapstructure:"apicontroller"`
	TLS                  TLSConfig                     `mapstructure:"tls"`
	Agent                AgentConfig                   `mapstructure:"agent"`
	Controller           ControllerConfig              `mapstructure:"controller"`
	MySQL                MySQLConfig                   `mapstructure:"mysql"`
	SQLite               SQLiteConfig                  `mapstructure:"sqlite"`
	CAdvisor             ServiceConfig                 `mapstructure:"cadvisor"`
	InfluxDB             ServiceConfig                 `mapstructure:"influxdb"`
	Network              NetworkConfig                 `mapstructure:"network"`
	Registry             RegistryConfig                `mapstructure:"registry"`
	Ksimc                KsimcConfig                   `mapstructure:"ksimc"`
	NodeProWhiteComList  NodeProWhiteComListConfig     `mapstructure:"nodeprowhitecomlist"`
	NetworkProcWhitelist NetworkProcessWhitelistConfig `mapstructure:"network_proc_whitelist"`
	Alert                AlertConfig                   `mapstructure:"alert"`
	Perf                 PerfConfig                    `mapstructure:"perf"`
}

type ProcessInfo struct {
	PID       int
	ExePath   string
	StartTime time.Time
	ParentPID int
	User      string
}

var (
	bootTime     time.Time // 缓存启动时间
	bootTimeErr  error     // 缓存错误信息
	bootTimeOnce sync.Once // 控制初始化逻辑仅执行一次
)

// btime 开机后不会变，执行一次即可
func getSystemBootTime() (time.Time, error) {
	// 使用 sync.Once 确保初始化逻辑只执行一次
	bootTimeOnce.Do(func() {
		data, err := ioutil.ReadFile("/proc/stat")
		if err != nil {
			bootTimeErr = err
			return
		}

		scanner := bufio.NewScanner(bytes.NewReader(data))
		found := false
		for scanner.Scan() {
			if strings.HasPrefix(scanner.Text(), "btime") {
				fields := strings.Fields(scanner.Text())
				if len(fields) < 2 {
					bootTimeErr = fmt.Errorf("invalid btime format")
					return
				}
				sec, err := strconv.ParseInt(fields[1], 10, 64)
				if err != nil {
					bootTimeErr = err
					return
				}
				bootTime = time.Unix(sec, 0)
				found = true
				break
			}
		}
		if !found {
			bootTimeErr = fmt.Errorf("btime not found")
		}
	})

	return bootTime, bootTimeErr
}

func getClockTicks() int64 {
	return 100 // 固定时钟频率
}

func getProcessStartTime(pid int) (time.Time, error) {
	data, err := ioutil.ReadFile(fmt.Sprintf("/proc/%d/stat", pid))
	if err != nil {
		return time.Time{}, err
	}

	pidStr := string(data)
	end := strings.LastIndex(pidStr, ")")
	if end == -1 {
		return time.Time{}, fmt.Errorf("invalid stat format")
	}

	fields := strings.Fields(pidStr[end+1:])
	if len(fields) < 20 {
		return time.Time{}, fmt.Errorf("invalid stat format")
	}

	clkTicks, err := strconv.ParseInt(fields[19], 10, 64)
	if err != nil {
		return time.Time{}, err
	}

	clkTCK := getClockTicks()
	bootTime, err := getSystemBootTime()
	if err != nil {
		return time.Time{}, err
	}
	return bootTime.Add(time.Duration(clkTicks*1000/clkTCK) * time.Millisecond), nil
}

func getProcessUID(pid int) (string, error) {
	statusFile := fmt.Sprintf("/proc/%d/status", pid)
	data, err := ioutil.ReadFile(statusFile)
	if err != nil {
		return "", err
	}

	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		if strings.HasPrefix(scanner.Text(), "Uid:") {
			fields := strings.Fields(scanner.Text())
			if len(fields) >= 2 {
				return fields[1], nil
			}
		}
	}
	return "", fmt.Errorf("UID not found")
}

func isGUIProcess(exePath string) bool {
	keywords := []string{
		"Xorg", "lightdm", "gdm", "sddm", "gnome-shell", "kwin",
		"mate-session", "xfce4-session", "kiran-session", "marco", "caja",
	}
	exeName := filepath.Base(exePath)
	for _, kw := range keywords {
		if strings.EqualFold(exeName, kw) {
			return true
		}
	}
	return false
}

func getBootProcesses() ([]ProcessInfo, error) {
	bootTime, err := getSystemBootTime()
	if err != nil {
		return nil, err
	}

	procDir := "/proc"
	entries, err := ioutil.ReadDir(procDir)
	if err != nil {
		return nil, err
	}

	var processes []ProcessInfo

	for _, entry := range entries {
		pid, err := strconv.Atoi(entry.Name())
		if err != nil || !entry.IsDir() {
			continue
		}

		ppid, _ := getParentPID(pid)
		startTime, _ := getProcessStartTime(pid)

		// 处理可执行路径读取错误
		exePath, err := os.Readlink(fmt.Sprintf("/proc/%d/exe", pid))
		if err != nil {
			continue
			// exePath = fmt.Sprintf("[error: %s]", err)
		}

		uid, _ := getProcessUID(pid)

		// 关键修改：只过滤启动时间早于系统启动时间的进程
		if startTime.Before(bootTime) {
			continue
		}

		// 调整过滤逻辑
		isSystemdChild := ppid == 1
		isGUI := isGUIProcess(exePath)
		isUserProcess := false
