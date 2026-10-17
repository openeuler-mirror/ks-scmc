package internal

import (
	"context"
	"fmt"
	"io"
	"regexp"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
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
	"scmc/model/registry"
	"scmc/rpc"
	pb "scmc/rpc/pb/container"
	"scmc/server/args"
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
		r.MemorySwap = -1 // prevent contaienr start error
	}

	return r
}

func ensureImage(imageRef string) error {
	exist, err := docker.IsImageExist(imageRef)
	if err != nil {
		log.Infof("docker.IsImageExist err=%v", err)
		return rpc.ErrInternal
	}
	if exist {
		return nil
	}

	exist, err = registry.IsImageExist(imageRef)
	if err != nil {
		log.Infof("registry.IsImageExist err=%v", err)
		return err
	}
	if !exist {
		log.Infof("image[%v] not exist in registry", imageRef)
		return rpc.ErrInternal
	}

	if err = docker.PullImage(imageRef); err != nil {
		log.Infof("pull image[%v] err: %v", imageRef, err)
		return err
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
		if err := kysec.CleanFileAccess(id); err != nil {
			log.Warnf("%v CleanFileAccess err:%v", id, err)
			return rpc.ErrContainerFileProtection
		}
		if err := kysec.CleanWhiteList(id); err != nil {
			log.Warnf("%v CleanWhiteList err:%v", id, err)
			return rpc.ErrContainerProcProtection
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

	if sec.DisableCmdOperation {
		if err := authz.AddSensitiveContainers(name, id); err != nil {
			log.Warnf("AddSensitiveContainers err=%v", err)
			return rpc.ErrContainerCmdOperation
		}
	} else {
		if fromUpdate {
			if err := authz.DelSensitiveContainers(name, id); err != nil {
				log.Warnf("DelSensitiveContainers err=%v", err)
				return rpc.ErrContainerCmdOperation
			}
		}

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

	// configs.EnableGraphic 默认开启，后面应用里有图形化选项
	if err := containerGraphicSetup(configs.Name, &config, &hostConfig); err != nil {
		log.Infof("containerGraphicSetup err=%v", err)
		return "", rpc.ErrInternal
	}
	config.Labels["KS_SCMC_GRAPHIC"] = "1"

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
			if err := checkNetworkInfo(v.Interface, "", &v.IpAddress, int(v.IpPrefixLen)); err != nil {
				return "", err
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

	if in.ContainerId == "" || (in.ResourceLimit == nil && in.RestartPolicy == nil && in.Networks == nil && in.SecurityConfig == nil) {
		return nil, rpc.ErrInvalidArgument
	}

	cfgs, err := litedb.GetContainerConfig(in.ContainerId)
	if err != nil {
		log.Infof("GetContainerConfig err=%v", err)
		return nil, rpc.ErrDatabaseFail
	}

	if in.Networks != nil {
		for _, v := range in.Networks {
			if err := checkNetworkInfo(v.Interface, in.ContainerId, &v.IpAddress, int(v.IpPrefixLen)); err != nil {
				return nil, err
			}
		}
	}

	inspectConfigs, err := s.inspect(in.ContainerId, false)
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
			for k := range info.NetworkSettings.Networks {
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

	cfgs.SecurityConfig = in.SecurityConfig
	if err := litedb.UpdateContainerConfig(cfgs); err != nil {
		log.Infof("UpdateContainerConfig err=%v", err)
		return nil, rpc.ErrDatabaseFail
	}

	return &reply, nil
}

func (s *ContainerServer) remove(cli *client.Client, containerID string, force bool) error {
	if cli == nil {
		c, err := docker.Client()
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

	// TODO remove container configs
	// TODO remove container backups
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

	var images []string
	for _, b := range backups {
		images = append(images, b.ImageRef)
	}

	docker.RemoveImage(false, images...)
	litedb.RemoveContainerBackup(backups)

	return nil
}

func (s *ContainerServer) inspect(id string, backup bool) (*pb.ContainerConfigs, error) {
	cli, err := docker.Client()
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
		// if image tag exists, use the tag, if not use image id
		if _, _, err := cli.ImageInspectWithRaw(context.Background(), info.Config.Image); err == nil {
			configs.Image = info.Config.Image
		}

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
		for iface, m := range info.NetworkSettings.Networks {
			if isPredefinedNetwork(iface) {
				continue // 需要忽略默认创建的网络连接
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
				Interface:   iface,
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
	data, err := litedb.GetContainerConfig(in.ContainerId)
	if err != nil {
		log.Infof("db get container configs id=%v err=%v", in.ContainerId, err)
		return nil, rpc.ErrDatabaseFail
	}

	configs, err := s.inspect(in.ContainerId, false)
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

	ci, err := cli.ContainerInspect(context.Background(), cfgs.ContainerID)
	if err != nil {
		log.Infof("ContainerInspect err=%v", err)
		return nil, rpc.ErrInternal
	}

	if ci.Image == b.ImageRef {
		return nil, rpc.ErrRemoveContainerBackupWhenRunning
	}

	if err := litedb.RemoveContainerBackup(b); err != nil {
		log.Warnf("db remove backup record err=%v", err)
		return nil, rpc.ErrInternal
	}

	docker.RemoveImage(false, b.ImageRef)

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

	if !containerIsRunningWithCli(ctx, cli, app.ContainerID) {
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

	if !containerIsRunningWithCli(ctx, cli, app.ContainerID) {
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
