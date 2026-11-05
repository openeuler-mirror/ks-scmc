package internal

import (
	"context"
	 _"errors"
	 _"time"
	 _"sort"
	 _"strings"
	 _"sync"

	//log "github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"scmc/model/db"
	"scmc/rpc"
	pb "scmc/rpc/pb/ipc"
	"scmc/server/args"
)

type IpcBaseServer struct {
	pb.UnimplementedIpcBaseServer
}

/*
 * 返回 IPC 基础容器数据
 * in.NodeId (int64) 
 */
func (s *IpcBaseServer) List(ctx context.Context, in *pb.IpcListRequest) (*pb.IpcListReply, error) {
	/*
	var nodes []*db.NodeInfo
	if in.NodeId == -1 {
		var err error
		if nodes, err = db.ListNodes(); err != nil {
			log.Warnf("db.ListNodes err = %v", err)
			return nil, rpc.ErrDatabaseFail
		}
	} else {
		n, err := db.QueryNodeByID(in.NodeId)
		if err != nil {
			if err == db.ErrRecordNotFound {
				return nil, rpc.ErrNotFound
			}
			log.Warnf("db.QueryNodeByID err = %v", err)
			return nil, rpc.ErrDatabaseFail
		}
		nodes = append(nodes, n)
	}

	// 请求数据 
	var reply pb.IpcListReply
	for _, n := range nodes {
		conn, err := getAgentConn(n.Address)
		if err != nil {
			if len(nodes) == 1 {
				return nil, rpc.ErrInternal
			}
			continue
		}
		r, err := pb.NewIpcBaseClient(conn).List(ctx, in)
		if err != nil {
			log.Warnf("IpcBase.List node=%+v, err=%v", n, err)
			if len(nodes) == 1 {
				return nil, rpc.ErrInternal
			}
			continue
		}
		// 这里怎么处理 
	}
	*/
	var reply pb.IpcListReply
	return &reply, nil
}

/*
 * 创建 IPC 基础容器
 * in.NodeId (int64)
 * in.Name (string)
 */
func (s *IpcBaseServer) Create(ctx context.Context, in *pb.IpcCreateRequest) (*pb.IpcCreateReply, error) {
	/* 检查容器名称 */
	if !args.ContainerName.Check(in.Name) {
		return nil, status.Errorf(codes.InvalidArgument, "IPC名称参数错误")
	}

	/* 查询节点是否存在 */
	n, err := db.QueryNodeByID(in.NodeId)
	if err != nil {
		if err == db.ErrRecordNotFound {
			return nil, rpc.ErrNotFound
		}
		return nil, rpc.ErrDatabaseFail
	}

	/* 检查镜像文件是否存在 */
	if err := CheckImageLegal(in.NodeId, "busybox:kysec"); err != nil {
		return nil, err
	}

	conn, err := getAgentConn(n.Address)
	if err != nil {
		return nil, rpc.ErrInternal
	}

	/* 给 Agent 发送请求 */
	return pb.NewIpcBaseClient(conn).Create(ctx, in)
}

/*
 * 删除 IPC 基础容器数据
 */
func (s *IpcBaseServer) Remove(ctx context.Context, in *pb.IpcRemoveRequest) (*pb.IpcRemoveReply, error) {
	var reply pb.IpcRemoveReply
	return &reply, nil
}
