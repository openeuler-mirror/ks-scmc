package internal

import (
	"context"
	"errors"
	"io"
	"sort"
	"strings"
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

type ExportBackupStream struct {
	ss pb.Container_ExportBackupClient
}

func (s ExportBackupStream) Read() ([]byte, error) {
	r, err := s.ss.Recv()
	if err != nil {
		return nil, err
	}

	return r.Data, nil
}

func (s ExportBackupStream) Context() context.Context {
	return s.ss.Context()
}

type ContainerServer struct {
	pb.UnimplementedContainerServer
}

func (s *ContainerServer) List(ctx context.Context, in *pb.ListRequest) (*pb.ListReply, error) {
	nodes, err := db.ListNodes()
	if err != nil {
		return nil, rpc.ErrInternal
	}

	nodeIDs := in.NodeIds
	if len(nodeIDs) == 0 {
		for _, n := range nodes {
			nodeIDs = append(nodeIDs, n.ID)
		}
	}

	var (
		mtx     sync.Mutex
		wg      sync.WaitGroup
		reply   pb.ListReply
		counter = countInt64(nodeIDs)
	)

	for _, n := range nodes {
		if counter[n.ID] == 0 {
			continue
		}

		wg.Add(1)
		go func(n *db.NodeInfo) {
			defer wg.Done()
			conn, err := getAgentConn(n.Address)
			if err != nil {
				mtx.Lock()
				defer mtx.Unlock()
				reply.FailNodes = append(reply.FailNodes, n.Address)
				return
			}

			r, err := pb.NewContainerClient(conn).List(ctx, in)
			if err != nil {
				log.Warnf("container.List node=%+v, err=%v", n, err)
				mtx.Lock()
				defer mtx.Unlock()
				reply.FailNodes = append(reply.FailNodes, n.Address)
				return
			}

			for i := range r.Containers {
				r.Containers[i].NodeId = n.ID
				r.Containers[i].NodeAddress = n.Address
			}
			mtx.Lock()
			defer mtx.Unlock()
			reply.Containers = append(reply.Containers, r.Containers...)
		}(n)
	}

	wg.Wait()
	sort.Slice(reply.Containers, func(i, j int) bool {
		c0 := reply.Containers[i]
		c1 := reply.Containers[j]
		if c0.NodeId != c1.NodeId {
			return c0.NodeId < c1.NodeId
		}

		var n0, n1 string
		if c0.Info != nil {
			n0 = c0.Info.Name
		}
		if c1.Info != nil {
			n1 = c1.Info.Name
		}
		return strings.Compare(n0, n1) < 0
	})
	return &reply, nil
}

func (s *ContainerServer) Create(ctx context.Context, in *pb.CreateRequest) (*pb.CreateReply, error) {
	log.Debugf("container createReply=%v+\n",*in)
	if in.Configs == nil || in.NodeId <= 0 {
		return nil, rpc.ErrInvalidArgument
	} else if !args.ContainerName.Check(in.Configs.Name) {
		return nil, status.Errorf(codes.InvalidArgument, "容器名参数错误")
	} else if !args.ContainerDesc.Check(in.Configs.Desc) {
		return nil, status.Errorf(codes.InvalidArgument, "容器描述参数错误")
	}
	if in.Configs.ResouceLimit != nil {
		r := in.Configs.ResouceLimit
		if !args.Float64Resource.Check(r.CpuLimit, r.DiskLimit, r.MemoryLimit, r.MemorySoftLimit) ||
			!args.Int64Resource.Check(r.CpuPrio) {
			return nil, rpc.ErrContainerResourceArgs
		}
	}

	n, err := db.QueryNodeByID(in.NodeId)
	if err != nil {
		if err == db.ErrRecordNotFound {
			return nil, rpc.ErrNotFound
		}
		return nil, rpc.ErrDatabaseFail
	}

	if err := CheckImageLegal(in.NodeId, in.Configs.Image); err != nil {
		return nil, err
	}

	conn, err := getAgentConn(n.Address)
	if err != nil {
		return nil, rpc.ErrInternal
	}

	return pb.NewContainerClient(conn).Create(ctx, in)
}

func (s *ContainerServer) Start(ctx context.Context, in *pb.StartRequest) (*pb.StartReply, error) {
	if len(in.Ids) <= 0 {
		return nil, rpc.ErrInvalidArgument
	}

	var reply pb.StartReply
	for _, c := range in.Ids {
		containerIDs := uniqueString(c.ContainerIds)
		if c.NodeId <= 0 || len(containerIDs) <= 0 {
			log.Warnf("start container ErrInvalidArgument")
			continue
		}

		n, err := db.QueryNodeByID(c.NodeId)
		if err != nil {
			if err == db.ErrRecordNotFound {
				log.Warnf("start container ErrRecordNotFound")
				continue
			}
			log.Warnf("start container ErrInternal")
			continue
		}

		conn, err := getAgentConn(n.Address)
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
					ContainerIds: containerIDs,
				},
			},
		}

		subReply, err := cli.Start(ctx_, &request)
		if err != nil {
			log.Warnf("start container ErrInternal: %v", err)
			continue
		}

		for i := range subReply.FailInfos {
			subReply.FailInfos[i].NodeInfo = n.Address
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
	if len(in.Ids) <= 0 {
		return nil, rpc.ErrInvalidArgument
	}

	var reply pb.StopReply
	for _, c := range in.Ids {
		containerIDs := uniqueString(c.ContainerIds)
		if c.NodeId <= 0 || len(containerIDs) <= 0 {
			log.Warnf("stop container ErrInvalidArgument")
			continue
		}

		n, err := db.QueryNodeByID(c.NodeId)
		if err != nil {
			if err == db.ErrRecordNotFound {
				log.Warnf("stop container ErrRecordNotFound")
				continue
			}
			log.Warnf("stop container ErrInternal")
			continue
		}

		conn, err := getAgentConn(n.Address)
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
					ContainerIds: containerIDs,
				},
			},
		}

		subReply, err := cli.Stop(ctx_, &request)
		if err != nil {
			log.Warnf("stop container ErrInternal: %v", err)
			continue
		}

		for i := range subReply.FailInfos {
			subReply.FailInfos[i].NodeInfo = n.Address
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
	if len(in.Ids) <= 0 {
		return nil, rpc.ErrInvalidArgument
	}

	var reply pb.RestartReply
	for _, c := range in.Ids {
		containerIDs := uniqueString(c.ContainerIds)
		if c.NodeId <= 0 || len(containerIDs) <= 0 {
			log.Warnf("restart container ErrInvalidArgument")
			continue
		}

		n, err := db.QueryNodeByID(c.NodeId)
		if err != nil {
			if err == db.ErrRecordNotFound {
				log.Warnf("restart container ErrRecordNotFound")
				continue
			}
			log.Warnf("restart container ErrInternal")
			continue
		}

		conn, err := getAgentConn(n.Address)
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
					ContainerIds: containerIDs,
				},
			},
		}

		subReply, err := cli.Restart(ctx_, &request)
		if err != nil {
			log.Warnf("restart container ErrInternal: %v", err)
			continue
		}

		for i := range subReply.FailInfos {
			subReply.FailInfos[i].NodeInfo = n.Address
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
	if len(in.Ids) <= 0 {
		return nil, rpc.ErrInvalidArgument
	}

	var reply pb.RemoveReply
	for _, c := range in.Ids {
		containerIDs := uniqueString(c.ContainerIds)
		if c.NodeId <= 0 || len(containerIDs) <= 0 {
			log.Warnf("remove container ErrInvalidArgument")
			continue
		}

		n, err := db.QueryNodeByID(c.NodeId)
		if err != nil {
			if err == db.ErrRecordNotFound {
				log.Warnf("remove container ErrRecordNotFound")
				continue
			}
			log.Warnf("remove container ErrInternal")
			continue
		}

		conn, err := getAgentConn(n.Address)
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
					ContainerIds: containerIDs,
				},
			},
		}

		subReply, err := cli.Remove(ctx_, &request)
		if err != nil {
			log.Warnf("remove container ErrInternal: %v", err)
			continue
		}

		for i := range subReply.FailInfos {
			subReply.FailInfos[i].NodeInfo = n.Address
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

	n, err := db.QueryNodeByID(in.NodeId)
	if err != nil {
		if err == db.ErrRecordNotFound {
			return nil, rpc.ErrNotFound
		}
		return nil, rpc.ErrDatabaseFail
	}

	conn, err := getAgentConn(n.Address)
	if err != nil {
		return nil, rpc.ErrInternal
	}

	return pb.NewContainerClient(conn).Inspect(ctx, in)
}

