package internal

import (
	"context"
	"regexp"
	//"errors"
	//"sort"
	//"strings"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/google/uuid"

	"scmc/model/docker"
	"scmc/model/litedb"
	"scmc/rpc"

	pb "scmc/rpc/pb/ipc"
	// "scmc/server/args"
)

type IpcBaseServer struct {
	pb.UnimplementedIpcBaseServer
}

/*
 * 返回 IPC 基础容器数据
 * 直接从数据库里面拉取数据
 */
func (s *IpcBaseServer) List(ctx context.Context, in *pb.IpcListRequest) (*pb.IpcListReply, error) {
	ipcAllData, err := litedb.IpcConfigList()
	if err != nil {
		log.Warnf("IpcConfigList err: %v", err)
		return nil, rpc.ErrInternal
	}

	var rsp pb.IpcListReply
	for _, c := range ipcAllData {
		info := pb.IpcContainerData{
			Name:      c.ContainerName,
			Id:        c.ContainerID,
			StartTime: c.CreatedAt,
		}
		ipcMapcfg, err := litedb.IpcmapConfigByIPCName([]string{c.ContainerName})
		if err == nil {
			for _, d := range ipcMapcfg {
				info.Containers = append(info.Containers, d.ContainerName)
			}
		}
		rsp.IpcnameIds = append(rsp.IpcnameIds, &pb.IpcContainerData {Name: info.Name, Id: info.Id, StartTime: info.StartTime, Containers: info.Containers})
	}

	/*
	for _, c := range ipcAllData {
		info := pb.IpcContainerData {
			Name : c.ContainerName,
			Id : c.ContainerID,
			StartTime : c.CreatedAt,
		}

		if c.IPC != nil && len(*c.IPC) > 0 {
			for key, value := range *c.IPC {
				info.Containers = append(info.Containers, &pb.IpcContainerPair{Name: key, Id: value})
			}
		}

		rsp.IpcnameIds = append(rsp.IpcnameIds, &pb.IpcContainerData{Name:info.Name, Id:info.Id, StartTime: info.StartTime, Containers:info.Containers})
	} */

	return &rsp, nil
}

// 创建 IPC 基础容器
func (s *IpcBaseServer) create(Name string) (string, error) {
	if !regexp.MustCompile(containerNamePattern).MatchString(Name) {
		return "", status.Errorf(codes.InvalidArgument, "IPC名参数错误")
	}
	cli, err := docker.Client()
	if err != nil {
		return "", rpc.ErrInternal
	}

	if err := ensureImage("ipcbase:v1.0"); err != nil {
		return "", status.Errorf(codes.Internal, "检查IPC镜像信息失败")
	}

	uuid := uuid.New().String()

	// 容器配置
	config := &container.Config {
		Image: "ipcbase:v1.0", //使用指定的镜像文件
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
		log.Warnf("ContainerStart: %v", err)
		return "", transDockerError(err)
	}

	return resp.ID, nil
}

/*
 * 创建 IPC 基础容器, 创建成功后写数据库
 * in.NodeId (int64) 
 * in.Name (string)
 */
func (s *IpcBaseServer) Create(ctx context.Context, in *pb.IpcCreateRequest) (*pb.IpcCreateReply, error) {
	id, err := s.create(in.Name)

	// 创建容器失败了
	if err != nil {
		return nil, err
	}

	// 创建成功了，写数据库 
	cfg := litedb.IpcConfig {
		ContainerID : id,
		ContainerName : in.Name,
	}

	err = litedb.CreateIpcConfig(&cfg)
	if err != nil {
		log.Warnf("CreateIpcConfig error: %v, name:%v, id:%v", err, in.Name, id)
	}

	return &pb.IpcCreateReply{Name: in.Name, ContainerId:id, StartTime:cfg.CreatedAt} , nil
}


func IpcStopAndRemoveContainer(id string) {
	cli, err := docker.Client()
	if err != nil {
		log.Warnf("IpcStopAndRemoveContainer err: %v", err)
		return
	}

	// 停止容器
	var timeout = time.Second
	err = cli.ContainerStop(context.Background(), id, &timeout)
	if err != nil {
		log.Warnf("ContainerStop id:%v err:%v", id, err)
		//return
	}

	// 删除容器
	opts := types.ContainerRemoveOptions{RemoveVolumes: true, Force: true}
	err = cli.ContainerRemove(context.Background(), id, opts);
	if err != nil {
		log.Warnf("IPC ContainerRemove id:%v, err:%v", id, err)
		return
	}

	return
}

/*
 * 删除 IPC 基础容器数据
 */
func (s *IpcBaseServer) Remove(ctx context.Context, in *pb.IpcRemoveRequest) (*pb.IpcRemoveReply, error) {
	// 查询数据库，获取 ID， 停止容器，删除容器
	if len(in.IpcName) <= 0 {
		log.Warnf("IPC Remove InvalidArgument")
		return nil, rpc.ErrInvalidArgument
	}

	for _, id := range in.IpcName {
		cfg, err := litedb.GetIpcConfigByName(id)
		if err != nil {
			continue
		}

		ipcMapcfg, err := litedb.IpcmapConfigByIPCName([]string{cfg.ContainerName})
		if err != nil || len(ipcMapcfg) == 0 { // err 不为 nil 说明 数据库里面没有记录，可以删除
			containerID := cfg.ContainerID
			IpcStopAndRemoveContainer(containerID)

			//删除数据库里面的记录
			litedb.IpcConfigRemoveByID(cfg.ID)
		} else {
			log.Warnf("Find Success ContainerName: %v, len: %v", cfg.ContainerName, len(ipcMapcfg))
		}
	}

	return &pb.IpcRemoveReply{}, nil
}


// 判断是否在允许中，如果在允许中就不用重启了
func isRunning(ID string) bool {
	cli, err := docker.Client()
	if err != nil {
		return false
	}

	info, _, err := cli.ContainerInspectWithRaw(context.Background(), ID, true)
    if err != nil {
        return false
    }

	if info.State != nil {
		return info.State.Status == "running"
	}

	return false;
}

func startIpcContainer(ID string) {
	if isRunning(ID) {
		return;
	}

	cli, err := docker.Client()
	if err != nil {
		return 
	}

	for i := 0; i < 5; i++ {
		if err := cli.ContainerStart(context.Background(), ID, types.ContainerStartOptions{}); err != nil {
			log.Warnf("IPC ContainerStart: id: %v, err :%v", ID, err)
			time.Sleep(time.Second * 5)
		} else {
			return;
		}
	}
}

// 开机的时候启动所有 IPC 容器
func StartAllIpcContainer() {
	ipcAllData, err := litedb.IpcConfigList()
	if err != nil {
		log.Warnf("IpcConfigList err: %v", err)
		return
	}

	var wg sync.WaitGroup
	for _, c := range ipcAllData {
		wg.Add(1)
		go func(ID string) {
			defer wg.Done()
			startIpcContainer(ID) // 启动容器
		}(c.ContainerID)
	}

	wg.Wait();
	log.Warnf("All IPC Container Start Success!")
}
