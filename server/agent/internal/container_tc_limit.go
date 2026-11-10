package internal

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"

	"scmc/model/docker"
	"scmc/model/litedb"
)

// 通过容器ID获取容器里面 docker 网卡
func (s *ContainerServer) GetContainerNIC(containerID string) (map[string]string, error) {
	cli, err := docker.Client()
	if err != nil {
		return nil, fmt.Errorf("failed to create docker client: %w", err)
	}

	// 获取容器详情
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	containerJSON, err := cli.ContainerInspect(ctx, containerID)
	if err != nil {
		return nil, fmt.Errorf("failed to inspect container: %w", err)
	}

	networks := containerJSON.NetworkSettings.Networks
	if len(networks) == 0 {
		return nil, fmt.Errorf("no networks found for container")
	}

	result := make(map[string]string) // mac -> network name
	for netName, info := range networks {
		if info.MacAddress != "" {
			mac := strings.ToLower(info.MacAddress)
			result[mac] = netName
		}
	}

	return result, nil
}

// 获取容器里面的网卡信息
func (s *ContainerServer) GetContainerInterfacesWithMAC(containerID string) ([]struct{ Name, MAC string }, error) {
	pidOut, err := s.GetContainerPid(containerID)
	if err != nil {
		return nil, err
	}

	pid := strings.TrimSpace(fmt.Sprintf("%d", pidOut))
	if pid == "0" {
		return nil, fmt.Errorf("container not running")
	}

	netns := "/proc/" + pid + "/ns/net"
	if _, err := os.Stat(netns); os.IsNotExist(err) {
		return nil, fmt.Errorf("netns not found")
	}

	// 在容器 netns 中执行 ip -br link show
	cmd := exec.Command("nsenter", "--net="+netns, "ip", "-br", "link", "show")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("nsenter failed: %w", err)
	}

	var interfaces []struct{ Name, MAC string }
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 3 {
			continue
		}
		iface := fields[0]
		mac := strings.ToLower(fields[2])

		if iface == "lo" || mac == "00:00:00:00:00:00" {
			continue
		}

		if !strings.HasPrefix(iface, "eth") { // 是否为 eth 开头的网卡
			continue
		}

		// 去掉 @ifXX 后缀
		if idx := strings.Index(iface, "@"); idx != -1 {
			iface = iface[:idx]
		}

		interfaces = append(interfaces, struct{ Name, MAC string }{iface, mac})
	}

	return interfaces, nil
}

func (s *ContainerServer) ContainerTCConfig(containerID string, Nic string) {
	macToNet, err := s.GetContainerNIC(containerID)
	if err != nil {
		log.Warnf("GetContainerNIC(%s), Error: %v\n", containerID, err)
		return
	}

	if len(macToNet) == 0 {
		log.Warnf("No network interfaces found.")
		return
	}

	interfaces, err := s.GetContainerInterfacesWithMAC(containerID)
	if err != nil {
		log.Warnf("GetContainerInterfacesWithMAC(%s), Error getting container interfaces: %v\n", containerID, err)
		return
	}

	for _, iface := range interfaces {
		netName := macToNet[iface.MAC]
		if netName == "" || strings.TrimSpace(netName) != strings.TrimSpace(Nic) {
			continue
		}

		// 查询数据库获取网卡配置信息
		data, err := litedb.GetNetworkLimitByNic(containerID, Nic)
		if err != nil {
			return
		}

		if data.ContainerInterfaceName != iface.Name {
			data.ContainerInterfaceName = iface.Name
			litedb.UpdateNetworkLimit(data)
		}

		err2 := s.ApplyMultiInterfaceLimit(containerID, iface.Name, data.NetWbw, data.NetRbw)
		if err2 != nil {
			log.Warnf("TC err: %v, containerID: %s, iface.Name: %v, NetWbw: %v, NetRbw: %v", err2, iface.Name, data.NetWbw, data.NetRbw)
		}
	}
}

// 删除容器里面的 ifb 网卡
func delContainerNicIfb(containerID string, iface string) {
	if iface == "" {
		log.Debugf("Invalid parameter")
		return
	}

	netns, err := getNetnsPath(containerID)
	if err != nil {
		log.Debugf("Get netns err: %v", err)
		return
	}

	ifbName, err := ifbNameForInterface(iface)
	if err != nil {
		log.Debugf("Get ifbName err: %v", err)
		return
	}

	runInNetns(netns, "ip", "link", "del", ifbName)
}

func (s *ContainerServer) ApplyMultiInterfaceLimit(containerID, iface string, EgressMbps, IngressMbps float64) error {
	if EgressMbps < 0 {
		EgressMbps = 0
	}
	if IngressMbps < 0 {
		IngressMbps = 0
	}

	if err := ensureIFBModuleLoaded(); err != nil {
		log.Warning("Warning: IFB module may not be loaded")
	}
	netns, err := getNetnsPath(containerID)
	ifbName, err := ifbNameForInterface(iface)
	if err != nil {
		return err
	}
	cleanupBandwidthRules(netns, iface)

	if EgressMbps > 0 {
		egressStr := mbpsToTcRate(EgressMbps)
		if err := limitEgress(netns, iface, egressStr); err != nil {
			return fmt.Errorf("egress on %s: %w", iface, err)
		}
	}

	if IngressMbps > 0 {
		if err := setupIFBForIngress(netns, ifbName); err != nil {
			return fmt.Errorf("setup IFB for %s: %w", iface, err)
		}

		if err := redirectIngressToIFB(netns, iface, ifbName); err != nil {
			return fmt.Errorf("redirect for %s: %w", iface, err)
		}

		ingressStr := mbpsToTcRate(IngressMbps)
		if err := limitIngressViaIFB(netns, ifbName, ingressStr); err != nil {
			return fmt.Errorf("ingress on %s: %w", iface, err)
		}
	}

	return nil
}

