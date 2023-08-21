package internal

import (
	"context"
	"net"
	"strings"

	"github.com/docker/docker/api/types"
	log "github.com/sirupsen/logrus"

	"ksc-mcube/rpc"
	pb "ksc-mcube/rpc/pb/network"
)

type NetworkServer struct {
	pb.UnimplementedNetworkServer
}

func parseSubnet(s string) (*pb.Subnet, error) {
	if len(s) == 0 {
		return nil, nil
	}

	_, info, err := net.ParseCIDR(s)
	if err != nil {
		return nil, err
	}

	n, _ := info.Mask.Size()
	return &pb.Subnet{
		Addr:      info.IP.String(),
		PrefixLen: int32(n),
	}, nil
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
		if strings.ToLower(i.Driver) != "bridge" {
			continue
		}
		if len(i.IPAM.Config) != 1 {
			log.Warnf("bridge network[%s] unexpect config %v", i.Name, i.IPAM.Config)
			continue
		}

		conf := i.IPAM.Config[0]

		subnet, err := parseSubnet(conf.Subnet)
		if err != nil {
			log.Warnf("bridge network[%s] parse subnet=%v return err=%v", i.Name, conf.Subnet, err)
		}

		ipRange, err := parseSubnet(conf.IPRange)
		if err != nil {
			log.Warnf("bridge network[%s] parse ip_range=%v return err=%v", i.Name, conf.IPRange, err)
		}

		reply.BridgeIf = append(reply.BridgeIf, &pb.BridgeNetworkInterface{
			Name:    i.Name,
			Subnet:  subnet,
			IpRange: ipRange,
			Gateway: conf.Gateway,
		})
	}
	return &reply, nil
}
