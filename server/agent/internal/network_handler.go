package internal

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"bufio"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	log "github.com/sirupsen/logrus"
	"github.com/vishvananda/netlink"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"scmc/model/docker"
	"scmc/model/iptables"
	"scmc/rpc"
	pb "scmc/rpc/pb/network"
)

const defaultContainerChain = "INPUT"
const defaultNodeChain = "OUTPUT"
const defaultPolicy = "ACCEPT"

const VirtualNicBase = "/sys/devices/virtual/net"

type NetworkServer struct {
	pb.UnimplementedNetworkServer
}

func parseSubnet(s string) (string, int, error) {
	if len(s) == 0 {
		return "", 0, nil
	}

	ip, info, err := net.ParseCIDR(s)
	if err != nil {
		return "", 0, err
	}

	n, _ := info.Mask.Size()
	return ip.String(), int(n), nil
}

func isVirtualNic(name string) (bool, error) {
	_, err := os.Stat(filepath.Join(VirtualNicBase, name))
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		log.Warnf("os.Stat(%v/%v) err=%v", VirtualNicBase, name, err)
		return false, err
	}
	return true, nil
}

func (s *NetworkServer) listDockerIfaces(ctx context.Context) ([]*pb.NetworkInterface, error) {
	cli, err := docker.Client()
	if err != nil {
		return nil, rpc.ErrInternal
	}

	ns, err := cli.NetworkList(ctx, types.NetworkListOptions{})
	if err != nil {
		log.Warnf("NetworkList: %v", err)
		return nil, transDockerError(err)
	}

	ifaces := make([]*pb.NetworkInterface, 0, len(ns))
	for _, n := range ns {
		// 不显示默认网卡
		if isPredefinedNetwork(n.Name) {
			continue
		}
		if len(n.IPAM.Config) != 1 {
			log.Warnf("bridge network[%s] unexpect config %v", n.Name, n.IPAM.Config)
			continue
		}

		conf := n.IPAM.Config[0]
		addr, masklen, err := parseSubnet(conf.Subnet)
		if err != nil {
			log.Warnf("bridge network[%s] parse subnet=%v return err=%v", n.Name, conf.Subnet, err)
		}

		detail, err := cli.NetworkInspect(ctx, n.Name, types.NetworkInspectOptions{})
		if err != nil {
			log.Warnf("NetworkInspect: %v", err)
		}

		var linkIfs string
		if detail.Options != nil {
			linkIfs = detail.Options["parent"]
		}

		ifaces = append(ifaces, &pb.NetworkInterface{
			Name:      n.Name,
			BindReal:  linkIfs,
			IpAddress: addr,
			IpMaskLen: int32(masklen),
			Gateway:   conf.Gateway,
			IsUp:      true,
		})
	}

	return ifaces, nil
}

func (s *NetworkServer) ifaceGateway(name string) (string, error) {
	link, err := netlink.LinkByName(name)
	if err != nil {
		log.Infof("netlink.LinkByName(%s) err=%v", name, err)
		return "", err
	}

	routes, err := netlink.RouteList(link, netlink.FAMILY_V4)
	if err != nil {
		log.Infof("link=%s netlink.RouteList %v", name, err)
		return "", err
	}

	var gw string
	for _, r := range routes {
		if r.Gw != nil {
			gw = r.Gw.String()
		}
	}
	if gw == "" {
		gw = "0.0.0.0"
	}
	return gw, nil
}

