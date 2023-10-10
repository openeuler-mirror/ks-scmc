package internal

import (
	"context"
	"net"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/network"
	log "github.com/sirupsen/logrus"

	"scmc/model"
	"scmc/rpc"
	pb "scmc/rpc/pb/network"
)

const defaultContainerChain = "INPUT"
const defaultNodeChain = "OUTPUT"
const defaultPolicy = "ACCEPT"

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

func (s *NetworkServer) List(ctx context.Context, in *pb.ListRequest) (*pb.ListReply, error) {
	reply := pb.ListReply{}

	cli, err := model.DockerClient()
	if err != nil {
		return nil, rpc.ErrInternal
	}

	list, err := cli.NetworkList(context.Background(), types.NetworkListOptions{})
	if err != nil {
		log.Warnf("NetworkList: %v", err)
		return nil, transDockerError(err)
	}

	for _, i := range list {
		//不显示默认网卡
		if i.Name == "bridge" || i.Name == "host" || i.Name == "none" {
			continue
		}
		if len(i.IPAM.Config) != 1 {
			log.Warnf("bridge network[%s] unexpect config %v", i.Name, i.IPAM.Config)
			continue
		}

		conf := i.IPAM.Config[0]
		addr, masklen, err := parseSubnet(conf.Subnet)
		if err != nil {
			log.Warnf("bridge network[%s] parse subnet=%v return err=%v", i.Name, conf.Subnet, err)
		}

		inspect, err := cli.NetworkInspect(context.Background(), i.Name, types.NetworkInspectOptions{})
		if err != nil {
			log.Warnf("NetworkInspect: %v", err)
		}

		var linkIfs string
		if inspect.Options != nil {
			linkIfs = inspect.Options["parent"]
		}

		var containers []*pb.ContainerNetwork
		for containerId, cntrInfo := range inspect.Containers {
			caddr, cmasklen, _ := parseSubnet(cntrInfo.IPv4Address)
			containerInfo := pb.ContainerNetwork{
				ContainerId: containerId,
				IpAddress:   caddr,
				IpMaskLen:   int32(cmasklen),
				MacAddress:  cntrInfo.MacAddress,
			}
			containers = append(containers, &containerInfo)
		}

		netInfo := pb.NetworkInterface{
			Name:       i.Name,
			BindReal:   linkIfs,
			IpAddress:  addr,
			IpMaskLen:  int32(masklen),
			Gateway:    conf.Gateway,
			IsUp:       true,
			Containers: containers,
		}

		reply.VirtualIfs = append(reply.VirtualIfs, &netInfo)
	}

	return &reply, nil
}

func (s *NetworkServer) Connect(ctx context.Context, in *pb.ConnectRequest) (*pb.ConnectReply, error) {
	reply := pb.ConnectReply{}

	if in.Interface == "" || in.ContainerId == "" || in.IpAddress == "" {
		return nil, rpc.ErrInvalidArgument
	}

	cli, err := model.DockerClient()
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

	return &reply, err
}

func (s *NetworkServer) Disconnect(ctx context.Context, in *pb.DisconnectRequest) (*pb.DisconnectReply, error) {
	reply := pb.DisconnectReply{}

	if in.Interface == "" || in.ContainerId == "" {
		return nil, rpc.ErrInvalidArgument
	}

	cli, err := model.DockerClient()
	if err != nil {
		return nil, rpc.ErrInternal
	}

	if err = cli.NetworkDisconnect(context.Background(), in.Interface, in.ContainerId, true); err != nil {
		log.Warnf("NetworkDisconnect: %v", err)
		return nil, transDockerError(err)
	}

	return &reply, err
}

func (s *NetworkServer) ListIPtables(ctx context.Context, in *pb.ListIPtablesRequest) (*pb.ListIPtablesReply, error) {
	reply := pb.ListIPtablesReply{}

	who := model.OperateNode
	pid := 0
	if in.ContainerId != "" {
		cli, err := model.DockerClient()
		if err != nil {
			return nil, rpc.ErrInternal
		}

		info, err := cli.ContainerInspect(context.Background(), in.ContainerId)
		if err != nil || info.State.Pid == 0 {
			log.Warnf("ContainerInspect err(%v) or container not start", err)
			return nil, transDockerError(err)
		}
		who = model.OperateContainer
		pid = info.State.Pid
	}

	list, err := model.ListRules(who, pid)
	if err != nil {
		return nil, rpc.ErrInternal
	}

	for _, chains := range list {
		var chainRule pb.ChianRule
		chainRule.Chain = chains.Chain
		for _, rule := range chains.Rules {
			info := &pb.RuleInfo{
				Source:       rule.Source,
				Destination:  rule.Destination,
				Protocol:     rule.Protocol,
				SrcPort:      rule.SrcPort,
				DestPort:     rule.DestPort,
				InInterface:  rule.InInterface,
				OutInterface: rule.OutInterface,
				Policy:       rule.Policy,
			}

			chainRule.Rule = append(chainRule.Rule, info)
		}

		reply.ChainRules = append(reply.ChainRules, &chainRule)
	}
	return &reply, nil

}