func (s *ContainerServer) Update(ctx context.Context, in *pb.UpdateRequest) (*pb.UpdateReply, error) {
	if in.NodeId <= 0 || in.ContainerId == "" {
		return nil, rpc.ErrInvalidArgument
	}
	if in.ResourceLimit != nil {
		r := in.ResourceLimit
		if !args.Float64Resource.Check(r.CpuLimit, r.DiskLimit, r.MemoryLimit, r.MemorySoftLimit) ||
			!args.Int64Resource.Check(r.CpuPrio, r.DiskPrio) {
			return nil, rpc.ErrContainerResourceArgs
		}
	}

	n, err := db.QueryNodeByID(in.NodeId)
	if err != nil {
		if err == db.ErrRecordNotFound {
			return nil, rpc.ErrNotFound
		}
		return nil, rpc.ErrDatabaseFail
	}

	conn, err := getAgentConn(n.Address)
	if err != nil {
		return nil, rpc.ErrInternal
	}

	return pb.NewContainerClient(conn).Update(ctx, in)
}

func (s *ContainerServer) MonitorHistory(ctx context.Context, in *pb.MonitorHistoryRequest) (*pb.MonitorHistoryReply, error) {
	n, err := db.QueryNodeByID(in.NodeId)
	if err != nil {
		if err == db.ErrRecordNotFound {
			return nil, rpc.ErrNotFound
		}
		return nil, rpc.ErrDatabaseFail
	}

	conn, err := getAgentConn(n.Address)
	if err != nil {
		return nil, rpc.ErrInternal
	}

	return pb.NewContainerClient(conn).MonitorHistory(ctx, in)
}

