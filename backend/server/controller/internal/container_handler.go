package internal

import (
	"context"
	"time"

	log "github.com/sirupsen/logrus"

	"ksc-mcube/model"
	"ksc-mcube/rpc"
	pb "ksc-mcube/rpc/pb/container"
)

type ContainerServer struct {
	pb.UnimplementedContainerServer
}

func (s *ContainerServer) List(ctx context.Context, in *pb.ListRequest) (*pb.ListReply, error) {
	log.Infof("Received: %v", in)
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

		cli := pb.NewContainerClient(conn)
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
		defer cancel()
		subReply, err := cli.List(ctx, in)
		if err != nil {
			log.Warnf("get container list ID=%v address=%v: %v", node.ID, node.Address, err)
			return nil, rpc.ErrInternal
		}

		log.Debugf("subReply: %+v", subReply)
		for i := range subReply.Containers {
			subReply.Containers[i].NodeId = node.ID
			subReply.Containers[i].NodeAddress = node.Address
			reply.Containers = append(reply.Containers, subReply.Containers[i])
		}
	}

	return &reply, nil
}

func (s *ContainerServer) Create(ctx context.Context, in *pb.CreateRequest) (*pb.CreateReply, error) {
	log.Infof("Received: %v", in)
	// reply := pb.CreateReply{}

	nodeInfo, err := model.QueryNodeByID(in.NodeId)
	if err != nil {
		if err == model.ErrDBRecordNotFound {
			return nil, rpc.ErrNotFound
		}
		return nil, rpc.ErrInternal
	}

	conn, err := getAgentConn(nodeInfo.Address)
	if err != nil {
		return nil, rpc.ErrInternal
	}

	cli := pb.NewContainerClient(conn)
	ctx_, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	agentReply, err := cli.Create(ctx_, in)
	if err != nil {
		log.Warnf("create container: %v", err)
		return nil, err
	}

	log.Debugf("create container agent reply: %+v", agentReply)
	return agentReply, nil
}

func (s *ContainerServer) Start(ctx context.Context, in *pb.StartRequest) (*pb.StartReply, error) {
	log.Infof("Received: %v", in)
	reply := pb.StartReply{}

	if in.NodeId <= 0 || len(in.ContainerIds) <= 0 {
		return nil, rpc.ErrInvalidArgument
	}

	nodeInfo, err := model.QueryNodeByID(in.NodeId)
	if err != nil {
		if err == model.ErrDBRecordNotFound {
			return nil, rpc.ErrNotFound
		}
		return nil, rpc.ErrInternal
	}

	conn, err := getAgentConn(nodeInfo.Address)
	if err != nil {
		return nil, rpc.ErrInternal
	}

	cli := pb.NewContainerClient(conn)
	ctx_, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	agentReply, err := cli.Start(ctx_, in)
	if err != nil {
		log.Warnf("start container: %v", err)
		return nil, rpc.ErrInternal
	}

	log.Debugf("start container agent reply: %+v", agentReply)
	return &reply, nil
}

func (s *ContainerServer) Stop(ctx context.Context, in *pb.StopRequest) (*pb.StopReply, error) {
	log.Infof("Received: %v", in)
	reply := pb.StopReply{}

	if in.NodeId <= 0 || len(in.ContainerIds) <= 0 {
		return nil, rpc.ErrInvalidArgument
	}

	nodeInfo, err := model.QueryNodeByID(in.NodeId)
	if err != nil {
		if err == model.ErrDBRecordNotFound {
			return nil, rpc.ErrNotFound
		}
		return nil, rpc.ErrInternal
	}

	conn, err := getAgentConn(nodeInfo.Address)
	if err != nil {
		return nil, rpc.ErrInternal
	}

	cli := pb.NewContainerClient(conn)
	ctx_, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	agentReply, err := cli.Stop(ctx_, in)
	if err != nil {
		log.Warnf("stop container: %v", err)
		return nil, rpc.ErrInternal
	}

	log.Debugf("stop container agent reply: %+v", agentReply)
	return &reply, nil
}

func (s *ContainerServer) Kill(ctx context.Context, in *pb.KillRequest) (*pb.KillReply, error) {
	log.Infof("Received: %v", in)
	reply := pb.KillReply{}

	if in.NodeId <= 0 || len(in.ContainerIds) <= 0 {
		return nil, rpc.ErrInvalidArgument
	}

	nodeInfo, err := model.QueryNodeByID(in.NodeId)
	if err != nil {
		if err == model.ErrDBRecordNotFound {
			return nil, rpc.ErrNotFound
		}
		return nil, rpc.ErrInternal
	}

	conn, err := getAgentConn(nodeInfo.Address)
	if err != nil {
		return nil, rpc.ErrInternal
	}

	cli := pb.NewContainerClient(conn)
	ctx_, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	agentReply, err := cli.Kill(ctx_, in)
	if err != nil {
		log.Warnf("kill container: %v", err)
		return nil, rpc.ErrInternal
	}

	log.Debugf("kill container agent reply: %+v", agentReply)
	return &reply, nil
}

