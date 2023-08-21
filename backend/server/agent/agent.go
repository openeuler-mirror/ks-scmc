package agent

import (
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"

	"ksc-mcube/rpc/pb/container"
	"ksc-mcube/rpc/pb/image"
	"ksc-mcube/rpc/pb/network"
	"ksc-mcube/rpc/pb/node"
	"ksc-mcube/server"
	"ksc-mcube/server/agent/internal"
)

func Server() (*grpc.Server, error) {
	opts := []grpc.ServerOption{
		grpc.ChainUnaryInterceptor(
			server.NewLogInterceptor().Unary(),
		),
		grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
			MinTime:             10 * time.Second, // If a client pings more than once every 10 seconds, terminate the connection
			PermitWithoutStream: true,             // Allow pings even when there are no active streams
		}),
		grpc.KeepaliveParams(keepalive.ServerParameters{
			MaxConnectionIdle:     60 * time.Second, // If a client is idle for 60 seconds, send a GOAWAY
			MaxConnectionAge:      30 * time.Minute, // If any connection is alive for more than 30 minutes, send a GOAWAY
			MaxConnectionAgeGrace: 5 * time.Second,  // Allow 5 seconds for pending RPCs to complete before forcibly closing connections
			Time:                  20 * time.Second, // Ping the client if it is idle for 20 seconds to ensure the connection is still active
			Timeout:               5 * time.Second,  // Wait 5 second for the ping ack before assuming the connection is dead
		}),
	}

	s := grpc.NewServer(opts...)

	container.RegisterContainerServer(s, &internal.ContainerServer{})
	image.RegisterImageServer(s, &internal.ImageServer{})
	network.RegisterNetworkServer(s, &internal.NetworkServer{})
	node.RegisterNodeServer(s, &internal.NodeServer{})

	return s, nil
}
