package internal

import (
	"context"
	"fmt"
	"strings"

	"ksc-mcube/rpc"
	pb "ksc-mcube/rpc/pb/network"

	"github.com/docker/docker/api/types"
	log "github.com/sirupsen/logrus"
)

type NetworkServer struct {
	pb.UnimplementedNetworkServer
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

		var (
			conf    = i.IPAM.Config[0]
			subnet  pb.Subnet
			ipRange *pb.Subnet
		)

		n, err := fmt.Sscanf(conf.Subnet, "%s/%d", &subnet.Addr, &subnet.PrefixLen)
		if err != nil || n != 2 {
			log.Warnf("bridge network[%s] parse subnet=%v return n=%v err=%v", i.Name, conf.Subnet, n, err)
		}

		if len(conf.IPRange) > 0 {
			ipRange = &pb.Subnet{}
			n, err = fmt.Sscanf(conf.IPRange, "%s/%d", &ipRange.Addr, &ipRange.PrefixLen)
			if err != nil || n != 2 {
				log.Warnf("bridge network[%s] parse ip_range=%v return n=%v err=%v", i.Name, conf.IPRange, n, err)
			}
		}

		reply.BridgeIf = append(reply.BridgeIf, &pb.BridgeNetworkInterface{
			Name:    i.Name,
			Subnet:  &subnet,
			IpRange: ipRange,
			Gateway: conf.Gateway,
		})
	}
	return &reply, nil
}