func (s *NetworkServer) listPhysicalIfaces() ([]*pb.NetworkInterface, error) {
	ifs, err := net.Interfaces()
	if err != nil {
		log.Warnf("net.Interface err=%v", err)
		return nil, rpc.ErrInternal
	}

	ifaces := make([]*pb.NetworkInterface, 0, len(ifs))
	for _, i := range ifs {
		log.Tracef("Network interface %+v", i)
		isVirt, err := isVirtualNic(i.Name)
		if isVirt || err != nil {
			continue
		}

		iface := &pb.NetworkInterface{
			Name:       i.Name,
			IsUp:       strings.Contains(i.Flags.String(), "up"),
			MacAddress: i.HardwareAddr.String(),
		}

		addrs, err := i.Addrs()
		if err != nil || len(addrs) == 0 {
			log.Warnf("retrieve addrs from interface %v err=%v", i, err)
			continue
		}

		addr := addrs[0]
		ip, ipNet, err := net.ParseCIDR(addr.String())
		if err != nil {
			log.Warnf("net.ParseCIDR(%s) err=%v", addr.String(), err)
			continue
		}

		if ipv4 := ip.To4(); ipv4 != nil {
			ones, _ := ipNet.Mask.Size()
			iface.IpAddress = ipv4.String()
			iface.IpMaskLen = int32(ones)

			var s []string
			for _, i := range ipNet.Mask {
				s = append(s, strconv.Itoa(int(i)))
			}
			iface.IpMask = strings.Join(s, ".")
		}
		iface.Gateway, _ = s.ifaceGateway(i.Name)
		ifaces = append(ifaces, iface)
	}
	return ifaces, nil
}

func (s *NetworkServer) List(ctx context.Context, in *pb.ListRequest) (*pb.ListReply, error) {
	dockerIfaces, err := s.listDockerIfaces(ctx)
	if err != nil {
		log.Errorf("listDockerIfaces: %v", err)
		return nil, err
	}

	physicalIfaces, err := s.listPhysicalIfaces()
	if err != nil {
		log.Errorf("listPhysicalIfaces: %v", err)
		return nil, err
	}

	return &pb.ListReply{
		RealIfs:    physicalIfaces,
		VirtualIfs: dockerIfaces,
	}, nil
}

func (s *NetworkServer) Connect(ctx context.Context, in *pb.ConnectRequest) (*pb.ConnectReply, error) {
	if in.Interface == "" || in.ContainerId == "" || in.IpAddress == "" {
		return nil, rpc.ErrInvalidArgument
	}

	cli, err := docker.Client()
	if err != nil {
		return nil, rpc.ErrInternal
	}

	var config = &network.EndpointSettings{
		NetworkID:   in.Interface,
		Gateway:     in.Gateway,
		IPAddress:   in.IpAddress,
		IPPrefixLen: int(in.IpPrefixLen),
		MacAddress:  in.MacAddress,
	}

	config.IPAMConfig = &network.EndpointIPAMConfig{
		IPv4Address: in.IpAddress,
	}

	err = cli.NetworkConnect(context.Background(), in.Interface, in.ContainerId, config)
	if err != nil {
		log.Warnf("NetworkConnect: %v", err)
		return nil, transDockerError(err)
	}

	return &pb.ConnectReply{}, err
}

func (s *NetworkServer) Disconnect(ctx context.Context, in *pb.DisconnectRequest) (*pb.DisconnectReply, error) {
	if in.Interface == "" || in.ContainerId == "" {
		return nil, rpc.ErrInvalidArgument
	}

	cli, err := docker.Client()
	if err != nil {
		return nil, rpc.ErrInternal
	}

	if err = cli.NetworkDisconnect(context.Background(), in.Interface, in.ContainerId, true); err != nil {
		log.Warnf("NetworkDisconnect: %v", err)
		return nil, transDockerError(err)
	}

	return &pb.DisconnectReply{}, err
}

