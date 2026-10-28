package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/docker/docker/pkg/jsonmessage"
	"io"
	"os"
	"reflect"
	"regexp"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/strslice"
	"github.com/docker/docker/client"
	"github.com/google/uuid"
	"github.com/shirou/gopsutil/mem"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"scmc/common"
	"scmc/model/authz"
	"scmc/model/db"
	"scmc/model/docker"
	"scmc/model/iptables"
	"scmc/model/kysec"
	"scmc/model/litedb"
	"scmc/model/opensnitch"
	"scmc/rpc"
	pb "scmc/rpc/pb/container"
	"scmc/server/args"
)

const (
	defaultCPUShares   = 1024
	maxCPUShares       = 3072
	maxBlkioWeight     = 1000
	defaultBlkioWeight = 0
	bufSize            = 4 * 1024 * 1024
	dockerTimeout      = 10 * time.Minute
)

//--cpushares不设置的话，docker默认是1024
//优先级为1,设置为默认值的1.25倍
//优先级为2,设置为默认值的1.5倍
//优先级为3,设置为默认值的2倍
//优先级为4,设置为默认值的2.5倍
//优先级为5,设置为默认值的3倍
var cpuShareMap = map[int64]int64{
	1: 1280,
	2: 1536,
	3: 2048,
	4: 2560,
	5: maxCPUShares,
}

var blkioWeightMap = map[int64]uint16{
	1: 600,
	2: 700,
	3: 800,
	4: 900,
	5: maxBlkioWeight,
}

//磁盘优先级=>blkioWeigh
func toBlkioWeight(diskPrio int64) uint16 {
	if diskPrio <= 0 {
		return defaultBlkioWeight
	}
	if v, ok := blkioWeightMap[diskPrio]; ok {
		return uint16(v)
	}
	return defaultBlkioWeight
}

//blkioWeight => 磁盘优先级
func fromBlkioWeight(blkioWeight uint16) int64 {

	if blkioWeight <= defaultBlkioWeight {
		return 0
	}
	for k, v := range blkioWeightMap {
		if v == blkioWeight {
			return k
		}
	}
	return 0
}

func toCPUShares(cpuPrio int64) int64 {
	//小于等于0,或大于5都返回defaultCPUShares
	if cpuPrio <= 0 {
		return defaultCPUShares
	}
	if v, ok := cpuShareMap[cpuPrio]; ok {
		return v
	}
	return defaultCPUShares
}

func fromCPUShares(cpuShares int64) int64 {

	if cpuShares <= defaultCPUShares {
		return 0
	}
	for k, v := range cpuShareMap {
		if v == cpuShares {
			return k
		}
	}
	return 0
}

func dockerResourceConfig(in *pb.ResourceLimit) container.Resources {
	var r container.Resources

	if in != nil {
		r.NanoCPUs = int64(in.CpuLimit * 1e9)
		if r.NanoCPUs == 0 {
			r.NanoCPUs = int64(numCPU()) * 1e9
		}
		r.CPUShares = toCPUShares(in.CpuPrio)
		r.BlkioWeight = toBlkioWeight(in.DiskPrio)
		r.Memory = int64(in.MemoryLimit * megaBytes)
		r.MemoryReservation = int64(in.MemorySoftLimit * megaBytes)
		r.MemorySwap = -1 // prevent contaienr start error
	}

	return r
}

func ensureImage(imageRef string) error {
	_, err := docker.IsImageExist(imageRef)
	if err != nil {
		log.Infof("docker.IsImageExist err=%v", err)
		return rpc.ErrInternal
	}

	return nil
}

type ContainerServer struct {
	pb.UnimplementedContainerServer
}

func (s *ContainerServer) List(ctx context.Context, in *pb.ListRequest) (*pb.ListReply, error) {
	cli, err := docker.Client()
	if err != nil {
		return nil, rpc.ErrInternal
	}

	stats, err := getContainerStats()
	if err != nil {
		log.Warnf("getContainerStats: %v", err)
	}

	opts := types.ContainerListOptions{All: in.GetListAll(), Size: true}
	containers, err := cli.ContainerList(context.Background(), opts)
	if err != nil {
		log.Warnf("ContainerList: %v", err)
		return nil, rpc.ErrInternal
	}

	var reply pb.ListReply
	for _, c := range containers {
		info := pb.ContainerInfo{
			Id:           c.ID,
			Image:        c.Image,
			ImageId:      c.ImageID,
			Command:      c.Command,
			State:        c.State,
			Created:      c.Created,
			Started:      containerStartedTime(ctx, cli, &c),
			Name:         containerName(c.Names),
			ResourceStat: containerResourceStat(stats, &c),
		}

		reply.Containers = append(reply.Containers, &pb.NodeContainer{Info: &info})
	}

	return &reply, nil
}

func (s *ContainerServer) setSecurityConfig(id, uuid, name string, pid int, fromUpdate bool, sec *pb.SecurityConfig) error {
	if sec == nil {
		return nil
	}

	if fromUpdate {
		if err := kysec.CleanAll(id, false); err != nil {
			log.Warnf("kysec.CleanAll: %v", err)
			return rpc.ErrContainerFileProtection
		}
	}

	if sec.ProcProtection != nil {
		var hashes []string
		for _, p := range sec.ProcProtection.ExeList {
			s, err := docker.ContainerFileMD5(id, p)
			if err != nil {
				log.Warnf("ContainerFileMD5 path=%v err=%v", p, err)
				return rpc.ErrContainerProcProtection
			}
			hashes = append(hashes, s)
		}

		if err := kysec.UpdateWhiteList(id, sec.ProcProtection.IsOn, hashes, []string{}); err != nil {
			log.Warnf("UpdateWhiteList %v err: %v", id, err)
			return rpc.ErrContainerProcProtection
		}
	}

	if sec.NprocProtection != nil {
		var inRule = &litedb.ProcProtection{
			IsOn:    sec.NprocProtection.IsOn,
			ExeList: sec.NprocProtection.ExeList,
		}

		if err := opensnitch.SaveRule(inRule, id, uuid); err != nil {
			log.Warnf("SaveRule id=%v err=%v", id, err)
			return rpc.ErrContainerNprocProtection
		}
	}

	if sec.FileProtection != nil {
		if err := kysec.UpdateFileAccess(id, sec.FileProtection.IsOn, sec.FileProtection.FileList, []string{}); err != nil {
			log.Warnf("UpdateFileAccess %v err: %v", id, err)
			return rpc.ErrContainerFileProtection
		}
	}

	if err := authz.HandleSensitiveContainer(sec.DisableCmdOperation, fromUpdate, name, id); err != nil {
		log.Warnf("HandleSensitiveContainer err=%v", err)
		return rpc.ErrContainerCmdOperation
	}

	if sec.NetworkRule != nil {
		var rules []*litedb.NetworkRule
		for _, v := range sec.NetworkRule.Rules {
			rules = append(rules, &litedb.NetworkRule{
				Protocols: v.Protocols,
				Addr:      v.Addr,
				Port:      v.Port,
			})
		}

		fileName := id + "-" + name
		if err := iptables.UpdateContainerIPtablesFile(fileName, sec.NetworkRule.IsOn, rules, pid); err != nil {
			log.Warnf("UpdateContainerIPtablesFile %v err: %v", fileName, err)
			return rpc.ErrContainerNetworkRule
		}
	}

	return nil
}

