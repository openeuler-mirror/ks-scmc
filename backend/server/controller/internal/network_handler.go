package internal

import (
	"context"
	"time"

	log "github.com/sirupsen/logrus"

	"scmc/model"
	"scmc/rpc"
	pb "scmc/rpc/pb/network"
)

type NetworkServer struct {
	pb.UnimplementedNetworkServer
}

func (s *NetworkServer) List(ctx context.Context, in *pb.ListRequest) (*pb.ListReply, error) {
	reply := pb.ListReply{}
	var nodeToQuery []*model.NodeInfo
	if in.NodeId == -1 {
		nodes, err := model.ListNodes()
		if err != nil {
			log.Warnf("query nodes: %v", err)
			return nil, rpc.ErrInternal
		}
		for i, _ := range nodes {
			nodeToQuery = append(nodeToQuery, &nodes[i])
		}
	} else {
		nodeInfo, err := model.QueryNodeByID(in.NodeId)
		if err != nil {
			if err == model.ErrRecordNotFound {
				return nil, rpc.ErrNotFound
			}
			return nil, rpc.ErrInternal
		}
		nodeToQuery = append(nodeToQuery, nodeInfo)
	}

	for _, node := range nodeToQuery {
		conn, err := getAgentConn(node.Address)
		if err != nil {
			return nil, rpc.ErrInternal
		}

		cli := pb.NewNetworkClient(conn)
		ctx_, cancel := context.WithTimeout(context.Background(), time.Second*10)
		defer cancel()
		in.NodeId = node.ID
		subReply, err := cli.List(ctx_, in)
		if err != nil {
			log.Warnf("get network list ID=%v address=%v: %v", node.ID, node.Address, err)
			return nil, rpc.ErrInternal
		}

		reply.RealIfs = append(reply.RealIfs, subReply.RealIfs...)
		reply.VirtualIfs = append(reply.VirtualIfs, subReply.VirtualIfs...)
	}

	return &reply, nil
}

func (s *NetworkServer) Connect(ctx context.Context, in *pb.ConnectRequest) (*pb.ConnectReply, error) {
	if in.NodeId <= 0 {
		return nil, rpc.ErrInvalidArgument
	}

	nodeInfo, err := model.QueryNodeByID(in.NodeId)
	if err != nil {
		if err == model.ErrRecordNotFound {
			return nil, rpc.ErrNotFound
		}
		return nil, rpc.ErrInternal
	}

	conn, err := getAgentConn(nodeInfo.Address)
	if err != nil {
		return nil, rpc.ErrInternal
	}

	cli := pb.NewNetworkClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	reply, err := cli.Connect(ctx, in)
	if err != nil {
		log.Warnf("network connect ID=%v address=%v: %v", nodeInfo.ID, nodeInfo.Address, err)
		return nil, rpc.ErrInternal
	}

	return reply, nil
}

func (s *NetworkServer) Disconnect(ctx context.Context, in *pb.DisconnectRequest) (*pb.DisconnectReply, error) {
	if in.NodeId <= 0 {
		return nil, rpc.ErrInvalidArgument
	}

	nodeInfo, err := model.QueryNodeByID(in.NodeId)
	if err != nil {
		if err == model.ErrRecordNotFound {
			return nil, rpc.ErrNotFound
		}
		return nil, rpc.ErrInternal
	}

	conn, err := getAgentConn(nodeInfo.Address)
	if err != nil {
		return nil, rpc.ErrInternal
	}

	cli := pb.NewNetworkClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	reply, err := cli.Disconnect(ctx, in)
	if err != nil {
		log.Warnf("network disconnect ID=%v address=%v: %v", nodeInfo.ID, nodeInfo.Address, err)
		return nil, rpc.ErrInternal
	}

	return reply, nil
}

func (s *NetworkServer) ListIPtables(ctx context.Context, in *pb.ListIPtablesRequest) (*pb.ListIPtablesReply, error) {
	if in.NodeId <= 0 {
		return nil, rpc.ErrInvalidArgument
	}

	nodeInfo, err := model.QueryNodeByID(in.NodeId)
	if err != nil {
		if err == model.ErrRecordNotFound {
			return nil, rpc.ErrNotFound
		}
		return nil, rpc.ErrInternal
	}

	conn, err := getAgentConn(nodeInfo.Address)
	if err != nil {
		return nil, rpc.ErrInternal
	}

	cli := pb.NewNetworkClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	reply, err := cli.ListIPtables(ctx, in)
	if err != nil {
		log.Warnf("ListIPtables ID=%v address=%v: %v", nodeInfo.ID, nodeInfo.Address, err)
		return nil, rpc.ErrInternal
	}

	return reply, nil
}

