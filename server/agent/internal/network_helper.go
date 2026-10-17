package internal

import (
	"context"
	"encoding/binary"
	"fmt"
	"net"
	"os"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"scmc/common"
	"scmc/model/docker"
	"scmc/model/iptables"
	pb "scmc/rpc/pb/container"
)

func ipv4ToUint(ipaddr net.IP) uint32 {
	if ipaddr.To4() == nil {
		return 0
	}
	return binary.BigEndian.Uint32(ipaddr.To4())
}

func ipStrToUint(ipaddr string) uint32 {
	ip := net.ParseIP(ipaddr)
	if ip == nil {
		return 0
	}

	return ipv4ToUint(ip)
}

func uintToIPv4(ipaddr uint32) net.IP {
	ip := make(net.IP, net.IPv4len)

	// Proceed conversion
	binary.BigEndian.PutUint32(ip, ipaddr)

	return ip
}

func isPredefinedNetwork(name string) bool {
	return name == "bridge" || name == "host" || name == "none"
}

func getContainerIPs(skipContainerID, matchIface string) (map[uint32]bool, error) {
	cli, err := docker.Client()
	if err != nil {
		return nil, err
	}

	opts := types.ContainerListOptions{All: true, Size: true}
	containers, err := cli.ContainerList(context.Background(), opts)
	if err != nil {
		log.Warnf("ContainerList: %v", err)
		return nil, err
	}

	containerIPs := make(map[uint32]bool)
	for _, c := range containers {
		if skipContainerID == c.ID {
			continue // (self) container id to skip
		}

		for k, n := range c.NetworkSettings.Networks {
			if k != matchIface {
				continue
			}

			var ipStr string
			if n.IPAMConfig != nil && n.IPAMConfig.IPv4Address != "" {
				ipStr = n.IPAMConfig.IPv4Address
			} else if n.IPAddress != "" {
				ipStr = n.IPAddress
			}
			containerIPs[ipStrToUint(ipStr)] = true
		}
	}
	return containerIPs, nil
}

func checkIPConflict(iface types.NetworkResource, containerId string, ipAddr *string, masklen int) bool {
	if iface.IPAM.Config == nil || len(iface.IPAM.Config) < 1 {
		return false
	}

	containerIPs, err := getContainerIPs(containerId, iface.Name)
	if err != nil {
		return false
	}

	_, subnet, _ := net.ParseCIDR(iface.IPAM.Config[0].Subnet)
	if *ipAddr != "" { // 处理用户输入的IP
		if masklen == 0 {
			masklen, _ = subnet.Mask.Size()
		}
		ipNet, _, _ := net.ParseCIDR(fmt.Sprintf("%s/%d", *ipAddr, masklen))
		if !subnet.Contains(ipNet) {
			log.Warnf("ipnet=%v not int subnet=%v", ipNet, subnet)
			return false // ip不在子网段内
		}

		//判断IP是否已使用
		ip := ipStrToUint(*ipAddr)
		if _, ok := containerIPs[ip]; ok {
			log.Warnf("ip=%v is used by other container", *ipAddr)
			return false // ip已被其他容器使用
		}
		return true
	}

	// 没有输入IP，分配IP
	for i := ipv4ToUint(subnet.IP) + uint32(2); ; i++ {
		if _, ok := containerIPs[i]; ok {
			continue
		}

		ip := uintToIPv4(i)
		if !subnet.Contains(ip) {
			break
		}
		*ipAddr = ip.String()
		return true
	}

	log.Warnf("subnet=%v has no free ips", subnet)
	return false
}

func checkNetworkInfo(ifaceName, containerId string, ipAddr *string, masklen int) error {
	cli, err := docker.Client()
	if err != nil {
		return err
	}

	// 检查网卡是否存在
	if isPredefinedNetwork(ifaceName) {
		log.Errorf("Unsupported interface=%v", ifaceName)
		return status.Error(codes.Internal, "网卡未找到")
	}

	iface, err := cli.NetworkInspect(context.Background(), ifaceName, types.NetworkInspectOptions{})
	if err != nil {
		log.Errorf("NetworkInspect networkID=%v err=%v", iface, err)
		return status.Error(codes.Internal, "网卡未找到")
	}

	if *ipAddr == "" && iface.Driver == "bridge" {
		// bridge 网卡通过docker自动分配IP
		return nil
	}

	if !checkIPConflict(iface, containerId, ipAddr, masklen) {
		return status.Error(codes.InvalidArgument, "IP冲突")
	}

	return nil
}

