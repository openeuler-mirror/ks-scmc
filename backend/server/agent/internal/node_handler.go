package internal

import (
	"context"

	pb "ksc-mcube/rpc/pb/node"
)

type NodeServer struct {
	pb.UnimplementedNodeServer
}

func (s *NodeServer) Status(ctx context.Context, in *pb.StatusRequest) (*pb.StatusReply, error) {
	reply := pb.StatusReply{}

	_, err := dockerCli()
	if err != nil {
		reply.StatusList = append(reply.StatusList, &pb.NodeStatus{State: int64(pb.NodeState_Unknown)})
	} else {
		reply.StatusList = append(reply.StatusList, &pb.NodeStatus{
			State: int64(pb.NodeState_Online),
			// total containers
			// running containers
			// cpu_usage
			// memory usage
		})
	}

	return &reply, nil
}
