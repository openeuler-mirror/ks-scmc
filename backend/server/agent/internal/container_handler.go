package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"ksc-mcube/rpc"
	pb "ksc-mcube/rpc/pb/container"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	log "github.com/sirupsen/logrus"
)

type ContainerServer struct {
	pb.UnimplementedContainerServer
}

func (s *ContainerServer) List(ctx context.Context, in *pb.ListRequest) (*pb.ListReply, error) {
	log.Infof("Received: %v", in)
	reply := pb.ListReply{}

	cli, err := dockerCli()
	if err != nil {
		return nil, rpc.ErrInternal
	}

	opts := types.ContainerListOptions{All: in.GetListAll(), Size: true}
	containers, err := cli.ContainerList(context.Background(), opts)
	if err != nil {
		log.Warnf("ContainerList: %v", err)
		return nil, rpc.ErrInternal
	}

	for _, c := range containers {
		log.Debugf("%+v", c)
		info := pb.ContainerInfo{
			Id:         c.ID,
			Image:      c.Image,
			ImageId:    c.ImageID,
			Command:    c.Command,
			State:      c.State,
			SizeRw:     c.SizeRw,
			SizeRootFs: c.SizeRootFs,
		}

		if len(c.Names) > 0 && strings.HasPrefix(c.Names[0], "/") {
			info.Name = c.Names[0][1:]
		}
		reply.Containers = append(reply.Containers, &pb.NodeContainer{Info: &info})
	}

	return &reply, nil
}

func (s *ContainerServer) Create(ctx context.Context, in *pb.CreateRequest) (*pb.CreateReply, error) {
	log.Infof("Received: %v", in)
	reply := pb.CreateReply{}

	cli, err := dockerCli()
	if err != nil {
		return nil, rpc.ErrInternal
	}

	// TODO check args

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
			Privileged:  in.HostConfig.Privileged,
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

	if len(in.NetworkConfig) > 0 {
		networkConfig = &network.NetworkingConfig{
			EndpointsConfig: make(map[string]*network.EndpointSettings, len(in.NetworkConfig)),
		}
		for k, v := range in.NetworkConfig {
			networkConfig.EndpointsConfig[k] = &network.EndpointSettings{
				IPAMConfig: &network.EndpointIPAMConfig{},
			}
			if v.IpamConfig != nil {
				networkConfig.EndpointsConfig[k].IPAMConfig.IPv4Address = v.IpamConfig.Ipv4Address
				networkConfig.EndpointsConfig[k].IPAMConfig.IPv6Address = v.IpamConfig.Ipv6Address
			}
		}
	}

	body, err := cli.ContainerCreate(context.Background(), &config, hostConfig, networkConfig, nil, in.Name)
	if err != nil {
		log.Warnf("ContainerCreate: %v", err)
		return nil, transDockerError(err)
	}

	reply.ContainerId = body.ID
	return &reply, nil
}

