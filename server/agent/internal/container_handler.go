package internal

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/shirou/gopsutil/mem"
	log "github.com/sirupsen/logrus"

	"scmc/model"
	"scmc/rpc"
	pb "scmc/rpc/pb/container"
)

const defaultCPUShares = 1024

func toCPUShares(cpuPrio int64) int64 {
	if cpuPrio <= 0 {
		return defaultCPUShares
	}
	return cpuPrio + defaultCPUShares // default cpu shares 1024
}

func fromCPUShares(cpuShares int64) int64 {
	if cpuShares <= defaultCPUShares {
		return 0
	}
	return cpuShares - defaultCPUShares
}

func dockerResourceConfig(in *pb.ResourceLimit) container.Resources {
	var r container.Resources

	if in != nil {
		r.NanoCPUs = int64(in.CpuLimit * 1e9)
		if r.NanoCPUs == 0 {
			r.NanoCPUs = int64(numCPU()) * 1e9
		}
		r.CPUShares = toCPUShares(in.CpuPrio)
		r.Memory = int64(in.MemoryLimit * megaBytes)
		r.MemoryReservation = int64(in.MemorySoftLimit * megaBytes)
		r.MemorySwap = int64(in.MemoryLimit * megaBytes) // prevent error: Memory limit should be smaller than already set memoryswap limit
	}

	return r
}

func ensureLocalImage(cli *client.Client, image string) error {
	list, err := cli.ImageList(context.Background(), types.ImageListOptions{})
	if err != nil {
		log.Warnf("ImageList: %v", err)
		return nil
	}
	log.Debugf("image: %v", image)
	for _, i := range list {
		log.Debugf("i.ID: %v", i.ID)
		if i.ID == image {
			return nil
		}
		for _, s := range i.RepoTags {
			log.Debugf("repotag: %v", s)
			if s == image {
				return nil
			}
		}
	}

	return errors.New("image is not in local")
}

func ensureImage(cli *client.Client, image string) error {
	err := ensureLocalImage(cli, image)
	if err != nil {
		log.Warnf("IsLocalImageExist err: %v", err)
	}

	imageExists, err := model.IsImageExist(image)
	if err != nil {
		log.Debugf("IsImageExist err: %v", err)
		return err
	}

	if !imageExists {
		return errors.New("image is not in registry")
	}

	if err = model.PullImage(image); err != nil {
		log.Errorf("pull image[%v] err: %v", image, err)
		return err
	}
	return nil
}

type ContainerServer struct {
	pb.UnimplementedContainerServer
}

