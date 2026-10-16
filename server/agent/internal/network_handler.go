package internal

import (
	"context"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"

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

func (s *NetworkServer) List(ctx context.Context, in *pb.ListRequest) (*pb.ListReply, error) {
	var reply pb.ListReply

	cli, err := docker.Client()
	if err != nil {
		return nil, rpc.ErrInternal
	}

	ns, err := cli.NetworkList(context.Background(), types.NetworkListOptions{})
	if err != nil {
		log.Warnf("NetworkList: %v", err)
		return nil, transDockerError(err)
	}

	for _, n := range ns {
		//不显示默认网卡
		if n.Name == "bridge" || n.Name == "host" || n.Name == "none" {
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

		detail, err := cli.NetworkInspect(context.Background(), n.Name, types.NetworkInspectOptions{})
		if err != nil {
			log.Warnf("NetworkInspect: %v", err)
		}

		var linkIfs string
		if detail.Options != nil {
			linkIfs = detail.Options["parent"]
		}

		/*
			var containers []*pb.ContainerNetwork
			for cid, c := range detail.Containers {
				caddr, cmasklen, _ := parseSubnet(c.IPv4Address)
				containers = append(containers, &pb.ContainerNetwork{
					ContainerId: cid,
					IpAddress:   caddr,
					IpMaskLen:   int32(cmasklen),
					MacAddress:  c.MacAddress,
				})
			}
		*/

		reply.VirtualIfs = append(reply.VirtualIfs, &pb.NetworkInterface{
			Name:      n.Name,
			BindReal:  linkIfs,
			IpAddress: addr,
			IpMaskLen: int32(masklen),
			Gateway:   conf.Gateway,
			IsUp:      true,
			// Containers: containers,
		})
	}

	ifs, err := net.Interfaces()
	if err != nil {
		log.Warnf("net.Interface err=%v", err)
		return nil, rpc.ErrInternal
	}

	for _, i := range ifs {
		log.Debugf("Network interface %+v", i)
		virtual, err := isVirtualNic(i.Name)
		if virtual || err != nil {
			continue
		}

		realIf := &pb.NetworkInterface{
			Name:       i.Name,
			IsUp:       strings.Contains(i.Flags.String(), "up"),
			MacAddress: i.HardwareAddr.String(),
		}

		addrs, err := i.Addrs()
		if err != nil {
			log.Warnf("retrieve addrs from interface %v err=%v", i, err)
		}

		if len(addrs) > 0 {
			addr := addrs[0]
			ip, ipNet, err := net.ParseCIDR(addr.String())
			if err != nil {
				log.Warnf("net.ParseCIDR(%s) err=%v", addr.String(), err)
				continue
			}

			if ipv4 := ip.To4(); ipv4 != nil {
				ones, _ := ipNet.Mask.Size()
				realIf.IpAddress = ipv4.String()
				realIf.IpMaskLen = int32(ones)

				var s []string
				for _, i := range ipNet.Mask {
					s = append(s, strconv.Itoa(int(i)))
				}
				realIf.IpMask = strings.Join(s, ".")
			}

			// retrive gateway
			link, err := netlink.LinkByName(i.Name)
			if err != nil {
				log.Infof("netlink.LinkByName(%s) err=%v", i.Name, err)
				continue
			}

			routes, err := netlink.RouteList(link, netlink.FAMILY_V4)
			if err != nil {
				log.Infof("link=%s netlink.RouteList %v", i.Name, err)
				continue
			}

			for _, r := range routes {
				if r.Gw != nil {
					realIf.Gateway = r.Gw.String()
				}
			}
			if realIf.Gateway == "" {
				realIf.Gateway = "0.0.0.0"
			}
		}
		reply.RealIfs = append(reply.RealIfs, realIf)
	}
	return &reply, nil
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

func (s *NetworkServer) createNetwork(name, subnet, gateway, parent string) error {
	driver := "bridge"
	if parent != "" {
		driver = "macvlan"
	}

	cli, err := docker.Client()
	if err != nil {
		return rpc.ErrInternal
	}

	data := types.NetworkCreate{
		Driver: driver,
		IPAM: &network.IPAM{
			Config: []network.IPAMConfig{
				{
					Subnet:  subnet,
					Gateway: gateway,
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
	return nil
}

func (s *NetworkServer) removeNetwork(name string) error {
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

	_, err = cli.NetworkInspect(context.Background(), name, types.NetworkInspectOptions{})
	if err != nil {
		log.Warnf("NetworkInspect err=%v", err)
		if client.IsErrNotFound(err) {
			return rpc.ErrNotFound
		}
		return rpc.ErrInternal
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