func (s *NetworkServer) ListIPtables(ctx context.Context, in *pb.ListIPtablesRequest) (*pb.ListIPtablesReply, error) {
	who := iptables.OperateNode
	pid := 0
	if in.ContainerId != "" {
		cli, err := docker.Client()
		if err != nil {
			return nil, rpc.ErrInternal
		}

		info, err := cli.ContainerInspect(context.Background(), in.ContainerId)
		if err != nil || info.State.Pid == 0 {
			log.Warnf("ContainerInspect err(%v) or container not start", err)
			return nil, transDockerError(err)
		}
		who = iptables.OperateContainer
		pid = info.State.Pid
	}

	list, err := iptables.ListRules(who, pid)
	if err != nil {
		return nil, rpc.ErrInternal
	}

	var reply pb.ListIPtablesReply

	for _, chains := range list {
		var chainRule pb.ChianRule
		chainRule.Chain = chains.Chain
		for _, rule := range chains.Rules {
			chainRule.Rule = append(chainRule.Rule, &pb.RuleInfo{
				Source:       rule.Source,
				Destination:  rule.Destination,
				Protocol:     rule.Protocol,
				SrcPort:      rule.SrcPort,
				DestPort:     rule.DestPort,
				InInterface:  rule.InInterface,
				OutInterface: rule.OutInterface,
				Policy:       rule.Policy,
			})
		}

		reply.ChainRules = append(reply.ChainRules, &chainRule)
	}

	return &reply, nil
}

func (s *NetworkServer) EnableIPtables(ctx context.Context, in *pb.EnableIPtablesRequest) (*pb.EnableIPtablesReply, error) {
	if in.ContainerId != "" {
		cli, err := docker.Client()
		if err != nil {
			return nil, rpc.ErrInternal
		}

		info, err := cli.ContainerInspect(context.Background(), in.ContainerId)
		if err != nil || info.State.Pid == 0 {
			log.Warnf("ContainerInspect err(%v) or container not start", err)
			return nil, transDockerError(err)
		}

		var name string
		if strings.HasPrefix(info.Name, "/") && len(strings.Split(info.Name[1:], "/")) == 1 {
			name = info.Name[1:]
		}
		fileName := info.ID + "-" + name

		if in.Enable {
			iptables.EnableContainer(fileName, info.State.Pid)
		} else {
			iptables.DisableContainer(fileName, info.State.Pid)
		}
	} else {
		if in.Enable {
			linkifs := getLinkIfs()
			for _, ifs := range linkifs {
				iptables.EnableNode(ifs)
			}
		} else {
			iptables.DisableNode()
		}
	}

	return &pb.EnableIPtablesReply{}, nil

}

func (s *NetworkServer) CreateIPtables(ctx context.Context, in *pb.CreateIPtablesRequest) (*pb.CreateIPtablesReply, error) {
	if in.Rule == nil {
		return nil, rpc.ErrInvalidArgument
	}

	who := iptables.OperateNode
	pid := 0
	chain := defaultNodeChain

	if in.ContainerId != "" {
		cli, err := docker.Client()
		if err != nil {
			return nil, rpc.ErrInternal
		}

		info, err := cli.ContainerInspect(context.Background(), in.ContainerId)
		if err != nil || info.State.Pid == 0 {
			log.Warnf("ContainerInspect err(%v) or container not start", err)
			return nil, transDockerError(err)
		}
		who = iptables.OperateContainer
		pid = info.State.Pid
		chain = defaultContainerChain
	}

	if in.Chain != "" {
		chain = in.Chain
	}
	policy := defaultPolicy
	if in.Rule.Policy != "" {
		policy = in.Rule.Policy
	}
	rule := iptables.RuleInfo{
		Source:       in.Rule.Source,
		Destination:  in.Rule.Destination,
		Protocol:     in.Rule.Protocol,
		SrcPort:      in.Rule.SrcPort,
		DestPort:     in.Rule.DestPort,
		InInterface:  in.Rule.InInterface,
		OutInterface: in.Rule.OutInterface,
		Policy:       policy,
	}

	if err := iptables.AddRule(who, in.ContainerId, pid, chain, rule); err != nil {
		log.Warnf("AddRule: %v", err)
		return nil, rpc.ErrInternal
	}

	return &pb.CreateIPtablesReply{}, nil
}

