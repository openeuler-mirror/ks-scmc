package internal

import (
	"context"
	"time"

	"ksc-mcube/model"
	"ksc-mcube/rpc"
	pb "ksc-mcube/rpc/pb/network"

	log "github.com/sirupsen/logrus"
)

type NetworkServer struct {
	pb.UnimplementedNetworkServer
}

func (s *NetworkServer) List(ctx context.Context, in *pb.ListRequest) (*pb.ListReply, error) {
	reply := pb.ListReply{}

	nodes, err := model.ListNodes()
	if err != nil {
		log.Warnf("query nodes: %v", err)
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
		log.Infof("node ID=%v not found", nodeID)
		return nil, rpc.ErrNotFound
	next:
	}

	for _, node := range nodeToQuery {
		conn, err := getAgentConn(node.Address)
		if err != nil {
			return nil, rpc.ErrInternal
		}

		cli := pb.NewNetworkClient(conn)
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
		defer cancel()
		subReply, err := cli.List(ctx, in)
		if err != nil {
			log.Warnf("get network list ID=%v address=%v: %v", node.ID, node.Address, err)
			return nil, rpc.ErrInternal
		}

		log.Debugf("subReply: %+v", subReply)
		for i := range subReply.BridgeIf {
			subReply.BridgeIf[i].NodeId = node.ID
			reply.BridgeIf = append(reply.BridgeIf, subReply.BridgeIf[i])
		}
		for i := range subReply.RealIf {
			subReply.RealIf[i].NodeId = node.ID
			reply.RealIf = append(reply.RealIf, subReply.RealIf[i])
		}
	}

	return &reply, nil
}