func (s *ContainerServer) Start(ctx context.Context, in *pb.StartRequest) (*pb.StartReply, error) {
	log.Infof("Received: %v", in)
	reply := pb.StartReply{}

	cli, err := dockerCli()
	if err != nil {
		return nil, rpc.ErrInternal
	}

	opts := types.ContainerStartOptions{}
	for _, id := range in.ContainerIds {
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
	log.Infof("Received: %v", in)
	reply := pb.StopReply{}

	cli, err := dockerCli()
	if err != nil {
		return nil, rpc.ErrInternal
	}

	for _, id := range in.ContainerIds {
		if err := cli.ContainerStop(context.Background(), id, nil); err != nil { // TODO timeout
			log.Warnf("ContainerStop: id=%v %v", id, err)
			return nil, rpc.ErrInternal
		}
		reply.OkIds = append(reply.OkIds, id)
	}

	return &reply, nil
}

func (s *ContainerServer) Kill(ctx context.Context, in *pb.KillRequest) (*pb.KillReply, error) {
	log.Infof("Received: %v", in)
	reply := pb.KillReply{}

	cli, err := dockerCli()
	if err != nil {
		return nil, rpc.ErrInternal
	}

	for _, id := range in.ContainerIds {
		if err := cli.ContainerKill(context.Background(), id, ""); err != nil { // TODO signal
			log.Warnf("ContainerKill: id=%v %v", id, err)
			return nil, rpc.ErrInternal
		}
		reply.OkIds = append(reply.OkIds, id)
	}

	return &reply, nil
}

func (s *ContainerServer) Restart(ctx context.Context, in *pb.RestartRequest) (*pb.RestartReply, error) {
	log.Infof("Received: %v", in)
	reply := pb.RestartReply{}

	cli, err := dockerCli()
	if err != nil {
		return nil, rpc.ErrInternal
	}

	for _, id := range in.ContainerIds {
		if err := cli.ContainerRestart(context.Background(), id, nil); err != nil { // TODO timeout
			log.Warnf("ContainerRestart: id=%v %v", id, err)
			return nil, rpc.ErrInternal
		}
		reply.OkIds = append(reply.OkIds, id)
	}

	return &reply, nil
}

func (s *ContainerServer) Update(ctx context.Context, in *pb.UpdateRequest) (*pb.UpdateReply, error) {
	log.Infof("Received: %v", in)
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
	log.Infof("Received: %v", in)
	reply := pb.RemoveReply{}

	cli, err := dockerCli()
	if err != nil {
		return nil, rpc.ErrInternal
	}

	// TODO follow user input
	opts := types.ContainerRemoveOptions{
		RemoveVolumes: true,
		RemoveLinks:   false,
		Force:         false,
	}

	for _, id := range in.ContainerIds {
		if err := cli.ContainerRemove(context.Background(), id, opts); err != nil {
			log.Warnf("ContainerRemove: id=%v %v", id, err)
			return nil, rpc.ErrInternal
		}
		reply.OkIds = append(reply.OkIds, id)
	}

	return &reply, nil
}

func (s *ContainerServer) Inspect(ctx context.Context, in *pb.InspectRequest) (*pb.InspectReply, error) {
	log.Infof("Received: %v", in)
	reply := pb.InspectReply{}

	cli, err := dockerCli()
	if err != nil {
		return nil, rpc.ErrInternal
	}

	info, err := cli.ContainerInspect(context.Background(), in.ContainerId)
	if err != nil {
		log.Warnf("ContainerInspect: %v", err)
		return nil, transDockerError(err)
	}

	reply.Info = &pb.ContainerInfo{
		Id:    info.ID,
		Name:  info.Name,
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
	if info.NetworkSettings.Networks != nil {
		reply.NetworkSettings = make(map[string]*pb.EndpointSetting)
		for k, v := range info.NetworkSettings.Networks {
			reply.NetworkSettings[k] = &pb.EndpointSetting{
				NetworkId:           v.NetworkID,
				EndpointId:          v.EndpointID,
				Gateway:             v.Gateway,
				IpAddress:           v.IPAddress,
				IpPrefixLen:         int32(v.IPPrefixLen),
				Ipv6Gateway:         v.IPv6Gateway,
				GlobalIpv6Address:   v.GlobalIPv6Address,
				GlobalIpv6PrefixLen: int32(v.GlobalIPv6PrefixLen),
				MacAddress:          v.MacAddress,
			}
		}
	}

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

func (s *ContainerServer) Status(ctx context.Context, in *pb.StatusRequest) (*pb.StatusReply, error) {
	log.Infof("Received: %v", in)
	reply := pb.StatusReply{Status: &pb.ContainerStatus{Id: in.ContainerId}}

	if in.ContainerId == "" {
		return nil, rpc.ErrInvalidArgument
	}

	cli, err := dockerCli()
	if err != nil {
		return nil, rpc.ErrInternal
	}

	info, err := cli.ContainerInspect(context.Background(), in.ContainerId)
	if err != nil {
		log.Warnf("ContainerInspect: %v", err)
		return nil, transDockerError(err)
	}

	if info.State == nil {
		reply.Status.Status = "unknown"
		return &reply, nil
	}

	reply.Status.Status = info.State.Status
	if !info.State.Running {
		return &reply, nil
	}

	stats, err := cli.ContainerStatsOneShot(context.Background(), in.ContainerId)
	if err != nil {
		log.Printf("ContainerStatsOneShot: %v", err)
		return nil, transDockerError(err)
	}

	defer stats.Body.Close()

	var v types.StatsJSON
	if err := json.NewDecoder(stats.Body).Decode(&v); err != nil {
		log.Printf("decode stats json: %v", err)
		return nil, rpc.ErrInternal
	}

	reply.Status = calculateContainerStat(&v)
	reply.Status.Id = in.ContainerId
	return &reply, nil
}