func (s *NetworkServer) ModifyIPtables(ctx context.Context, in *pb.ModifyIPtablesRequest) (*pb.ModifyIPtablesReply, error) {
	who := iptables.OperateNode
	pid := 0
	oldChain := defaultNodeChain
	newChain := defaultNodeChain

	if in.ContainerId != "" {
		cli, err := docker.Client()
		if err != nil {
			return nil, rpc.ErrInternal
		}

		info, err := cli.ContainerInspect(context.Background(), in.ContainerId)
		if err != nil || info.State.Pid == 0 {
			log.Warnf("ContainerInspect err(%v) or container not start", err)
			return nil, transDockerError(err)
		}
		who = iptables.OperateContainer
		pid = info.State.Pid
		oldChain = defaultContainerChain
		newChain = defaultContainerChain
	}

	if in.OldChain != "" {
		oldChain = in.OldChain
	}
	oldPolicy := defaultPolicy
	if in.OldRule.Policy != "" {
		oldPolicy = in.OldRule.Policy
	}
	oldrule := iptables.RuleInfo{
		Source:       in.OldRule.Source,
		Destination:  in.OldRule.Destination,
		Protocol:     in.OldRule.Protocol,
		SrcPort:      in.OldRule.SrcPort,
		DestPort:     in.OldRule.DestPort,
		InInterface:  in.OldRule.InInterface,
		OutInterface: in.OldRule.OutInterface,
		Policy:       oldPolicy,
	}

	if in.NewChain != "" {
		newChain = in.NewChain
	}
	newPolicy := defaultPolicy
	if in.NewRule.Policy != "" {
		newPolicy = in.NewRule.Policy
	}
	newrule := iptables.RuleInfo{
		Source:       in.NewRule.Source,
		Destination:  in.NewRule.Destination,
		Protocol:     in.NewRule.Protocol,
		SrcPort:      in.NewRule.SrcPort,
		DestPort:     in.NewRule.DestPort,
		InInterface:  in.NewRule.InInterface,
		OutInterface: in.NewRule.OutInterface,
		Policy:       newPolicy,
	}

	if err := iptables.ModifyRule(who, in.ContainerId, pid, oldChain, oldrule, newChain, newrule); err != nil {
		log.Warnf("ModifyContainerRule: %v", err)
		return nil, rpc.ErrInternal
	}

	return &pb.ModifyIPtablesReply{}, nil
}

func (s *NetworkServer) RemoveIPtables(ctx context.Context, in *pb.RemoveIPtablesRequest) (*pb.RemoveIPtablesReply, error) {
	if in.Rule == nil {
		return nil, rpc.ErrInvalidArgument
	}
	who := iptables.OperateNode
	pid := 0
	chain := defaultNodeChain

	if in.ContainerId != "" {
		cli, err := docker.Client()
		if err != nil {
			return nil, rpc.ErrInternal
		}

		info, err := cli.ContainerInspect(context.Background(), in.ContainerId)
		if err != nil || info.State.Pid == 0 {
			log.Warnf("ContainerInspect err(%v) or container not start", err)
			return nil, transDockerError(err)
		}
		who = iptables.OperateContainer
		pid = info.State.Pid
		chain = defaultContainerChain
	}

	if in.Chain != "" {
		chain = in.Chain
	}
	policy := defaultPolicy
	if in.Rule.Policy != "" {
		policy = in.Rule.Policy
	}
	rule := iptables.RuleInfo{
		Source:       in.Rule.Source,
		Destination:  in.Rule.Destination,
		Protocol:     in.Rule.Protocol,
		SrcPort:      in.Rule.SrcPort,
		DestPort:     in.Rule.DestPort,
		InInterface:  in.Rule.InInterface,
		OutInterface: in.Rule.OutInterface,
		Policy:       policy,
	}

	if err := iptables.DelRule(who, in.ContainerId, pid, chain, rule); err != nil {
		log.Warnf("DelRule: %v", err)
		return nil, rpc.ErrInternal
	}

	return &pb.RemoveIPtablesReply{}, nil
}