func (s *ContainerServer) create(configs *pb.ContainerConfigs) (string, error) {
	if configs == nil || configs.Image == "" {
		return "", rpc.ErrInvalidArgument
	} else if !regexp.MustCompile(containerNamePattern).MatchString(configs.Name) {
		return "", status.Errorf(codes.InvalidArgument, "容器名参数错误")
	}

	cli, err := docker.Client()
	if err != nil {
		return "", rpc.ErrInternal
	}

	if err := ensureImage(configs.Image); err != nil {
		return "", status.Errorf(codes.Internal, "检查镜像信息失败")
	}

	config := container.Config{
		Image: configs.Image,
		Labels: map[string]string{
			"KS_SCMC_DESC": configs.Desc,
			"KS_SCMC_UUID": configs.Uuid,
		},
		Env:         createContainerEnvs(configs.Envs, configs.Uuid),
		AttachStdin: true, // AttachStdin, Tty: make sure container like OS(ubuntu, centos) will not exit after start
		Tty:         true,
	}

	if configs.Cmd != "" {
		var cmdSlice strslice.StrSlice = strings.Fields(configs.Cmd)
		config.Cmd = cmdSlice
	}

	hostConfig := container.HostConfig{
		Privileged: false, // force non-privileged
	}

	for _, m := range configs.Mounts {
		hostConfig.Mounts = append(hostConfig.Mounts, mount.Mount{
			Type:     mount.Type(m.Type),
			Source:   m.Source,
			Target:   m.Target,
			ReadOnly: m.ReadOnly,
		})
	}

	// configs.EnableGraphic 默认开启，后面应用里有图形化选项
	if err := containerGraphicSetup(configs.Name, &config, &hostConfig); err != nil {
		log.Infof("containerGraphicSetup err=%v", err)
		return "", rpc.ErrInternal
	}

	if configs.RestartPolicy != nil {
		hostConfig.RestartPolicy.Name = configs.RestartPolicy.Name
		hostConfig.RestartPolicy.MaximumRetryCount = int(configs.RestartPolicy.MaxRetry)
	}

	if configs.ResouceLimit != nil {
		hostConfig.Resources = dockerResourceConfig(configs.ResouceLimit)

		if configs.ResouceLimit.DiskLimit > 0.0 {
			config.Labels["KS_SCMC_DISK_LIMIT"] = fmt.Sprintf("%f", configs.ResouceLimit.DiskLimit)
			hostConfig.StorageOpt = map[string]string{
				"size": fmt.Sprintf("%fM", configs.ResouceLimit.DiskLimit),
			}
		}
	}

	networkConfig := &network.NetworkingConfig{
		EndpointsConfig: make(map[string]*network.EndpointSettings, len(configs.Networks)),
	}
	for _, v := range configs.Networks {
		if err := checkNetworkInfo(v.Interface, "", &v.IpAddress, int(v.IpPrefixLen)); err != nil {
			return "", err
		}

		networkConfig.EndpointsConfig[v.Interface] = &network.EndpointSettings{
			IPAMConfig: &network.EndpointIPAMConfig{
				IPv4Address: v.IpAddress,
			},
			IPAddress:   v.IpAddress,
			IPPrefixLen: int(v.IpPrefixLen),
			MacAddress:  v.MacAddress,
			Gateway:     v.Gateway,
		}
	}

	var networkConfigCreate *network.NetworkingConfig
	if len(configs.Networks) > 0 {
		ifs := configs.Networks[0].Interface
		networkConfigCreate = &network.NetworkingConfig{
			EndpointsConfig: map[string]*network.EndpointSettings{
				ifs: networkConfig.EndpointsConfig[ifs],
			},
		}
	}

	body, err := cli.ContainerCreate(context.Background(), &config, &hostConfig, networkConfigCreate, nil, configs.Name)
	if err != nil {
		log.Warnf("ContainerCreate: %v", err)
		return "", transDockerError(err)
	}

	if len(configs.Networks) > 1 {
		for i := 1; i < len(configs.Networks); i++ {
			ifs := configs.Networks[i].Interface
			network := networkConfig.EndpointsConfig[ifs]
			err = cli.NetworkConnect(context.Background(), ifs, body.ID, network)
			if err != nil {
				log.Warnf("NetworkConnect: %v", err)
			}
		}
	}

	if err := s.setSecurityConfig(body.ID, configs.Uuid, configs.Name, 0, false, configs.SecurityConfig); err != nil {
		log.Warnf("setSecurityConfig failed err=%v", err)
		if e := s.remove(cli, body.ID, true); e != nil {
			log.Warnf("after setSecurityConfig failed remove container=%v err=%v", body.ID, e)
			return body.ID, err
		}
		return "", err
	}

	return body.ID, nil
}

func (s *ContainerServer) Create(ctx context.Context, in *pb.CreateRequest) (*pb.CreateReply, error) {
	if in.Configs == nil {
		return nil, rpc.ErrInvalidArgument
	}

	uuid := uuid.New().String()
	cfgs := litedb.ContainerConfig{
		UUID:           uuid,
		ContainerName:  in.Configs.Name,
		SecurityConfig: in.Configs.SecurityConfig,
	}

	if err := litedb.CreateContainerConfig(&cfgs); err != nil {
		return nil, rpc.ErrInternal
	}

	in.Configs.Uuid = uuid
	id, err := s.create(in.Configs)

	if err != nil {
		litedb.RemoveContainerConfigByID(cfgs.ID)
	} else {
		cfgs.ContainerID = id
		litedb.UpdateContainerConfig(&cfgs)
	}

	return &pb.CreateReply{ContainerId: id}, err
}

func (s *ContainerServer) Start(ctx context.Context, in *pb.StartRequest) (*pb.StartReply, error) {
	if (len(in.Ids)) <= 0 || len(in.Ids[0].ContainerIds) <= 0 {
		return nil, rpc.ErrInvalidArgument
	}

	cli, err := docker.Client()
	if err != nil {
		return nil, rpc.ErrInternal
	}

	var (
		reply pb.StartReply
		opts  types.ContainerStartOptions
	)

	for _, id := range in.Ids[0].ContainerIds {
		if err := cli.ContainerStart(context.Background(), id, opts); err != nil {
			log.Warnf("ContainerStart: id=%v %v", id, err)
			var faileReason string
			if strings.HasSuffix(err.Error(), "Address already in use") {
				faileReason = "IP已被占用"
			} else if s, _ := status.FromError(err); s != nil {
				faileReason = s.Message()
			}
			reply.FailInfos = append(reply.FailInfos, &pb.ContainerFailInfo{
				ContainerId: id,
				FailReason:  faileReason,
			})
			continue
		}

		reply.OkIds = append(reply.OkIds, id)
	}

	return &reply, nil
}