func (s *ContainerServer) Restart(ctx context.Context, in *pb.RestartRequest) (*pb.RestartReply, error) {
	log.Infof("Received: %v", in)
	reply := pb.RestartReply{}

	if in.NodeId <= 0 || len(in.ContainerIds) <= 0 {
		return nil, rpc.ErrInvalidArgument
	}

	nodeInfo, err := model.QueryNodeByID(in.NodeId)
	if err != nil {
		if err == model.ErrDBRecordNotFound {
			return nil, rpc.ErrNotFound
		}
		return nil, rpc.ErrInternal
	}

	conn, err := getAgentConn(nodeInfo.Address)
	if err != nil {
		return nil, rpc.ErrInternal
	}

	cli := pb.NewContainerClient(conn)
	ctx_, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	agentReply, err := cli.Restart(ctx_, in)
	if err != nil {
		log.Warnf("restart container: %v", err)
		return nil, rpc.ErrInternal
	}

	log.Debugf("restart container agent reply: %+v", agentReply)
	return &reply, nil
}

func (s *ContainerServer) Remove(ctx context.Context, in *pb.RemoveRequest) (*pb.RemoveReply, error) {
	log.Infof("Received: %v", in)
	reply := pb.RemoveReply{}

	if in.NodeId <= 0 || len(in.ContainerIds) <= 0 {
		return nil, rpc.ErrInvalidArgument
	}

	nodeInfo, err := model.QueryNodeByID(in.NodeId)
	if err != nil {
		if err == model.ErrDBRecordNotFound {
			return nil, rpc.ErrNotFound
		}
		return nil, rpc.ErrInternal
	}

	conn, err := getAgentConn(nodeInfo.Address)
	if err != nil {
		return nil, rpc.ErrInternal
	}

	cli := pb.NewContainerClient(conn)
	ctx_, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	agentReply, err := cli.Remove(ctx_, in)
	if err != nil {
		log.Warnf("remove container: %v", err)
		return nil, rpc.ErrInternal
	}

	log.Debugf("remove container agent reply: %+v", agentReply)
	return &reply, nil
}

func (s *ContainerServer) Inspect(ctx context.Context, in *pb.InspectRequest) (*pb.InspectReply, error) {
	log.Infof("Received: %v", in)
	// reply := pb.InspectReply{}

	if in.NodeId <= 0 || in.ContainerId == "" {
		return nil, rpc.ErrInvalidArgument
	}

	nodeInfo, err := model.QueryNodeByID(in.NodeId)
	if err != nil {
		if err == model.ErrDBRecordNotFound {
			return nil, rpc.ErrNotFound
		}
		return nil, rpc.ErrInternal
	}

	conn, err := getAgentConn(nodeInfo.Address)
	if err != nil {
		return nil, rpc.ErrInternal
	}

	cli := pb.NewContainerClient(conn)
	ctx_, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	agentReply, err := cli.Inspect(ctx_, in)
	if err != nil {
		log.Warnf("Inspect container: %v", err)
		return nil, rpc.ErrInternal
	}

	log.Debugf("Inspect container agent reply: %+v", agentReply)
	return agentReply, nil
}

func (s *ContainerServer) Status(ctx context.Context, in *pb.StatusRequest) (*pb.StatusReply, error) {
	log.Infof("Received: %v", in)
	// reply := pb.StatusReply{}

	if in.NodeId <= 0 || in.ContainerId == "" {
		return nil, rpc.ErrInvalidArgument
	}

	nodeInfo, err := model.QueryNodeByID(in.NodeId)
	if err != nil {
		if err == model.ErrDBRecordNotFound {
			return nil, rpc.ErrNotFound
		}
		return nil, rpc.ErrInternal
	}

	conn, err := getAgentConn(nodeInfo.Address)
	if err != nil {
		return nil, rpc.ErrInternal
	}

	cli := pb.NewContainerClient(conn)
	ctx_, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	agentReply, err := cli.Status(ctx_, in)
	if err != nil {
		log.Warnf("Status container: %v", err)
		return nil, rpc.ErrInternal
	}

	log.Debugf("Status container agent reply: %+v", agentReply)
	return agentReply, nil
}

func (s *ContainerServer) Update(ctx context.Context, in *pb.UpdateRequest) (*pb.UpdateReply, error) {
	log.Infof("Received: %v", in)

	if in.NodeId <= 0 || in.ContainerId == "" {
		return nil, rpc.ErrInvalidArgument
	}

	nodeInfo, err := model.QueryNodeByID(in.NodeId)
	if err != nil {
		if err == model.ErrDBRecordNotFound {
			return nil, rpc.ErrNotFound
		}
		return nil, rpc.ErrInternal
	}

	conn, err := getAgentConn(nodeInfo.Address)
	if err != nil {
		return nil, rpc.ErrInternal
	}

	cli := pb.NewContainerClient(conn)
	ctx_, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	agentReply, err := cli.Update(ctx_, in)
	if err != nil {
		log.Warnf("Update container: %v", err)
		return nil, rpc.ErrInternal
	}

	log.Debugf("Update container agent reply: %+v", agentReply)
	return agentReply, nil
}
