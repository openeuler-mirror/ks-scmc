package internal

import (
	"context"
	"net"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/network"
	log "github.com/sirupsen/logrus"

	"scmc/model"
	"scmc/rpc"
	pb "scmc/rpc/pb/network"
)

type NetworkServer struct {
	pb.UnimplementedNetworkServer
}

func parseSubnet(s string) (string, int, error) {
	if len(s) == 0 {
		return "", 0, nil
	}

	_, info, err := net.ParseCIDR(s)
	if err != nil {
		return "", 0, err
	}

	n, _ := info.Mask.Size()
	return info.IP.String(), int(n), nil
}

func (s *NetworkServer) List(ctx context.Context, in *pb.ListRequest) (*pb.ListReply, error) {
	reply := pb.ListReply{}

	cli, err := dockerCli()
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

	cli, err := dockerCli()
	if err != nil {
		return nil, rpc.ErrInternal
	}

	masklen := model.TransMask(in.IpMask)
	var config *network.EndpointSettings
	config = &network.EndpointSettings{
		NetworkID:   in.Interface,
		Gateway:     in.Gateway,
		IPAddress:   in.IpAddress,
		IPPrefixLen: masklen,
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

	cli, err := dockerCli()
	if err != nil {
		return nil, rpc.ErrInternal
	}

	if err = cli.NetworkDisconnect(context.Background(), in.Interface, in.ContainerId, true); err != nil {
		log.Warnf("NetworkDisconnect: %v", err)
		return nil, transDockerError(err)
	}

	return &reply, err
}
