package controller

import (
	"ksc-mcube/rpc/pb/container"
	"ksc-mcube/rpc/pb/image"
	"ksc-mcube/rpc/pb/network"
	"ksc-mcube/rpc/pb/node"
	"ksc-mcube/rpc/pb/user"
	"ksc-mcube/server/controller/internal"

	"google.golang.org/grpc"
)

func Register(s *grpc.Server) {
	container.RegisterContainerServer(s, &internal.ContainerServer{})
	image.RegisterImageServer(s, &internal.ImageServer{})
	network.RegisterNetworkServer(s, &internal.NetworkServer{})

	node.RegisterNodeServer(s, &internal.NodeServer{})
	user.RegisterUserServer(s, &internal.UserServer{})
}
