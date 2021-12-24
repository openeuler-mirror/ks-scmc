package internal

import (
	"context"
	"time"

	log "github.com/sirupsen/logrus"

	"ksc-mcube/model"
	"ksc-mcube/rpc/errno"
	pb "ksc-mcube/rpc/pb/node"
)

type NodeServer struct {
	pb.UnimplementedNodeServer
}

func (s *NodeServer) List(ctx context.Context, in *pb.ListRequest) (*pb.ListReply, error) {
	log.Infof("Received: %v", in)
	nodes, err := model.ListNodes()
	if err != nil {
		return &pb.ListReply{Header: errno.InternalError}, nil
	}

	reply := pb.ListReply{Header: errno.OK}
	for _, node := range nodes {
		reply.Nodes = append(reply.Nodes, &pb.NodeInfo{
			Id:      node.ID,
			Name:    node.Name,
			Address: node.Address,
			Comment: node.Comment,
		})
	}
	return &reply, nil
}

func (s *NodeServer) Create(ctx context.Context, in *pb.CreateRequest) (*pb.CreateReply, error) {
	log.Infof("Received: %v", in)

	if in.Name == "" || in.Address == "" {
		return &pb.CreateReply{Header: errno.InvalidArgument}, nil
	}

	// connect agent address, check is alive

	if err := model.CreateNode(in.Name, in.Address, in.Comment); err != nil {
		if err == model.ErrDuplicateKey {
			return &pb.CreateReply{Header: errno.AlreadyExists}, nil
		}
		return &pb.CreateReply{Header: errno.InternalError}, nil
	}

	return &pb.CreateReply{Header: errno.OK}, nil
}

func (s *NodeServer) Remove(ctx context.Context, in *pb.RemoveRequest) (*pb.RemoveReply, error) {
	log.Infof("Received: %v", in)
	if err := model.RemoveNode(in.Ids); err != nil {
		return &pb.RemoveReply{Header: errno.InternalError}, nil
	}
	return &pb.RemoveReply{Header: errno.OK}, nil
}

func (s *NodeServer) Status(ctx context.Context, in *pb.StatusRequest) (*pb.StatusReply, error) {
	reply := pb.StatusReply{Header: errno.InternalError}

	nodes, err := model.ListNodes()
	if err != nil {
		return &reply, nil
	}

	var nodeToQuery []*model.NodeInfo
	for _, nodeID := range in.NodeIds {
		for i, node := range nodes {
			if node.ID == nodeID {
				nodeToQuery = append(nodeToQuery, &nodes[i])
				goto next
			}
		}
		log.Warnf("node ID=%v not found", nodeID)
		reply.Header = errno.NotFound
		return &reply, nil
	next:
	}

	for _, node := range nodeToQuery {
		conn, err := getAgentConn(node.Address)
		if err != nil {
			log.Warnf("Failed to connect to agent service")
			reply.StatusList = append(reply.StatusList, &pb.NodeStatus{
				NodeId: node.ID,
				State:  int64(pb.NodeState_Offline),
			})
			continue
		}

		cli := pb.NewNodeClient(conn)
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
		defer cancel()
		subReply, err := cli.Status(ctx, in)
		if err != nil {
			log.Warnf("get node status ID=%v address=%v: %v", node.ID, node.Address, err)
			return &reply, nil
		}

		log.Debugf("subReply: %+v", subReply)
		for i, _ := range subReply.StatusList {
			subReply.StatusList[i].NodeId = node.ID
			reply.StatusList = append(reply.StatusList, subReply.StatusList[i])
		}
	}

	reply.Header = errno.OK
	return &reply, nil
}