/* 删除 iptables 相关信息 */
func deleteIptablesInfo(routeInfo , parent_nic string) {
    if len(routeInfo) == 0 {
        return
    }

    var iptbInfo string
    var cird string
    parts := strings.Fields(routeInfo)
    for _, part := range parts {
         _, _, err := net.ParseCIDR(part)
        if err == nil {
            iptbInfo = "iptables -t nat -D POSTROUTING -s " + part  + " -o " + parent_nic + " -j MASQUERADE"
            cird = part
        }
    }

    /* 执行命令 */
    if len(iptbInfo) > 0 {
	    cmd := exec.Command(
            "iptables",
            "-t", "nat",
            "-D", "POSTROUTING",
            "-s", cird,
            "-o", parent_nic,
            "-j", "MASQUERADE",
        )

        err := cmd.Run()
        if err != nil {
            log.Warnf("Run Command=%s, err=%v", iptbInfo, err)
        }
    }
}


func insertAndDeleteLines(parent_nic string, filePath string, startStr, endStr string, newContents []string) error {
	// 打开文件
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file: %v", filePath)
	}
	defer file.Close()

	// 读取文件内容
	lines, err := readLines(file)
	if err != nil {
		return fmt.Errorf("failed to read file: %v", filePath)
	}

    var routeInfo string
	// 查找开始和结束的行索引
	startIndex, endIndex := -1, -1
	for i, line := range lines {
		if strings.Contains(line, startStr) {
			startIndex = i
		}
		if strings.Contains(line, endStr) && startIndex != -1 {
			endIndex = i
			break // 找到结束索引后退出循环
		}
        if strings.Contains(line, "ip route add") {
            routeInfo = line
        }
	}

    // 删除添加的 Iptables 信息
    if len(routeInfo) > 0 {
        deleteIptablesInfo(routeInfo, parent_nic)
    }


	// 检查是否找到了开始和结束的行
	if startIndex == -1 || endIndex == -1 {
		// 如果没有找到开始或结束标记，只插入新内容到文件末尾
		lines = append(lines, newContents...)
	} else {
		// 删除指定行之间的内容，包括开始和结束行，并插入新内容
		lines = append(lines[:startIndex], append(newContents, lines[endIndex+1:]...)...)
	}

	// 写回文件
	err = writeLines(filePath, lines)
	if err != nil {
		return fmt.Errorf("failed to write file: %v", filePath)
	}

	return nil
}

func readLines(file *os.File) ([]string, error) {
	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return lines, nil
}

func writeLines(filePath string, lines []string) error {
	tempFile, err := os.Create(filePath + ".tmp")
	if err != nil {
		return err
	}
	defer tempFile.Close()

	writer := bufio.NewWriter(tempFile)
	for _, line := range lines {
		_, err := writer.WriteString(line + "\n")
		if err != nil {
			return err
		}
	}
	writer.Flush()

	// 替换原文件
	return os.Rename(filePath+".tmp", filePath)
}

func firstUsableIP(network string) (net.IP, error) {
	// 将CIDR表示法转换为IPNet结构
	_, ipnet, err := net.ParseCIDR(network)
	if err != nil {
		return nil, fmt.Errorf("failed to parse network: %v", network)
	}

	// 获取网络的第一个IP地址（通常是网络地址）
	firstIP := ipnet.IP

	// 将IP地址转换为可增加的切片形式
	ipv4 := firstIP.To4()
	if ipv4 == nil {
		return nil, fmt.Errorf("failed to get first ip of network: %v", network)
	}

	// 增加1以得到第一个可用的IP地址
	incrementIP(ipv4)

	return ipv4, nil
}

