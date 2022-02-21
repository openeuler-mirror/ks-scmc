package main

import (
	"log"
	"net"
	"os"

	"google.golang.org/grpc"

	"ksc-mcube/common"
	"ksc-mcube/server/agent"
	"ksc-mcube/server/controller"
)

const (
	agentMode      = "agent"
	controllerMode = "controller"
)

func main() {
	if len(os.Args) != 3 {
		usage()
	}

	mode := os.Args[1]
	configFile := os.Args[2]
	if mode != agentMode && mode != controllerMode {
		usage()
	}

	if err := common.LoadConfig(configFile); err != nil {
		log.Fatalf("load config file=%s error=%v", configFile, err)
	}

	// TODO check log dir, create when it not exists.
	common.InitLogger(mode)

	var (
		addr   string
		err    error
		server *grpc.Server
	)

	if mode == agentMode {
		server, err = agent.Server()
		addr = common.Config.Agent.Addr()
	} else {
		server, err = controller.Server()
		addr = common.Config.Controller.Addr()
	}

	if err != nil {
		log.Fatalf("init service, error=%v", err)
	}

	lis, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("failed to listen addr=%s error=%v", addr, err)
	}

	// log.Printf("%s server listening at %v", mode, lis.Addr())
	if err := server.Serve(lis); err != nil {
		log.Fatalf("failed to serve, error=%v", err)
	}
}

func usage() {
	log.Fatalf("Usage: %s controller|agent CONFIG_FILE", os.Args[0])
}
