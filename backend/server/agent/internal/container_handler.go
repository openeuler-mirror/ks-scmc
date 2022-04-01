package internal

import (
	"context"
	"fmt"
	"runtime"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	"github.com/shirou/gopsutil/mem"
	log "github.com/sirupsen/logrus"

	"scmc/model"
	"scmc/rpc"
	pb "scmc/rpc/pb/container"
)

type ContainerServer struct {
	pb.UnimplementedContainerServer
}

func (s *ContainerServer) List(ctx context.Context, in *pb.ListRequest) (*pb.ListReply, error) {
	reply := pb.ListReply{}

	cli, err := dockerCli()
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
			Id:         c.ID,
			Image:      c.Image,
			ImageId:    c.ImageID,
			Command:    c.Command,
			State:      c.State,
			SizeRw:     c.SizeRw,
			SizeRootFs: c.SizeRootFs,
			Labels:     c.Labels,
			Created:    c.Created,
		}

		// 参考docker cli实现 去掉link特性连接的其他容器名
		for _, name := range c.Names {
			if strings.HasPrefix(name, "/") && len(strings.Split(name[1:], "/")) == 1 {
				info.Name = name[1:]
				break
			}
		}

		if stats != nil {
			if stat, ok := stats[c.ID]; ok {
				info.ResourceStat = stat
			}
		}

		reply.Containers = append(reply.Containers, &pb.NodeContainer{Info: &info})
	}

	return &reply, nil
}

