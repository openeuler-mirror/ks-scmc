package internal

import (
	"context"
	"fmt"
	"github.com/docker/docker/api/types/filters"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/client"
	log "github.com/sirupsen/logrus"

	"scmc/common"
	"scmc/model/docker"
	"scmc/model/db"
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
type DockerManager struct {
	client       *client.Client
	tcController *TrafficController
	cancel       context.CancelFunc
	startOnce    sync.Once  // 保证启动逻辑只执行一次
	mu           sync.Mutex // 状态锁
	isRunning    bool       // 运行状态标识
}

var (
	instance *DockerManager
	once     sync.Once
)

func GetDockerManager(cli *client.Client) *DockerManager {
	once.Do(func() {
		tc := NewTrafficController()
		instance = &DockerManager{
			client:       cli,
			tcController: tc,
		}
	})
	return instance
}
func (dm *DockerManager) Start() {
	dm.startOnce.Do(func() {
		dm.mu.Lock()
		defer dm.mu.Unlock()

		if !dm.isRunning {
			go dm.startEventLoop() // 真正的启动逻辑
			dm.isRunning = true
		}
	})
}
func (dm *DockerManager) startEventLoop() {
	ctx, cancel := context.WithCancel(context.Background())
	dm.cancel = cancel
	options := types.EventsOptions{
		Filters: filters.NewArgs(
			filters.Arg("type", events.ContainerEventType),
		),
	}
	for {
		msgChan, errs := dm.client.Events(ctx, options)
		select {
		case msg := <-msgChan:
			dm.handleEvent(msg)
		case err := <-errs:
			log.Printf("Docker event error: %v", err)
		case <-ctx.Done():
			return
		}
	}
}
func (dm *DockerManager) Stop() {
	if dm.cancel != nil {
		dm.cancel()
	}
}

func (dm *DockerManager) handleEvent(msg events.Message) {
	if msg.Type != events.ContainerEventType {
		return
	}

	switch msg.Action {
	case "start", "restart":
		// 获取限速配置
		limit, err := db.QueryContainerLimitsByContainerId(msg.Actor.ID)
		if err != nil {
			log.Errorf("Events: start or restart.Can not get container limit config,err: %v", err)
			return
		}

		// 应用限速规则
		if pid, err := getContainerPID(msg.Actor.ID); err == nil {
			log.Errorf("Events: start or restart.Can not get container pid,err: %v", err)
			if pid == 0 {
				log.Errorf("container pid err.")
				return
			}
			dm.tcController.ApplyLimit(pid, limit.UploadSpeed, limit.DownSpeed)
		}

	case "die", "destroy":
		// 清理规则
		if pid, err := getContainerPID(msg.Actor.ID); err == nil {
			log.Errorf("Events: die or destroy.Can not get container limit config,err: %v", err)
			if pid == 0 {
				log.Errorf("Events: die or destroy.container pid err.")
				return
			}
			dm.tcController.RemoveLimit(pid)
		}
		//dm.dbManager.DeleteLimit(event.ID)
	}
}

type TrafficController struct{}

func NewTrafficController() *TrafficController {
	return &TrafficController{}
}
func (tc *TrafficController) ApplyLimit(pid int, upload, download float64) error {
	if upload == 0 && download == 0 {
		return tc.RemoveLimit(pid)
	}

	if upload > 0 {
		if err := setupIngress(pid, upload); err != nil {
			return err
		}
	}

	if download > 0 {
		if err := setupEgress(pid, download); err != nil {
			return err
		}
	}
	return nil
}
func (tc *TrafficController) RemoveLimit(pid int) error {
	cmds := []string{
		fmt.Sprintf("nsenter -t %d -n tc qdisc del dev eth0 root", pid),
		fmt.Sprintf("nsenter -t %d -n tc qdisc del dev eth0 ingress", pid),
		fmt.Sprintf("nsenter -t %d -n tc qdisc del dev ifb0 root", pid),
		fmt.Sprintf("nsenter -t %d -n ip link del ifb0", pid),
	}

	for _, cmd := range cmds {
		if err := exec.Command("sh", "-c", cmd).Run(); err != nil {
			// 忽略"Device or resource busy"等错误
			log.Errorf("cmd: %v. exec error: %v", cmd, err)
		}
	}
	return nil
}
func setupIngress(pid int, rate float64) error {
	if err := createIFB(pid); err != nil {
		return err
	}

	cmds := []string{
		fmt.Sprintf("nsenter -t %d -n tc qdisc add dev eth0 handle ffff: ingress", pid),
		fmt.Sprintf("nsenter -t %d -n tc filter add dev eth0 parent ffff: protocol ip u32 match u32 0 0 action mirred egress redirect dev ifb0", pid),
		fmt.Sprintf("nsenter -t %d -n tc qdisc add dev ifb0 root tbf rate %dmbit latency 100ms burst %d",
			pid, rate*8, calculateBurst(rate)),
	}

	for _, cmd := range cmds {
		if err := exec.Command("sh", "-c", cmd).Run(); err != nil {
			log.Errorf("ingress setup failed: %v", err)
			return fmt.Errorf("ingress setup failed: %v", err)
		}
	}
	return nil
}
func setupEgress(pid int, rate float64) error {
	cmd := fmt.Sprintf("nsenter -t %d -n tc qdisc add dev eth0 root tbf rate %dmbit latency 100ms burst %d",
		pid, rate*8, calculateBurst(rate))
	return exec.Command("sh", "-c", cmd).Run()
}
func createIFB(pid int) error {
	cmds := []string{
		"modprobe ifb",
		fmt.Sprintf("ip link add ifb0 type ifb"),
		fmt.Sprintf("ip link set ifb0 netns %d", pid),
		fmt.Sprintf("nsenter -t %d -n ip link set ifb0 up", pid),
	}

	for _, cmd := range cmds {
		if err := exec.Command("sh", "-c", cmd).Run(); err != nil {
			return fmt.Errorf("IFB creation failed: %v", err)
		}
	}
	return nil
}

func calculateBurst(rate float64) float64 {
	if rate < 1 {
		return 1000
	}
	return rate * 125000 // 1MB/s = 1,000,000 bytes
}
func getContainerPID(containerID string) (int, error) {
	// 实现获取容器PID的逻辑（通过docker inspect）
	// 返回PID和可能的错误
	s := &ContainerServer{}
	pid, err := s.GetContainerPid(containerID)
	if err != nil {
		return 0, err
	}
	return pid, nil
}