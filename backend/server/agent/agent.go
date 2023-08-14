package agent

import (
	"ksc-mcube/rpc/pb/container"
	"ksc-mcube/rpc/pb/node"
	"ksc-mcube/server/agent/internal"

	"google.golang.org/grpc"
)

func Register(s *grpc.Server) {
	container.RegisterContainerServer(s, &internal.ContainerServer{})
	node.RegisterNodeServer(s, &internal.NodeServer{})
}