func (s *ContainerServer) ListTemplate(ctx context.Context, in *pb.ListTemplateRequest) (*pb.ListTemplateReply, error) {
	page, data, err := db.ListTemplate(ctx, in.PerPage, in.NextPage)
	if err != nil {
		log.Infof("db.ListTemplate err=%v", err)
		return nil, rpc.ErrDatabaseFail
	}

	reply := pb.ListTemplateReply{
		CurPage:    page.CurPage,
		PerPage:    page.PerPage,
		TotalPages: page.TotalPages,
		TotalRows:  page.TotalRows,
	}

	for _, t := range data {
		reply.Data = append(reply.Data, &pb.ContainerTemplate{
			Id:     t.Id,
			Conf:   t.ConfigJSON,
			NodeId: t.NodeId,
		})
	}

	return &reply, nil
}

func CheckImageLegal(nodeId int64, image string) error {
	images, err := db.GetImageMap()
	if err != nil {
		return rpc.ErrDatabaseFail
	}

	n, err := db.QueryNodeByID(nodeId)
	if err != nil {
		if err == db.ErrRecordNotFound {
			return rpc.ErrNotFound
		}
		return rpc.ErrDatabaseFail
	}

	conn, err := getAgentConn(n.Address)
	if err != nil {
		return rpc.ErrInternal
	}

	backupImages, err := ListBackupImage(conn)
	if err != nil {
		log.Warnf("ListBackupImage err=%v", err)
		return rpc.ErrInternal
	}
	if images[image] == nil && !backupImages[image] {
		return status.Errorf(codes.InvalidArgument, "容器镜像非法")
	}

	return nil
}

