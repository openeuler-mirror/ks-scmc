package common

import "flag"

var (
	Host           string
	AgentPort      uint
	ControllerPort uint

	GraphicConfigBaseDir string
	CAdvisorAddr         string
)

func init() {
	flag.StringVar(&Host, "host", "0.0.0.0", "Server listening host")
	flag.UintVar(&AgentPort, "agent-port", 10051, "Listening port of agent server")
	flag.UintVar(&ControllerPort, "controller-port", 10050, "Listening port of controller server")
	flag.StringVar(&GraphicConfigBaseDir, "graphic-conf-base", "/var/lib/ksc-mcube/containers", "Container graphic config base directory")
	flag.StringVar(&CAdvisorAddr, "cadvisor-addr", "127.0.0.1:8080", "Address to cadvisor service")
}
