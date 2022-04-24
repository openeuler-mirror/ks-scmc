package main

import (
	"context"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "scmc/rpc/pb/authz"
)

type AuthzServer struct {
	pb.UnimplementedAuthzServer
}

func (s *AuthzServer) UpdateConfig(ctx context.Context, in *pb.UpdateConfigRequest) (*pb.UpdateConfigReply, error) {
	switch in.Action {
	case int64(pb.AUTHZ_ACTION_ADD_SENSITIVE_CONTAINER):
		globalConfig.addSensitiveContainers(in.ContainerIds)
	case int64(pb.AUTHZ_ACTION_DEL_SENSITIVE_CONTAINER):
		globalConfig.delSensitiveContainers(in.ContainerIds)
	default:
		status.Error(codes.InvalidArgument, "Invalid argument")
	}

	return &pb.UpdateConfigReply{}, nil
}

func initRPCServer(sockAddr string) error {
	lis, err := net.Listen("unix", sockAddr)
	if err != nil {
		return nil
	}

	s := grpc.NewServer()
	pb.RegisterAuthzServer(s, &AuthzServer{})
	go s.Serve(lis)

	return nil
}
