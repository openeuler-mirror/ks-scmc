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
	"github.com/docker/docker/client"
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

func ensureImage(cli *client.Client, image string) error {
	list, err := cli.ImageList(context.Background(), types.ImageListOptions{})
	if err != nil {
		log.Warnf("ImageList: %v", err)
		return nil
	}

	for _, i := range list {
		if i.ID == image {
			return nil
		}
		for _, s := range i.RepoTags {
			if s == image {
				return nil
			}
		}
	}

	imageExists, err := model.IsImageExist(image)
	if err != nil {
		return err
	} else if !imageExists {
		return rpc.ErrInvalidArgument
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
			Id:      c.ID,
			Image:   c.Image,
			ImageId: c.ImageID,
			Command: c.Command,
			State:   c.State,
			Created: c.Created,
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

		diskStat := &pb.DiskStat{
			Used: float64(c.SizeRootFs) / megaBytes,
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

func (s *ContainerServer) Create(ctx context.Context, in *pb.CreateRequest) (*pb.CreateReply, error) {
	if in.Configs == nil || in.Configs.Image == "" || !containerNamePattern.MatchString(in.Configs.Name) {
		return nil, rpc.ErrInvalidArgument
	}

	cli, err := dockerCli()
	if err != nil {
		return nil, rpc.ErrInternal
	}

	if err := ensureImage(cli, in.Configs.Image); err != nil {
		if _, ok := status.FromError(err); ok {
			return nil, err
		}
		return nil, rpc.ErrInternal
	}

	config := container.Config{
		Image: in.Configs.Image, // should check
		Labels: map[string]string{
			"KS_SCMC_DESC": in.Configs.Desc,
			"KS_SCMC_UUID": in.Configs.Uuid,
		},
	}
	hostConfig := container.HostConfig{
		Privileged: false, // force non-privileged
	}
	var networkConfig *network.NetworkingConfig

	for k, v := range in.Configs.Envs {
		config.Env = append(config.Env, fmt.Sprintf("%s=%s", k, v))
	}
	config.Env = append(config.Env, fmt.Sprintf("KS_SCMC_UUID=%s", in.Configs.Uuid))

	if in.Configs.EnableGraphic {
		if err := containerGraphicSetup(in.Configs.Name, &config, &hostConfig); err != nil {
			log.Infof("containerGraphicSetup err=%v", err)
			return nil, rpc.ErrInternal
		}
		config.Labels["KS_SCMC_GRAPHIC"] = "1"
	}

	for _, m := range in.Configs.Mounts {
		hostConfig.Mounts = append(hostConfig.Mounts, mount.Mount{
			Type:     mount.Type(m.Type),
			Source:   m.Source,
			Target:   m.Target,
			ReadOnly: m.ReadOnly,
			// Consistency: mount.Consistency(m.Consistency),
		})
	}

	if in.Configs.RestartPolicy != nil {
		hostConfig.RestartPolicy.Name = in.Configs.RestartPolicy.Name
		hostConfig.RestartPolicy.MaximumRetryCount = int(in.Configs.RestartPolicy.MaxRetry)
	}

	if in.Configs.ResouceLimit != nil {
		hostConfig.Resources = container.Resources{
			NanoCPUs:          int64(in.Configs.ResouceLimit.CpuLimit * 10e9),
			CPUShares:         toCPUShares(in.Configs.ResouceLimit.CpuPrio),
			Memory:            int64(in.Configs.ResouceLimit.MemoryLimit * megaBytes),
			MemoryReservation: int64(in.Configs.ResouceLimit.MemorySoftLimit * megaBytes),
		}

		if in.Configs.ResouceLimit.DiskLimit > 0.0 {
			config.Labels["KS_SCMC_DISK_LIMIT"] = fmt.Sprintf("%f", in.Configs.ResouceLimit.DiskLimit)
			hostConfig.StorageOpt = map[string]string{
				"size": fmt.Sprintf("%fM", in.Configs.ResouceLimit.DiskLimit),
			}
		}
	}

	var networkConfigCreate *network.NetworkingConfig
	if len(in.Configs.Networks) > 0 {
		networkConfig = &network.NetworkingConfig{
			EndpointsConfig: make(map[string]*network.EndpointSettings, len(in.Configs.Networks)),
		}
		for _, v := range in.Configs.Networks {
			if v.IpAddress != "" {
				if !checkConfilt(v.Interface, v.IpAddress, int(v.IpPrefixLen)) {
					return nil, rpc.ErrInvalidArgument
				}
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

		ifs := in.Configs.Networks[0].Interface
		networkConfigCreate = &network.NetworkingConfig{
			EndpointsConfig: make(map[string]*network.EndpointSettings, 1),
		}
		networkConfigCreate.EndpointsConfig[ifs] = networkConfig.EndpointsConfig[ifs]
	}

	body, err := cli.ContainerCreate(context.Background(), &config, &hostConfig, networkConfigCreate, nil, in.Configs.Name)
	if err != nil {
		log.Warnf("ContainerCreate: %v", err)
		return nil, transDockerError(err)
	}

	if len(in.Configs.Networks) > 1 {
		for i := 1; i < len(in.Configs.Networks); i++ {
			ifs := in.Configs.Networks[i].Interface
			network := networkConfig.EndpointsConfig[ifs]
			err = cli.NetworkConnect(context.Background(), ifs, body.ID, network)
			if err != nil {
				log.Warnf("NetworkConnect: %v", err)
			}
		}
	}

	reply := pb.CreateReply{
		ContainerId: body.ID,
	}
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

	if in.ContainerId == "" || (in.ResourceLimit == nil && in.RestartPolicy == nil && in.Networks == nil && in.SecurityConfig == nil) {
		return nil, rpc.ErrInvalidArgument
	}

	cli, err := dockerCli()
	if err != nil {
		return nil, rpc.ErrInternal
	}

	config := container.UpdateConfig{}
	if in.ResourceLimit != nil {
		config.Resources = container.Resources{
			NanoCPUs:          int64(in.ResourceLimit.CpuLimit * 10e9),
			CPUShares:         toCPUShares(in.ResourceLimit.CpuPrio),
			Memory:            int64(in.ResourceLimit.MemoryLimit * megaBytes),
			MemoryReservation: int64(in.ResourceLimit.MemorySoftLimit * megaBytes),
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

	if in.Networks != nil {
		info, err := cli.ContainerInspect(context.Background(), in.ContainerId)
		if err != nil {
			log.Warnf("ContainerInspectWithRaw: %v", err)
			return nil, transDockerError(err)
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
	}

	return &reply, nil
}

func (s *ContainerServer) Inspect(ctx context.Context, in *pb.InspectRequest) (*pb.InspectReply, error) {

	cli, err := dockerCli()
	if err != nil {
		return nil, rpc.ErrInternal
	}

	info, _, err := cli.ContainerInspectWithRaw(context.Background(), in.ContainerId, true)
	if err != nil {
		log.Warnf("ContainerInspectWithRaw: %v", err)
		return nil, transDockerError(err)
	}

	image := info.Image
	if info.Config != nil {
		image = info.Config.Image
	}

	reply := pb.InspectReply{
		Configs: &pb.ContainerConfigs{
			ContainerId:  info.ID,
			Name:         strings.TrimPrefix(info.Name, "/"),
			Image:        image,
			ResouceLimit: &pb.ResourceLimit{},
		},
	}

	if info.State != nil {
		reply.Configs.Status = info.State.Status
	}

	if info.Config != nil {
		if v, ok := info.Config.Labels["KS_SCMC_DESC"]; ok {
			reply.Configs.Desc = v
		}

		if v, ok := info.Config.Labels["KS_SCMC_GRAPHIC"]; ok {
			if v == "1" {
				reply.Configs.EnableGraphic = true
			}
		}

		if len(info.Config.Env) > 0 {
			reply.Configs.Envs = make(map[string]string, len(info.Config.Env))
			for _, e := range info.Config.Env {
				parts := strings.SplitN(e, "=", 2)
				if len(parts) >= 2 {
					reply.Configs.Envs[parts[0]] = parts[1]
				}
			}
		}

		for _, m := range info.Mounts {
			reply.Configs.Mounts = append(reply.Configs.Mounts, &pb.Mount{
				Type:     string(m.Type),
				Source:   m.Source,
				Target:   m.Destination,
				ReadOnly: !m.RW,
			})
		}
	}

	if info.HostConfig != nil {
		reply.Configs.RestartPolicy = &pb.RestartPolicy{
			Name:     info.HostConfig.RestartPolicy.Name,
			MaxRetry: int32(info.HostConfig.RestartPolicy.MaximumRetryCount),
		}

		reply.Configs.ResouceLimit = &pb.ResourceLimit{
			CpuLimit:        float64(info.HostConfig.Resources.NanoCPUs) / 10e9,
			CpuPrio:         fromCPUShares(info.HostConfig.Resources.CPUShares),
			MemoryLimit:     float64(info.HostConfig.Resources.Memory) / megaBytes,
			MemorySoftLimit: float64(info.HostConfig.Resources.MemoryReservation) / megaBytes,
		}

		if s, ok := info.HostConfig.StorageOpt["size"]; ok {
			fmt.Sscanf(s, "%fM", &reply.Configs.ResouceLimit.DiskLimit)
		}
	}

	if info.NetworkSettings != nil {
		for k, m := range info.NetworkSettings.Networks {
			if k == "bridge" {
				continue // "bridge" 是默认创建的网络连接, 需要忽略
			}
			reply.Configs.Networks = append(reply.Configs.Networks, &pb.NetworkConfig{
				Interface:   k,
				ContainerId: info.ID,
				IpAddress:   m.IPAddress,
				IpPrefixLen: int32(m.IPPrefixLen),
				MacAddress:  m.MacAddress,
				Gateway:     m.Gateway,
			})
		}
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
