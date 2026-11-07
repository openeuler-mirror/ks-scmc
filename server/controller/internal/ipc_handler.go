package internal

import (
	"context"
	 _"errors"
	 _"time"
	 _"sort"
	 _"strings"
	 _"sync"

	log "github.com/sirupsen/logrus"
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
	/* 查询节点是否存在 */
	n, err := db.QueryNodeByID(in.NodeId)
	if err != nil {
		if err == db.ErrRecordNotFound {
			return nil, rpc.ErrNotFound
		}
		return nil, rpc.ErrDatabaseFail
	}

	conn, err := getAgentConn(n.Address)
	if err != nil {
		log.Warnf("IPC getAgentConn err:%v", err)
		return nil, rpc.ErrInternal
	}

	/* 给 Agent 发送请求 */
	return pb.NewIpcBaseClient(conn).List(ctx, in)
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
	// 检查镜像文件是否存在 
	if err := CheckImageLegal(in.NodeId, "ipcbase:v1.0"); err != nil {
		return nil, err
	}

	conn, err := getAgentConn(n.Address)
	if err != nil {
		log.Warnf("IPC getAgentConn err:%v", err)
		return nil, rpc.ErrInternal
	}

	/* 给 Agent 发送请求 */
	return pb.NewIpcBaseClient(conn).Create(ctx, in)
}

/*
 * 删除 IPC 基础容器数据
 */
func (s *IpcBaseServer) Remove(ctx context.Context, in *pb.IpcRemoveRequest) (*pb.IpcRemoveReply, error) {
	/* 查询节点是否存在 */
	n, err := db.QueryNodeByID(in.NodeId)
	if err != nil {
		if err == db.ErrRecordNotFound {
			return nil, rpc.ErrNotFound
		}
		return nil, rpc.ErrDatabaseFail
	}

	if len(in.IpcName) <= 0 {
		return nil, status.Errorf(codes.InvalidArgument, "IPC名列表不能为空!")
	}

	conn, err := getAgentConn(n.Address)
	if err != nil {
		return nil, rpc.ErrInternal
	}

	/* 给 Agent 发送请求 */
	return pb.NewIpcBaseClient(conn).Remove(ctx, in)
}