func (s *ContainerServer) parallelProcessing(operate string, contaienrIds []string) ([]string, []*pb.ContainerFailInfo, error) {
	cli, err := docker.Client()
	if err != nil {
		return nil, nil, rpc.ErrInternal
	}

	var (
		timeout   = time.Second
		wg        sync.WaitGroup
		lock      sync.Mutex
		okIDs     []string
		failInfos []*pb.ContainerFailInfo
	)

	for _, id := range contaienrIds {
		wg.Add(1)
		go func(containerID string) {
			defer wg.Done()
			var err error
			if operate == "ContainerStop" {
				err = cli.ContainerStop(context.Background(), containerID, &timeout)
			} else if operate == "ContainerRestart" {
				err = cli.ContainerRestart(context.Background(), containerID, &timeout)
			} else {
				log.Warnf("%v: id=%v ", operate, containerID)
				return
			}

			if err != nil {
				log.Warnf("%v: id=%v %v", operate, containerID, err)
				var faileReason string
				if s, _ := status.FromError(err); s != nil {
					faileReason = s.Message()
				}
				lock.Lock()
				failInfos = append(failInfos, &pb.ContainerFailInfo{
					ContainerId: containerID,
					FailReason:  faileReason,
				})
				lock.Unlock()
				return
			}
			okIDs = append(okIDs, containerID)
		}(id)
	}

	wg.Wait()
	return okIDs, failInfos, nil
}

func (s *ContainerServer) Stop(ctx context.Context, in *pb.StopRequest) (*pb.StopReply, error) {
	if (len(in.Ids)) <= 0 || len(in.Ids[0].ContainerIds) <= 0 {
		return nil, rpc.ErrInvalidArgument
	}

	okIDs, failInfos, err := s.parallelProcessing("ContainerStop", in.Ids[0].ContainerIds)
	if err != nil {
		return nil, err
	}

	return &pb.StopReply{
		OkIds:     okIDs,
		FailInfos: failInfos,
	}, nil
}

func (s *ContainerServer) Kill(ctx context.Context, in *pb.KillRequest) (*pb.KillReply, error) {
	if (len(in.Ids)) <= 0 || len(in.Ids[0].ContainerIds) <= 0 {
		return nil, rpc.ErrInvalidArgument
	}

	cli, err := docker.Client()
	if err != nil {
		return nil, rpc.ErrInternal
	}

	var reply pb.KillReply
	for _, id := range in.Ids[0].ContainerIds {
		if err := cli.ContainerKill(context.Background(), id, ""); err != nil {
			log.Warnf("ContainerKill: id=%v %v", id, err)
			var faileReason string
			if s, _ := status.FromError(err); s != nil {
				faileReason = s.Message()
			}
			reply.FailInfos = append(reply.FailInfos, &pb.ContainerFailInfo{
				ContainerId: id,
				FailReason:  faileReason,
			})
			continue
		}
		reply.OkIds = append(reply.OkIds, id)
	}

	return &reply, nil
}

func (s *ContainerServer) Restart(ctx context.Context, in *pb.RestartRequest) (*pb.RestartReply, error) {
	if (len(in.Ids)) <= 0 || len(in.Ids[0].ContainerIds) <= 0 {
		return nil, rpc.ErrInvalidArgument
	}

	okIDs, failInfos, err := s.parallelProcessing("ContainerRestart", in.Ids[0].ContainerIds)
	if err != nil {
		return nil, err
	}

	return &pb.RestartReply{
		OkIds:     okIDs,
		FailInfos: failInfos,
	}, nil
}

func (s *ContainerServer) Update(ctx context.Context, in *pb.UpdateRequest) (*pb.UpdateReply, error) {
	var reply pb.UpdateReply

	log.Warnf("---------- container update")

	if in.ContainerId == "" || (in.ResourceLimit == nil && in.RestartPolicy == nil && in.Networks == nil && in.SecurityConfig == nil) {
		return nil, rpc.ErrInvalidArgument
	}

	cfgs, err := litedb.GetContainerConfig(in.ContainerId)
	if err != nil {
		log.Infof("GetContainerConfig err=%v", err)
		return nil, rpc.ErrDatabaseFail
	}

	for _, v := range in.Networks {
		if err := checkNetworkInfo(v.Interface, in.ContainerId, &v.IpAddress, int(v.IpPrefixLen)); err != nil {
			return nil, err
		}
	}

	inspectConfigs, pid, err := s.inspect(in.ContainerId, false)
	if err != nil {
		log.Warnf("inspect container=%v err=%v", in.ContainerId, err)
		return nil, rpc.ErrInternal
	}

	cli, err := docker.Client()
	if err != nil {
		return nil, rpc.ErrInternal
	}

	config := container.UpdateConfig{
		Resources: dockerResourceConfig(in.ResourceLimit),
	}

	if in.RestartPolicy != nil {
		config.RestartPolicy = container.RestartPolicy{
			Name:              in.RestartPolicy.Name,
			MaximumRetryCount: int(in.RestartPolicy.MaxRetry),
		}
	}

	body, err := cli.ContainerUpdate(context.Background(), in.ContainerId, config)
	if err != nil {
		log.Warnf("ContainerUpdate: %v", err)
		return nil, transDockerError(err)
	}

	if len(body.Warnings) > 0 {
		log.Infof("ContainerUpdate result warnings: %v", body.Warnings)
	}

	updateNetwork(ctx, cli, in.ContainerId, in.Networks)

	if err := s.setSecurityConfig(in.ContainerId, inspectConfigs.Uuid, inspectConfigs.Name, pid, true, in.SecurityConfig); err != nil {
		log.Warnf("Update setSecurityConfig err=%v", err)
		return nil, err
	}

	cfgs.SecurityConfig = in.SecurityConfig
	if err := litedb.UpdateContainerConfig(cfgs); err != nil {
		log.Infof("UpdateContainerConfig err=%v", err)
		return nil, rpc.ErrDatabaseFail
	}

	var change int
	var envs []string

	if !reflect.DeepEqual(inspectConfigs.Envs, in.Envs) {
		change |= docker.EnvBit
		for k, v := range in.Envs {
			envs = append(envs, k+"="+v)
			//磁盘读写速度、磁盘读写频率
			if k == "disk_wio" || k == "disk_rio" || k == "disk_rbw" || k == "disk_wbw" {
				if err := docker.SetContainerDiskLimit(k, in.ContainerId, v); err != nil {
					return nil, err
				}
			}
		}
		log.Debugf("%v envs change, %+v, %+v", inspectConfigs.ContainerId, in.Envs, envs)
	}

	mountMap, equal := docker.CompareMounts(inspectConfigs.Mounts, in.Mounts)
	if !equal {
		change |= docker.MountBit
		log.Debugf("%v mounts change", inspectConfigs.ContainerId)
	}

	log.Warnf("---------- 1, change: %v docker.NoneBit: %v", change, docker.NoneBit)
	if change != docker.NoneBit {
		if err = docker.ModifyContainerConfig(inspectConfigs.ContainerId, change, envs, mountMap); err != nil {
			log.Infof("UpdateContainerConfig err=%v", err)
			return nil, rpc.ErrInternal
		}
		log.Infof("status:%v", inspectConfigs.Status)
		var timeout = time.Second
		if inspectConfigs.Status == "running" {
			if err = cli.ContainerRestart(context.Background(), in.ContainerId, &timeout); err != nil {
				log.Warnf("restart id=%v %v", in.ContainerId, err)
			}
		} else {
			if err := cli.ContainerStart(context.Background(), in.ContainerId, types.ContainerStartOptions{}); err != nil {
				log.Warnf("start id=%v %v", in.ContainerId, err)
			}
			if err := cli.ContainerStop(context.Background(), in.ContainerId, &timeout); err != nil {
				log.Warnf("stop id=%v %v", in.ContainerId, err)
			}
		}
		log.Infof("modify %v end, change:%v", in.ContainerId, change)
	}

	return &reply, nil
}

