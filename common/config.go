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