// incrementIP 将IP地址增加1
func incrementIP(ip net.IP) {
	for i := len(ip) - 1; i >= 0; i-- {
		ip[i]++
		if ip[i] != 0 {
			break
		}
	}
}

// 设置虚拟网卡的 MAC 地址	
func setVirtualNicMac(nicName string) {
	nicMAC := getMacJSON(nicName)
	if nicMAC == "" {
		updateJSON(nicName, getMacAddr(nicName))
	} else {
		setMACcmd := exec.Command("ifconfig", nicName, "hw", "ether", nicMAC)
		setMACcmd.Run()
	}
}

func (s *NetworkServer) createNetwork(name, subnet, gateway, parent string) error {
	driver := "bridge"
	if parent != "" {
		driver = "macvlan"
	}

	cli, err := docker.Client()
	if err != nil {
		return rpc.ErrInternal
	}

	macvlan_ip, masklen, _ := parseSubnet(subnet)
	_, sb, _ := net.ParseCIDR(subnet)
	subnet = sb.String()

	data := types.NetworkCreate{
		Driver: driver,
		IPAM: &network.IPAM{
			Config: []network.IPAMConfig{
				{
					Subnet:  subnet,
					Gateway: macvlan_ip,
				},
			},
		},
		Options: map[string]string{
			"parent":                         parent,
			"com.docker.network.bridge.name": "scmc-" + name,
		},
	}

	_, err = cli.NetworkCreate(context.Background(), name, data)
	if err != nil {
		log.Warnf("NetworkCreate name=%v err=%v", name, err)
		return err
	}

	if parent == "" {
		return nil
	}
/*
	firstip, err_ip := firstUsableIP(subnet)
	if err_ip != nil {
		log.Warnf("NetworkCreate name=%v err=%v", name, err_ip)
		return err_ip
	}
*/

	path := "/etc/sysconfig/network-scripts/ifcfg-" + parent
	begin_flag := "# ks-scmc config begin, donot modify"
	end_flag := "# ks-scmc config end, donot modify"

	newlines := []string{}
	add := ""
	host_name := name + "-host"

	newlines = append(newlines, begin_flag)
	add = fmt.Sprintf("ip link set %s promisc on", parent)
	newlines = append(newlines, add)

	add = fmt.Sprintf("ip link add link %s %s type macvlan mode bridge", parent, host_name)
	newlines = append(newlines, add)

	add = fmt.Sprintf("ip link set dev %s up", host_name)
	newlines = append(newlines, add)

	/*
	* #74606 修复并且验证创建 macvlan 的时候没有设置子网掩码的问题！
	* https://ones.kylinsec.com.cn:5080/project/#/team/6B7ew4b7/task/X6NeBChCU3lw1DK8
	*/
	// _, masklen, _ := parseSubnet(subnet)

	//add = fmt.Sprintf("ip addr add %s/%d dev %s", firstip, masklen, host_name)
	add = fmt.Sprintf("ip addr add %s/%d dev %s", macvlan_ip, masklen, host_name)
	newlines = append(newlines, add)

	add = fmt.Sprintf("ip route add %s dev %s", subnet, host_name)
	newlines = append(newlines, add)
	newlines = append(newlines, end_flag)

    /*
    // 添加一条 iptables 信息, 不能写入文件，会导致执行很多次
    // add = fmt.Sprintf("iptables -t nat -A POSTROUTING -s %s -o %s -j MASQUERADE", subnet, parent)
	// newlines = append(newlines, add)
    */
	err = insertAndDeleteLines(parent, path, begin_flag, end_flag, newlines)
	if err != nil {
		log.Warnf("NetworkCreate name=%v err=%v", name, err)
		return err
	}

    /* 添加规则 */
    iptbInfo := fmt.Sprintf("iptables -t nat -A POSTROUTING -s %s -o %s -j MASQUERADE", subnet, parent)
    cmd := exec.Command(
        "iptables",
        "-t", "nat",
        "-A", "POSTROUTING",
        "-s", subnet,
        "-o", parent,
        "-j", "MASQUERADE",
    )
    err = cmd.Run()
    if err != nil {
        log.Warnf("Run Command: %s, err= %v", iptbInfo, err)
    }
    /* 添加规则 */

	cmd = exec.Command("systemctl", "restart", "network")
	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("failed to execute command: %v", cmd)
	}

	log.Debugf("NetworkCreate parent=%v name=%v host_name=%v gateway=%v subnet=%v", parent, name, host_name, gateway, subnet)
	return nil
}