func updateNetwork(ctx context.Context, cli *client.Client, containerID string, networks []*pb.NetworkConfig) error {
	if len(networks) == 0 {
		return nil
	}

	info, err := cli.ContainerInspect(context.Background(), containerID)
	if err != nil {
		log.Warnf("ContainerInspect: %v", err)
		return transDockerError(err)
	}

	if info.NetworkSettings == nil {
		for _, v := range networks {
			dockerNetworkConnect(ctx, cli, containerID, v)
		}
		return nil
	}

	m := make(map[string]bool, len(networks))
	for _, v := range networks {
		m[v.Interface] = true
	}
	for k := range info.NetworkSettings.Networks {
		if _, ok := m[k]; ok {
			continue
		}
		dockerNetworkDisconnect(ctx, cli, containerID, k) // 被删除的连接
	}

	for _, v := range networks {
		_, ok := info.NetworkSettings.Networks[v.Interface]
		if !ok {
			dockerNetworkConnect(ctx, cli, containerID, v) // 新连接
			continue
		}

		// 修改IP
		if v.IpAddress != info.NetworkSettings.Networks[v.Interface].IPAddress {
			dockerNetworkDisconnect(ctx, cli, containerID, v.Interface)
			dockerNetworkConnect(ctx, cli, containerID, v)
		}
	}
	return nil
}

func (s *ContainerServer) remove(cli *client.Client, containerID string, force bool) error {
	if cli == nil {
		c, err := docker.Client()
		if err != nil {
			return rpc.ErrInternal
		}
		cli = c
	}

	configs, _, err := s.inspect(containerID, false)
	if err != nil {
		log.Warnf("inspect container=%v err=%v", containerID, err)
		return err
	}

	if !force && configs.Status == "running" {
		return rpc.ErrRemoveContainerWhenRunning
	}

	opts := types.ContainerRemoveOptions{RemoveVolumes: true, Force: force}
	if err := cli.ContainerRemove(context.Background(), containerID, opts); err != nil {
		log.Warnf("ContainerRemove: id=%v %v", containerID, err)
		return rpc.ErrInternal
	}

	if err := kysec.CleanFileAccess(containerID); err != nil {
		log.Warnf("CleanFileAccess container=%v err=%v", containerID, err)
	}
	if err := kysec.CleanWhiteList(containerID); err != nil {
		log.Warnf("CleanWhiteList container=%v err=%v", containerID, err)
	}

	if err := opensnitch.RemoveRule(containerID); err != nil {
		log.Warnf("RemoveRule container=%v err=%v", containerID, err)
	}

	if err := removeContainerGraphicSetup(containerID); err != nil {
		log.Warnf("removeContainerGraphicSetup container=%v err=%v", containerID, err)
	}

	fileName := containerID + "-" + configs.Name
	iptables.ContainerRemveIPtables(fileName)

	if err := authz.DelSensitiveContainers(configs.Name, containerID); err != nil {
		log.Warnf("DelSensitiveContainers err=%v", err)
	}

	return nil
}

func (s *ContainerServer) Remove(ctx context.Context, in *pb.RemoveRequest) (*pb.RemoveReply, error) {
	if (len(in.Ids)) <= 0 || len(in.Ids[0].ContainerIds) <= 0 {
		return nil, rpc.ErrInvalidArgument
	}

	cli, err := docker.Client()
	if err != nil {
		return nil, rpc.ErrInternal
	}

	var reply pb.RemoveReply
	for _, id := range in.Ids[0].ContainerIds {
		err := s.remove(cli, id, false)
		if err != nil {
			var faileReason string
			if s, _ := status.FromError(err); s != nil {
				faileReason = s.Message()
			}
			reply.FailInfos = append(reply.FailInfos, &pb.ContainerFailInfo{
				ContainerId: id,
				FailReason:  faileReason,
			})
			reply.FailIds = append(reply.FailIds, id)
			continue
		}
		litedb.RemoveContainerConfig([]string{id})

		reply.OkIds = append(reply.OkIds, id)
	}

	litedb.RemoveContainerConfig(reply.OkIds)
	go s.removeContainerBackups(reply.OkIds)

	return &reply, nil
}

func (s *ContainerServer) removeContainerBackups(ids []string) error {
	backups, err := litedb.QueryContainerBackupByContainer(ids)
	if err != nil {
		log.Warnf("QueryContainerBackupByContainer err=%v", err)
		return err
	}

	for _, b := range backups {
		docker.RemoveImage(context.Background(), b.ImageRef, false)
	}

	litedb.RemoveContainerBackup(backups)
	return nil
}

func (s *ContainerServer) inspect(id string, backup bool) (*pb.ContainerConfigs, int, error) {
	cli, err := docker.Client()
	if err != nil {
		return nil, 0, rpc.ErrInternal
	}

	info, _, err := cli.ContainerInspectWithRaw(context.Background(), id, true)
	if err != nil {
		log.Warnf("ContainerInspectWithRaw: %v", err)
		return nil, 0, transDockerError(err)
	}

	configs := &pb.ContainerConfigs{
		ContainerId:  info.ID,
		Name:         strings.TrimPrefix(info.Name, "/"),
		Image:        info.Image,
		ResouceLimit: &pb.ResourceLimit{},
	}

	var pid int
	if info.State != nil {
		configs.Status = info.State.Status
		pid = info.State.Pid
	}

	if info.Config != nil {
		s.inspectConfig(cli, info.Config, configs)
	}

	for _, m := range info.Mounts {
		configs.Mounts = append(configs.Mounts, &pb.Mount{
			Type:     string(m.Type),
			Source:   m.Source,
			Target:   m.Destination,
			ReadOnly: !m.RW,
		})
	}

	if info.HostConfig != nil {
		configs.RestartPolicy = &pb.RestartPolicy{
			Name:     info.HostConfig.RestartPolicy.Name,
			MaxRetry: int32(info.HostConfig.RestartPolicy.MaximumRetryCount),
		}

		configs.ResouceLimit = &pb.ResourceLimit{
			CpuLimit:        float64(info.HostConfig.Resources.NanoCPUs) / 1e9,
			CpuPrio:         fromCPUShares(info.HostConfig.Resources.CPUShares),
			DiskPrio:        fromBlkioWeight(info.HostConfig.Resources.BlkioWeight),
			MemoryLimit:     float64(info.HostConfig.Resources.Memory) / megaBytes,
			MemorySoftLimit: float64(info.HostConfig.Resources.MemoryReservation) / megaBytes,
		}

		if s, ok := info.HostConfig.StorageOpt["size"]; ok {
			fmt.Sscanf(s, "%fM", &configs.ResouceLimit.DiskLimit)
		}
	}

	if info.NetworkSettings != nil {
		s.inspectNetwork(info.ID, backup, info.NetworkSettings, configs)
	}

	if info.Config.Cmd != nil {
		configs.Cmd = strings.Join(info.Config.Cmd, " ")
	}

	return configs, pid, nil
}