func mbpsToTcRate(mbps float64) string {
	if mbps <= 0 {
		mbps = 1
	}

	// 使用 'f' 格式，保留最多 3 位小数，然后去除尾随零和小数点
	s := fmt.Sprintf("%.3f", mbps)
	s = strings.TrimRight(s, "0")
	s = strings.TrimRight(s, ".")
	return s + "mbit"
}

// ifbNameForInterface 返回 eth0 -> ifb0, eth1 -> ifb1
func ifbNameForInterface(iface string) (string, error) {
	if !strings.HasPrefix(iface, "eth") {
		return "", fmt.Errorf("unsupported interface prefix: %s", iface)
	}
	numStr := strings.TrimPrefix(iface, "eth")
	if _, err := strconv.Atoi(numStr); err != nil {
		return "", fmt.Errorf("invalid interface number in %s", iface)
	}
	return "ifb" + numStr, nil
}

// getNetnsPath returns the network namespace path of a running container
func getNetnsPath(containerID string) (string, error) {
	cli, err := docker.Client()
	if err != nil {
		return "", fmt.Errorf("failed to create docker client: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	info, err := cli.ContainerInspect(ctx, containerID)
	if err != nil {
		return "", err
	}

	if !info.State.Running {
		return "", fmt.Errorf("container is not running")
	}

	pid := info.State.Pid
	return fmt.Sprintf("/proc/%d/ns/net", pid), nil
}

// runInNetns runs a command inside the given network namespace
func runInNetns(netns string, args ...string) error {
	cmd := exec.Command("nsenter", append([]string{"--net=" + netns}, args...)...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("command %v failed: %w (output: %s)", args, err, string(output))
	}
	return nil
}

// ensureIFBModuleLoaded tries to load the ifb kernel module
func ensureIFBModuleLoaded() error {
	exec.Command("modprobe", "ifb", "numifbs=16").Run()
	return nil
}

// cleanupBandwidthRules removes existing tc rules and IFB device for an interface
func cleanupBandwidthRules(netns, iface string) error {
	ifbName, err := ifbNameForInterface(iface)
	if err != nil {
		return err
	}

	runInNetns(netns, "tc", "qdisc", "del", "dev", iface, "root")
	runInNetns(netns, "tc", "qdisc", "del", "dev", iface, "ingress")
	runInNetns(netns, "ip", "link", "del", ifbName)
	return nil
}

// limitEgress applies egress (upload) bandwidth limit
func limitEgress(netns, iface, rate string) error {
	if err := runInNetns(netns, "tc", "qdisc", "add", "dev", iface, "root", "handle", "1:", "htb", "default", "10"); err != nil {
		return fmt.Errorf("add root qdisc on %s: %w", iface, err)
	}
	if err := runInNetns(netns, "tc", "class", "add", "dev", iface, "parent", "1:", "classid", "1:1", "htb", "rate", rate); err != nil {
		return fmt.Errorf("set egress rate on %s: %w", iface, err)
	}
	return nil
}

// setupIFBForIngress creates and brings up an IFB device
func setupIFBForIngress(netns, ifbName string) error {
	if err := runInNetns(netns, "ip", "link", "add", ifbName, "type", "ifb"); err != nil {
		return fmt.Errorf("create %s: %w", ifbName, err)
	}
	if err := runInNetns(netns, "ip", "link", "set", ifbName, "up"); err != nil {
		return fmt.Errorf("bring up %s: %w", ifbName, err)
	}
	return nil
}

// redirectIngressToIFB redirects ingress traffic to IFB
func redirectIngressToIFB(netns, iface, ifbName string) error {
	if err := runInNetns(netns, "tc", "qdisc", "add", "dev", iface, "ingress"); err != nil {
		return fmt.Errorf("add ingress qdisc on %s: %w", iface, err)
	}
	if err := runInNetns(netns,
		"tc", "filter", "add", "dev", iface, "parent", "ffff:",
		"protocol", "all", "u32", "match", "u32", "0", "0",
		"action", "mirred", "egress", "redirect", "dev", ifbName); err != nil {
		return fmt.Errorf("redirect %s to %s: %w", iface, ifbName, err)
	}
	return nil
}

// limitIngressViaIFB applies ingress (download) limit on IFB
func limitIngressViaIFB(netns, ifbName, rate string) error {
	if err := runInNetns(netns, "tc", "qdisc", "add", "dev", ifbName, "root", "handle", "2:", "htb"); err != nil {
		return fmt.Errorf("add root qdisc on %s: %w", ifbName, err)
	}
	if err := runInNetns(netns,
		"tc", "class", "add", "dev", ifbName, "parent", "2:", "classid", "2:1", "htb", "rate", rate); err != nil {
		return fmt.Errorf("set ingress rate on %s: %w", ifbName, err)
	}
	return nil
}
