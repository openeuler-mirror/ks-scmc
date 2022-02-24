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
	reply := pb.ListReply{}

	nodes, err := model.ListNodes()
	if err != nil {
		log.Warnf("query nodes: %v", err)
		return nil, rpc.ErrInternal
	}

	var nodeToQuery []*model.NodeInfo
	for _, nodeID := range uniqueInt64(in.NodeIds) {
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

		// log.Debugf("subReply: %+v", subReply)
		for i := range subReply.Containers {
			subReply.Containers[i].NodeId = node.ID
			subReply.Containers[i].NodeAddress = node.Address
			reply.Containers = append(reply.Containers, subReply.Containers[i])
		}
	}

	return &reply, nil
}

func (s *ContainerServer) Create(ctx context.Context, in *pb.CreateRequest) (*pb.CreateReply, error) {
	// reply := pb.CreateReply{}

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
	reply := pb.StartReply{}

	if len(in.Ids) <= 0 {
		return nil, rpc.ErrInvalidArgument
	}

	for _, c := range in.Ids {
		nodeId, containerIds := c.NodeId, uniqueString(c.ContainerIds)

		if nodeId <= 0 || len(containerIds) <= 0 {
			log.Warnf("start container ErrInvalidArgument")
			continue
		}

		nodeInfo, err := model.QueryNodeByID(nodeId)
		if err != nil {
			if err == model.ErrRecordNotFound {
				log.Warnf("start container ErrRecordNotFound")
				continue
			}
			log.Warnf("start container ErrInternal")
			continue
		}

		conn, err := getAgentConn(nodeInfo.Address)
		if err != nil {
			log.Warnf("start container ErrInternal")
			continue
		}

		cli := pb.NewContainerClient(conn)
		ctx_, cancel := context.WithTimeout(context.Background(), time.Second*10)
		defer cancel()

		request := pb.StartRequest{
			Ids: []*pb.ContainerIdList{
				{
					ContainerIds: containerIds,
				},
			},
		}

		agentReply, err := cli.Start(ctx_, &request)
		if err != nil {
			log.Warnf("start container ErrInternal: %v", err)
			continue
		}

		log.Debugf("start container agent reply: %+v", agentReply)
	}

	return &reply, nil
}

func (s *ContainerServer) Stop(ctx context.Context, in *pb.StopRequest) (*pb.StopReply, error) {
	reply := pb.StopReply{}

	if len(in.Ids) <= 0 {
		return nil, rpc.ErrInvalidArgument
	}

	for _, c := range in.Ids {
		nodeId, containerIds := c.NodeId, uniqueString(c.ContainerIds)

		if nodeId <= 0 || len(containerIds) <= 0 {
			log.Warnf("stop container ErrInvalidArgument")
			continue
		}

		nodeInfo, err := model.QueryNodeByID(nodeId)
		if err != nil {
			if err == model.ErrRecordNotFound {
				log.Warnf("stop container ErrRecordNotFound")
				continue
			}
			log.Warnf("stop container ErrInternal")
			continue
		}

		conn, err := getAgentConn(nodeInfo.Address)
		if err != nil {
			log.Warnf("stop container ErrInternal")
			continue
		}

		cli := pb.NewContainerClient(conn)
		ctx_, cancel := context.WithTimeout(context.Background(), time.Second*10)
		defer cancel()

		request := pb.StopRequest{
			Ids: []*pb.ContainerIdList{
				{
					ContainerIds: containerIds,
				},
			},
		}

		agentReply, err := cli.Stop(ctx_, &request)
		if err != nil {
			log.Warnf("stop container ErrInternal: %v", err)
			continue
		}

		log.Debugf("stop container agent reply: %+v", agentReply)
	}

	return &reply, nil
}

func (s *ContainerServer) Kill(ctx context.Context, in *pb.KillRequest) (*pb.KillReply, error) {
	reply := pb.KillReply{}
	if len(in.Ids) <= 0 {
		return nil, rpc.ErrInvalidArgument
	}

	for _, c := range in.Ids {
		nodeId, containerIds := c.NodeId, uniqueString(c.ContainerIds)

		if nodeId <= 0 || len(containerIds) <= 0 {
			log.Warnf("kill container ErrInvalidArgument")
			continue
		}

		nodeInfo, err := model.QueryNodeByID(nodeId)
		if err != nil {
			if err == model.ErrRecordNotFound {
				log.Warnf("kill container ErrRecordNotFound")
				continue
			}
			log.Warnf("kill container ErrInternal")
			continue
		}

		conn, err := getAgentConn(nodeInfo.Address)
		if err != nil {
			log.Warnf("kill container ErrInternal")
			continue
		}

		cli := pb.NewContainerClient(conn)
		ctx_, cancel := context.WithTimeout(context.Background(), time.Second*10)
		defer cancel()

		request := pb.KillRequest{
			Ids: []*pb.ContainerIdList{
				{
					ContainerIds: containerIds,
				},
			},
		}

		agentReply, err := cli.Kill(ctx_, &request)
		if err != nil {
			log.Warnf("kill container ErrInternal: %v", err)
			continue
		}

		log.Debugf("kill container agent reply: %+v", agentReply)
	}

	return &reply, nil
}

