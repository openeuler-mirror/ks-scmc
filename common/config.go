package common

import (
	"fmt"
	"net/url"
	"os"

	"github.com/spf13/viper"
)

type AgentConfig struct {
	Host                      string `mapstructure:"host"`
	Port                      uint   `mapstructure:"port"`
	ContainerExtraDataBasedir string `mapstructure:"container-extra-data-basedir"`
	ContainerBackupBasedir    string `mapstructure:"container-backup-basedir"`
	OpensnitchRuleDir         string `mapstructure:"opensnitch-rule-dir"`
	AuthzSock                 string `mapstructure:"authz-sock"`
}

func (m *AgentConfig) Addr() string {
	return fmt.Sprintf("%s:%d", m.Host, m.Port)
}

type ControllerConfig struct {
	Host        string `mapstructure:"host"`
	Port        uint   `mapstructure:"port"`
	VirtualIf   string `mapstructure:"virtual-if"`
	VirtualIP   string `mapstructure:"virtual-ip"`
	ImageDir    string `mapstructure:"image-dir"`
	ImageSigner string `mapstructure:"image-signer"`
	CheckAuth   bool   `mapstructure:"check-auth"`
	CheckPerm   bool   `mapstructure:"check-perm"`
	// cert
}

func (m *ControllerConfig) Addr() string {
	return fmt.Sprintf("%s:%d", m.Host, m.Port)
}

type MySQLConfig struct {
	Addr     string `mapstructure:"addr"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	DB       string `mapstructure:"db"`
	Debug    bool   `mapstructure:"debug"`
}

func (m *MySQLConfig) DSN() string {
	q := url.Values{
		"charset": []string{"utf8mb4"},
		"timeout": []string{"10s"},
	}
	// return fmt.Sprintf("%s:%s@tcp(%s)/%s?charset=utf8mb4&timeout=10s", m.User, m.Password, m.Addr, m.DB)
	return fmt.Sprintf("%s:%s@tcp(%s)/%s?%s", m.User, m.Password, m.Addr, m.DB, q.Encode())
}

type LogConfig struct {
	Basedir string `mapstructure:"basedir"`
	Level   string `mapstructure:"level"`
	Stdout  bool   `mapstructure:"stdout"`
}

type ServiceConfig struct {
	Addr string `mapstructure:"addr"`
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

var Config struct {
	Log        LogConfig        `mapstructure:"log"`
	TLS        TLSConfig        `mapstructure:"tls"`
	Agent      AgentConfig      `mapstructure:"agent"`
	Controller ControllerConfig `mapstructure:"controller"`
	MySQL      MySQLConfig      `mapstructure:"mysql"`
	CAdvisor   ServiceConfig    `mapstructure:"cadvisor"`
	InfluxDB   ServiceConfig    `mapstructure:"influxdb"`
	Network    NetworkConfig    `mapstructure:"network"`
	Registry   RegistryConfig   `mapstructure:"registry"`
}

func setDefault() {
	viper.SetDefault("log.basedir", "/var/log/ks-scmc")
	viper.SetDefault("log.level", "info")
	viper.SetDefault("log.stdout", false)

	viper.SetDefault("tls.enable", false)

	viper.SetDefault("agent.host", "0.0.0.0")
	viper.SetDefault("agent.port", 10051)
	viper.SetDefault("agent.container-extra-data-basedir", "/var/lib/ks-scmc/containers")
	viper.SetDefault("agent.container-backup-basedir", "/var/lib/ks-scmc/backups")
	viper.SetDefault("agent.opensnitch-rule-dir", "/etc/opensnitchd/rules")
	viper.SetDefault("agent.authz-sock", "/var/lib/ks-scmc/authz.sock")

	viper.SetDefault("controller.host", "0.0.0.0")
	viper.SetDefault("controller.port", 10050)
	viper.SetDefault("controller.image-dir", "/var/lib/ks-scmc/images")
	viper.SetDefault("controller.image-signer", "/var/lib/ks-scmc/images/public-key.txt")
	viper.SetDefault("controller.check-auth", true)
	viper.SetDefault("controller.check-perm", true)

	viper.SetDefault("mysql.addr", "127.0.0.1:3306")
	viper.SetDefault("mysql.user", "root")
	viper.SetDefault("mysql.password", "12345678")
	viper.SetDefault("mysql.db", "ks-scmc")
	viper.SetDefault("mysql.debug", false)

	viper.SetDefault("cadvisor.addr", "127.0.0.1:8080")

	viper.SetDefault("influxdb.addr", "127.0.0.1:8086")

	viper.SetDefault("network.iptables_json_file", "/var/lib/ks-scmc/networks/iptables/enable.json")
	viper.SetDefault("network.iptables_path", "/var/lib/ks-scmc/networks/iptables")

	viper.SetDefault("registry.secure", false)
	viper.SetDefault("registry.addr", "127.0.0.1:5000")
	viper.SetDefault("registry.username", "")
	viper.SetDefault("registry.password", "")

}

func LoadConfig(path string) error {
	viper.SetConfigType("toml")
	setDefault()

	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	if err := viper.ReadConfig(f); err != nil {
		return err
	}

	if err := viper.Unmarshal(&Config); err != nil {
		return err
	}

	return nil
}

func NeedCheckAuth() bool {
	return Config.Controller.CheckAuth
}

func NeedCheckPerm() bool {
	return Config.Controller.CheckPerm
}