func (s *ContainerServer) inspectConfig(cli *client.Client, config *container.Config, ret *pb.ContainerConfigs) {
	// if image tag exists, use the tag, if not use image id
	if _, _, err := cli.ImageInspectWithRaw(context.Background(), config.Image); err == nil {
		ret.Image = config.Image
	}

	if v, ok := config.Labels["KS_SCMC_DESC"]; ok {
		ret.Desc = v
	}

	if v, ok := config.Labels["KS_SCMC_UUID"]; ok {
		ret.Uuid = v
	}

	if len(config.Env) > 0 {
		ret.Envs = make(map[string]string, len(config.Env))
		for _, e := range config.Env {
			parts := strings.SplitN(e, "=", 2)
			if len(parts) >= 2 {
				ret.Envs[parts[0]] = parts[1]
			}
		}
	}
}

func (s *ContainerServer) inspectNetwork(id string, backup bool, settings *types.NetworkSettings, ret *pb.ContainerConfigs) {
	for iface, m := range settings.Networks {
		if isPredefinedNetwork(iface) {
			continue // 需要忽略默认创建的网络连接
		}

		var ipAddr string
		if m.IPAMConfig != nil {
			ipAddr = m.IPAMConfig.IPv4Address // 返回用户填的IP地址
		}

		if !backup && m.IPAddress != "" {
			ipAddr = m.IPAddress // 非备份情况下，当用户没填IP地址，返回运行时IP
		}

		ret.Networks = append(ret.Networks, &pb.NetworkConfig{
			Interface:   iface,
			ContainerId: id,
			IpAddress:   ipAddr,
			IpPrefixLen: int32(m.IPPrefixLen),
			MacAddress:  m.MacAddress,
			Gateway:     m.Gateway,
		})
	}
}

func (s *ContainerServer) Inspect(ctx context.Context, in *pb.InspectRequest) (*pb.InspectReply, error) {
	data, err := litedb.GetContainerConfig(in.ContainerId)
	if err != nil {
		log.Infof("db get container configs id=%v err=%v", in.ContainerId, err)
		return nil, rpc.ErrDatabaseFail
	}

	configs, _, err := s.inspect(in.ContainerId, false)
	if err != nil {
		return nil, err
	}
	configs.SecurityConfig = data.SecurityConfig

	return &pb.InspectReply{Configs: configs}, nil
}

func (s *ContainerServer) MonitorHistory(ctx context.Context, in *pb.MonitorHistoryRequest) (*pb.MonitorHistoryReply, error) {
	now := time.Now()
	if in.StartTime >= in.EndTime || in.Interval < 1 || in.StartTime < now.Add(-time.Hour*24*10).Unix() {
		log.Info("MonitorHistory invalid time args")
		return nil, rpc.ErrInvalidArgument
	}

	containerName := "/" // query influxdb need container name, "/" for query the host

	memInfo, err := mem.VirtualMemory()
	if err != nil {
		log.Infof("VirtualMemory err=%v", err)
		return nil, err
	}

	rscLimit := pb.ResourceLimit{
		CpuLimit:    float64(numCPU()),
		MemoryLimit: float64(memInfo.Total) / megaBytes,
	}

	if in.ContainerId != "" {
		cli, err := docker.Client()
		if err != nil {
			return nil, rpc.ErrInternal
		}
		info, err := cli.ContainerInspect(context.Background(), in.ContainerId)
		if err != nil {
			log.Infof("ContainerInspect:  %v", err)
			return nil, transDockerError(err)
		}

		containerName = strings.TrimPrefix(info.Name, "/")
		if info.HostConfig != nil {
			if info.HostConfig.NanoCPUs > 0 {
				rscLimit.CpuLimit = float64(info.HostConfig.NanoCPUs) / 1e9
			}
			if info.HostConfig.Memory > 0 {
				rscLimit.MemoryLimit = float64(info.HostConfig.Memory) / megaBytes
			}
			if info.HostConfig.MemoryReservation > 0 {
				rscLimit.MemorySoftLimit = float64(info.HostConfig.MemoryReservation) / megaBytes
			}
		}
	}

	r, err := influxdbQuery(in.StartTime, in.EndTime, uint(in.Interval), containerName)
	if err != nil {
		log.Infof("query influxdb error=%v", err)
		return nil, rpc.ErrInternal
	}

	r.RscLimit = &rscLimit
	return r, nil
}

func (*ContainerServer) CreateBackup(ctx context.Context, in *pb.CreateBackupRequest) (*pb.CreateBackupReply, error) {
	if in.ContainerId == "" {
		return nil, rpc.ErrInvalidArgument
	}
	if in.BackupName == "" {
		now := time.Now()
		in.BackupName = now.Format("20060102150405") + fmt.Sprintf("%d", now.Nanosecond()/1000000)
	}

	if !args.BackupDesc.Check(in.BackupDesc) {
		return nil, status.Errorf(codes.InvalidArgument, "参数错误：备份描述无效")
	} else if !args.ImageVersion.Check(in.BackupName) {
		return nil, status.Errorf(codes.InvalidArgument, "参数错误：备份版本名无效")
	}

	cfgs, err := litedb.GetContainerConfig(in.ContainerId)
	if err != nil {
		log.Infof("GetContainerConfigs err=%v", err)
		return nil, rpc.ErrInternal
	}

	b, err := litedb.CreateContainerBackup(cfgs.UUID, in.BackupName, in.BackupDesc)
	if err != nil {
		if strings.HasPrefix(err.Error(), "UNIQUE constraint failed") {
			return nil, status.Errorf(codes.InvalidArgument, "已存在同名备份")
		}

		return nil, rpc.ErrInternal
	}

	b.ContainerID = in.ContainerId
	go addBackupJob(b)

	return &pb.CreateBackupReply{}, nil
}

func (*ContainerServer) UpdateBackup(ctx context.Context, in *pb.UpdateBackupRequest) (*pb.UpdateBackupReply, error) {
	if !args.BackupDesc.Check(in.BackupDesc) {
		return nil, status.Errorf(codes.InvalidArgument, "参数错误：备份描述无效")
	}

	b, err := litedb.QueryContainerBackupByID(in.Id)
	if err != nil {
		log.Infof("db.QueryContainerBackupByID err=%v", err)
		if err == db.ErrRecordNotFound {
			return nil, rpc.ErrNotFound
		}
		return nil, rpc.ErrInternal
	}

	b.BackupDesc = in.BackupDesc
	if err := litedb.UpdateContainerBackup(b); err != nil {
		log.Infof("db.UpdateContainerBackup err=%v", err)
		return nil, rpc.ErrInternal
	}

	return &pb.UpdateBackupReply{}, nil
}