func (s *NetworkServer) EnableIPtables(ctx context.Context, in *pb.EnableIPtablesRequest) (*pb.EnableIPtablesReply, error) {
	reply := pb.EnableIPtablesReply{}

	if in.ContainerId != "" {
		cli, err := model.DockerClient()
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
			model.EnableContainerIPtables(fileName, info.State.Pid)
		} else {
			model.DisableContainerIPtables(fileName, info.State.Pid)
		}
	} else {
		if in.Enable {
			linkifs := getLinkIfs()
			for _, ifs := range linkifs {
				model.EnableNodeIPtables(ifs)
			}
		} else {
			model.DisableNodeIPtables()
		}
	}

	return &reply, nil

}

func (s *NetworkServer) CreateIPtables(ctx context.Context, in *pb.CreateIPtablesRequest) (*pb.CreateIPtablesReply, error) {
	reply := pb.CreateIPtablesReply{}
	if in.Rule == nil {
		return nil, rpc.ErrInvalidArgument
	}

	who := model.OperateNode
	pid := 0
	chain := defaultNodeChain

	if in.ContainerId != "" {
		cli, err := model.DockerClient()
		if err != nil {
			return nil, rpc.ErrInternal
		}

		info, err := cli.ContainerInspect(context.Background(), in.ContainerId)
		if err != nil || info.State.Pid == 0 {
			log.Warnf("ContainerInspect err(%v) or container not start", err)
			return nil, transDockerError(err)
		}
		who = model.OperateContainer
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
	rule := model.RuleInfo{
		Source:       in.Rule.Source,
		Destination:  in.Rule.Destination,
		Protocol:     in.Rule.Protocol,
		SrcPort:      in.Rule.SrcPort,
		DestPort:     in.Rule.DestPort,
		InInterface:  in.Rule.InInterface,
		OutInterface: in.Rule.OutInterface,
		Policy:       policy,
	}

	if err := model.AddRule(who, in.ContainerId, pid, chain, rule); err != nil {
		log.Warnf("AddRule: %v", err)
		return nil, rpc.ErrInternal
	}

	return &reply, nil
}

func (s *NetworkServer) ModifyIPtables(ctx context.Context, in *pb.ModifyIPtablesRequest) (*pb.ModifyIPtablesReply, error) {
	reply := pb.ModifyIPtablesReply{}
	if in.OldRule == nil || in.NewRule == nil {
		return nil, rpc.ErrInvalidArgument
	}

	who := model.OperateNode
	pid := 0
	oldChain := defaultNodeChain
	newChain := defaultNodeChain

	if in.ContainerId != "" {
		cli, err := model.DockerClient()
		if err != nil {
			return nil, rpc.ErrInternal
		}

		info, err := cli.ContainerInspect(context.Background(), in.ContainerId)
		if err != nil || info.State.Pid == 0 {
			log.Warnf("ContainerInspect err(%v) or container not start", err)
			return nil, transDockerError(err)
		}
		who = model.OperateContainer
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
	oldrule := model.RuleInfo{
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
	newrule := model.RuleInfo{
		Source:       in.NewRule.Source,
		Destination:  in.NewRule.Destination,
		Protocol:     in.NewRule.Protocol,
		SrcPort:      in.NewRule.SrcPort,
		DestPort:     in.NewRule.DestPort,
		InInterface:  in.NewRule.InInterface,
		OutInterface: in.NewRule.OutInterface,
		Policy:       newPolicy,
	}

	if err := model.ModifyRule(who, in.ContainerId, pid, oldChain, oldrule, newChain, newrule); err != nil {
		log.Warnf("ModifyContainerRule: %v", err)
		return nil, rpc.ErrInternal
	}

	return &reply, nil
}

func (s *NetworkServer) RemoveIPtables(ctx context.Context, in *pb.RemoveIPtablesRequest) (*pb.RemoveIPtablesReply, error) {
	reply := pb.RemoveIPtablesReply{}
	if in.Rule == nil {
		return nil, rpc.ErrInvalidArgument
	}
	who := model.OperateNode
	pid := 0
	chain := defaultNodeChain

	if in.ContainerId != "" {
		cli, err := model.DockerClient()
		if err != nil {
			return nil, rpc.ErrInternal
		}

		info, err := cli.ContainerInspect(context.Background(), in.ContainerId)
		if err != nil || info.State.Pid == 0 {
			log.Warnf("ContainerInspect err(%v) or container not start", err)
			return nil, transDockerError(err)
		}
		who = model.OperateContainer
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
	rule := model.RuleInfo{
		Source:       in.Rule.Source,
		Destination:  in.Rule.Destination,
		Protocol:     in.Rule.Protocol,
		SrcPort:      in.Rule.SrcPort,
		DestPort:     in.Rule.DestPort,
		InInterface:  in.Rule.InInterface,
		OutInterface: in.Rule.OutInterface,
		Policy:       policy,
	}

	if err := model.DelRule(who, in.ContainerId, pid, chain, rule); err != nil {
		log.Warnf("DelRule: %v", err)
		return nil, rpc.ErrInternal
	}

	return &reply, nil
}
