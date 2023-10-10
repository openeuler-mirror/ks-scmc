package internal

import (
	"context"
	"scmc/model"
	"scmc/rpc"
	pb "scmc/rpc/pb/logging"

	log "github.com/sirupsen/logrus"
)

type LoggingServer struct {
	pb.UnimplementedLoggingServer
}

func (s *LoggingServer) ListRuntime(ctx context.Context, in *pb.ListRuntimeRequest) (*pb.ListRuntimeReply, error) {
	if in.StartTime < 0 || in.StartTime > in.EndTime {
		log.Info("ListRuntime invalid timestamp args.")
		return nil, rpc.ErrInvalidArgument
	}

	p, data, err := model.ListRuntimeLog(in.PageSize, in.PageNo, in.StartTime, in.EndTime, in.NodeId, in.EventModule)
	if err != nil {
		log.Infof("ListRuntime db err=%v", err)
		return nil, rpc.ErrInternal
	}

	var reply pb.ListRuntimeReply
	for _, l := range data {
		reply.Logs = append(reply.Logs, &pb.RuntimeLog{
			Id:          l.ID,
			NodeId:      l.NodeId,
			NodeInfo:    l.NodeInfo,
			Username:    l.Username,
			EventType:   l.EventType,
			EventModule: l.EventModule,
			Detail:      l.Detail,
			Target:      l.Target,
			StatusCode:  l.StatusCode,
			Error:       l.Error,
			CreatedAt:   l.CreatedAt,
			UpdatedAt:   l.UpdatedAt,
		})
	}
	reply.PageNo = p.PageNo
	reply.PageSize = p.PageSize
	reply.TotalPages = p.TotalPages

	return &reply, nil
}

func (s *LoggingServer) ListWarn(ctx context.Context, in *pb.ListWarnRequest) (*pb.ListWarnReply, error) {
	p, data, err := model.ListWarnLog(in.PageSize, in.PageNo, in.NodeId, in.EventModule)
	if err != nil {
		log.Infof("ListRuntime db err=%v", err)
		return nil, rpc.ErrInternal
	}

	var reply pb.ListWarnReply
	for _, w := range data {
		reply.Logs = append(reply.Logs, &pb.WarnLog{
			Id:            w.ID,
			NodeId:        w.NodeId,
			NodeInfo:      w.NodeInfo,
			EventType:     w.EventType,
			EventModule:   w.EventModule,
			ContainerName: w.ContainerName,
			Detail:        w.Detail,
			HaveRead:      w.HaveRead,
			CreatedAt:     w.CreatedAt,
			UpdatedAt:     w.UpdatedAt,
		})
	}
	reply.PageNo = p.PageNo
	reply.PageSize = p.PageSize
	reply.TotalPages = p.TotalPages

	return &reply, nil
}

func (s *LoggingServer) ReadWarn(ctx context.Context, in *pb.ReadWarnRequest) (*pb.ReadWarnReply, error) {
	if len(in.Ids) > 50 {
		log.Infof("ReadWarn input id length=%v", len(in.Ids))
		return nil, rpc.ErrInvalidArgument
	}

	if err := model.SetWarnLogRead(in.Ids); err != nil {
		log.Infof("model.SetWarnLogRead err=%v", err)
		return nil, rpc.ErrInternal
	}

	return &pb.ReadWarnReply{}, nil
}
