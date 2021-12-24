package server

import (
	"context"
	"fmt"
	"log"

	"ksc-mcube/rpc/errno"
	pb "ksc-mcube/rpc/pb/container"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
)

type ContainerServer struct {
	pb.UnimplementedContainerServer
}

func (s *ContainerServer) List(ctx context.Context, in *pb.ListRequest) (*pb.ListReply, error) {
	log.Printf("Received: %v", in)
	reply := pb.ListReply{Header: errno.InternalError}

	cli, err := dockerCli()
	if err != nil {
		return &reply, nil
	}

	containers, err := cli.ContainerList(context.Background(), types.ContainerListOptions{All: in.GetListAll()})
	if err != nil {
		log.Printf("ContainerList: %v", err)
	}

	for _, c := range containers {
		fmt.Printf("%s %s\n", c.ID[:10], c.Image)
		reply.Containers = append(reply.Containers, &pb.ListReply_NodeContainer{
			Info: &pb.ContainerInfo{
				Id:    c.ID,
				Names: c.Names,
				Image: c.Image,
			},
		})
	}

	reply.Header = errno.OK
	return &reply, nil
}

func (s *ContainerServer) Create(ctx context.Context, in *pb.CreateRequest) (*pb.CreateReply, error) {
	log.Printf("Received: %v", in)
	reply := pb.CreateReply{Header: errno.InternalError}

	cli, err := dockerCli()
	if err != nil {
		return &reply, nil
	}

	var envs []string
	for k, v := range in.Config.Env {
		envs = append(envs, fmt.Sprintf("%s=%s", k, v))
	}
	config := container.Config{
		Hostname:   in.Config.Hostname,
		Domainname: in.Config.DomainName,
		User:       in.Config.User,
		Env:        envs,
		Cmd:        in.Config.Cmd,
		Image:      in.Config.Image, // should check
		Entrypoint: in.Config.Entrypoint,
	}

	var mounts []mount.Mount
	for _, m := range in.HostConfig.Mounts {
		mounts = append(mounts, mount.Mount{
			Type:        mount.Type(m.Type),
			Source:      m.Source,
			Target:      m.Target,
			ReadOnly:    m.ReadOnly,
			Consistency: mount.Consistency(m.Consistency),
		})
	}
	hostConfig := container.HostConfig{
		NetworkMode:   container.NetworkMode(in.HostConfig.NetworkMode),
		RestartPolicy: container.RestartPolicy{in.HostConfig.RestartPolicy.Name, int(in.HostConfig.RestartPolicy.MaxRetry)},
		AutoRemove:    in.HostConfig.AutoRemove,
		// IpcMode: ,
		Mounts: mounts,
	}

	// networkConfig := network.NetworkingConfig{}
	_, err = cli.ContainerCreate(context.Background(), &config, &hostConfig, nil, nil, in.Name)
	if err != nil {
		log.Printf("ContainerCreate: %v", err)
		return &reply, nil
	}

	reply.Header = errno.OK
	return &reply, nil
}

func (s *ContainerServer) Start(ctx context.Context, in *pb.StartRequest) (*pb.StartReply, error) {
	log.Printf("Received: %v", in)

	reply := pb.StartReply{Header: errno.InternalError}

	cli, err := dockerCli()
	if err != nil {
		return &reply, nil
	}

	opts := types.ContainerStartOptions{}
	if err := cli.ContainerStart(context.Background(), in.ContainerId, opts); err != nil {
		log.Printf("ContainerList: %v", err)
		return &reply, nil
	}

	reply.Header = errno.OK
	return &reply, nil
}

func (s *ContainerServer) Stop(ctx context.Context, in *pb.StopRequest) (*pb.StopReply, error) {
	log.Printf("Received: %v", in)

	reply := pb.StopReply{Header: errno.InternalError}

	cli, err := dockerCli()
	if err != nil {
		return &reply, nil
	}

	if err := cli.ContainerStop(context.Background(), in.ContainerId, nil); err != nil { // TODO timeout
		log.Printf("ContainerList: %v", err)
		return &reply, nil
	}

	reply.Header = errno.OK
	return &reply, nil
}

func (s *ContainerServer) Kill(ctx context.Context, in *pb.KillRequest) (*pb.KillReply, error) {
	log.Printf("Received: %v", in)

	return &pb.KillReply{Header: errno.OK}, nil
}

func (s *ContainerServer) Restart(ctx context.Context, in *pb.RestartRequest) (*pb.RestartReply, error) {
	log.Printf("Received: %v", in)

	return &pb.RestartReply{Header: errno.OK}, nil
}

func (s *ContainerServer) Update(ctx context.Context, in *pb.UpdateRequest) (*pb.UpdateReply, error) {
	log.Printf("Received: %v", in)

	return &pb.UpdateReply{Header: errno.OK}, nil
}

func (s *ContainerServer) Remove(ctx context.Context, in *pb.RemoveRequest) (*pb.RemoveReply, error) {
	log.Printf("Received: %v", in)

	return &pb.RemoveReply{Header: errno.OK}, nil
}