func (*ContainerServer) RemoveBackup(ctx context.Context, in *pb.RemoveBackupRequest) (*pb.RemoveBackupReply, error) {
	b, err := litedb.QueryContainerBackupByID(in.Id)
	if err != nil {
		log.Infof("db.QueryContainerBackupByID err=%v", err)
		if err == db.ErrRecordNotFound {
			return nil, rpc.ErrNotFound
		}
		return nil, rpc.ErrInternal
	}

	cfgs, err := litedb.GetContainerConfigByUUID(b.UUID)
	if err != nil {
		log.Infof("db.GetContainerConfigsByUUID err=%v", err)
		if err == db.ErrRecordNotFound {
			return nil, rpc.ErrNotFound
		}
		return nil, rpc.ErrInternal
	}

	cli, err := docker.Client()
	if err != nil {
		return nil, rpc.ErrInternal
	}

	ci, err := cli.ContainerInspect(ctx, cfgs.ContainerID)
	if err != nil {
		log.Infof("ContainerInspect err=%v", err)
		return nil, rpc.ErrInternal
	}

	if ci.Config.Image == b.ImageRef {
		return nil, rpc.ErrRemoveContainerBackupWhenRunning
	}

	if err := litedb.RemoveContainerBackup(b); err != nil {
		log.Warnf("db remove backup record err=%v", err)
		return nil, rpc.ErrInternal
	}

	docker.RemoveImage(ctx, b.ImageRef, false)

	return &pb.RemoveBackupReply{}, nil
}

func (s *ContainerServer) ResumeBackup(ctx context.Context, in *pb.ResumeBackupRequest) (*pb.ResumeBackupReply, error) {
	b, err := litedb.QueryContainerBackupByID(in.BackupId)
	if err != nil {
		log.Infof("db.QueryContainerBackupByID err=%v", err)
		if err == db.ErrRecordNotFound {
			return nil, rpc.ErrNotFound
		}
		return nil, rpc.ErrInternal
	}

	if b.Status != int8(pb.BACKUP_STATUS_SUCCEED) && b.ContainerID != in.ContainerId {
		log.Infof("invalid backup=%+v target container=%v", b, in.ContainerId)
		return nil, rpc.ErrInvalidArgument
	}

	backups, err := litedb.QueryContainerBackupByUUID(b.UUID)
	if err != nil {
		log.Infof("QueryContainerBackupByUUID uuid=%v err=%v", b.UUID, err)
		return nil, rpc.ErrInternal
	}

	cfgs, err := litedb.GetContainerConfigByUUID(b.UUID)
	if err != nil {
		log.Warnf("db get container configs uuid=%v err=%v", b.UUID, err)
		return nil, rpc.ErrInternal
	}

	cli, err := docker.Client()
	if err != nil {
		return nil, rpc.ErrInternal
	}

	// inspect container
	configs, _, err := s.inspect(in.ContainerId, true)
	if err != nil {
		log.Warnf("inspect container=%v err=%v", in.ContainerId, err)
		return nil, rpc.ErrInternal
	}

	// remove old container
	if err := s.remove(cli, in.ContainerId, true); err != nil {
		log.Warnf("remove container=%v err=%v", in.ContainerId, err)
		return nil, rpc.ErrInternal
	}

	// create new contaienr
	configs.Image = b.ImageRef
	configs.SecurityConfig = cfgs.SecurityConfig
	id, err := s.create(configs)
	if err != nil {
		log.Warnf("create container err=%v", err)
		return nil, rpc.ErrInternal
	}

	cfgs.ContainerID = id
	litedb.UpdateContainerConfig(cfgs)

	for _, b := range backups {
		b.ContainerID = id
	}
	if err := litedb.UpdateContainerBackup(backups); err != nil {
		log.Infof("db.UpdateContainerBackup err=%v", err)
		return nil, rpc.ErrInternal
	}

	return &pb.ResumeBackupReply{ContainerId: id}, nil
}