func (s *ContainerServer) CreateTemplate(ctx context.Context, in *pb.CreateTemplateRequest) (*pb.CreateTemplateReply, error) {
	if in.Data == nil || in.Data.Conf == nil || in.Data.NodeId <= 0 {
		return nil, rpc.ErrInvalidArgument
	} else if !args.ContainerName.Check(in.Data.Conf.Name) {
		return nil, status.Errorf(codes.InvalidArgument, "模板名参数错误")
	} else if !args.ContainerDesc.Check(in.Data.Conf.Desc) {
		return nil, status.Errorf(codes.InvalidArgument, "模板描述参数错误")
	}

	if err := CheckImageLegal(in.Data.NodeId, in.Data.Conf.Image); err != nil {
		return nil, err
	}

	if in.Data.Conf.ResouceLimit != nil {
		r := in.Data.Conf.ResouceLimit
		if !args.Float64Resource.Check(r.CpuLimit, r.DiskLimit, r.MemoryLimit, r.MemorySoftLimit) ||
			!args.Int64Resource.Check(r.CpuPrio) {
			return nil, rpc.ErrContainerResourceArgs
		}
	}

	id, err := db.CreateTemplate(ctx, &db.ContainerTemplates{
		Name:       in.Data.Conf.Name,
		ConfigJSON: in.Data.Conf,
		NodeId:     in.Data.NodeId,
	})
	if err != nil {
		if err == db.ErrDuplicateKey {
			return nil, rpc.ErrAlreadyExists
		}
		log.Warnf("db.CreateTemplate err=%v", err)
		return nil, rpc.ErrDatabaseFail
	}

	return &pb.CreateTemplateReply{
		Id: id,
	}, nil
}

func (s *ContainerServer) UpdateTemplate(ctx context.Context, in *pb.UpdateTemplateRequest) (*pb.UpdateTemplateReply, error) {
	if in.Data == nil || in.Data.Conf == nil || in.Data.NodeId <= 0 || in.Data.Id <= 0 {
		return nil, rpc.ErrInvalidArgument
	} else if !args.ContainerName.Check(in.Data.Conf.Name) {
		return nil, status.Errorf(codes.InvalidArgument, "模板名参数错误")
	} else if !args.ContainerDesc.Check(in.Data.Conf.Desc) {
		return nil, status.Errorf(codes.InvalidArgument, "模板描述参数错误")
	}
	if in.Data.Conf.ResouceLimit != nil {
		r := in.Data.Conf.ResouceLimit
		if !args.Float64Resource.Check(r.CpuLimit, r.DiskLimit, r.MemoryLimit, r.MemorySoftLimit) ||
			!args.Int64Resource.Check(r.CpuPrio) {
			return nil, rpc.ErrContainerResourceArgs
		}
	}

	data, err := db.FindTemplate(in.Data.Id)
	if err != nil {
		if err == db.ErrRecordNotFound {
			return nil, rpc.ErrNotFound
		}
		return nil, rpc.ErrDatabaseFail
	}

	data.Name = in.Data.Conf.Name
	data.ConfigJSON = in.Data.Conf
	data.NodeId = in.Data.NodeId
	if err := db.UpdateTemplate(ctx, data); err != nil {
		log.Infof("db.UpdateTemplate err=%v", err)
		return nil, rpc.ErrInternal
	}

	return &pb.UpdateTemplateReply{}, nil
}

func (s *ContainerServer) RemoveTemplate(ctx context.Context, in *pb.RemoveTemplateRequest) (*pb.RemoveTemplateReply, error) {
	var reply pb.RemoveTemplateReply
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
		if err == db.ErrRecordNotFound {
			return nil, rpc.ErrNotFound
		}
		log.Infof("db.FindTemplate err=%v", err)
		return nil, rpc.ErrDatabaseFail
	}

	return &pb.InspectTemplateReply{
		Data: &pb.ContainerTemplate{
			Id:     data.Id,
			Conf:   data.ConfigJSON,
			NodeId: data.NodeId,
		},
	}, nil
}

