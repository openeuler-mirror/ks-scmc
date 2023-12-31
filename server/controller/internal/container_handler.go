package internal

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"scmc/common"
	"scmc/model"
	"scmc/rpc"
	pb "scmc/rpc/pb/container"
	"scmc/rpc/pb/user"
	"scmc/server"
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
		go func(ctx context.Context, node *model.NodeInfo) {
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
	} else if !isValidContainerName(in.Configs.Name) {
		return nil, status.Errorf(codes.InvalidArgument, "容器名参数错误")
	} else if !isValidContainerDesc(in.Configs.Desc) {
		return nil, status.Errorf(codes.InvalidArgument, "容器描述参数错误")
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

	cfgs := model.ContainerConfigs{
		NodeID:        nodeInfo.ID,
		UUID:          uuid.New().String(),
		ContainerName: in.Configs.Name,
	}

	if in.Configs.SecurityConfig != nil {
		if !isEmptySecurityConfig(in.Configs.SecurityConfig) && common.NeedCheckAuth() && common.NeedCheckPerm() {
			perms, _ := ctx.Value("PERMS").([]*user.Permission)
			if !server.HasPerm(user.PERMISSION_CONTAINER_CONF_SEC, perms) {
				return nil, rpc.ErrContainerSecurityConfigNoPerm
			}
		}

		if data, err := json.Marshal(in.Configs.SecurityConfig); err != nil {
			log.Warnf("Marshal SecurityConfig err: %v", err)
			return nil, rpc.ErrInternal
		} else {
			cfgs.SecurityConfig = string(data)
		}
	}

	if err := model.CreateContainerConfigs(&cfgs); err != nil {
		log.Infof("Create: CreateContainerConfigs err=%v", err)
		return nil, rpc.ErrInternal
	}

	in.Configs.Uuid = cfgs.UUID

	cli := pb.NewContainerClient(conn)
	ctx_, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	agentReply, err := cli.Create(ctx_, in)
	if err != nil && agentReply == nil {
		log.Warnf("Create container: %v", err)
		model.RemoveContainerConfigsByID(cfgs.ID)
		return nil, err
	}

	cfgs.ContainerID = agentReply.ContainerId
	if err := model.UpdateContainerConfigs(&cfgs); err != nil {
		log.Infof("Create: UpdateContainerConfigs data=%+v err=%v", cfgs, err)
		// still return OK
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

		configs, err := model.GetContainerConfigsList(nodeId, containerIds)
		if err != nil {
			log.Warnf("model.GetContainerConfigsList err=%v", err)
			continue
		}

		var imageIds []string
		var uuids []string
		for _, v := range configs {
			backup, err := model.QueryContainerBackupByUUID(v.UUID)
			if err != nil {
				log.Warnf("model.QueryContainerBackupByID err=%v", err)
				continue
			}

			uuids = append(uuids, v.UUID)
			for _, v := range backup {
				imageIds = append(imageIds, v.ImageID)
			}
		}

		cli := pb.NewContainerClient(conn)
		ctx_, cancel := context.WithTimeout(context.Background(), time.Second*10)
		defer cancel()

		request := pb.RemoveRequest{
			Ids: []*pb.ContainerIdList{
				{
					ContainerIds:   containerIds,
					BackupImageIds: imageIds,
				},
			},
		}

		subReply, err := cli.Remove(ctx_, &request)
		if err != nil {
			log.Warnf("remove container ErrInternal: %v", err)
			// TODO remove container configs and backups
			continue
		}

		for i := range subReply.FailInfos {
			subReply.FailInfos[i].NodeInfo = nodeInfo.Address
		}
		reply.FailInfos = append(reply.FailInfos, subReply.FailInfos...)
		reply.OkIds = append(reply.OkIds, subReply.OkIds...)

		if err = model.RemoveContainerConfigs(c.NodeId, containerIds); err != nil {
			log.Warnf("remove container configs err: %v", err)
		}

		if err = model.RemoveContainerBackupByUUID(uuids); err != nil {
			log.Warnf("remove container backup err: %v", err)
		}
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
		return nil, err
	}

	data, err := model.GetContainerConfigs(in.NodeId, in.ContainerId)
	if err != nil {
		log.Infof("db get container configs id=%v err=%v", in.ContainerId, err)
	} else {
		var secCfgs pb.SecurityConfig
		if err := json.Unmarshal([]byte(data.SecurityConfig), &secCfgs); err != nil {
			log.Infof("unmarshal security config err=%v", err)
		} else {
			if agentReply.Configs != nil {
				agentReply.Configs.SecurityConfig = &secCfgs
			}
		}
	}
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
	if common.NeedCheckAuth() && common.NeedCheckPerm() {
		perms, _ := ctx.Value("PERMS").([]*user.Permission)

		ctx_, cancel := context.WithTimeout(context.Background(), time.Second*10)
		defer cancel()
		inspect, err := cli.Inspect(ctx_, &pb.InspectRequest{ContainerId: in.ContainerId})
		if err != nil {
			log.Warnf("Container.Update inspect container=%v err=%v", in.ContainerId, err)
			return nil, err
		}

		if containerBasicConfigDiff(inspect.Configs, in) && !server.HasPerm(user.PERMISSION_CONTAINER_CONF_BASIC, perms) {
			log.Infof("no permission to update container basic config")
			return nil, rpc.ErrContainerBasicConfigNoPerm
		}

		cfgs, err := model.GetContainerConfigs(in.NodeId, in.ContainerId)
		if err != nil {
			log.Infof("db get container configs id=%v err=%v", in.ContainerId, err)
		} else {
			var secCfgs pb.SecurityConfig
			if err := json.Unmarshal([]byte(cfgs.SecurityConfig), &secCfgs); err != nil {
				log.Infof("unmarshal security config err=%v", err)
			} else {
				if containerSecurityConfigDiff(&secCfgs, in.SecurityConfig) && !server.HasPerm(user.PERMISSION_CONTAINER_CONF_SEC, perms) {
					log.Infof("no permission to update container security config")
					return nil, rpc.ErrContainerSecurityConfigNoPerm
				}
			}
		}
	}

	ctx_, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	agentReply, err := cli.Update(ctx_, in)
	if err != nil {
		log.Warnf("Update container: %v", err)
		return nil, err
	}

	if in.SecurityConfig != nil {
		if data, err := json.Marshal(in.SecurityConfig); err != nil {
			log.Warnf("Marshal SecurityConfig err: %v", err)
		} else {
			if containerConfigs, err := model.GetContainerConfigs(in.NodeId, in.ContainerId); err == nil {
				containerConfigs.SecurityConfig = string(data)
				if err := model.UpdateContainerConfigs(containerConfigs); err != nil {
					log.Infof("Update: UpdateContainerConfigs node_id=%v container_id=%s err=%v", in.NodeId, in.ContainerId, err)
				}
			}
		}
	}

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

func (s *ContainerServer) ListTemplate(ctx context.Context, in *pb.ListTemplateRequest) (*pb.ListTemplateReply, error) {
	reply := &pb.ListTemplateReply{}

	pageinfo, templates, err := model.ListTemplate(ctx, in.PerPage, in.NextPage)
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
	} else if !isValidContainerName(in.Data.Conf.Name) {
		return nil, status.Errorf(codes.InvalidArgument, "模板名参数错误")
	} else if !isValidContainerDesc(in.Data.Conf.Desc) {
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

	id, err = model.CreateTemplate(ctx, id, data.Conf.Name, confbyte, data.NodeId)
	if err != nil {
		log.Println(err)
	}
	reply.Id = id

	return &reply, nil
}

func (s *ContainerServer) UpdateTemplate(ctx context.Context, in *pb.UpdateTemplateRequest) (*pb.UpdateTemplateReply, error) {
	if in.Data == nil || in.Data.Conf == nil || in.Data.NodeId <= 0 {
		return nil, rpc.ErrInvalidArgument
	} else if !isValidContainerName(in.Data.Conf.Name) {
		return nil, status.Errorf(codes.InvalidArgument, "模板名参数错误")
	} else if !isValidContainerDesc(in.Data.Conf.Desc) {
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
	err = model.UpdateTemplate(ctx, id, data.Conf.Name, confbyte, data.NodeId)
	if err != nil {
		log.Println(err)
	}

	return &reply, nil
}

func (s *ContainerServer) RemoveTemplate(ctx context.Context, in *pb.RemoveTemplateRequest) (*pb.RemoveTemplateReply, error) {
	reply := pb.RemoveTemplateReply{}
	if ids := in.GetIds(); ids != nil {
		err := model.RemoveTemplate(ctx, in.Ids)
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

	data, err := model.FindTemplate(in.Id)
	if err != nil {
		log.Infof("model.FindTemplate err=%v", err)
		if err == model.ErrRecordNotFound {
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

	if !isValidBackupDesc(in.BackupDesc) {
		return nil, status.Errorf(codes.InvalidArgument, "参数错误：备份描述无效")
	}

	configs, err := model.GetContainerConfigs(in.NodeId, in.ContainerId)
	if err != nil {
		log.Infof("model.GetContainerConfigs err=%v", err)
		return nil, rpc.ErrInternal
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

	now := time.Now()
	backupName := now.Format("20060102150405") + fmt.Sprintf("%d", now.Nanosecond()/1000000)
	backup, err := model.CreateContainerBackup(in.NodeId, configs.UUID, backupName, in.BackupDesc)
	if err != nil {
		return nil, rpc.ErrInternal
	}

	cli := pb.NewContainerClient(conn)
	ctx_, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	_, err = cli.AddBackupJob(ctx_, &pb.AddBackupJobRequest{
		Id:          backup.ID,
		ContainerId: in.ContainerId,
		BackupName:  backupName,
	})
	if err != nil {
		// model set backup failed
		backup.Status = 2
		model.UpdateContainerBackup(backup)
		return nil, rpc.ErrInternal
	}

	model.UpdateContainerBackup(backup)

	return &pb.CreateBackupReply{}, nil

}

func (*ContainerServer) UpdateBackup(ctx context.Context, in *pb.UpdateBackupRequest) (*pb.UpdateBackupReply, error) {
	if !isValidBackupDesc(in.BackupDesc) {
		return nil, status.Errorf(codes.InvalidArgument, "参数错误：备份描述无效")
	}

	data, err := model.QueryContainerBackupByID(in.Id)
	if err != nil {
		log.Infof("model.QueryContainerBackupByID err=%v", err)
		if err == model.ErrRecordNotFound {
			return nil, rpc.ErrNotFound
		}
		return nil, rpc.ErrInternal
	}

	data.BackupDesc = in.BackupDesc
	if err := model.UpdateContainerBackup(data); err != nil {
		log.Infof("model.UpdateContainerBackup err=%v", err)
		return nil, rpc.ErrInternal
	}

	return &pb.UpdateBackupReply{}, nil
}

func (*ContainerServer) RemoveBackup(ctx context.Context, in *pb.RemoveBackupRequest) (*pb.RemoveBackupReply, error) {
	backup, err := model.QueryContainerBackupByID(in.Id)
	if err != nil {
		log.Infof("model.QueryContainerBackupByID err=%v", err)
		if err == model.ErrRecordNotFound {
			return nil, rpc.ErrNotFound
		}
		return nil, rpc.ErrInternal
	}

	cc, err := model.GetContainerConfigsByUUID(backup.UUID)
	if err != nil {
		log.Infof("model.GetContainerConfigsByUUID err=%v", err)
		if err == model.ErrRecordNotFound {
			return nil, rpc.ErrNotFound
		}
		return nil, rpc.ErrInternal
	}

	if cc.NodeID != backup.NodeID {
		return nil, rpc.ErrInternal
	}

	nodeInfo, err := model.QueryNodeByID(backup.NodeID)
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

	ci, err := cli.Inspect(ctx_, &pb.InspectRequest{
		NodeId:      backup.ID,
		ContainerId: cc.ContainerID,
	})
	if err != nil {
		return nil, rpc.ErrInternal
	}

	if ci.Configs.GetImage() == backup.ImageRef {
		return nil, rpc.ErrRemoveContainerBackupWhenRunning
	}

	if _, err := cli.RemoveBackup(ctx_, &pb.RemoveBackupRequest{
		Id:       in.Id,
		ImageRef: backup.ImageRef,
	}); err != nil {
		log.Warnf("agent RemoveBackup image=%v err=%v", backup.ImageRef, err)
		return nil, rpc.ErrInternal
	}

	if err := model.RemoveContainerBackup(backup); err != nil {
		log.Warnf("db remove backup record err=%v", err)
		return nil, rpc.ErrInternal
	}

	return &pb.RemoveBackupReply{}, nil
}

func (*ContainerServer) ResumeBackup(ctx context.Context, in *pb.ResumeBackupRequest) (*pb.ResumeBackupReply, error) {
	backup, err := model.QueryContainerBackupByID(in.BackupId)
	if err != nil {
		log.Infof("model.QueryContainerBackupByID err=%v", err)
		if err == model.ErrRecordNotFound {
			return nil, rpc.ErrNotFound
		}
		return nil, rpc.ErrInternal
	}

	if backup.Status != int8(pb.BACKUP_STATUS_SUCCEED) {
		log.Infof("backup invalid status %+v", backup)
		return nil, rpc.ErrInvalidArgument
	}

	cfgs, err := model.GetContainerConfigsByUUID(backup.UUID)
	if err != nil {
		log.Warnf("db get container configs uuid=%v err=%v", backup.UUID, err)
		return nil, rpc.ErrInternal
	}

	var secCfg pb.SecurityConfig
	if err := json.Unmarshal([]byte(cfgs.SecurityConfig), &secCfg); err != nil {
		log.Infof("json unmarshal security config err=%v", err)
	}

	nodeInfo, err := model.QueryNodeByID(backup.NodeID)
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
	subReply, err := cli.ResumeBackup(ctx_, &pb.ResumeBackupRequest{
		ContainerId:    in.ContainerId,
		ImageRef:       backup.ImageRef,
		SecurityConfig: &secCfg,
	})
	if err != nil {
		log.Warnf("agent ResumeBackup failed err=%v", err)
		return nil, err
	}

	cfgs.ContainerID = subReply.ContainerId
	model.UpdateContainerConfigs(cfgs)

	return subReply, nil
}

func (*ContainerServer) ListBackup(ctx context.Context, in *pb.ListBackupRequest) (*pb.ListBackupReply, error) {
	configs, err := model.GetContainerConfigs(in.NodeId, in.ContainerId)
	if err != nil {
		log.Infof("model.GetContainerConfigs err=%v", err)
		return nil, rpc.ErrInternal
	}

	data, err := model.QueryContainerBackupByUUID(configs.UUID)
	if err != nil {
		log.Infof("model.QueryContainerBackupByUUID err=%v", err)
		return nil, rpc.ErrInternal
	}

	var reply pb.ListBackupReply
	for _, b := range data {
		reply.Data = append(reply.Data, &pb.ContainerBackup{
			Id:         b.ID,
			Uuid:       b.UUID,
			BackupName: b.BackupName,
			BackupDesc: b.BackupDesc,
			ImageRef:   b.ImageRef,
			ImageSize:  b.ImageSize,
			Status:     int64(b.Status),
			CreatedAt:  b.CreatedAt,
		})
	}

	return &reply, nil
}