func (s *NetworkServer) EnableIPtables(ctx context.Context, in *pb.EnableIPtablesRequest) (*pb.EnableIPtablesReply, error) {
	if in.NodeId <= 0 {
		return nil, rpc.ErrInvalidArgument
	}

	nodeInfo, err := model.QueryNodeByID(in.NodeId)
	if err != nil {
		if err == model.ErrRecordNotFound {
			return nil, rpc.ErrNotFound
		}
		return nil, rpc.ErrInternal
	}

	conn, err := getAgentConn(nodeInfo.Address)
	if err != nil {
		return nil, rpc.ErrInternal
	}

	cli := pb.NewNetworkClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	reply, err := cli.EnableIPtables(ctx, in)
	if err != nil {
		log.Warnf("EnableIPtables ID=%v address=%v: %v", nodeInfo.ID, nodeInfo.Address, err)
		return nil, rpc.ErrInternal
	}

	return reply, nil
}

func (s *NetworkServer) CreateIPtables(ctx context.Context, in *pb.CreateIPtablesRequest) (*pb.CreateIPtablesReply, error) {
	if in.NodeId <= 0 {
		return nil, rpc.ErrInvalidArgument
	}

	nodeInfo, err := model.QueryNodeByID(in.NodeId)
	if err != nil {
		if err == model.ErrRecordNotFound {
			return nil, rpc.ErrNotFound
		}
		return nil, rpc.ErrInternal
	}

	conn, err := getAgentConn(nodeInfo.Address)
	if err != nil {
		return nil, rpc.ErrInternal
	}

	cli := pb.NewNetworkClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	reply, err := cli.CreateIPtables(ctx, in)
	if err != nil {
		log.Warnf("CreateIPtables ID=%v address=%v: %v", nodeInfo.ID, nodeInfo.Address, err)
		return nil, rpc.ErrInternal
	}

	return reply, nil
}

func (s *NetworkServer) ModifyIPtables(ctx context.Context, in *pb.ModifyIPtablesRequest) (*pb.ModifyIPtablesReply, error) {
	if in.NodeId <= 0 {
		return nil, rpc.ErrInvalidArgument
	}

	nodeInfo, err := model.QueryNodeByID(in.NodeId)
	if err != nil {
		if err == model.ErrRecordNotFound {
			return nil, rpc.ErrNotFound
		}
		return nil, rpc.ErrInternal
	}

	conn, err := getAgentConn(nodeInfo.Address)
	if err != nil {
		return nil, rpc.ErrInternal
	}

	cli := pb.NewNetworkClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	reply, err := cli.ModifyIPtables(ctx, in)
	if err != nil {
		log.Warnf("ModifyIPtables ID=%v address=%v: %v", nodeInfo.ID, nodeInfo.Address, err)
		return nil, rpc.ErrInternal
	}

	return reply, nil
}

func (s *NetworkServer) RemoveIPtables(ctx context.Context, in *pb.RemoveIPtablesRequest) (*pb.RemoveIPtablesReply, error) {
	if in.NodeId <= 0 {
		return nil, rpc.ErrInvalidArgument
	}

	nodeInfo, err := model.QueryNodeByID(in.NodeId)
	if err != nil {
		if err == model.ErrRecordNotFound {
			return nil, rpc.ErrNotFound
		}
		return nil, rpc.ErrInternal
	}

	conn, err := getAgentConn(nodeInfo.Address)
	if err != nil {
		return nil, rpc.ErrInternal
	}

	cli := pb.NewNetworkClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	reply, err := cli.RemoveIPtables(ctx, in)
	if err != nil {
		log.Warnf("RemoveIPtables ID=%v address=%v: %v", nodeInfo.ID, nodeInfo.Address, err)
		return nil, rpc.ErrInternal
	}

	return reply, nil
}
