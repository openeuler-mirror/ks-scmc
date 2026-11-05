package internal

import (
	"context"
	"regexp"
	//"errors"
	//"sort"
	//"strings"
	//"sync"
	//"time"

	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/google/uuid"

	"scmc/model/docker"
	"scmc/rpc"
	pb "scmc/rpc/pb/ipc"
	// "scmc/server/args"
)

type IpcBaseServer struct {
	pb.UnimplementedIpcBaseServer
}

/*
 * 返回 IPC 基础容器数据
 */
func (s *IpcBaseServer) List(ctx context.Context, in *pb.IpcListRequest) (*pb.IpcListReply, error) {
	return nil, nil
}


func (s *IpcBaseServer) create(Name string) (string, error) {
	if !regexp.MustCompile(containerNamePattern).MatchString(Name) {
		return "", status.Errorf(codes.InvalidArgument, "IPC名参数错误")
	}
	cli, err := docker.Client()
	if err != nil {
		return "", rpc.ErrInternal
	}

	if err := ensureImage("busybox:kysec"); err != nil {
		return "", status.Errorf(codes.Internal, "检查IPC镜像信息失败")
	}

	uuid := uuid.New().String()

	// 容器配置
	config := &container.Config {
		Image: "busybox:kysec", //使用指定的镜像文件
		Labels: map[string] string {
			"KS_SCMC_DESC": "ipc base container",
			"KS_SCMC_UUID": uuid,
		},
		AttachStdin:true,
		Tty:true,
		Cmd: []string{"sleep", "infinity"}, // 容器启动命令
	}

	// 主机配置
	hostConfig :=&container.HostConfig {
		IpcMode: "shareable",
		Privileged: false,
	}

	// 创建容器
	resp, err := cli.ContainerCreate (
		context.Background(),
		config,
		hostConfig,
		nil, //网络配置
		nil, //平台配置
		"ipc." + Name, // 容器名称
	)

	if err != nil {
		log.Warnf("ContainerCreate: %v", err)
		return "", transDockerError(err)
	}

	if err := cli.ContainerStart(context.Background(), resp.ID, types.ContainerStartOptions{}); err != nil {
		return "", err
	}

	return resp.ID, nil
}

/*
 * 创建 IPC 基础容器
 * in.NodeId (int64) 
 * in.Name (string)
 */
func (s *IpcBaseServer) Create(ctx context.Context, in *pb.IpcCreateRequest) (*pb.IpcCreateReply, error) {
	id, err := s.create(in.Name)
	if err != nil {

	}
	_  = id
	return nil, nil
}

/*
 * 删除 IPC 基础容器数据
 */
func (s *IpcBaseServer) Remove(ctx context.Context, in *pb.IpcRemoveRequest) (*pb.IpcRemoveReply, error) {
	return nil, nil
}
