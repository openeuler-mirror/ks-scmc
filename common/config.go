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
