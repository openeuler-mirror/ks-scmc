package internal

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"scmc/model/db"
	"scmc/rpc"
	pb "scmc/rpc/pb/container"
	"scmc/server/args"
)

type ContainerServer struct {
	pb.UnimplementedContainerServer
}

func (s *ContainerServer) List(ctx context.Context, in *pb.ListRequest) (*pb.ListReply, error) {
	reply := pb.ListReply{}

	nodes, err := db.ListNodes()
	if err != nil {
		log.Warnf("query nodes: %v", err)
		return nil, rpc.ErrInternal
	}

	var nodeToQuery []*db.NodeInfo
	if in.NodeIds == nil {
		for i := range nodes {
			nodeToQuery = append(nodeToQuery, &nodes[i])
		}
	} else {
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
	}

	var (
		mtx sync.Mutex
		wg  sync.WaitGroup
	)
	for _, node := range nodeToQuery {
		wg.Add(1)
		go func(ctx context.Context, node *db.NodeInfo) {
			defer wg.Done()
			conn, err := getAgentConn(node.Address)
			if err != nil {
				log.Warnf("get agent Conn address=%v: %v", node.Address, err)

				mtx.Lock()
				defer mtx.Unlock()
				reply.FailNodes = append(reply.FailNodes, node.Address)
				return
			}

			cli := pb.NewContainerClient(conn)
			ctx_, cancel := context.WithTimeout(ctx, time.Second*10)
			defer cancel()
			subReply, err := cli.List(ctx_, in)
			if err != nil {
				log.Warnf("get container list ID=%v address=%v: %v", node.ID, node.Address, err)

				mtx.Lock()
				defer mtx.Unlock()
				reply.FailNodes = append(reply.FailNodes, node.Address)
				return
			}

			for i := range subReply.Containers {
				subReply.Containers[i].NodeId = node.ID
				subReply.Containers[i].NodeAddress = node.Address
			}
			mtx.Lock()
			defer mtx.Unlock()
			reply.Containers = append(reply.Containers, subReply.Containers...)
		}(ctx, node)
	}

	wg.Wait()

	return &reply, nil
}

