package controller

import (
	"fmt"
	"os"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/reflection"

	"scmc/common"
	"scmc/rpc/middleware/audit"
	"scmc/rpc/middleware/auth"
	"scmc/rpc/middleware/debug"
	"scmc/rpc/pb/container"
	"scmc/rpc/pb/image"
	"scmc/rpc/pb/logging"
	"scmc/rpc/pb/network"
	"scmc/rpc/pb/node"
	"scmc/rpc/pb/system"
	"scmc/rpc/pb/user"
	"scmc/server/controller/internal"
	"scmc/server/controller/tasks"
	"scmc/server/controller/thriftapi"
)



func Server() (*grpc.Server, error) {
	tlsCredentials, err := common.LoadTLSCredentials()
	if err != nil {
		return nil, fmt.Errorf("cannot load TLS credentials: %w", err)
	}

	s := grpc.NewServer(
		grpc.ChainUnaryInterceptor(debug.UnaryInterceptor(), auth.UnaryInterceptor(), audit.UnaryInterceptor()),
		grpc.ChainStreamInterceptor(debug.StreamInterceptor(), auth.StreamInterceptor(), audit.StreamInterceptor()),
		grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
			MinTime:             20 * time.Second, // If a client pings more than once every 10 seconds, terminate the connection
			PermitWithoutStream: true,             // Allow pings even when there are no active streams
		}),
		grpc.KeepaliveParams(keepalive.ServerParameters{
			MaxConnectionIdle:     60 * time.Second, // If a client is idle for 60 seconds, send a GOAWAY
			MaxConnectionAge:      30 * time.Minute, // If any connection is alive for more than 30 minutes, send a GOAWAY
			MaxConnectionAgeGrace: 5 * time.Second,  // Allow 5 seconds for pending RPCs to complete before forcibly closing connections
			Time:                  20 * time.Second, // Ping the client if it is idle for 20 seconds to ensure the connection is still active
			Timeout:               5 * time.Second,  // Wait 5 second for the ping ack before assuming the connection is dead
		}),
		grpc.Creds(tlsCredentials),
	)

	container.RegisterContainerServer(s, &internal.ContainerServer{})
	image.RegisterImageServer(s, &internal.ImageServer{})
	network.RegisterNetworkServer(s, &internal.NetworkServer{})
	node.RegisterNodeServer(s, &internal.NodeServer{})
	user.RegisterUserServer(s, &internal.UserServer{})
	logging.RegisterLoggingServer(s, &internal.LoggingServer{})
	system.RegisterSystemServer(s, &internal.SystemServer{})

	go internal.RuntimeLogWriter()
	go thriftapi.ThriftapiMain()

	tasks.AddTask(
		&tasks.SyncImages{},
		&tasks.IllegalContainer{},
		&tasks.NodeStatus{},
		&tasks.UpdateIptables{},
	)

	if os.Getenv("GRPC_REFLECTION") == "1" {
		reflection.Register(s) // for grpc debugging tools
	}
	return s, nil
}