func (s *ContainerServer) Create(ctx context.Context, in *pb.CreateRequest) (*pb.CreateReply, error) {
	reply := pb.CreateReply{}

	// TODO check args
	if in.Config.Image == "" || !containerNamePattern.MatchString(in.Name) {
		return nil, rpc.ErrInvalidArgument
	}

	cli, err := dockerCli()
	if err != nil {
		return nil, rpc.ErrInternal
	}

	list, err := cli.ImageList(context.Background(), types.ImageListOptions{})
	if err != nil {
		log.Warnf("ImageList: %v", err)
		return nil, transDockerError(err)
	}

	imageExist := false
	for _, image := range list {
		for _, s := range image.RepoTags {
			if s == in.Config.Image {
				imageExist = true
				break
			}
		}
	}

	if !imageExist {
		if err = model.Pull(in.Config.Image); err != nil {
			log.Errorf("pull image[%v] err: %v", in.Config.Image, err)
			return nil, rpc.ErrInternal
		}
	}

	var (
		envs          []string
		mounts        []mount.Mount
		config        container.Config
		hostConfig    *container.HostConfig
		networkConfig *network.NetworkingConfig
	)

	for k, v := range in.Config.Env {
		envs = append(envs, fmt.Sprintf("%s=%s", k, v))
	}

	config = container.Config{
		Hostname:   in.Config.Hostname,
		Domainname: in.Config.DomainName,
		User:       in.Config.User,
		Env:        envs,
		Image:      in.Config.Image, // should check
		Entrypoint: in.Config.Entrypoint,
		Cmd:        in.Config.Cmd,
		Labels:     in.Config.Labels,
	}

	if in.HostConfig != nil {
		for _, m := range in.HostConfig.Mounts {
			mounts = append(mounts, mount.Mount{
				Type:     mount.Type(m.Type),
				Source:   m.Source,
				Target:   m.Target,
				ReadOnly: m.ReadOnly,
				// Consistency: mount.Consistency(m.Consistency),
			})
		}

		hostConfig = &container.HostConfig{
			NetworkMode: container.NetworkMode(in.HostConfig.NetworkMode),
			AutoRemove:  in.HostConfig.AutoRemove,
			IpcMode:     container.IpcMode(in.HostConfig.IpcMode),
			Mounts:      mounts,
			Privileged:  false, // force no privilege
			StorageOpt:  in.HostConfig.StorageOpt,
		}

		if in.HostConfig.RestartPolicy != nil {
			hostConfig.RestartPolicy = container.RestartPolicy{
				Name:              in.HostConfig.RestartPolicy.Name,
				MaximumRetryCount: int(in.HostConfig.RestartPolicy.MaxRetry),
			}
		}

		if in.HostConfig.ResourceConfig != nil {
			hostConfig.Resources = container.Resources{
				NanoCPUs:          in.HostConfig.ResourceConfig.NanoCpus,
				CPUShares:         in.HostConfig.ResourceConfig.CpuShares,
				Memory:            in.HostConfig.ResourceConfig.MemLimit,
				MemoryReservation: in.HostConfig.ResourceConfig.MemSoftLimit,
			}

			for _, d := range in.HostConfig.ResourceConfig.Devices {
				hostConfig.Resources.Devices = append(hostConfig.Resources.Devices, container.DeviceMapping{
					PathOnHost:        d.PathOnHost,
					PathInContainer:   d.PathInContainer,
					CgroupPermissions: d.CgroupPermissions,
				})
			}
		}
	}

	// do not use docker default bridge network
	// networkConfig := &network.NetworkingConfig{
	// 	EndpointsConfig: map[string]*network.EndpointSettings{"none": {}},
	// }

	containerNet := make(map[string]*model.ContainerNetwork)
	virtualInfo := make(map[string]*model.VirNicInfo)
	if len(in.NetworkConfig) > 0 {
		networkConfig = &network.NetworkingConfig{
			EndpointsConfig: make(map[string]*network.EndpointSettings, len(in.NetworkConfig)),
		}
		for _, v := range in.NetworkConfig {
			virinfo := getVirNicInfo(v.Interface)
			containerIP := v.IpAddress
			masklen := virinfo.IPMaskLen
			nextIP := virinfo.MinAvailableIP
			if v.IpMask != "" {
				masklen = model.TransMask(v.IpMask)
			}

			if v.IpAddress == "" {
				containerIP, nextIP = assignIP(v.Interface, virinfo)
			} else {
				if conflic := model.IsConflict(v.Interface, v.IpAddress, masklen); conflic {
					return nil, rpc.ErrInvalidArgument
				}
			}

			networkConfig.EndpointsConfig[v.Interface] = &network.EndpointSettings{
				IPAMConfig: &network.EndpointIPAMConfig{},
			}

			networkConfig.EndpointsConfig[v.Interface].IPAMConfig.IPv4Address = containerIP
			networkConfig.EndpointsConfig[v.Interface].IPAddress = containerIP
			networkConfig.EndpointsConfig[v.Interface].IPPrefixLen = masklen
			networkConfig.EndpointsConfig[v.Interface].MacAddress = v.MacAddress
			networkConfig.EndpointsConfig[v.Interface].Gateway = v.Gateway
			containerNet[v.Interface] = &model.ContainerNetwork{
				IpAddress: containerIP,
				MaskLen:   masklen,
			}
			virtualInfo[v.Interface] = &model.VirNicInfo{
				IPAddress:      virinfo.IPAddress,
				IPMaskLen:      virinfo.IPMaskLen,
				MinAvailableIP: nextIP,
			}
		}
	}

	if in.EnableGraphic {
		if hostConfig == nil {
			hostConfig = &container.HostConfig{}
		}
		containerGraphicSetup(in.Name, &config, hostConfig)
	}

	body, err := cli.ContainerCreate(context.Background(), &config, hostConfig, networkConfig, nil, in.Name)
	if err != nil {
		log.Warnf("ContainerCreate: %v", err)
		return nil, transDockerError(err)
	}

	for k, v := range containerNet {
		v.ForShell = fmt.Sprintf("%s/%s/%s/%d", body.ID, k, v.IpAddress, v.MaskLen)
	}
	cntrs := make(map[string]*model.ContainerNic)
	cntrs[body.ID] = &model.ContainerNic{
		ContainerNetworks: containerNet,
	}
	netinfo := &model.NetworkInfo{
		Containers: cntrs,
		VirtualNic: virtualInfo,
	}
	model.AddJSON(body.ID, netinfo)

	reply.ContainerId = body.ID
	return &reply, nil
}