func ContainerWhiteCongig() error {
	cli, err := docker.Client()
	if err != nil {
		return err
	}

	options := types.EventsOptions{}
	eventq, errq := cli.Events(context.Background(), options)
	if err != nil {
		log.Warnf("Events: %v", err)
		return err
	}

	eventProcessor := func(e events.Message) {
		switch e.Status {
		case "start":
			{
				info, err := cli.ContainerInspect(context.Background(), e.ID)
				if err != nil || info.State.Pid == 0 {
					log.Warnf("%v call ContainerInspect err(%v) or container not start", e.ID, err)
				} else {
					var name string
					if e.Actor.Attributes != nil {
						name = e.Actor.Attributes["name"]
					}
					log.Infof("%v-%v: pid(%v)", e.ID, name, info.State.Pid)
					fileName := e.ID + "-" + name
					iptables.ContainerWhitelistInitialization(fileName, info.State.Pid)
				}
			}
			/*
				case "destroy":
					{
						var name string
						if e.Actor.Attributes != nil {
							name = e.Actor.Attributes["name"]
						}
						fileName := e.ID + "-" + name
						model.ContainerRemveIPtables(fileName)
					}
			*/
		}
	}

	go func() {
		for {
			select {
			case evt := <-eventq:
				eventProcessor(evt)
			case err := <-errq:
				log.Errorf("error getting events from daemon: %v", err)
				return
			}
		}
	}()

	return nil
}

func getLinkIfs() []string {
	var linkIfs []string
	cli, err := docker.Client()
	if err != nil {
		return linkIfs
	}

	list, err := cli.NetworkList(context.Background(), types.NetworkListOptions{})
	if err != nil {
		log.Warnf("NetworkList: %v", err)
		return linkIfs
	}

	for _, i := range list {
		//因网络访问控制权限是影响指定网卡的对应网段，未防止主机不能通外网，因此只处理桥接网络，不处理macvlan网络
		if i.Driver != "bridge" {
			continue
		}

		//默认网卡
		if isPredefinedNetwork(i.Name) {
			continue
		}

		if len(i.IPAM.Config) != 1 {
			log.Warnf("bridge network[%s] unexpect config %v", i.Name, i.IPAM.Config)
			continue
		}

		inspect, err := cli.NetworkInspect(context.Background(), i.Name, types.NetworkInspectOptions{})
		if err != nil {
			log.Warnf("NetworkInspect: %v", err)
		}

		if inspect.Options == nil {
			continue
		}

		ifs := inspect.Options["com.docker.network.bridge.name"]
		if ifs == "" {
			continue
		}

		linkIfs = append(linkIfs, ifs)
	}

	return linkIfs
}

func NodeWhitelistConfig() error {
	IPtablesPath := common.Config.Network.IPtablesPath
	_, err := os.Stat(IPtablesPath)
	if err != nil {
		if os.IsNotExist(err) {
			err := os.MkdirAll(IPtablesPath, 0666)
			if err != nil {
				log.Warnf("mkdir %v err: %v", IPtablesPath, err)
			}
		}
	}

	linkifs := getLinkIfs()
	for _, ifs := range linkifs {
		iptables.NodeWhitelistInit(ifs)
	}

	return nil
}

func dockerNetworkDisconnect(ctx context.Context, cli *client.Client, containerID, iface string) error {
	err := cli.NetworkDisconnect(ctx, iface, containerID, true)
	if err != nil {
		log.Warnf("NetworkDisconnect: %v", err)
	}
	return err
}

func dockerNetworkConnect(ctx context.Context, cli *client.Client, containerID string, n *pb.NetworkConfig) error {
	config := &network.EndpointSettings{
		NetworkID:   n.Interface,
		Gateway:     n.Gateway,
		IPAddress:   n.IpAddress,
		IPPrefixLen: int(n.IpPrefixLen),
		MacAddress:  n.MacAddress,
		IPAMConfig: &network.EndpointIPAMConfig{
			IPv4Address: n.IpAddress,
		},
	}

	err := cli.NetworkConnect(ctx, n.Interface, containerID, config)
	if err != nil {
		log.Warnf("NetworkConnect: %v", err)
		return transDockerError(err)
	}
	return nil
}
