package main

import (
	"flag"
	"log"
	"net"

	"google.golang.org/grpc"

	"ksc-mcube/controller/server"
	"ksc-mcube/rpc/pb/container"
	"ksc-mcube/rpc/pb/node"
	// user "ksc-mcube/rpc/pb/user"
)

var (
	listenAddr string
)

func main() {
	flag.Parse()
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
}
