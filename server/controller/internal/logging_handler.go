package internal

import (
	"context"
	"strconv"

	log "github.com/sirupsen/logrus"

	"scmc/model/db"
	"scmc/model/warnlog"
	"scmc/rpc"
	pb "scmc/rpc/pb/logging"
)

type LoggingServer struct {
	pb.UnimplementedLoggingServer
}

func (s *LoggingServer) SubscribeWarn(in *pb.SubscribeWarnRequest, stream pb.Logging_SubscribeWarnServer) error {

	ch := warnlog.Subscribe()
	defer warnlog.Evict(ch)

	for {
		select {
		case message := <-ch:
			var reply pb.SubscribeWarnReply
			if m, ok := message.([]*db.WarnLog); ok {
				for _, log := range m {
					reply.Logs = append(reply.Logs, &pb.WarnLog{
						Id:            log.ID,
						NodeId:        log.NodeId,
						NodeInfo:      log.NodeInfo,
						EventType:     log.EventType,
						EventModule:   log.EventModule,
						ContainerName: log.ContainerName,
						Detail:        log.Detail,
						HaveRead:      log.HaveRead,
						CreatedAt:     log.CreatedAt,
						UpdatedAt:     log.UpdatedAt,
					})
				}

				if err := stream.Send(&reply); err != nil {
					return err
				}
			}
		case <-stream.Context().Done():
			return nil
		}
	}

}

func (s *LoggingServer) ListRuntime(ctx context.Context, in *pb.ListRuntimeRequest) (*pb.ListRuntimeReply, error) {

	if in.StartTime < 0 || in.StartTime > in.EndTime {
		log.Info("ListRuntime invalid timestamp args.")
		return nil, rpc.ErrInvalidArgument
	}

	var f *db.LogFilter
	if in.Filter != nil {
		if in.Filter.Property != "target" && in.Filter.Property != "detail" {
			return nil, rpc.ErrInvalidArgument
		}
		f = &db.LogFilter{
			Property: in.Filter.Property,
			Query:    in.Filter.Query,
			Fuzzy:    in.Filter.Fuzzy,
		}
	}

	p, data, err := db.ListRuntimeLog(in.PageSize, in.PageNo, in.StartTime, in.EndTime, in.NodeId, in.EventModule, f)
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

	p, data, err := db.ListWarnLog(in.PageSize, in.PageNo, in.NodeId, in.EventModule)
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
	nodes, err := db.ListNodes()
	if err == nil {
		for _, n := range nodes {
			reply.TotalWarnnings += n.UnreadWarn
		}
	}

	return &reply, nil
}

func (s *LoggingServer) ReadWarn(ctx context.Context, in *pb.ReadWarnRequest) (*pb.ReadWarnReply, error) {

	if len(in.Ids) > 50 {
		log.Infof("ReadWarn input id length=%v", len(in.Ids))
		return nil, rpc.ErrInvalidArgument
	}

	if err := db.SetWarnLogRead(in.Ids); err != nil {
		log.Infof("db.SetWarnLogRead err=%v", err)
		return nil, rpc.ErrInternal
	}

	return &pb.ReadWarnReply{}, nil
}

func (s *LoggingServer) SetLog(ctx context.Context, in *pb.SetLogRequest) (*pb.SetLogReply, error) {

	if in.SaveLogDays < 1 || in.SaveLogDays > 999 {
		log.Infof("SetLog invalid parameter save_log_days: %v", in.SaveLogDays)
		return nil, rpc.ErrInvalidArgument
	}

	if err := db.SetLog("saveLogDays", strconv.FormatInt(in.SaveLogDays, 10)); err != nil {
		log.Infof("db.SetLog err=%v", err)
		return nil, rpc.ErrInternal
	}

	return &pb.SetLogReply{}, nil
}

func (s *LoggingServer) GetLog(ctx context.Context, in *pb.GetLogRequest) (*pb.GetLogReply, error) {

	value, err := db.GetLog("saveLogDays")
	if err != nil {
		log.Infof("db.GetLog err=%v", err)
		return nil, rpc.ErrInternal
	}

	num, err := strconv.ParseInt(value, 10, 64)
	if err != nil || num < 1 || num > 999 {
		log.Warnf("db.GetLog value invalid, err: %v value: %v", err, value)
		return nil, rpc.ErrInternal
	}

	return &pb.GetLogReply{SaveLogDays: num,}, nil
}