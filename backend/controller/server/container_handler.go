package server

import (
	"context"
	"log"
	"time"

	"ksc-mcube/model"
	"ksc-mcube/rpc/errno"
	pb "ksc-mcube/rpc/pb/container"
)

type ContainerServer struct {
	pb.UnimplementedContainerServer
}

func (s *ContainerServer) List(ctx context.Context, in *pb.ListRequest) (*pb.ListReply, error) {
	log.Printf("Received: %v", in)
	reply := pb.ListReply{Header: errno.InternalError}

	nodes, err := model.ListNodes()
	if err != nil {
		log.Printf("query nodes: %v", err)
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
		log.Printf("node ID=%v not found", nodeID)
		reply.Header = errno.NotFound
		return &reply, nil
	next:
	}

	for _, node := range nodeToQuery {
		conn, err := getAgentConn(node.Address)
		if err != nil {
			return &reply, nil
		}

		cli := pb.NewContainerClient(conn)
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
		defer cancel()
		subReply, err := cli.List(ctx, in)
		if err != nil {
			log.Printf("get container list ID=%v address=%v: %v", node.ID, node.Address, err)
			return &reply, nil
		}

		log.Printf("subReply: %+v", subReply)
		for i, _ := range subReply.Containers {
			subReply.Containers[i].NodeId = node.ID
			subReply.Containers[i].NodeAddress = node.Address
			reply.Containers = append(reply.Containers, subReply.Containers[i])
		}
	}

	reply.Header = errno.OK
	return &reply, nil
}