func (*ContainerServer) ListBackup(ctx context.Context, in *pb.ListBackupRequest) (*pb.ListBackupReply, error) {
	cfgs, err := litedb.GetContainerConfig(in.ContainerId)
	if err != nil {
		log.Infof("GetContainerConfigs err=%v", err)
		return nil, rpc.ErrInternal

	}
	data, err := litedb.QueryContainerBackupByUUID(cfgs.UUID)
	if err != nil {
		log.Infof("db.QueryContainerBackupByUUID err=%v", err)
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

func (*ContainerServer) ListBackupImage(ctx context.Context, in *pb.ListBackupImageRequest) (*pb.ListBackupImageReply, error) {
	backups, err := litedb.ListContainerBackup()
	if err != nil {
		return nil, rpc.ErrDatabaseFail
	}

	var r pb.ListBackupImageReply
	for _, b := range backups {
		r.Images = append(r.Images, b.ImageRef)
	}

	return &r, nil
}

func addBackupJob(b *litedb.ContainerBackup) {
	cli, err := docker.Client()
	if err != nil {
		log.Warnf("create docker cli err=%v", err)
		b.Status = 2
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), common.Config.Agent.BackupTimeoutDuration())
	defer cancel()

	info, err := cli.ContainerInspect(ctx, b.ContainerID)
	if err != nil {
		log.Warnf("inspect container=%v err=%v", b.ContainerID, err)
		b.Status = 2
		if err := litedb.UpdateContainerBackup(b); err != nil {
			log.Errorf("db.UpdateContainerBackup err=%v", err)
		}
		return
	}

	if info.Config == nil {
		log.Warnf("inspect container=%v .Config=nil", b.ContainerID)
		b.Status = 2
		if err := litedb.UpdateContainerBackup(b); err != nil {
			log.Errorf("db.UpdateContainerBackup err=%v", err)
		}
		return
	}

	b.ImageRef = strings.Split(info.Config.Image, ":")[0] + ":" + b.BackupName
	newImage, err := cli.ContainerCommit(ctx, b.ContainerID, types.ContainerCommitOptions{
		Reference: b.ImageRef,
		Pause:     true,
	})
	if err != nil {
		log.Warnf("commit container=%v err=%v", b.ContainerID, err)
		b.Status = 2
		if err := litedb.UpdateContainerBackup(b); err != nil {
			log.Errorf("db.UpdateContainerBackup err=%v", err)
		}
		return
	}

	imageInfo, _, err := cli.ImageInspectWithRaw(ctx, newImage.ID)
	if err != nil {
		log.Warnf("inspect image=%v err=%v", newImage.ID, err)
		b.Status = 2
	} else {
		log.Debugf("backup id=%v finished", b.ID)
		b.ImageID = strings.TrimPrefix(newImage.ID, "sha256:")
		b.ImageSize = imageInfo.Size
		b.Status = 1
	}

	if err := litedb.UpdateContainerBackup(b); err != nil {
		log.Errorf("db.UpdateContainerBackup err=%v", err)
	}
}

func (*ContainerServer) ExportBackup(in *pb.ExportBackupRequest, stream pb.Container_ExportBackupServer) error {
	b, err := litedb.QueryContainerBackupByID(in.BackupId)
	if err != nil {
		log.Warnf("QueryContainerBackupByID err=%v", err)
		return rpc.ErrInternal

	}

	if b.Status != int8(pb.BACKUP_STATUS_SUCCEED) {
		return status.Error(codes.Internal, "备份失败，无法导出")
	}

	cli, err := docker.Client()
	if err != nil {
		return rpc.ErrInternal
	}

	img, _, err := cli.ImageInspectWithRaw(context.Background(), b.ImageRef)
	if err != nil {
		if docker.IsErrNotFound(err) {
			return rpc.ErrNotFound
		}
		log.Warnf("ImageInspectWithRaw err=%v", err)
		return rpc.ErrInternal
	}

	r, err := cli.ImageSave(context.Background(), []string{b.ImageRef})
	if err != nil {
		log.Warnf("ImageSave err=%v", err)
		return rpc.ErrInternal
	}
	defer r.Close()

	chunk := pb.ExportBackupReply{
		ImgSize:    img.Size,
		ImgName:    strings.Split(b.ImageRef, ":")[0],
		ImgVersion: in.ImgVersion,
	}
	if err := stream.Send(&chunk); err != nil {
		log.Warnf("ExportBackup send image info err=%v", err)
		return rpc.ErrInternal
	}

	chunk.ImgSize = 0
	chunk.ImgName = ""
	chunk.ImgVersion = ""
	buf := make([]byte, 16*1024)
	for {
		n, err := r.Read(buf)
		if err != nil {
			if err == io.EOF {
				break
			}
			log.Warnf("read image content err=%v", err)
			return rpc.ErrInternal
		}
		chunk.Data = buf[:n]
		if err := stream.Send(&chunk); err != nil {
			log.Warnf("ExportBackup send stream err=%v", err)
			return rpc.ErrInternal
		}
	}
	return nil
}

func (*ContainerServer) InspectBackup(ctx context.Context, in *pb.InspectBackupRequest) (*pb.InspectBackupReply, error) {
	b, err := litedb.QueryContainerBackupByID(in.BackupId)
	if err != nil {
		log.Warnf("GetContainerConfigs err=%v", err)
		return nil, rpc.ErrInternal

	}

	var reply pb.InspectBackupReply
	reply.Backup = &pb.ContainerBackup{
		Id:         b.ID,
		Uuid:       b.UUID,
		BackupName: b.BackupName,
		BackupDesc: b.BackupDesc,
		ImageRef:   b.ImageRef,
		ImageSize:  b.ImageSize,
		Status:     int64(b.Status),
		CreatedAt:  b.CreatedAt,
	}

	return &reply, nil
}

func (*ContainerServer) AddAppEntry(ctx context.Context, in *pb.AddAppEntryRequest) (*pb.AddAppEntryReply, error) {
	if in.App == nil {
		return nil, rpc.ErrInvalidArgument
	}

	if err := litedb.AddAppEntry(&litedb.AppEntry{
		ContainerID: in.App.ContainerId,
		Name:        in.App.Name,
		ExePath:     in.App.ExePath,
		IsGUI:       in.App.IsGui,
	}); err != nil {
		log.Warnf("AddAppEntry err=%v", err)
		return nil, rpc.ErrDatabaseFail
	}

	return &pb.AddAppEntryReply{}, nil
}

func (*ContainerServer) RemoveAppEntry(ctx context.Context, in *pb.RemoveAppEntryRequest) (*pb.RemoveAppEntryReply, error) {
	if len(in.AppIds) == 0 {
		return nil, rpc.ErrInvalidArgument
	}

	for _, appId := range in.AppIds {
		e, err := litedb.GetAppEntryByID(appId)
		if err != nil {
			log.Warnf("litedb.GetAppEntryByID err=%v", err)
			return nil, rpc.ErrDatabaseFail
		}

		if appIsRunning(ctx, e.IDResponse) {
			return nil, status.Errorf(codes.Internal, "%v正在运行,无法删除", e.Name)
		}
	}

	if err := litedb.RemoveAppEntry(in.AppIds); err != nil {
		log.Warnf("litedb.RemoveAppEntry err=%v", err)
		return nil, rpc.ErrDatabaseFail
	}

	return &pb.RemoveAppEntryReply{}, nil
}

func (*ContainerServer) UpdateAppEntry(ctx context.Context, in *pb.UpdateAppEntryRequest) (*pb.UpdateAppEntryReply, error) {
	if in.App == nil {
		return nil, rpc.ErrInvalidArgument
	}

	e, err := litedb.GetAppEntryByID(in.App.Id)
	if err != nil {
		log.Warnf("litedb.GetAppEntryByID err=%v", err)
		return nil, rpc.ErrDatabaseFail
	}

	if appIsRunning(ctx, e.IDResponse) {
		return nil, status.Error(codes.Internal, "应用正在运行")
	}

	e.ExePath = in.App.ExePath
	e.Name = in.App.Name
	e.IsGUI = in.App.IsGui
	if err := litedb.UpdateAppEntry(e); err != nil {
		log.Warnf("litedb.UpdateAppEntry err=%v", err)
		return nil, rpc.ErrDatabaseFail
	}

	return &pb.UpdateAppEntryReply{}, nil
}

func (*ContainerServer) ListAppEntry(ctx context.Context, in *pb.ListAppEntryRequest) (*pb.ListAppEntryReply, error) {
	if in.ContainerId == "" {
		return nil, rpc.ErrInvalidArgument
	}

	entries, err := litedb.ListAppEntry(in.ContainerId)
	if err != nil {
		log.Warnf("litedb.ListAppEntry err=%v", err)
		return nil, rpc.ErrDatabaseFail
	}

	var reply pb.ListAppEntryReply
	for _, e := range entries {
		reply.Apps = append(reply.Apps, &pb.AppEntry{
			Id:          e.ID,
			ContainerId: e.ContainerID,
			Name:        e.Name,
			ExePath:     e.ExePath,
			IsGui:       e.IsGUI,
			IsRunning:   appIsRunning(ctx, e.IDResponse),
		})
	}

	return &reply, nil
}

func (*ContainerServer) RunAppEntry(ctx context.Context, in *pb.RunAppEntryRequest) (*pb.RunAppEntryReply, error) {
	app, err := litedb.GetAppEntryByID(in.AppId)
	if err != nil {
		log.Warnf("litedb.GetAppEntryByID err=%v", err)
		return nil, rpc.ErrDatabaseFail
	}

	if app.IsGUI {
		return nil, status.Error(codes.InvalidArgument, "无法运行GUI应用")
	}

	cli, err := docker.Client()
	if err != nil {
		log.Warnf("docker.Client err:%s", err)
		return nil, rpc.ErrInternal
	}

	if !containerIsRunning(ctx, cli, app.ContainerID) {
		return nil, status.Error(codes.InvalidArgument, "容器未运行")
	}

	if appIsRunningWithCli(ctx, cli, app.IDResponse) {
		return nil, status.Error(codes.Internal, "应用正在运行")
	}

	idRes, err := cli.ContainerExecCreate(ctx, app.ContainerID, types.ExecConfig{
		Cmd: strings.Split(app.ExePath, " "),
	})
	if err != nil {
		log.Warnf("ContainerExecCreate err=%v", err)
		return nil, rpc.ErrInternal
	}
	if idRes.ID == "" {
		log.Warnln("ContainerExecCreate exec ID empty")
		return nil, rpc.ErrInternal
	}

	if err := cli.ContainerExecStart(ctx, idRes.ID, types.ExecStartCheck{
		Detach: true,
		Tty:    false,
	}); err != nil {
		log.Warnf("ContainerExecStart err=%v", err)
		return nil, rpc.ErrInternal
	}

	app.IDResponse = idRes.ID
	if err := litedb.UpdateAppEntry(app); err != nil {
		log.Warnf("litedb.UpdateAppEntry err=%v", err)
		return nil, rpc.ErrDatabaseFail
	}

	return &pb.RunAppEntryReply{}, nil
}

func (*ContainerServer) KillAppEntry(ctx context.Context, in *pb.KillAppEntryRequest) (*pb.KillAppEntryReply, error) {
	app, err := litedb.GetAppEntryByID(in.AppId)
	if err != nil {
		log.Warnf("litedb.GetAppEntryByID err=%v", err)
		return nil, rpc.ErrDatabaseFail
	}

	if app.IsGUI {
		return nil, status.Error(codes.InvalidArgument, "无法停止GUI应用")
	}

	cli, err := docker.Client()
	if err != nil {
		log.Warnf("docker.Client err:%s", err)
		return nil, rpc.ErrInternal
	}

	if !containerIsRunning(ctx, cli, app.ContainerID) {
		return nil, status.Error(codes.InvalidArgument, "容器未运行")
	}

	execIns, err := cli.ContainerExecInspect(ctx, app.IDResponse)
	if err != nil {
		log.Warnf("cli.ContainerExecInspect err:%s", err)
		return nil, rpc.ErrInternal
	}

	if !execIns.Running {
		return nil, status.Error(codes.Internal, "应用未运行")
	}

	err = syscall.Kill(execIns.Pid, syscall.SIGKILL)
	if err != nil {
		log.Warnf("cli.ContainerExecInspect err:%s", err)
		return nil, status.Error(codes.Internal, "停止应用失败")
	}

	return &pb.KillAppEntryReply{}, nil
}
func (*ContainerServer) Pause(ctx context.Context, in *pb.PauseRequest) (*pb.PauseReply, error) {
	if in.ContainerId == "" {
		return nil, rpc.ErrInvalidArgument
	}
	cli, err := docker.Client()
	if err != nil {
		return nil, rpc.ErrInternal
	}
	if err = cli.ContainerPause(ctx, in.ContainerId); err != nil {
		log.Errorf("ContainerPause: id=%v %v", in.ContainerId, err)
		return nil, transDockerError(err)
	}
	return &pb.PauseReply{}, nil
}
func (*ContainerServer) Unpause(ctx context.Context, in *pb.UnpauseRequest) (*pb.UnpauseReply, error) {
	if in.ContainerId == "" {
		return nil, rpc.ErrInvalidArgument
	}
	cli, err := docker.Client()
	if err != nil {
		return nil, rpc.ErrInternal
	}
	if err = cli.ContainerUnpause(ctx, in.ContainerId); err != nil {
		log.Errorf("ContainerPause: id=%v %v", in.ContainerId, err)
		return nil, transDockerError(err)
	}
	return &pb.UnpauseReply{}, nil
}
func (*ContainerServer) LoadImage(in *pb.LoadRequest, stream pb.Container_LoadImageServer) error {
	if _, err := os.Stat(in.TarPath); err != nil {
		log.Errorf("file not found err: %v", err)
		return rpc.ErrInternal
	}
	pr, pw := io.Pipe()
	defer pr.Close()
	go func() {
		defer pw.Close()
		file, err := os.Open(in.TarPath)
		if err != nil {
			log.Errorf("open file failed: %v", err)
			pw.CloseWithError(rpc.ErrNotFound)
			return
		}
		defer file.Close()
		buf := make([]byte, bufSize)
		for {
			n, err := file.Read(buf)
			if err != nil && err != io.EOF {
				log.Errorf("read tar file error: %v", err)
				pw.CloseWithError(err)
				return
			}
			if n > 0 {
				if _, err := pw.Write(buf[:n]); err != nil {
					log.Errorf("write to buf error: %v", err)
					return
				}
				if err == io.EOF {
					log.Infof("write to buf finished.")
					return
				}
			}
		}
	}()
	dockerCli, err := docker.ClientWithTimeout(dockerTimeout)
	if err != nil {
		log.Errorf("docker cli init error: %v", err)
	}
	defer dockerCli.Close()
	ctx, cancel := context.WithTimeout(stream.Context(), dockerTimeout)
	defer cancel()
	resp, err := dockerCli.ImageLoad(ctx, pr, false)
	if err != nil {
		log.Errorf("image loaded error: %v", err)
	}
	defer resp.Body.Close()
	decoder := json.NewDecoder(resp.Body)
	var lastPercent float32
	for {
		var msg jsonmessage.JSONMessage
		if err = decoder.Decode(&msg); err != nil {
			if err == io.EOF {
				return stream.Send(&pb.LoadProgress{
					Update: &pb.LoadProgress_Result{
						Result: &pb.LoadResult{
							Success: true,
							Detail:  "image load success",
						},
					},
				})
			}
			log.Errorf("response resolve failed: %v", err)
			return rpc.ErrInternal
		}
		if msg.Error != nil {
			return stream.Send(&pb.LoadProgress{
				Update: &pb.LoadProgress_Result{
					Result: &pb.LoadResult{
						Success: false,
						Detail:  msg.Error.Message,
					},
				},
			})
		}
		// 处理进度更新
		if msg.Progress != nil {
			currentPercent := calculatePercent(msg.Progress)
			if currentPercent != lastPercent {
				lastPercent = currentPercent
				if err := stream.Send(&pb.LoadProgress{
					Update: &pb.LoadProgress_Percent{Percent: currentPercent},
				}); err != nil {
					log.Error("image load progress update failed: %v", err)
					return rpc.ErrInternal
				}
			}
		}
		if msg.Stream != "" {
			if err := stream.Send(&pb.LoadProgress{
				Update: &pb.LoadProgress_Message{Message: msg.Stream},
			}); err != nil {
				log.Error("image load message update failed: %v", err)
				return rpc.ErrInternal
			}
		}
	}
}
func calculatePercent(p *jsonmessage.JSONProgress) float32 {
	if p.Total == 0 {
		return 0
	}
	return float32(p.Current) / float32(p.Total) * 100
}