func (s *ContainerServer) Start(ctx context.Context, in *pb.StartRequest) (*pb.StartReply, error) {
	reply := pb.StartReply{}

	if (len(in.Ids)) <= 0 || len(in.Ids[0].ContainerIds) <= 0 {
		return nil, rpc.ErrInvalidArgument
	}

	cli, err := dockerCli()
	if err != nil {
		return nil, rpc.ErrInternal
	}

	opts := types.ContainerStartOptions{}
	for _, id := range in.Ids[0].ContainerIds {
		if err := cli.ContainerStart(context.Background(), id, opts); err != nil {
			log.Warnf("ContainerStart: id=%v %v", id, err)
			return nil, rpc.ErrInternal
		}

		reply.OkIds = append(reply.OkIds, id)
	}

	reply.OkIds = nil
	return &reply, nil
}

func (s *ContainerServer) Stop(ctx context.Context, in *pb.StopRequest) (*pb.StopReply, error) {
	reply := pb.StopReply{}
	if (len(in.Ids)) <= 0 || len(in.Ids[0].ContainerIds) <= 0 {
		return nil, rpc.ErrInvalidArgument
	}

	cli, err := dockerCli()
	if err != nil {
		return nil, rpc.ErrInternal
	}

	for _, id := range in.Ids[0].ContainerIds {
		if err := cli.ContainerStop(context.Background(), id, nil); err != nil { // TODO timeout
			log.Warnf("ContainerStop: id=%v %v", id, err)
			return nil, rpc.ErrInternal
		}
		reply.OkIds = append(reply.OkIds, id)
	}

	return &reply, nil
}