func (s *ContainerServer) List(ctx context.Context, in *pb.ListRequest) (*pb.ListReply, error) {
	reply := pb.ListReply{}

	cli, err := model.DockerClient()
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

	for _, c := range containers {
		info := pb.ContainerInfo{
			Id:      c.ID,
			Image:   c.Image,
			ImageId: c.ImageID,
			Command: c.Command,
			State:   c.State,
			Created: c.Created,
		}

		if c.State != "created" {
			ci, err := cli.ContainerInspect(ctx, c.ID)
			if err != nil {
				log.Warnf("ContainerInspect id=%v err=%v", c.ID, err)
				// dont return error
			} else {
				if ci.State != nil {
					startedAt, err := time.ParseInLocation(time.RFC3339Nano, ci.State.StartedAt, time.UTC)
					if err != nil {
						log.Warnf("ParseInLocation time=%v err=%v", ci.State.StartedAt, err)
					} else {
						info.Started = startedAt.Unix()
					}
				}
			}
		}

		// 参考docker cli实现 去掉link特性连接的其他容器名
		for _, name := range c.Names {
			if strings.HasPrefix(name, "/") && len(strings.Split(name[1:], "/")) == 1 {
				info.Name = name[1:]
				break
			}
		}

		initResourceStat := func() {
			info.ResourceStat = &pb.ResourceStat{
				CpuStat: &pb.CpuStat{},
				MemStat: &pb.MemoryStat{},
			}
		}
		if c.State == "running" {
			initResourceStat()
		}

		if stats != nil {
			if stat, ok := stats[c.ID]; ok {
				if info.ResourceStat == nil {
					initResourceStat()
				}
				if stat.CpuStat != nil {
					info.ResourceStat.CpuStat = stat.CpuStat
				}
				if stat.MemStat != nil {
					info.ResourceStat.MemStat = stat.MemStat
				}
			}
		}

		diskStat := &pb.DiskStat{
			Used: float64(c.SizeRootFs) / 1e6, // 与docker client保持一致
		}

		if v, ok := c.Labels["KS_SCMC_DISK_LIMIT"]; ok {
			fmt.Sscanf(v, "%f", &diskStat.Limit)
		}

		if info.ResourceStat != nil {
			info.ResourceStat.DiskStat = diskStat
		} else {
			info.ResourceStat = &pb.ResourceStat{
				DiskStat: diskStat,
			}
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
		if err := model.CleanFileAccess(id); err != nil {
			log.Warnf("%v CleanFileAccess err:%v", id, err)
			return rpc.ErrContainerFileProtection
		}
		if err := model.CleanWhiteList(id); err != nil {
			log.Warnf("%v CleanWhiteList err:%v", id, err)
			return rpc.ErrContainerProcProtection
		}
	}

	if sec.ProcProtection != nil {
		exeHashList, err := generateMD5Slice(id, sec.ProcProtection.ExeList)
		if err != nil {
			return rpc.ErrContainerProcProtection
		}
		if err := model.UpdateWhiteList(id, sec.ProcProtection.IsOn, exeHashList, []string{}); err != nil {
			log.Warnf("UpdateWhiteList %v err: %v", id, err)
			return rpc.ErrContainerProcProtection
		}
	}

	if sec.NprocProtection != nil {
		var inRule = &model.ProcProtection{
			IsOn:    sec.NprocProtection.IsOn,
			ExeList: sec.NprocProtection.ExeList,
		}

		if err := model.SaveOpensnitchRule(inRule, id, uuid); err != nil {
			log.Warnf("SaveOpensnitchRule id=%v err=%v", id, err)
			return rpc.ErrContainerNprocProtection
		}
	}

	if sec.FileProtection != nil {
		if err := model.UpdateFileAccess(id, sec.FileProtection.IsOn, sec.FileProtection.FileList, []string{}); err != nil {
			log.Warnf("UpdateFileAccess %v err: %v", id, err)
			return rpc.ErrContainerFileProtection
		}
	}

	if sec.DisableCmdOperation {
		if err := model.AddSensitiveContainers(name, id); err != nil {
			log.Warnf("AddSensitiveContainers err=%v", err)
			return rpc.ErrContainerCmdOperation
		}
	} else {
		if fromUpdate {
			if err := model.DelSensitiveContainers(name, id); err != nil {
				log.Warnf("DelSensitiveContainers err=%v", err)
				return rpc.ErrContainerCmdOperation
			}
		}

	}

	if sec.NetworkRule != nil {
		var rules []model.NetworkRule
		for _, v := range sec.NetworkRule.Rules {
			rules = append(rules, model.NetworkRule{
				Protocols: v.Protocols,
				Addr:      v.Addr,
				Port:      v.Port,
			})
		}

		fileName := id + "-" + name
		if err := model.UpdateContainerIPtablesFile(fileName, sec.NetworkRule.IsOn, rules, pid); err != nil {
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

	cli, err := model.DockerClient()
	if err != nil {
		return "", rpc.ErrInternal
	}

	if err := ensureLocalImage(cli, configs.Image); err != nil {
		if _, ok := status.FromError(err); ok {
			return "", err
		}
		return "", rpc.ErrInternal
	}

	config := container.Config{
		Image: configs.Image,
		Labels: map[string]string{
			"KS_SCMC_DESC": configs.Desc,
			"KS_SCMC_UUID": configs.Uuid,
		},
		AttachStdin: true, // AttachStdin, Tty: make sure container like OS(ubuntu, centos) will not exit after start
		Tty:         true,
	}
	hostConfig := container.HostConfig{
		Privileged: false, // force non-privileged
	}
	var networkConfig *network.NetworkingConfig

	for k, v := range configs.Envs {
		config.Env = append(config.Env, fmt.Sprintf("%s=%s", k, v))
	}
	config.Env = append(config.Env, fmt.Sprintf("KS_SCMC_UUID=%s", configs.Uuid))

	for _, m := range configs.Mounts {
		hostConfig.Mounts = append(hostConfig.Mounts, mount.Mount{
			Type:     mount.Type(m.Type),
			Source:   m.Source,
			Target:   m.Target,
			ReadOnly: m.ReadOnly,
		})
	}

	if configs.EnableGraphic {
		if err := containerGraphicSetup(configs.Name, &config, &hostConfig); err != nil {
			log.Infof("containerGraphicSetup err=%v", err)
			return "", rpc.ErrInternal
		}
		config.Labels["KS_SCMC_GRAPHIC"] = "1"
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

	var networkConfigCreate *network.NetworkingConfig
	if len(configs.Networks) > 0 {
		networkConfig = &network.NetworkingConfig{
			EndpointsConfig: make(map[string]*network.EndpointSettings, len(configs.Networks)),
		}
		for _, v := range configs.Networks {
			if !checkNetworkInfo("", v.Interface, &v.IpAddress, int(v.IpPrefixLen)) {
				return "", rpc.ErrInvalidArgument
			}

			networkConfig.EndpointsConfig[v.Interface] = &network.EndpointSettings{
				IPAMConfig: &network.EndpointIPAMConfig{},
			}

			networkConfig.EndpointsConfig[v.Interface].IPAMConfig.IPv4Address = v.IpAddress
			networkConfig.EndpointsConfig[v.Interface].IPAddress = v.IpAddress
			networkConfig.EndpointsConfig[v.Interface].IPPrefixLen = int(v.IpPrefixLen)
			networkConfig.EndpointsConfig[v.Interface].MacAddress = v.MacAddress
			networkConfig.EndpointsConfig[v.Interface].Gateway = v.Gateway
		}

		ifs := configs.Networks[0].Interface
		networkConfigCreate = &network.NetworkingConfig{
			EndpointsConfig: make(map[string]*network.EndpointSettings, 1),
		}
		networkConfigCreate.EndpointsConfig[ifs] = networkConfig.EndpointsConfig[ifs]
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
	id, err := s.create(in.Configs)
	return &pb.CreateReply{ContainerId: id}, err
}

func (s *ContainerServer) Start(ctx context.Context, in *pb.StartRequest) (*pb.StartReply, error) {
	reply := pb.StartReply{}

	if (len(in.Ids)) <= 0 || len(in.Ids[0].ContainerIds) <= 0 {
		return nil, rpc.ErrInvalidArgument
	}

	cli, err := model.DockerClient()
	if err != nil {
		return nil, rpc.ErrInternal
	}

	opts := types.ContainerStartOptions{}
	for _, id := range in.Ids[0].ContainerIds {
		if err := cli.ContainerStart(context.Background(), id, opts); err != nil {
			log.Warnf("ContainerStart: id=%v %v", id, err)
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

func (s *ContainerServer) parallelProcessing(operate string, contaienrIds []string) ([]string, []*pb.ContainerFailInfo, error) {
	cli, err := model.DockerClient()
	if err != nil {
		return nil, nil, rpc.ErrInternal
	}

	timeout := time.Second
	var wg sync.WaitGroup
	var lock sync.Mutex
	var okIds []string
	var failInfos []*pb.ContainerFailInfo

	for _, id := range contaienrIds {
		wg.Add(1)
		go func(containerId string) {
			defer wg.Done()
			var err error
			if "ContainerStop" == operate {
				err = cli.ContainerStop(context.Background(), containerId, &timeout)
			} else if "ContainerRestart" == operate {
				err = cli.ContainerRestart(context.Background(), containerId, &timeout)
			} else {
				log.Warnf("%v: id=%v ", operate, containerId)
				return
			}

			if err != nil {
				log.Warnf("%v: id=%v %v", operate, containerId, err)
				var faileReason string
				if s, _ := status.FromError(err); s != nil {
					faileReason = s.Message()
				}
				lock.Lock()
				failInfos = append(failInfos, &pb.ContainerFailInfo{
					ContainerId: containerId,
					FailReason:  faileReason,
				})
				lock.Unlock()
				return
			}
			okIds = append(okIds, containerId)
		}(id)
	}

	wg.Wait()
	return okIds, failInfos, nil
}

func (s *ContainerServer) Stop(ctx context.Context, in *pb.StopRequest) (*pb.StopReply, error) {
	reply := pb.StopReply{}
	if (len(in.Ids)) <= 0 || len(in.Ids[0].ContainerIds) <= 0 {
		return nil, rpc.ErrInvalidArgument
	}

	okIds, failInfos, err := s.parallelProcessing("ContainerStop", in.Ids[0].ContainerIds)
	if err != nil {
		return nil, err
	}

	reply.OkIds = append(reply.OkIds, okIds...)
	reply.FailInfos = append(reply.FailInfos, failInfos...)

	return &reply, nil
}

func (s *ContainerServer) Kill(ctx context.Context, in *pb.KillRequest) (*pb.KillReply, error) {
	reply := pb.KillReply{}
	if (len(in.Ids)) <= 0 || len(in.Ids[0].ContainerIds) <= 0 {
		return nil, rpc.ErrInvalidArgument
	}

	cli, err := model.DockerClient()
	if err != nil {
		return nil, rpc.ErrInternal
	}

	for _, id := range in.Ids[0].ContainerIds {
		if err := cli.ContainerKill(context.Background(), id, ""); err != nil { // TODO signal
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
	reply := pb.RestartReply{}
	if (len(in.Ids)) <= 0 || len(in.Ids[0].ContainerIds) <= 0 {
		return nil, rpc.ErrInvalidArgument
	}

	okIds, failInfos, err := s.parallelProcessing("ContainerRestart", in.Ids[0].ContainerIds)
	if err != nil {
		return nil, err
	}

	reply.OkIds = append(reply.OkIds, okIds...)
	reply.FailInfos = append(reply.FailInfos, failInfos...)

	return &reply, nil
}

func (s *ContainerServer) Update(ctx context.Context, in *pb.UpdateRequest) (*pb.UpdateReply, error) {
	reply := pb.UpdateReply{}

	if in.ContainerId == "" || (in.ResourceLimit == nil && in.RestartPolicy == nil && in.Networks == nil && in.SecurityConfig == nil) {
		return nil, rpc.ErrInvalidArgument
	}

	if in.Networks != nil {
		for _, v := range in.Networks {
			if !checkNetworkInfo(in.ContainerId, v.Interface, &v.IpAddress, int(v.IpPrefixLen)) {
				return nil, rpc.ErrInvalidArgument
			}
		}
	}

	inspectConfigs, err := s.inspect(in.ContainerId, false)
	if err != nil {
		log.Warnf("inspect container=%v err=%v", in.ContainerId, err)
		return nil, rpc.ErrInternal
	}

	cli, err := model.DockerClient()
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

	var pid int
	if in.Networks != nil {
		info, err := cli.ContainerInspect(context.Background(), in.ContainerId)
		if err != nil {
			log.Warnf("ContainerInspect: %v", err)
			return nil, transDockerError(err)
		}
		if info.State != nil {
			pid = info.State.Pid
		}

		if info.NetworkSettings != nil {
			m := make(map[string]struct{})
			for _, v := range in.Networks {
				m[v.Interface] = struct{}{}
			}
			for k, _ := range info.NetworkSettings.Networks {
				if _, ok := m[k]; !ok {
					if err = cli.NetworkDisconnect(context.Background(), k, in.ContainerId, true); err != nil {
						log.Warnf("NetworkDisconnect: %v", err)
					}
				}
			}

			for _, v := range in.Networks {
				_, ok := info.NetworkSettings.Networks[v.Interface]
				if ok {
					//修改
					if v.IpAddress != info.NetworkSettings.Networks[v.Interface].IPAddress {
						if err = cli.NetworkDisconnect(context.Background(), v.Interface, in.ContainerId, true); err != nil {
							log.Warnf("NetworkDisconnect: %v", err)
						}
					} else {
						continue
					}
				}
				config := &network.EndpointSettings{
					NetworkID:   v.Interface,
					Gateway:     v.Gateway,
					IPAddress:   v.IpAddress,
					IPPrefixLen: int(v.IpPrefixLen),
					MacAddress:  v.MacAddress,
				}

				config.IPAMConfig = &network.EndpointIPAMConfig{
					IPv4Address: v.IpAddress,
				}

				err = cli.NetworkConnect(context.Background(), v.Interface, in.ContainerId, config)
				if err != nil {
					log.Warnf("NetworkConnect: %v", err)
					return nil, transDockerError(err)
				}
			}
		} else {
			for _, v := range in.Networks {
				config := &network.EndpointSettings{
					NetworkID:   v.Interface,
					Gateway:     v.Gateway,
					IPAddress:   v.IpAddress,
					IPPrefixLen: int(v.IpPrefixLen),
					MacAddress:  v.MacAddress,
				}

				config.IPAMConfig = &network.EndpointIPAMConfig{
					IPv4Address: v.IpAddress,
				}

				err = cli.NetworkConnect(context.Background(), v.Interface, in.ContainerId, config)
				if err != nil {
					log.Warnf("NetworkConnect: %v", err)
					return nil, transDockerError(err)
				}
			}
		}
	}

	if err := s.setSecurityConfig(in.ContainerId, inspectConfigs.Uuid, inspectConfigs.Name, pid, true, in.SecurityConfig); err != nil {
		log.Warnf("Update setSecurityConfig err=%v", err)
		return nil, err
	}

	return &reply, nil
}

func (s *ContainerServer) remove(cli *client.Client, containerID string, force bool) error {
	if cli == nil {
		c, err := model.DockerClient()
		if err != nil {
			return rpc.ErrInternal
		}
		cli = c
	}

	configs, err := s.inspect(containerID, false)
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

	if err := model.CleanFileAccess(containerID); err != nil {
		log.Warnf("CleanFileAccess container=%v err=%v", containerID, err)
	}
	if err := model.CleanWhiteList(containerID); err != nil {
		log.Warnf("CleanWhiteList container=%v err=%v", containerID, err)
	}

	if err := model.RemoveOpensnitchRule(containerID); err != nil {
		log.Warnf("RemoveOpensnitchRule container=%v err=%v", containerID, err)
	}

	if err := removeContainerGraphicSetup(containerID); err != nil {
		log.Warnf("removeContainerGraphicSetup container=%v err=%v", containerID, err)
	}

	fileName := containerID + "-" + configs.Name
	model.ContainerRemveIPtables(fileName)

	if err := model.DelSensitiveContainers(configs.Name, containerID); err != nil {
		log.Warnf("DelSensitiveContainers err=%v", err)
	}

	return nil
}

func (s *ContainerServer) Remove(ctx context.Context, in *pb.RemoveRequest) (*pb.RemoveReply, error) {
	if (len(in.Ids)) <= 0 || len(in.Ids[0].ContainerIds) <= 0 {
		return nil, rpc.ErrInvalidArgument
	}

	cli, err := model.DockerClient()
	if err != nil {
		return nil, rpc.ErrInternal
	}

	reply := pb.RemoveReply{}
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
		reply.OkIds = append(reply.OkIds, id)
	}

	return &reply, nil
}

func (s *ContainerServer) inspect(id string, backup bool) (*pb.ContainerConfigs, error) {
	cli, err := model.DockerClient()
	if err != nil {
		return nil, rpc.ErrInternal
	}

	info, _, err := cli.ContainerInspectWithRaw(context.Background(), id, true)
	if err != nil {
		log.Warnf("ContainerInspectWithRaw: %v", err)
		return nil, transDockerError(err)
	}

	configs := &pb.ContainerConfigs{
		ContainerId:  info.ID,
		Name:         strings.TrimPrefix(info.Name, "/"),
		Image:        info.Image,
		ResouceLimit: &pb.ResourceLimit{},
	}

	if info.State != nil {
		configs.Status = info.State.Status
	}

	if info.Config != nil {
		configs.Image = info.Config.Image
		if v, ok := info.Config.Labels["KS_SCMC_DESC"]; ok {
			configs.Desc = v
		}

		if v, ok := info.Config.Labels["KS_SCMC_GRAPHIC"]; ok {
			if v == "1" {
				configs.EnableGraphic = true
			}
		}

		if v, ok := info.Config.Labels["KS_SCMC_UUID"]; ok {
			configs.Uuid = v
		}

		if len(info.Config.Env) > 0 {
			configs.Envs = make(map[string]string, len(info.Config.Env))
			for _, e := range info.Config.Env {
				parts := strings.SplitN(e, "=", 2)
				if len(parts) >= 2 {
					configs.Envs[parts[0]] = parts[1]
				}
			}
		}

		for _, m := range info.Mounts {
			configs.Mounts = append(configs.Mounts, &pb.Mount{
				Type:     string(m.Type),
				Source:   m.Source,
				Target:   m.Destination,
				ReadOnly: !m.RW,
			})
		}
	}

	if info.HostConfig != nil {
		configs.RestartPolicy = &pb.RestartPolicy{
			Name:     info.HostConfig.RestartPolicy.Name,
			MaxRetry: int32(info.HostConfig.RestartPolicy.MaximumRetryCount),
		}

		configs.ResouceLimit = &pb.ResourceLimit{
			CpuLimit:        float64(info.HostConfig.Resources.NanoCPUs) / 1e9,
			CpuPrio:         fromCPUShares(info.HostConfig.Resources.CPUShares),
			MemoryLimit:     float64(info.HostConfig.Resources.Memory) / megaBytes,
			MemorySoftLimit: float64(info.HostConfig.Resources.MemoryReservation) / megaBytes,
		}

		if s, ok := info.HostConfig.StorageOpt["size"]; ok {
			fmt.Sscanf(s, "%fM", &configs.ResouceLimit.DiskLimit)
		}
	}

	if info.NetworkSettings != nil {
		for k, m := range info.NetworkSettings.Networks {
			if k == "bridge" {
				continue // "bridge" 是默认创建的网络连接, 需要忽略
			}

			var ipAddr string
			if m.IPAMConfig != nil {
				// 返回用户填的IP地址
				ipAddr = m.IPAMConfig.IPv4Address
			}

			if !backup && m.IPAddress != "" {
				// 非备份情况下，当用户没填IP地址，返回运行时IP
				ipAddr = m.IPAddress
			}

			configs.Networks = append(configs.Networks, &pb.NetworkConfig{
				Interface:   k,
				ContainerId: info.ID,
				IpAddress:   ipAddr,
				IpPrefixLen: int32(m.IPPrefixLen),
				MacAddress:  m.MacAddress,
				Gateway:     m.Gateway,
			})
		}
	}

	return configs, nil
}

func (s *ContainerServer) Inspect(ctx context.Context, in *pb.InspectRequest) (*pb.InspectReply, error) {
	configs, err := s.inspect(in.ContainerId, false)
	if err != nil {
		return nil, err
	}
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
		cli, err := model.DockerClient()
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

func (*ContainerServer) RemoveBackup(ctx context.Context, in *pb.RemoveBackupRequest) (*pb.RemoveBackupReply, error) {
	cli, err := model.DockerClient()
	if err != nil {
		return nil, rpc.ErrInternal
	}

	if _, err := cli.ImageRemove(context.Background(), in.ImageRef, types.ImageRemoveOptions{}); err != nil {
		log.Warnf("remove image=%v err=%v", in.ImageRef, err)
		return nil, rpc.ErrInternal
	}

	return &pb.RemoveBackupReply{}, nil
}

func (s *ContainerServer) ResumeBackup(ctx context.Context, in *pb.ResumeBackupRequest) (*pb.ResumeBackupReply, error) {
	cli, err := model.DockerClient()
	if err != nil {
		return nil, rpc.ErrInternal
	}

	// inspect container
	configs, err := s.inspect(in.ContainerId, true)
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
	configs.Image = in.ImageRef
	configs.SecurityConfig = in.SecurityConfig
	id, err := s.create(configs)
	if err != nil {
		log.Warnf("create container err=%v", err)
		return nil, rpc.ErrInternal
	}

	return &pb.ResumeBackupReply{ContainerId: id}, nil
}

func (s *ContainerServer) AddBackupJob(ctx context.Context, in *pb.AddBackupJobRequest) (*pb.AddBackupJobReply, error) {
	if in.Id <= 0 || in.ContainerId == "" || in.BackupName == "" {
		return nil, rpc.ErrInvalidArgument
	}

	if err := model.AddContainerBackupJob(in.Id, in.ContainerId, in.BackupName); err != nil {
		log.Infof("model.AddContainerBackupJob err=%v", err)
		return nil, rpc.ErrInternal
	}

	return &pb.AddBackupJobReply{}, nil
}

func (s *ContainerServer) GetBackupJob(ctx context.Context, in *pb.GetBackupJobRequest) (*pb.GetBackupJobReply, error) {
	if in.Id <= 0 {
		return nil, rpc.ErrInvalidArgument
	}

	job, err := model.GetContainerBackupJob(in.Id)
	if err != nil {
		log.Infof("GetContainerBackupJob err=%v", err)
		return nil, rpc.ErrInternal
	}

	return &pb.GetBackupJobReply{
		Id:          job.ID,
		ContainerId: job.ContainerID,
		BackupName:  job.BackupName,
		ImageRef:    job.ImageRef,
		ImageId:     job.ImageID,
		ImageSize:   job.ImageSize,
		Status:      job.Status,
	}, nil
}

func (s *ContainerServer) DelBackupJob(ctx context.Context, in *pb.DelBackupJobRequest) (*pb.DelBackupJobReply, error) {
	if in.Id <= 0 {
		return nil, rpc.ErrInvalidArgument
	}

	model.DelContainerBackupJob(in.Id)
	return &pb.DelBackupJobReply{}, nil
}
