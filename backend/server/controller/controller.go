package controller

import (
	"fmt"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"

	"scmc/common"
	"scmc/model"
	"scmc/rpc/pb/container"
	"scmc/rpc/pb/image"
	"scmc/rpc/pb/logging"
	"scmc/rpc/pb/network"
	"scmc/rpc/pb/node"
	"scmc/rpc/pb/security"
	"scmc/rpc/pb/user"
	"scmc/server"
	"scmc/server/controller/internal"
)

func Server() (*grpc.Server, error) {
	opts := []grpc.ServerOption{
		grpc.ChainUnaryInterceptor(server.NewLogInterceptor(true).Unary()),
		grpc.ChainUnaryInterceptor(server.NewAuthInterceptor().Unary()),
		// grpc.ChainStreamInterceptor(server.NewLogInterceptor().Streams()),
		grpc.ChainStreamInterceptor(server.NewAuthInterceptor().Streams()),
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

	tlsCredentials, err := common.LoadTLSCredentials()
	if err != nil {
		return nil, fmt.Errorf("cannot load TLS credentials: %w", err)
	}

	opts = append(opts, grpc.Creds(tlsCredentials))

	s := grpc.NewServer(opts...)

	container.RegisterContainerServer(s, &internal.ContainerServer{})
	image.RegisterImageServer(s, &internal.ImageServer{})
	network.RegisterNetworkServer(s, &internal.NetworkServer{})
	node.RegisterNodeServer(s, &internal.NodeServer{})
	user.RegisterUserServer(s, &internal.UserServer{})
	logging.RegisterLoggingServer(s, &internal.LoggingServer{})
	security.RegisterSecurityServer(s, &internal.SecurityServer{})

	go internal.NodeStatusMonitor()
	go internal.RuntimeLogWriter()
	go model.PushImgae()
	go internal.ResumeContainerConfigs()
	go internal.CheckContainerBackupJob()
	go internal.DetectIllegalContainer()

	return s, nil
}