/*
 * 获取虚拟网卡对应的物理网卡名称 
 * getPhysicalInterfaceName 因为是通过调用 shell 命令实现,
 * 效率会差些, 作为备选方案
 */
func getParentInterface(virtualNicName string) (string, error) {
	macvlanName := virtualNicName + "-host"
	sysPath := filepath.Join("/sys/class/net", macvlanName)

	// 查找 lower_* 符号链接
	lowerLinks, err := filepath.Glob(filepath.Join(sysPath, "lower_*"))
	if err != nil {
		return getPhysicalInterfaceName(virtualNicName)
	}

	if len(lowerLinks) == 0 {
		return getPhysicalInterfaceName(virtualNicName)
	}

	// 解析符号链接目标（例如：lower_eth0 -> ../eth0）
	linkPath, err := os.Readlink(lowerLinks[0])
	if err != nil {
		return getPhysicalInterfaceName(virtualNicName)
	}

	// 提取父接口名称（如 "../eth0" -> "eth0"）
	parentName := filepath.Base(linkPath)
	return parentName, nil
}

// getPhysicalInterfaceName 通过虚拟网卡名称获取其绑定的物理网卡名称
// 绑定物理网卡的 <virtualNicName>-host
// 没有绑定物理网卡的   scmc-<virtualNicName>
func getPhysicalInterfaceName(virtualNicName string) (string, error) {
	// 执行 ip link show 命令
	cmd := exec.Command("ip", "link", "show", virtualNicName + "-host")
	output, err := cmd.CombinedOutput()
	if err != nil {
		//return "", fmt.Errorf("failed to execute command: %v", cmd)
		// 测试没有绑定物理网卡的情况
		cmd = exec.Command("ip", "link", "show", "scmc-" + virtualNicName)
		output, err = cmd.CombinedOutput()
		if err == nil {
			return "", nil
		}
		return "", fmt.Errorf("failed to execute command: %v", cmd)
	}

	// 解析输出，查找物理接口名称
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "@") {
			// 提取物理接口名称
			physicalNicName := strings.Split(line, "@")[1]
			if strings.Contains(physicalNicName, ":") {
				physicalNicName = strings.Split(physicalNicName, ":")[0]
			}
			// 去除物理接口名称前后的空白字符和冒号
			physicalNicName = strings.TrimSpace(physicalNicName)
			return physicalNicName, nil
		}
	}

	return "", nil
}

func (s *NetworkServer) removeNetwork(name string) error {
	//parent_nic, err_res := getPhysicalInterfaceName(name)
	parent_nic, err_res := getParentInterface(name)
	if err_res != nil {
		log.Warnf("NetworkRemove name=%v err=%v", name, err_res)
		return err_res
	}

	if parent_nic != "" {
		path := "/etc/sysconfig/network-scripts/ifcfg-" + parent_nic
		begin_flag := "# ks-scmc config begin, donot modify"
		end_flag := "# ks-scmc config end, donot modify"
		newlines := []string{}

		err_res = insertAndDeleteLines(parent_nic, path, begin_flag, end_flag, newlines)
		if err_res != nil {
			log.Warnf("NetworkCreate name=%v err=%v", name, err_res)
			return err_res
		}

		cmd := exec.Command("ip", "link", "del", "link", parent_nic, name + "-host")
		err_res = cmd.Run()
		if err_res != nil {
			return fmt.Errorf("failed to execute command: %v", cmd)
		}
	}

	cli, err := docker.Client()
	if err != nil {
		return rpc.ErrInternal
	}

	err = cli.NetworkRemove(context.Background(), name)
	if err != nil {
		if client.IsErrNotFound(err) {
			return rpc.ErrNotFound
		}
		log.Warnf("NetworkRemove name=%v err=%v", name, err)
		return err
	}

	// 没有绑定物理网卡的情况
	if parent_nic == "" {
		cmd := exec.Command("ip", "link", "del", "link", "scmc-" + name)
		_ = cmd.Run()
	}

	return nil
}

