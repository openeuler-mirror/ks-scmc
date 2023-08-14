package common

import "flag"

var (
	Host           string
	AgentPort      uint
	ControllerPort uint
)

func init() {
	flag.StringVar(&Host, "host", "0.0.0.0", "Server listening host")
	flag.UintVar(&AgentPort, "agent-port", 10051, "Listening port of agent server")
	flag.UintVar(&ControllerPort, "controller-port", 10050, "Listening port of controller server")
}