func (s *ContainerServer) Kill(ctx context.Context, in *pb.KillRequest) (*pb.KillReply, error) {
	reply := pb.KillReply{}
	if (len(in.Ids)) <= 0 || len(in.Ids[0].ContainerIds) <= 0 {
		return nil, rpc.ErrInvalidArgument
	}

	cli, err := dockerCli()
	if err != nil {
		return nil, rpc.ErrInternal
	}

	for _, id := range in.Ids[0].ContainerIds {
		if err := cli.ContainerKill(context.Background(), id, ""); err != nil { // TODO signal
			log.Warnf("ContainerKill: id=%v %v", id, err)
			return nil, rpc.ErrInternal
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

	cli, err := dockerCli()
	if err != nil {
		return nil, rpc.ErrInternal
	}

	for _, id := range in.Ids[0].ContainerIds {
		if err := cli.ContainerRestart(context.Background(), id, nil); err != nil { // TODO timeout
			log.Warnf("ContainerRestart: id=%v %v", id, err)
			return nil, rpc.ErrInternal
		}
		reply.OkIds = append(reply.OkIds, id)
	}

	return &reply, nil
}

func (s *ContainerServer) Update(ctx context.Context, in *pb.UpdateRequest) (*pb.UpdateReply, error) {
	reply := pb.UpdateReply{}

	if in.ContainerId == "" || (in.ResourceConfig == nil && in.RestartPolicy == nil) {
		return nil, rpc.ErrInvalidArgument
	}

	cli, err := dockerCli()
	if err != nil {
		return nil, rpc.ErrInternal
	}

	config := container.UpdateConfig{}
	if in.ResourceConfig != nil {
		config.Resources = container.Resources{
			NanoCPUs:          in.ResourceConfig.NanoCpus,
			Memory:            in.ResourceConfig.MemLimit,
			MemoryReservation: in.ResourceConfig.MemSoftLimit,
		}
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

	return &reply, nil
}

func (s *ContainerServer) Remove(ctx context.Context, in *pb.RemoveRequest) (*pb.RemoveReply, error) {
	reply := pb.RemoveReply{}
	if (len(in.Ids)) <= 0 || len(in.Ids[0].ContainerIds) <= 0 {
		return nil, rpc.ErrInvalidArgument
	}

	cli, err := dockerCli()
	if err != nil {
		return nil, rpc.ErrInternal
	}

	opts := types.ContainerRemoveOptions{
		RemoveVolumes: in.RemoveVolumes,
	}

	for _, id := range in.Ids[0].ContainerIds {
		if err := cli.ContainerRemove(context.Background(), id, opts); err != nil {
			log.Warnf("ContainerRemove: id=%v %v", id, err)
			return nil, rpc.ErrInternal
		}
		reply.OkIds = append(reply.OkIds, id)
		model.DelContainerNetInfo(id)
	}

	return &reply, nil
}

func (s *ContainerServer) Inspect(ctx context.Context, in *pb.InspectRequest) (*pb.InspectReply, error) {
	reply := pb.InspectReply{}

	cli, err := dockerCli()
	if err != nil {
		return nil, rpc.ErrInternal
	}

	info, _, err := cli.ContainerInspectWithRaw(context.Background(), in.ContainerId, true)
	if err != nil {
		log.Warnf("ContainerInspectWithRaw: %v", err)
		return nil, transDockerError(err)
	}

	reply.Info = &pb.ContainerInfo{
		Id:    info.ID,
		Name:  strings.TrimPrefix(info.Name, "/"),
		Image: info.Image,
		State: info.State.Status,
	}

	if info.Config != nil {
		reply.Config = &pb.ContainerConfig{
			Hostname:        info.Config.Hostname,
			DomainName:      info.Config.Domainname,
			User:            info.Config.User,
			Image:           info.Config.Image,
			WorkingDir:      info.Config.WorkingDir,
			Entrypoint:      info.Config.Entrypoint,
			Cmd:             info.Config.Cmd,
			NetworkDisabled: info.Config.NetworkDisabled,
			Labels:          info.Config.Labels,
		}
		if len(info.Config.Env) > 0 {
			reply.Config.Env = make(map[string]string, len(info.Config.Env))
			for _, e := range info.Config.Env {
				parts := strings.SplitN(e, "=", 2)
				if len(parts) >= 2 {
					reply.Config.Env[parts[0]] = parts[1]
				}
			}
		}
	}
	if info.HostConfig != nil {
		reply.HostConfig = &pb.HostConfig{
			RestartPolicy: &pb.RestartPolicy{
				Name:     info.HostConfig.RestartPolicy.Name,
				MaxRetry: int32(info.HostConfig.RestartPolicy.MaximumRetryCount),
			},
			ResourceConfig: &pb.ResourceConfig{
				NanoCpus:     info.HostConfig.Resources.NanoCPUs,
				CpuShares:    info.HostConfig.Resources.CPUShares,
				MemLimit:     info.HostConfig.Resources.Memory,
				MemSoftLimit: info.HostConfig.Resources.MemoryReservation,
			},
		}
		if len(info.HostConfig.StorageOpt) > 0 {
			reply.HostConfig.StorageOpt = make(map[string]string, len(info.HostConfig.StorageOpt))
			for k, v := range info.HostConfig.StorageOpt {
				reply.HostConfig.StorageOpt[k] = v
			}
		}
	}

	// TODO retrive network infomation

	for _, m := range info.Mounts {
		reply.Mounts = append(reply.Mounts, &pb.Mount{
			Type:     string(m.Type),
			Source:   m.Source,
			Target:   m.Destination,
			ReadOnly: !m.RW,
		})
	}

	return &reply, nil
}

func (s *ContainerServer) MonitorHistory(ctx context.Context, in *pb.MonitorHistoryRequest) (*pb.MonitorHistoryReply, error) {
	now := time.Now()
	if in.StartTime >= in.EndTime || in.StartTime > now.Unix() || in.Interval < 1 || in.StartTime < now.Add(-time.Hour*24*10).Unix() {
		log.Info("MonitorHistory invalid time args")
		return nil, rpc.ErrInvalidArgument
	}

	containerName := "/" // query influxdb need container name, "/" for query the host

	numCPU := float64(runtime.NumCPU())
	memInfo, err := mem.VirtualMemory()
	if err != nil {
		log.Infof("VirtualMemory err=%v", err)
		return nil, err
	}

	rscLimit := pb.ResourceLimit{
		CpuLimit:    numCPU,
		MemoryLimit: float64(memInfo.Total) / megaBytes,
	}

	if in.ContainerId != "" {
		cli, err := dockerCli()
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
