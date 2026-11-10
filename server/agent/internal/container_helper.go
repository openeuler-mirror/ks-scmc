package internal

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/client"
	log "github.com/sirupsen/logrus"

	"scmc/common"
	"scmc/model/db"
	"scmc/model/docker"
	"scmc/model/litedb"
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
	if err := os.MkdirAll(socketDir, 0o755); err != nil {
		log.Warnf("mkdir %v: %v", socketDir, err)
		return err
	}

	authDir := filepath.Join(hostPath, "xauth")
	if err := os.MkdirAll(authDir, 0o755); err != nil {
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
			Used:  float64(c.SizeRw) / 1e6, // 与docker client保持一致
			Limit: float64(c.SizeRootFs) / 1e6,
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

// 返回 VIP
func getVIPAddr() (string, error) {
	vip := common.Config.Controller.VirtualIP
	vip = strings.TrimSpace(vip)
	if vip == "" {
		return "", fmt.Errorf("Invalid VIP format: [%s]", vip)
	}

	ip, _, err := net.ParseCIDR(vip) // 如果有子网掩码
	if err == nil {
		return ip.String(), nil
	}

	parsedIP := net.ParseIP(vip) // 测试没有子网掩码的情况
	if parsedIP == nil {
		return "", fmt.Errorf("Invalid VIP format: [%s]", vip)
	}

	return parsedIP.String(), nil
}

var ipv4Pattern = regexp.MustCompile(
	`^\s*` +
		`((25[0-5]|2[0-4]\d|1\d{2}|[1-9]?\d)\.){3}` +
		`(25[0-5]|2[0-4]\d|1\d{2}|[1-9]?\d)` +
		`(?:/\d{1,3})?` +
		`\s*$`,
)

// 从 Keepalived 配置文件中提取 VIP
func extractVIPFromConfig(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open file: %v", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	inVrrpInstance := false
	inVirtualIPBlock := false

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// 检查是否进入 vrrp_instance 块
		if strings.HasPrefix(line, "vrrp_instance") {
			inVrrpInstance = true
			continue
		}

		// 在 vrrp_instance 块内检查 virtual_ipaddress
		if inVrrpInstance {
			if strings.HasPrefix(line, "virtual_ipaddress") {
				inVirtualIPBlock = true
				continue
			}

			// 进入 virtual_ipaddress 块后提取 IP
			if inVirtualIPBlock {
				if strings.Contains(line, "}") {
					break // 退出 virtual_ipaddress 块
				}

				if ipv4Pattern.MatchString(line) {
					return strings.TrimSpace(strings.Split(line, "/")[0]), nil
				}
			}
		}
	}

	return "", fmt.Errorf("VIP not found in config")
}

// 通用切片比较函数
func sliceEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}

	// 如果接受乱序可排序后比较
	sort.Strings(a)
	sort.Strings(b)

	// 要求严格顺序匹配
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// 深度比较两个NetworkRule是否相等
func isRuleEqual(a, b *litedb.NetworkRule) bool {
	return a.Addr == b.Addr &&
		a.Port == b.Port &&
		sliceEqual(a.Protocols, b.Protocols)
}

// 检查规则是否存在（线性搜索）
func containsRule(rules []*litedb.NetworkRule, target *litedb.NetworkRule) bool {
	for _, rule := range rules {
		if isRuleEqual(rule, target) {
			return true
		}
	}
	return false
}

// 合并规则的交集
func UnionNetworkRules(arr1, arr2 []*litedb.NetworkRule) ([]*litedb.NetworkRule, error) {
	merged := make([]*litedb.NetworkRule, 0)
	seen := make(map[string]struct{}) // 用于去重的特征码集合

	// 遍历所有规则组合
	for _, r1 := range arr1 {
		for _, r2 := range arr2 {
			// 1. 检查IP交集
			ipNet, ok := getCommonIPNet(r1.Addr, r2.Addr)
			if !ok {
				continue
			}

			// 2. 检查端口交集
			commonPort := getCommonPort(r1.Port, r2.Port)
			if commonPort == ^uint32(0) { // 无有效端口交集
				continue
			}

			// 3. 检查协议交集
			commonProtocols := getCommonProtocols(r1.Protocols, r2.Protocols)
			if len(commonProtocols) == 0 {
				continue
			}

			newipnet := ipNet.String()
			if strings.HasSuffix(newipnet, "/32") {
				newipnet = strings.Split(newipnet, "/")[0]
			}

			// 构造新规则
			newRule := &litedb.NetworkRule{
				Addr:      newipnet,
				Port:      commonPort,
				Protocols: commonProtocols,
			}

			// 去重处理
			if sig := getRuleSignature(newRule); sig != "" {
				if _, exists := seen[sig]; !exists {
					seen[sig] = struct{}{}
					merged = append(merged, newRule)
				}
			}
		}
	}

	return merged, nil
}

// --- IP地址处理逻辑 ---
func parseCIDR(addr string) (*net.IPNet, error) {
	if !strings.Contains(addr, "/") { // 处理单IP地址
		addr += "/32" // IPv4掩码
	}
	_, ipNet, err := net.ParseCIDR(addr)
	return ipNet, err
}

// 获取IP网段交集
func getCommonIPNet(addr1, addr2 string) (*net.IPNet, bool) {
	ipNet1, err1 := parseCIDR(addr1)
	ipNet2, err2 := parseCIDR(addr2)
	if err1 != nil || err2 != nil {
		return nil, false
	}

	// 检查包含关系
	if ipNet1.Contains(ipNet2.IP) && ipNet2.Contains(ipNet1.IP) {
		return ipNet1, true // 等价网段
	} else if ipNet1.Contains(ipNet2.IP) {
		return ipNet2, true // ipNet2是子网
	} else if ipNet2.Contains(ipNet1.IP) {
		return ipNet1, true // ipNet1是子网
	}
	return nil, false
}

// --- 端口处理逻辑 ---
func getCommonPort(p1, p2 uint32) uint32 {
	if p1 == 0 {
		return p2
	}
	if p2 == 0 {
		return p1
	}
	if p1 == p2 {
		return p1
	}
	return ^uint32(0) // 无效标记
}

// --- 协议处理逻辑 ---
func getCommonProtocols(p1, p2 []string) []string {
	set := make(map[string]struct{})
	for _, p := range p1 {
		set[p] = struct{}{}
	}
	common := make([]string, 0)
	for _, p := range p2 {
		if _, ok := set[p]; ok {
			common = append(common, p)
		}
	}
	sort.Strings(common) // 排序保证特征码一致
	return common
}

// 生成规则唯一签名
func getRuleSignature(r *litedb.NetworkRule) string {
	if r == nil || len(r.Protocols) == 0 {
		return ""
	}
	return fmt.Sprintf("%s:%d:%s", r.Addr, r.Port, strings.Join(r.Protocols, ","))
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
		// dm.dbManager.DeleteLimit(event.ID)
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