func (*ContainerServer) CreateBackup(ctx context.Context, in *pb.CreateBackupRequest) (*pb.CreateBackupReply, error) {
	if in.NodeId <= 0 || in.ContainerId == "" {
		log.Infof("CreateBackup invalid input: %+v", in)
		return nil, rpc.ErrInvalidArgument
	}

	n, err := db.QueryNodeByID(in.NodeId)
	if err != nil {
		if err == db.ErrRecordNotFound {
			return nil, rpc.ErrNotFound
		}
		return nil, rpc.ErrDatabaseFail
	}

	conn, err := getAgentConn(n.Address)
	if err != nil {
		return nil, rpc.ErrInternal
	}

	return pb.NewContainerClient(conn).CreateBackup(ctx, in)
}

func (*ContainerServer) UpdateBackup(ctx context.Context, in *pb.UpdateBackupRequest) (*pb.UpdateBackupReply, error) {
	n, err := db.QueryNodeByID(in.NodeId)
	if err != nil {
		if err == db.ErrRecordNotFound {
			return nil, rpc.ErrNotFound
		}
		return nil, rpc.ErrDatabaseFail
	}

	conn, err := getAgentConn(n.Address)
	if err != nil {
		return nil, rpc.ErrInternal
	}

	return pb.NewContainerClient(conn).UpdateBackup(ctx, in)
}

func (*ContainerServer) RemoveBackup(ctx context.Context, in *pb.RemoveBackupRequest) (*pb.RemoveBackupReply, error) {
	n, err := db.QueryNodeByID(in.NodeId)
	if err != nil {
		if err == db.ErrRecordNotFound {
			return nil, rpc.ErrNotFound
		}
		return nil, rpc.ErrDatabaseFail
	}

	conn, err := getAgentConn(n.Address)
	if err != nil {
		return nil, rpc.ErrInternal
	}

	return pb.NewContainerClient(conn).RemoveBackup(ctx, in)
}

func (*ContainerServer) ResumeBackup(ctx context.Context, in *pb.ResumeBackupRequest) (*pb.ResumeBackupReply, error) {
	n, err := db.QueryNodeByID(in.NodeId)
	if err != nil {
		if err == db.ErrRecordNotFound {
			return nil, rpc.ErrNotFound
		}
		return nil, rpc.ErrDatabaseFail
	}

	conn, err := getAgentConn(n.Address)
	if err != nil {
		return nil, rpc.ErrInternal
	}

	return pb.NewContainerClient(conn).ResumeBackup(ctx, in)
}

func (*ContainerServer) ListBackup(ctx context.Context, in *pb.ListBackupRequest) (*pb.ListBackupReply, error) {
	n, err := db.QueryNodeByID(in.NodeId)
	if err != nil {
		if err == db.ErrRecordNotFound {
			return nil, rpc.ErrNotFound
		}
		return nil, rpc.ErrDatabaseFail
	}

	conn, err := getAgentConn(n.Address)
	if err != nil {
		return nil, rpc.ErrInternal
	}

	return pb.NewContainerClient(conn).ListBackup(ctx, in)
}

func (*ContainerServer) exportBackupImageToClient(r pb.Container_ExportBackupClient, w pb.Container_ExportBackupServer) error {
	for {
		if m, err := r.Recv(); err != nil {
			if err == io.EOF {
				return nil // read internal stream finished
			}
			log.Error("ExportBackup read from internal stream: ", err)
			return rpc.ErrInternal
		} else {
			if err := w.Send(m); err != nil {
				log.Error("ExportBackup send data to client: ", err)
				return rpc.ErrInternal
			}
		}
	}
}

