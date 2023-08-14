package controller

import (
	"ksc-mcube/rpc/pb/container"
	"ksc-mcube/rpc/pb/node"
	"ksc-mcube/server/controller/internal"

	"google.golang.org/grpc"
)

func Register(s *grpc.Server) {
	// user.RegisterUserServer(s, &internal.UserServer{})
	node.RegisterNodeServer(s, &internal.NodeServer{})
	container.RegisterContainerServer(s, &internal.ContainerServer{})
}
