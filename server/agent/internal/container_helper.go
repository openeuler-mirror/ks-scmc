package internal

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/client"
	log "github.com/sirupsen/logrus"

	"scmc/common"
	"scmc/model/docker"
	pb "scmc/rpc/pb/container"
)

const (
	containerSocketPath  = "/tmp/.X11-unix"
	containerAuthPath    = "/tmp/.xauth"
	authFile             = "Xauthority"
	containerNamePattern = `^[a-zA-Z0-9][a-zA-Z0-9_.-]+$`
)

func containerGraphicSetup(containerName string, config *container.Config, hostConfig *container.HostConfig) error {
	hostPath := filepath.Join(common.Config.Agent.ContainerExtraDataBasedir, containerName)
	socketDir := filepath.Join(hostPath, "socket")
	if err := os.MkdirAll(socketDir, 0755); err != nil {
		log.Warnf("mkdir %v: %v", socketDir, err)
		return err
	}

	authDir := filepath.Join(hostPath, "xauth")
	if err := os.MkdirAll(authDir, 0755); err != nil {
		log.Warnf("mkdir %v: %v", authDir, err)
		return err
	}

	// set environment for display, shared X11 socket/auth file
	config.Env = append(config.Env, "DISPLAY=:0", "XAUTHORITY="+filepath.Join(containerAuthPath, authFile))

	var mnts []mount.Mount
	for _, m := range hostConfig.Mounts {
		if m.Target == containerSocketPath || m.Target == containerAuthPath {
			continue // remove duplicate mnt point
		}
		mnts = append(mnts, m)
	}

	hostConfig.Mounts = append(mnts,
		mount.Mount{
			Type:   mount.TypeBind,
			Source: socketDir,
			Target: containerSocketPath,
		},
		mount.Mount{
			Type:   mount.TypeBind,
			Source: authDir,
			Target: containerAuthPath,
		},
	)

	return nil
}

func removeContainerGraphicSetup(containerName string) error {
	hostPath := filepath.Join(common.Config.Agent.ContainerExtraDataBasedir, containerName)
	return os.RemoveAll(hostPath)
}

func appIsRunning(ctx context.Context, idResponse string) bool {
	cli, err := docker.Client()
	if err != nil {
		log.Infof("docker.Client err:%s", err)
		return false
	}

	return appIsRunningWithCli(ctx, cli, idResponse)
}

func appIsRunningWithCli(ctx context.Context, cli *client.Client, idResponse string) bool {
	if idResponse == "" {
		return false
	}

	execins, err := cli.ContainerExecInspect(ctx, idResponse)
	if err != nil {
		log.Infof("cli.ContainerExecInspect err:%s", err)
		return false
	}

	return execins.Running
}

func containerIsRunning(ctx context.Context, cli *client.Client, containerID string) bool {
	ins, err := cli.ContainerInspect(ctx, containerID)
	if err != nil {
		log.Infof("cli.ContainerInspect err:%s", err)
		return false
	}

	if ins.State == nil {
		return false
	}

	return ins.State.Running
}

func containerStartedTime(ctx context.Context, cli *client.Client, c *types.Container) int64 {
	if c.State == "created" {
		return 0
	}

	info, err := cli.ContainerInspect(ctx, c.ID)
	if err != nil {
		log.Warnf("ContainerInspect id=%v err=%v", c.ID, err)
		return 0 // dont return error
	}

	if info.State != nil {
		startedAt, err := time.ParseInLocation(time.RFC3339Nano, info.State.StartedAt, time.UTC)
		if err != nil {
			log.Warnf("ParseInLocation time=%v err=%v", info.State.StartedAt, err)
			return 0
		}
		return startedAt.Unix()
	}
	return 0
}

func containerName(names []string) string {
	// 参考docker cli实现 去掉link特性连接的其他容器名
	for _, name := range names {
		if strings.HasPrefix(name, "/") && len(strings.Split(name[1:], "/")) == 1 {
			return name[1:]
		}
	}
	return ""
}

func containerResourceStat(realtimeStats map[string]*pb.ResourceStat, c *types.Container) *pb.ResourceStat {
	data := pb.ResourceStat{
		CpuStat: &pb.CpuStat{},
		MemStat: &pb.MemoryStat{},
		DiskStat: &pb.DiskStat{
			Used: float64(c.SizeRw) / 1e6, // 与docker client保持一致
		},
	}

	if v, ok := c.Labels["KS_SCMC_DISK_LIMIT"]; ok {
		fmt.Sscanf(v, "%f", &data.DiskStat.Limit)
	}

	if realtimeStats == nil {
		return &data
	}

	if stat, ok := realtimeStats[c.ID]; ok {
		if stat.CpuStat != nil {
			data.CpuStat = stat.CpuStat
		}
		if stat.MemStat != nil {
			data.MemStat = stat.MemStat
		}
		if stat.NetStat != nil {
			data.NetStat = stat.NetStat
		}
	}
	return &data
}

func createContainerEnvs(m map[string]string, uuid string) []string {
	ret := make([]string, 0, len(m)+1)
	for k, v := range m {
		ret = append(ret, fmt.Sprintf("%s=%s", k, v))
	}
	ret = append(ret, "KS_SCMC_UUID="+uuid)
	return ret
}
