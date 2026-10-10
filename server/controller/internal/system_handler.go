package internal

import (
	"context"
	"scmc/model/db"
	"scmc/model/session"
	"scmc/rpc"
	"scmc/rpc/middleware/auth"
	pb "scmc/rpc/pb/system"

	log "github.com/sirupsen/logrus"
)

type SystemServer struct {
	pb.UnimplementedSystemServer
}

func (s *SystemServer) Ping(ctx context.Context, req *pb.PingRequest) (*pb.PingReply, error) {
	return &pb.PingReply{}, nil
}

func (s *SystemServer) Subscribe(req *pb.SubscribeRequest, stream pb.System_SubscribeServer) error {
	ctx := stream.Context()

	userID, sessionKey, ok := auth.UserSessionFromContext(ctx)
	if !ok {
		log.Warnf("context no user session, ctx=%+v", ctx)
		return rpc.ErrInternal
	}

	sess, err := db.QuerySession(userID, sessionKey)
	if err != nil {
		log.Infof("QuerySession err=%v", err)
		return rpc.ErrDatabaseFail
	} else if sess == nil {
		log.Info("QuerySession session not found")
		return rpc.ErrInternal
	}

	c := make(chan int)
	defer close(c)

	session.Timers.Add(sessionKey, sess.ExpiredAt, c)

	<-c
	msg := pb.SubscribeReply{
		MsgType:    int64(pb.SubscribeMessage_UserSessionExpire),
		MsgContent: "用户会话已过期，或已在其他位置登录",
	}
	if err := stream.Send(&msg); err != nil {
		log.Warnf("Subscribe send to stream err=%v", err)
		return rpc.ErrInternal
	}

	return nil
}
