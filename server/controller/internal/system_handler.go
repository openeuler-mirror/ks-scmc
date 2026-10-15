package internal

import (
	"context"

	log "github.com/sirupsen/logrus"

	"scmc/model/db"
	"scmc/model/session"
	"scmc/rpc"
	"scmc/rpc/middleware/auth"
	pbnode "scmc/rpc/pb/node"
	pb "scmc/rpc/pb/system"
	pbuser "scmc/rpc/pb/user"
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

func (s *SystemServer) GetSecuritySwitch(ctx context.Context, in *pb.GetSecuritySwitchRequest) (*pb.GetSecuritySwitchReply, error) {
	isOn, err := db.GetSecuritySwitch()
	if err != nil {
		log.Warnf("get security switch  err: %v", err)
		return nil, err
	}
	return &pb.GetSecuritySwitchReply{
		IsOn: isOn,
	}, nil
}

// SetSecuritySwitch set global security switch
// note: Only let users with security administrator privileges call
func (s *SystemServer) SetSecuritySwitch(ctx context.Context, in *pb.SetSecuritySwitchRequest) (*pb.SetSecuritySwitchReply, error) {
	if err := db.SetSecuritySwitch(in.IsOn); err != nil {
		log.Warnf("set security switch  err: %v", err)
		return nil, err
	}

	return &pb.SetSecuritySwitchReply{
		IsOn: in.IsOn,
	}, nil
}

func (s *SystemServer) hasPerm(target pbuser.PERMISSION, perms []*db.Perm) bool {
	isOn, _ := db.GetSecuritySwitch()
	if !isOn {
		return true // 安全开关关闭 无需校验权限位
	}

	return auth.HasPerm(target, perms)
}

// Dashboard resturn system dashboard based on user permssion
func (s *SystemServer) Dashboard(ctx context.Context, in *pb.DashboardRequest) (*pb.DashboardReply, error) {
	uid, _, ok := auth.UserSessionFromContext(ctx)
	if !ok {
		log.Warnf("auth.UserSessionFromContext ctx=%+v", ctx)
		return nil, rpc.ErrInternal
	}

	role, err := db.QueryUserRole(uid)
	if err != nil {
		log.Infof("db.QueryUserRole uid=%v err=%v", uid, err)
		return nil, rpc.ErrDatabaseFail
	}

	var ns NodeServer
	nodeList, err := ns.List(ctx, &pbnode.ListRequest{})
	if err != nil {
		log.Infof("node.List err=%v", err)
		return nil, rpc.ErrInternal
	}

	imageStats, err := db.GetImageStats()
	if err != nil {
		log.Infof("db.GetImageStats err=%v", err)
		return nil, rpc.ErrInternal
	}

	templateCount, err := db.TemplateCount()
	if err != nil {
		log.Infof("db.TemplateCount err=%v", err)
		return nil, rpc.ErrInternal
	}

	var r pb.DashboardReply
	if s.hasPerm(pbuser.PERMISSION_NODE_INFO_READ, role.PermsJSON) {
		nodeStats(nodeList, &r)
	}

	if s.hasPerm(pbuser.PERMISSION_CONTAINER_INFO_READ, role.PermsJSON) {
		containerStats(templateCount, nodeList, &r)
	}

	if s.hasPerm(pbuser.PERMISSION_AUTID_WARN_READ, role.PermsJSON) {
		logStats(nodeList, &r)
	}

	if s.hasPerm(pbuser.PERMISSION_IMAGE_INFO_READ, role.PermsJSON) {
		r.Image = &pb.ImageStats{
			TotalCount: imageStats.TotalCount,
			TotalSize:  imageStats.TotalSize,
		}
	}

	if s.hasPerm(pbuser.PERMISSION_AUDIT_APPROVE_READ, role.PermsJSON) {
		r.Audit = &pb.AuditStats{
			ImageToApproveCount: imageStats.ToApproveCount,
		}
	}

	return &r, nil
}

func nodeStats(n *pbnode.ListReply, r *pb.DashboardReply) {
	r.Node = &pb.NodeStats{}

	for _, n := range n.Nodes {
		if n.Status != nil && n.Status.State == int64(pbnode.NodeState_Online) {
			r.Node.OnlineCount++
		} else {
			r.Node.OfflineCount++
		}
	}
}

func containerStats(t int64, n *pbnode.ListReply, r *pb.DashboardReply) {
	r.Container = &pb.ContainerStats{}

	for _, n := range n.Nodes {
		if n.Status != nil && n.Status.ContainerStat != nil {
			r.Container.TotalCount += n.Status.ContainerStat.Total
			r.Container.OnlineCount += n.Status.ContainerStat.Running
			r.Container.OfflineCount += (n.Status.ContainerStat.Total - n.Status.ContainerStat.Running)
		}
	}
	r.Container.TemplateCount = t
}

func logStats(n *pbnode.ListReply, r *pb.DashboardReply) {
	r.Log = &pb.LogStats{}

	for _, n := range n.Nodes {
		r.Log.UnreadWarnCount += n.UnreadWarn
	}
}