func (s *ContainerServer) Restart(ctx context.Context, in *pb.RestartRequest) (*pb.RestartReply, error) {
	reply := pb.RestartReply{}

	if len(in.Ids) <= 0 {
		return nil, rpc.ErrInvalidArgument
	}

	for _, c := range in.Ids {
		nodeId, containerIds := c.NodeId, uniqueString(c.ContainerIds)

		if nodeId <= 0 || len(containerIds) <= 0 {
			log.Warnf("restart container ErrInvalidArgument")
			continue
		}

		nodeInfo, err := model.QueryNodeByID(nodeId)
		if err != nil {
			if err == model.ErrRecordNotFound {
				log.Warnf("restart container ErrRecordNotFound")
				continue
			}
			log.Warnf("restart container ErrInternal")
			continue
		}

		conn, err := getAgentConn(nodeInfo.Address)
		if err != nil {
			log.Warnf("restart container ErrInternal")
			continue
		}

		cli := pb.NewContainerClient(conn)
		ctx_, cancel := context.WithTimeout(context.Background(), time.Second*10)
		defer cancel()

		request := pb.RestartRequest{
			Ids: []*pb.ContainerIdList{
				{
					ContainerIds: containerIds,
				},
			},
		}

		agentReply, err := cli.Restart(ctx_, &request)
		if err != nil {
			log.Warnf("restart container ErrInternal: %v", err)
			continue
		}

		log.Debugf("restart container agent reply: %+v", agentReply)
	}

	return &reply, nil
}

func (s *ContainerServer) Remove(ctx context.Context, in *pb.RemoveRequest) (*pb.RemoveReply, error) {
	reply := pb.RemoveReply{}

	if len(in.Ids) <= 0 {
		return nil, rpc.ErrInvalidArgument
	}

	for _, c := range in.Ids {
		nodeId, containerIds := c.NodeId, uniqueString(c.ContainerIds)
		log.Debugf("remove container, nodeId: %v", nodeId)

		if nodeId <= 0 || len(containerIds) <= 0 {
			log.Warnf("remove container ErrInvalidArgument")
			continue
		}

		nodeInfo, err := model.QueryNodeByID(nodeId)
		if err != nil {
			if err == model.ErrRecordNotFound {
				log.Warnf("remove container ErrRecordNotFound")
				continue
			}
			log.Warnf("remove container ErrInternal")
			continue
		}

		conn, err := getAgentConn(nodeInfo.Address)
		if err != nil {
			log.Warnf("remove container ErrInternal")
			continue
		}

		cli := pb.NewContainerClient(conn)
		ctx_, cancel := context.WithTimeout(context.Background(), time.Second*10)
		defer cancel()

		request := pb.RemoveRequest{
			Ids: []*pb.ContainerIdList{
				{
					ContainerIds: containerIds,
				},
			},
		}

		agentReply, err := cli.Remove(ctx_, &request)
		if err != nil {
			log.Warnf("remove container ErrInternal: %v", err)
			continue
		}

		log.Debugf("remove container agent reply: %+v", agentReply)
	}

	return &reply, nil
}

func (s *ContainerServer) Inspect(ctx context.Context, in *pb.InspectRequest) (*pb.InspectReply, error) {
	// reply := pb.InspectReply{}

	if in.NodeId <= 0 || in.ContainerId == "" {
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
	// reply := pb.StatusReply{}

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
	if in.NodeId <= 0 || in.ContainerId == "" {
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

func (s *ContainerServer) MonitorHistory(ctx context.Context, in *pb.MonitorHistoryRequest) (*pb.MonitorHistoryReply, error) {
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

	cli := pb.NewContainerClient(conn)
	ctx_, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	agentReply, err := cli.MonitorHistory(ctx_, in)
	if err != nil {
		log.Warnf("monitor history: %v", err)
		return nil, err
	}

	return agentReply, nil
}