func (s *ContainerServer) Create(ctx context.Context, in *pb.CreateRequest) (*pb.CreateReply, error) {
	if in.Configs == nil || in.NodeId <= 0 {
		return nil, rpc.ErrInvalidArgument
	} else if !args.IsValidContainerName(in.Configs.Name) {
		return nil, status.Errorf(codes.InvalidArgument, "容器名参数错误")
	} else if !args.IsValidContainerDesc(in.Configs.Desc) {
		return nil, status.Errorf(codes.InvalidArgument, "容器描述参数错误")
	}

	nodeInfo, err := db.QueryNodeByID(in.NodeId)
	if err != nil {
		if err == db.ErrRecordNotFound {
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
	if err != nil && agentReply == nil {
		log.Warnf("Create container: %v", err)
		return nil, err
	}

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

		nodeInfo, err := db.QueryNodeByID(nodeId)
		if err != nil {
			if err == db.ErrRecordNotFound {
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

		subReply, err := cli.Start(ctx_, &request)
		if err != nil {
			log.Warnf("start container ErrInternal: %v", err)
			continue
		}

		for i := range subReply.FailInfos {
			subReply.FailInfos[i].NodeInfo = nodeInfo.Address
		}
		reply.FailInfos = append(reply.FailInfos, subReply.FailInfos...)
		reply.OkIds = append(reply.OkIds, subReply.OkIds...)
	}

	// 操作对象只有一个出错时确保返回错误码
	if len(in.Ids) == 1 && len(reply.OkIds) == 0 {
		if len(reply.FailInfos) == 0 {
			return nil, rpc.ErrInternal
		}

		return nil, status.Errorf(codes.Internal, reply.FailInfos[0].FailReason)
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

		nodeInfo, err := db.QueryNodeByID(nodeId)
		if err != nil {
			if err == db.ErrRecordNotFound {
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

		subReply, err := cli.Stop(ctx_, &request)
		if err != nil {
			log.Warnf("stop container ErrInternal: %v", err)
			continue
		}

		for i := range subReply.FailInfos {
			subReply.FailInfos[i].NodeInfo = nodeInfo.Address
		}
		reply.FailInfos = append(reply.FailInfos, subReply.FailInfos...)
		reply.OkIds = append(reply.OkIds, subReply.OkIds...)
	}

	// 操作对象只有一个出错时确保返回错误码
	if len(in.Ids) == 1 && len(reply.OkIds) == 0 {
		if len(reply.FailInfos) == 0 {
			return nil, rpc.ErrInternal
		}

		return nil, status.Errorf(codes.Internal, reply.FailInfos[0].FailReason)
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

		nodeInfo, err := db.QueryNodeByID(nodeId)
		if err != nil {
			if err == db.ErrRecordNotFound {
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

		subReply, err := cli.Restart(ctx_, &request)
		if err != nil {
			log.Warnf("restart container ErrInternal: %v", err)
			continue
		}

		for i := range subReply.FailInfos {
			subReply.FailInfos[i].NodeInfo = nodeInfo.Address
		}
		reply.FailInfos = append(reply.FailInfos, subReply.FailInfos...)
		reply.OkIds = append(reply.OkIds, subReply.OkIds...)
	}

	// 操作对象只有一个出错时确保返回错误码
	if len(in.Ids) == 1 && len(reply.OkIds) == 0 {
		if len(reply.FailInfos) == 0 {
			return nil, rpc.ErrInternal
		}

		return nil, status.Errorf(codes.Internal, reply.FailInfos[0].FailReason)
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

		nodeInfo, err := db.QueryNodeByID(nodeId)
		if err != nil {
			if err == db.ErrRecordNotFound {
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

		subReply, err := cli.Remove(ctx_, &request)
		if err != nil {
			log.Warnf("remove container ErrInternal: %v", err)
			continue
		}

		for i := range subReply.FailInfos {
			subReply.FailInfos[i].NodeInfo = nodeInfo.Address
		}
		reply.FailInfos = append(reply.FailInfos, subReply.FailInfos...)
		reply.OkIds = append(reply.OkIds, subReply.OkIds...)
	}

	// 操作对象只有一个出错时确保返回错误码
	if len(in.Ids) == 1 && len(reply.OkIds) == 0 {
		if len(reply.FailInfos) == 0 {
			return nil, rpc.ErrInternal
		}

		return nil, status.Errorf(codes.Internal, reply.FailInfos[0].FailReason)
	}

	return &reply, nil
}

func (s *ContainerServer) Inspect(ctx context.Context, in *pb.InspectRequest) (*pb.InspectReply, error) {
	if in.NodeId <= 0 || in.ContainerId == "" {
		return nil, rpc.ErrInvalidArgument
	}

	nodeInfo, err := db.QueryNodeByID(in.NodeId)
	if err != nil {
		if err == db.ErrRecordNotFound {
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
		return nil, err
	}

	return agentReply, nil
}

func (s *ContainerServer) Update(ctx context.Context, in *pb.UpdateRequest) (*pb.UpdateReply, error) {
	if in.NodeId <= 0 || in.ContainerId == "" {
		return nil, rpc.ErrInvalidArgument
	}

	nodeInfo, err := db.QueryNodeByID(in.NodeId)
	if err != nil {
		if err == db.ErrRecordNotFound {
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
		return nil, err
	}

	return agentReply, nil
}

func (s *ContainerServer) MonitorHistory(ctx context.Context, in *pb.MonitorHistoryRequest) (*pb.MonitorHistoryReply, error) {
	nodeInfo, err := db.QueryNodeByID(in.NodeId)
	if err != nil {
		if err == db.ErrRecordNotFound {
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

func (s *ContainerServer) ListTemplate(ctx context.Context, in *pb.ListTemplateRequest) (*pb.ListTemplateReply, error) {
	reply := &pb.ListTemplateReply{}

	pageinfo, templates, err := db.ListTemplate(ctx, in.PerPage, in.NextPage)
	if err != nil {
		return nil, err
	}
	reply.CurPage = pageinfo.CurPage
	reply.PerPage = pageinfo.PerPage
	reply.TotalPages = pageinfo.TotalPages
	reply.TotalRows = pageinfo.TotalRows

	for _, template := range templates {
		// log.Println(template.Id, template.Name, template.Config_json)
		var containerConfig *pb.ContainerConfigs
		json.Unmarshal([]byte(template.ConfigJSON), &containerConfig)
		templatestruct := pb.ContainerTemplate{Id: template.Id, Conf: containerConfig, NodeId: template.NodeId}
		reply.Data = append(reply.Data, &templatestruct)
	}

	return reply, nil
}

func (s *ContainerServer) CreateTemplate(ctx context.Context, in *pb.CreateTemplateRequest) (*pb.CreateTemplateReply, error) {
	if in.Data == nil || in.Data.Conf == nil || in.Data.NodeId <= 0 {
		return nil, rpc.ErrInvalidArgument
	} else if !args.IsValidContainerName(in.Data.Conf.Name) {
		return nil, status.Errorf(codes.InvalidArgument, "模板名参数错误")
	} else if !args.IsValidContainerDesc(in.Data.Conf.Desc) {
		return nil, status.Errorf(codes.InvalidArgument, "模板描述参数错误")
	}

	reply := pb.CreateTemplateReply{}
	data := in.Data
	id := data.Id
	if id < 0 {
		reply.Id = -1
		return &reply, status.Errorf(codes.InvalidArgument, "id error")
	}

	confbyte, err := json.Marshal(data.Conf)
	if err != nil {
		return nil, err
	}

	id, err = db.CreateTemplate(ctx, id, data.Conf.Name, confbyte, data.NodeId)
	if err != nil {
		log.Println(err)
	}
	reply.Id = id

	return &reply, nil
}

func (s *ContainerServer) UpdateTemplate(ctx context.Context, in *pb.UpdateTemplateRequest) (*pb.UpdateTemplateReply, error) {
	if in.Data == nil || in.Data.Conf == nil || in.Data.NodeId <= 0 {
		return nil, rpc.ErrInvalidArgument
	} else if !args.IsValidContainerName(in.Data.Conf.Name) {
		return nil, status.Errorf(codes.InvalidArgument, "模板名参数错误")
	} else if !args.IsValidContainerDesc(in.Data.Conf.Desc) {
		return nil, status.Errorf(codes.InvalidArgument, "模板描述参数错误")
	}

	reply := pb.UpdateTemplateReply{}
	data := in.Data
	id := data.Id
	if id < 1 {
		return &reply, errors.New("id error")
	}

	confbyte, err := json.Marshal(data.Conf)
	if err != nil {
		return nil, err
	}
	err = db.UpdateTemplate(ctx, id, data.Conf.Name, confbyte, data.NodeId)
	if err != nil {
		log.Println(err)
	}

	return &reply, nil
}

func (s *ContainerServer) RemoveTemplate(ctx context.Context, in *pb.RemoveTemplateRequest) (*pb.RemoveTemplateReply, error) {
	reply := pb.RemoveTemplateReply{}
	if ids := in.GetIds(); ids != nil {
		err := db.RemoveTemplate(ctx, in.Ids)
		if err != nil {
			return nil, err
		}
	} else {
		return nil, errors.New("request is null")
	}

	return &reply, nil
}

func (*ContainerServer) InspectTemplate(ctx context.Context, in *pb.InspectTemplateRequest) (*pb.InspectTemplateReply, error) {
	if in.Id <= 0 {
		log.Infof("InspectTemplate invalid id=%v", in.Id)
		return nil, rpc.ErrInvalidArgument
	}

	data, err := db.FindTemplate(in.Id)
	if err != nil {
		log.Infof("db.FindTemplate err=%v", err)
		if err == db.ErrRecordNotFound {
			return nil, rpc.ErrNotFound
		}
		return nil, rpc.ErrInternal
	}

	var c pb.ContainerConfigs
	if err := json.Unmarshal([]byte(data.ConfigJSON), &c); err != nil {
		log.Warnf("parse json string '%s' err=%v", data.ConfigJSON, err)
		return nil, rpc.ErrInternal
	}

	reply := pb.InspectTemplateReply{
		Data: &pb.ContainerTemplate{
			Id:     data.Id,
			Conf:   &c,
			NodeId: data.NodeId,
		},
	}

	return &reply, nil
}

func (*ContainerServer) CreateBackup(ctx context.Context, in *pb.CreateBackupRequest) (*pb.CreateBackupReply, error) {
	if in.NodeId <= 0 || in.ContainerId == "" {
		log.Infof("CreateBackup invalid input: %+v", in)
		return nil, rpc.ErrInvalidArgument
	}

	nodeInfo, err := db.QueryNodeByID(in.NodeId)
	if err != nil {
		if err == db.ErrRecordNotFound {
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
	subReply, err := cli.CreateBackup(ctx_, in)
	if err != nil {
		log.Infof("agent CreateBackup err=%v", err)
		return nil, rpc.ErrInternal
	}

	return subReply, nil
}

func (*ContainerServer) UpdateBackup(ctx context.Context, in *pb.UpdateBackupRequest) (*pb.UpdateBackupReply, error) {
	if !args.IsValidBackupDesc(in.BackupDesc) {
		return nil, status.Errorf(codes.InvalidArgument, "参数错误：备份描述无效")
	}

	nodeInfo, err := db.QueryNodeByID(in.NodeId)
	if err != nil {
		if err == db.ErrRecordNotFound {
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
	subReply, err := cli.UpdateBackup(ctx_, in)
	if err != nil {
		log.Infof("agent UpdateBackup err=%v", err)
		return nil, rpc.ErrInternal
	}

	return subReply, nil
}

func (*ContainerServer) RemoveBackup(ctx context.Context, in *pb.RemoveBackupRequest) (*pb.RemoveBackupReply, error) {
	nodeInfo, err := db.QueryNodeByID(in.NodeId)
	if err != nil {
		if err == db.ErrRecordNotFound {
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
	subReply, err := cli.RemoveBackup(ctx_, in)
	if err != nil {
		log.Infof("agent RemoveBackup err=%v", err)
		return nil, rpc.ErrInternal
	}

	return subReply, nil
}

func (*ContainerServer) ResumeBackup(ctx context.Context, in *pb.ResumeBackupRequest) (*pb.ResumeBackupReply, error) {
	nodeInfo, err := db.QueryNodeByID(in.NodeId)
	if err != nil {
		if err == db.ErrRecordNotFound {
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
	subReply, err := cli.ResumeBackup(ctx_, in)
	if err != nil {
		log.Infof("agent ResumeBackup err=%v", err)
		return nil, rpc.ErrInternal
	}

	return subReply, nil
}

func (*ContainerServer) ListBackup(ctx context.Context, in *pb.ListBackupRequest) (*pb.ListBackupReply, error) {
	nodeInfo, err := db.QueryNodeByID(in.NodeId)
	if err != nil {
		if err == db.ErrRecordNotFound {
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
	subReply, err := cli.ListBackup(ctx_, in)
	if err != nil {
		log.Infof("agent ListBackup err=%v", err)
		return nil, rpc.ErrInternal
	}

	return subReply, nil
}
