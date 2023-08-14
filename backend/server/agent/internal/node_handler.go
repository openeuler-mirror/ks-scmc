package internal

import (
	"context"

	"ksc-mcube/rpc/errno"
	pb "ksc-mcube/rpc/pb/node"
)

type NodeServer struct {
	pb.UnimplementedNodeServer
}

func (s *NodeServer) List(ctx context.Context, in *pb.ListRequest) (*pb.ListReply, error) {
	return &pb.ListReply{Header: errno.Unimplemented}, nil
}

func (s *NodeServer) Create(ctx context.Context, in *pb.CreateRequest) (*pb.CreateReply, error) {
	return &pb.CreateReply{Header: errno.Unimplemented}, nil
}

func (s *NodeServer) Remove(ctx context.Context, in *pb.RemoveRequest) (*pb.RemoveReply, error) {
	return &pb.RemoveReply{Header: errno.Unimplemented}, nil
}

func (s *NodeServer) Status(ctx context.Context, in *pb.StatusRequest) (*pb.StatusReply, error) {
	reply := pb.StatusReply{Header: errno.InternalError}

	_, err := dockerCli()
	if err != nil {
		reply.Header = errno.OK
		reply.StatusList = append(reply.StatusList, &pb.NodeStatus{State: int64(pb.NodeState_Unknown)})
		return &reply, nil
	}

	reply.StatusList = append(reply.StatusList, &pb.NodeStatus{
		State: int64(pb.NodeState_Online),
		// total containers
		// running containers
		// cpu_usage
		// memory usage
	})

	reply.Header = errno.OK
	return &reply, nil
}