func (*ContainerServer) exportBackupImageToRegistry(img *db.ImageInfo, stream pb.Container_ExportBackupClient) error {
	f, err := NewImageFile(img.Name, img.Version, img.FileSize)
	if err != nil {
		log.Errorf("NewImageFile err=%v", err)
		return rpc.ErrInternal
	}

	if err := f.ReadStream(ExportBackupStream{stream}); err != nil {
		log.Errorf("Write stream data to image file err=%v", err)
		f.Destroy()
		return status.Errorf(codes.InvalidArgument, "读写镜像数据失败：%s", err.Error())
	}

	img.FilePath = f.Path()
	img.CheckSum = f.SHA256()
	img.FileSize = f.Size()
	img.ImageId = f.DockerID()

	if _, err := db.CreateImage(img); err != nil {
		log.Infof("db.CreateImages image=%+v err=%v", img, err)
		f.Destroy()
		return rpc.ErrDatabaseFail
	}

	is := ImageServer{}
	is.noticeNewImage(img)

	f.Close()
	return nil
}

func (s *ContainerServer) ExportBackup(in *pb.ExportBackupRequest, stream pb.Container_ExportBackupServer) error {
	if in.NodeId == 0 || in.BackupId == 0 {
		return rpc.ErrInvalidArgument
	}

	n, err := db.QueryNodeByID(in.NodeId)
	if err != nil {
		log.Infof("QueryNodeByID err=%v", err)
		if err == db.ErrRecordNotFound {
			return rpc.ErrNotFound
		}
		return rpc.ErrInternal
	}

	conn, err := getAgentConn(n.Address)
	if err != nil {
		log.Infof("getAgentConn err=%v", err)
		return rpc.ErrInternal
	}

	var img *db.ImageInfo
	if !in.IsDownload {
		if !args.ImageVersion.Check(in.ImgVersion) {
			return rpc.ErrInvalidArgument
		}

		approve := db.ApprovalWait
		ss, err := db.GetSecuritySwitch()
		if err != nil {
			return rpc.ErrInternal
		}
		if !ss {
			approve = db.ApprovalPass
		}

		b, err := pb.NewContainerClient(conn).InspectBackup(stream.Context(), &pb.InspectBackupRequest{BackupId: in.BackupId})
		if err != nil {
			log.Infof("InspectBackup err=%v", err)
			return err
		}

		if b.Backup == nil {
			return rpc.ErrInternal
		}
		imageName := strings.Split(b.Backup.ImageRef, ":")[0]

		// 检查相同版本的镜像是否已存在
		i, err := db.QueryImageByNameVersion(imageName, in.ImgVersion)
		if err != nil {
			log.Infof("QueryImageByNameVersion err=%v", err)
			return rpc.ErrDatabaseFail
		} else if i != nil {
			return status.Errorf(codes.InvalidArgument, "镜像版本已存在")
		}

		img = &db.ImageInfo{
			Name:           imageName,
			Version:        in.ImgVersion,
			Description:    in.Description,
			VerifyStatus:   db.VerifyPass,
			ApprovalStatus: int32(approve),
			FileSize:       imageMaxSize(),
		}
	}

	c, err := pb.NewContainerClient(conn).ExportBackup(stream.Context(), in)
	if err != nil {
		log.Infof("rpc ExportBackupJob err=%v", err)
		return err
	}

	// 客户端直接下载
	if in.IsDownload {
		return s.exportBackupImageToClient(c, stream)
	}

	// 上传镜像管理
	return s.exportBackupImageToRegistry(img, c)
}

func (*ContainerServer) AddAppEntry(ctx context.Context, in *pb.AddAppEntryRequest) (*pb.AddAppEntryReply, error) {
	if in.NodeId == 0 {
		return nil, rpc.ErrInvalidArgument
	}

	n, err := db.QueryNodeByID(in.NodeId)
	if err != nil {
		if err == db.ErrRecordNotFound {
			log.Infof("node info not found, id=%v", in.NodeId)
		}
		log.Infof("QueryNodeByID err=%v", err)
		return nil, rpc.ErrDatabaseFail
	}

	conn, err := getAgentConn(n.Address)
	if err != nil {
		log.Infof("getAgentConn error=%v", err)
		return nil, rpc.ErrInternal
	}

	return pb.NewContainerClient(conn).AddAppEntry(ctx, in)
}

