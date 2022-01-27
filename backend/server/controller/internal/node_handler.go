package internal

import (
	"context"
	"net"
	"regexp"
	"time"

	log "github.com/sirupsen/logrus"

	"ksc-mcube/model"
	"ksc-mcube/rpc"
	"ksc-mcube/rpc/pb/common"
	pb "ksc-mcube/rpc/pb/node"
)

type NodeServer struct {
	pb.UnimplementedNodeServer
}

func getNodeStatus(node *model.NodeInfo) (*pb.StatusReply, error) {
	conn, err := getAgentConn(node.Address)
	if err != nil {
		log.Warnf("Failed to connect to agent service, node=%+v", node)
		return nil, rpc.ErrInternal
	}

	cli := pb.NewNodeClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*2)
	defer cancel()
	r, err := cli.Status(ctx, &pb.StatusRequest{})
	if err != nil {
		log.Warnf("get node status ID=%v address=%v: %v", node.ID, node.Address, err)
		return nil, rpc.ErrInternal
	}

	return r, nil
}

func (s *NodeServer) List(ctx context.Context, in *pb.ListRequest) (*pb.ListReply, error) {
	reply := pb.ListReply{}

	nodes, err := model.ListNodes()
	if err != nil {
		return nil, rpc.ErrInternal
	}

	for _, node := range nodes {
		r, _ := getNodeStatus(&node)
		var s *pb.NodeStatus
		if r != nil && len(r.StatusList) > 0 {
			s = r.StatusList[0]
		}

		reply.Nodes = append(reply.Nodes, &pb.NodeInfo{
			Id:      node.ID,
			Name:    node.Name,
			Address: node.Address,
			Comment: node.Comment,
			Status:  s,
		})
	}
	return &reply, nil
}

func (s *NodeServer) Create(ctx context.Context, in *pb.CreateRequest) (*pb.CreateReply, error) {
	if in.Name == "" || in.Address == "" {
		return nil, rpc.ErrInvalidArgument
	}

	isIp := net.ParseIP(in.Address)

	isDomain, _ := regexp.MatchString("^[a-zA-Z][a-zA-Z0-9-.]{0,253}[a-zA-Z0-9]$", in.Address)
	log.Warnf("node_handle ip: [%v], [%v], [%v]", isIp, isDomain, in.Address)

	if isIp == nil && !isDomain {
		log.Warnf("node_handle ip err: %v", in.Address)
		return nil, rpc.ErrInvalidArgument
	}

	conn, err := getAgentConn(in.Address)
	if err != nil {
		return nil, rpc.ErrInternal
	}

	cli := pb.NewNodeClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	request := pb.StatusRequest{
		Header: &common.RequestHeader{},
	}

	_, err = cli.Status(ctx, &request)
	if err != nil {
		log.Warnf("Status: %v", err)
		return nil, rpc.ErrInternal
	}

	// connect agent address, check is alive

	if err := model.CreateNode(in.Name, in.Address, in.Comment); err != nil {
		if err == model.ErrDuplicateKey {
			return nil, rpc.ErrAlreadyExists
		}
		return nil, rpc.ErrInternal
	}

	return &pb.CreateReply{}, nil
}

func (s *NodeServer) Remove(ctx context.Context, in *pb.RemoveRequest) (*pb.RemoveReply, error) {
	if err := model.RemoveNode(in.Ids); err != nil {
		return nil, rpc.ErrInternal
	}
	return &pb.RemoveReply{}, nil
}

func (s *NodeServer) Status(ctx context.Context, in *pb.StatusRequest) (*pb.StatusReply, error) {
	reply := pb.StatusReply{}

	nodes, err := model.ListNodes()
	if err != nil {
		return nil, rpc.ErrInternal
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
		return nil, rpc.ErrNotFound
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
			return nil, rpc.ErrInternal
		}

		log.Debugf("subReply: %+v", subReply)
		for i := range subReply.StatusList {
			subReply.StatusList[i].NodeId = node.ID
			reply.StatusList = append(reply.StatusList, subReply.StatusList[i])
		}
	}

	return &reply, nil
}
