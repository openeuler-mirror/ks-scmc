package main

import (
	"flag"
	"log"
	"net"
	"os"

	"google.golang.org/grpc"

	"ksc-mcube/agent/server"
	"ksc-mcube/common"
	"ksc-mcube/rpc/pb/container"
	"ksc-mcube/rpc/pb/node"
	// user "ksc-mcube/rpc/pb/user"
)

var (
	listenAddr string
	logFile    string
)

func main() {
	if len(os.Args) < 2 {
		log.Fatalf("Usage: %s controller|agent ...", os.Args[0])
	}

	flag.Parse()
	common.InitLogger(true, logFile)

	lis, err := net.Listen("tcp", listenAddr)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer()
	// user.RegisterUserServer(s, &server.UserServer{})
	node.RegisterNodeServer(s, &server.NodeServer{})
	container.RegisterContainerServer(s, &server.ContainerServer{})

	log.Printf("server listening at %v", lis.Addr())
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

func init() {
	flag.StringVar(&listenAddr, "listen-addr", "0.0.0.0:10050", "Service listening address")
	flag.StringVar(&logFile, "logfile", "/var/log/ksc-mcube/agent.log", "log output file")
}
