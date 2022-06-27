package main

import (
	"context"
	"log"
	"net"
	"os"

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
	os.Remove(sockAddr) // avoid error "bind: address already in use"
	lis, err := net.Listen("unix", sockAddr)
	if err != nil {
		log.Printf("listen socket=%v err=%v", sockAddr, err)
		return err
	}

	s := grpc.NewServer()
	pb.RegisterAuthzServer(s, &AuthzServer{})
	go func() {
		err = s.Serve(lis)
	}()

	if err != nil {
		log.Printf("init rpc server err=%v", err)
		return err
	}

	return nil
}