func (s *NetworkServer) checkRemoveNetwork(name string) error {
	if name == "" {
		return rpc.ErrInvalidArgument
	}

	cli, err := docker.Client()
	if err != nil {
		return rpc.ErrInternal
	}

	nw, err := cli.NetworkInspect(context.Background(), name, types.NetworkInspectOptions{})
	if err != nil {
		log.Warnf("NetworkInspect err=%v", err)
		if client.IsErrNotFound(err) {
			return status.Error(codes.FailedPrecondition, "网卡不存在")
		}
		return rpc.ErrInternal
	}

	if len(nw.Containers) > 0 {
		return status.Error(codes.FailedPrecondition, "容器正在使用该网络，建议断开容器网络后再删除/更新")
	}

	return nil
}

func (s *NetworkServer) checkCreateNetwork(name, subnet, gateway, parent string) error {
	if name == "" {
		return rpc.ErrInvalidArgument
	}

	_, _, err := net.ParseCIDR(subnet)
	if err != nil {
		return status.Errorf(codes.InvalidArgument, "网段参数错误")
	}

	// ip := net.ParseIP(gateway)
	// if ip == nil || r.Contains(ip) {
	// 	return status.Errorf(codes.InvalidArgument, "网关参数错误")
	// }

	if parent != "" {
		_, err = net.InterfaceByName(parent)
		if err != nil {
			log.Warnf("InterfaceByName(%v) err=%v", parent, err)
			return rpc.ErrInternal
		}

		isVirtual, err := isVirtualNic(parent)
		if err != nil {
			return rpc.ErrInternal
		}
		if isVirtual {
			log.Infof("cannot use virtual nic %v", parent)
			return rpc.ErrInvalidArgument
		}
	}

	return nil
}

func (s *NetworkServer) CreateNic(ctx context.Context, in *pb.CreateNicRequest) (*pb.CreateNicReply, error) {
	if err := s.checkCreateNetwork(in.Name, in.Subnet, in.Gateway, in.Parent); err != nil {
		return nil, err
	}

	if err := s.createNetwork(in.Name, in.Subnet, in.Gateway, in.Parent); err != nil {
		return nil, err
	}

	return &pb.CreateNicReply{}, nil
}

func (s *NetworkServer) RemoveNic(ctx context.Context, in *pb.RemoveNicRequest) (*pb.RemoveNicReply, error) {
	if err := s.checkRemoveNetwork(in.Name); err != nil {
		return nil, err
	}

	if err := s.removeNetwork(in.Name); err != nil {
		return nil, err
	}

	return &pb.RemoveNicReply{}, nil
}

func (s *NetworkServer) UpdateNic(ctx context.Context, in *pb.UpdateNicRequest) (*pb.UpdateNicReply, error) {
	if err := s.checkRemoveNetwork(in.Name); err != nil {
		return nil, err
	}

	if err := s.checkCreateNetwork(in.Name, in.Subnet, in.Gateway, in.Parent); err != nil {
		return nil, err
	}

	if err := s.removeNetwork(in.Name); err != nil {
		return nil, err
	}

	if err := s.createNetwork(in.Name, in.Subnet, in.Gateway, in.Parent); err != nil {
		return nil, err
	}

	return &pb.UpdateNicReply{}, nil
}