func (*ContainerServer) RemoveAppEntry(ctx context.Context, in *pb.RemoveAppEntryRequest) (*pb.RemoveAppEntryReply, error) {
	if in.NodeId == 0 {
		return nil, rpc.ErrInvalidArgument
	}

	n, err := db.QueryNodeByID(in.NodeId)
	if err != nil {
		if err == db.ErrRecordNotFound {
			log.Infof("node info not found, id=%v", in.NodeId)
		}
		log.Infof("QueryNodeByID err=%v", err)
		return nil, rpc.ErrDatabaseFail
	}

	conn, err := getAgentConn(n.Address)
	if err != nil {
		log.Infof("getAgentConn error=%v", err)
		return nil, rpc.ErrInternal
	}

	return pb.NewContainerClient(conn).RemoveAppEntry(ctx, in)
}

func (*ContainerServer) UpdateAppEntry(ctx context.Context, in *pb.UpdateAppEntryRequest) (*pb.UpdateAppEntryReply, error) {
	if in.NodeId == 0 {
		return nil, rpc.ErrInvalidArgument
	}

	n, err := db.QueryNodeByID(in.NodeId)
	if err != nil {
		if err == db.ErrRecordNotFound {
			log.Infof("node info not found, id=%v", in.NodeId)
		}
		log.Infof("QueryNodeByID err=%v", err)
		return nil, rpc.ErrDatabaseFail
	}

	conn, err := getAgentConn(n.Address)
	if err != nil {
		log.Infof("getAgentConn error=%v", err)
		return nil, rpc.ErrInternal
	}

	return pb.NewContainerClient(conn).UpdateAppEntry(ctx, in)
}

func (*ContainerServer) ListAppEntry(ctx context.Context, in *pb.ListAppEntryRequest) (*pb.ListAppEntryReply, error) {
	if in.NodeId == 0 {
		return nil, rpc.ErrInvalidArgument
	}

	n, err := db.QueryNodeByID(in.NodeId)
	if err != nil {
		if err == db.ErrRecordNotFound {
			log.Infof("node info not found, id=%v", in.NodeId)
		}
		log.Infof("QueryNodeByID err=%v", err)
		return nil, rpc.ErrDatabaseFail
	}

	conn, err := getAgentConn(n.Address)
	if err != nil {
		log.Infof("getAgentConn error=%v", err)
		return nil, rpc.ErrInternal
	}

	return pb.NewContainerClient(conn).ListAppEntry(ctx, in)
}

func (*ContainerServer) RunAppEntry(ctx context.Context, in *pb.RunAppEntryRequest) (*pb.RunAppEntryReply, error) {
	if in.NodeId == 0 {
		return nil, rpc.ErrInvalidArgument
	}

	n, err := db.QueryNodeByID(in.NodeId)
	if err != nil {
		if err == db.ErrRecordNotFound {
			log.Infof("node info not found, id=%v", in.NodeId)
		}
		log.Infof("QueryNodeByID err=%v", err)
		return nil, rpc.ErrDatabaseFail
	}

	conn, err := getAgentConn(n.Address)
	if err != nil {
		log.Infof("getAgentConn error=%v", err)
		return nil, rpc.ErrInternal
	}

	return pb.NewContainerClient(conn).RunAppEntry(ctx, in)
}

func (*ContainerServer) KillAppEntry(ctx context.Context, in *pb.KillAppEntryRequest) (*pb.KillAppEntryReply, error) {
	if in.NodeId == 0 {
		return nil, rpc.ErrInvalidArgument
	}

	n, err := db.QueryNodeByID(in.NodeId)
	if err != nil {
		if err == db.ErrRecordNotFound {
			log.Infof("node info not found, id=%v", in.NodeId)
		}
		log.Infof("QueryNodeByID err=%v", err)
		return nil, rpc.ErrDatabaseFail
	}

	conn, err := getAgentConn(n.Address)
	if err != nil {
		log.Infof("getAgentConn error=%v", err)
		return nil, rpc.ErrInternal
	}

	return pb.NewContainerClient(conn).KillAppEntry(ctx, in)
}
