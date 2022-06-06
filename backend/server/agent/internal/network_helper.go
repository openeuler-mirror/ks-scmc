package internal

import (
	"context"
	"fmt"
	"net"
	"os"
	"scmc/common"
	"scmc/model"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/events"
	log "github.com/sirupsen/logrus"
)

func checkIfs(ifs string) bool {
	cli, err := model.DockerClient()
	if err != nil {
		return false
	}

	list, err := cli.NetworkList(context.Background(), types.NetworkListOptions{})
	if err != nil {
		log.Warnf("NetworkList: %v", err)
		return false
	}

	for _, i := range list {
		//不显示默认网卡
		if i.Name == "bridge" || i.Name == "host" || i.Name == "none" {
			continue
		}
		if ifs == i.Name {
			return false
		}
	}

	log.Errorf("%v not in docker nic", ifs)
	return true
}

func checkConflict(ifs, ipAddr string, masklen int, containerId string) bool {
	cli, err := model.DockerClient()
	if err != nil {
		return false
	}

	inspect, err := cli.NetworkInspect(context.Background(), ifs, types.NetworkInspectOptions{})
	if err != nil {
		log.Warnf("NetworkInspect: %v", err)
		return false
	}

	if inspect.IPAM.Config != nil {
		if len(inspect.IPAM.Config) >= 1 {
			conf := inspect.IPAM.Config[0]
			_, subnet, _ := net.ParseCIDR(conf.Subnet)
			if masklen == 0 {
				masklen, _ = subnet.Mask.Size()
			}
			ipAddr := fmt.Sprintf("%s/%d", ipAddr, masklen)
			ip, _, _ := net.ParseCIDR(ipAddr)
			if !subnet.Contains(ip) {
				log.Warnf("%v(%v) is not within the range of %v(%v)", conf.Subnet, subnet, ipAddr, ip)
				return false
			}
		}
	}

	opts := types.ContainerListOptions{All: true, Size: true}
	containers, err := cli.ContainerList(context.Background(), opts)
	if err != nil {
		log.Warnf("ContainerList: %v", err)
		return true
	}

	containerIPs := make(map[string]struct{})
	for _, c := range containers {
		if containerId == c.ID {
			continue
		}

		for _, n := range c.NetworkSettings.Networks {
			if n.IPAMConfig != nil && n.IPAMConfig.IPv4Address != "" {
				containerIPs[n.IPAMConfig.IPv4Address] = struct{}{}
			} else if n.IPAddress != "" {
				containerIPs[n.IPAddress] = struct{}{}
			}
		}
	}

	if _, ok := containerIPs[ipAddr]; ok {
		log.Warnf("ipAddr:%v is used", ipAddr)
		return false
	}

	return true
}

func ContainerWhiteCongig() error {
	cli, err := model.DockerClient()
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
					model.ContainerWhitelistInitialization(fileName, info.State.Pid)
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
	cli, err := model.DockerClient()
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
		if i.Name == "bridge" {
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
		model.NodeWhitelistInitialization(ifs)
	}

	return nil
}
