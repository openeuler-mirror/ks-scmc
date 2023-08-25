package internal

import (
	"context"
	"fmt"
	"scmc/model"
	"strconv"
	"strings"

	"github.com/docker/docker/api/types"
	log "github.com/sirupsen/logrus"
)

const startIPVal = 3
const endIPVal = 255 //不包含

func getSubnet(ifs string) (string, int, error) {
	var addr string
	var masklen int
	cli, err := dockerCli()
	if err != nil {
		return addr, masklen, err
	}

	list, err := cli.NetworkList(context.Background(), types.NetworkListOptions{})
	if err != nil {
		log.Warnf("NetworkList: %v", err)
		return addr, masklen, transDockerError(err)
	}

	for _, i := range list {
		if i.Name == ifs {
			if len(i.IPAM.Config) != 1 {
				log.Warnf("network[%s] unexpect config %v", i.Name, i.IPAM.Config)
				continue
			}

			conf := i.IPAM.Config[0]
			addr, masklen, err = parseSubnet(conf.Subnet)
			if err != nil {
				log.Warnf("network[%s] parse subnet=%v return err=%v", i.Name, conf.Subnet, err)
			}
		}
	}

	return addr, masklen, nil
}

func getVirNicInfo(ifs string) model.VirNicInfo {
	netinfo, _ := model.ReadJSON()
	if netinfo != nil {
		if netinfo.NodeNicIPAddr == nil {
			model.GetNodeIP()
		}
		if _, ok := netinfo.VirtualNic[ifs]; ok {
			return *netinfo.VirtualNic[ifs]
		}
	}

	addr, masklen, _ := getSubnet(ifs)
	ips := model.GetNodeIP()
	nextip := addr
	if addr != "" {
		splitaddr := strings.Split(addr, ".")
		for n := startIPVal; n < endIPVal; n++ {
			nextip = fmt.Sprintf("%s.%s.%s.%d", splitaddr[0], splitaddr[1], splitaddr[2], n)
			if _, ok := ips[nextip]; !ok {
				break
			}
		}
	}

	info := model.VirNicInfo{
		IPAddress:      addr,
		IPMaskLen:      masklen,
		MinAvailableIP: nextip,
	}

	log.Debugf("info [%+v][%v][%v][%v]", ips, addr, masklen, nextip)
	return info
}

func assignIP(ifs string, info model.VirNicInfo) (string, string) {
	var contrip, nextip string
	if info.IPAddress == "" {
		return contrip, nextip
	}

	var splitaddr []string

	n := startIPVal
	if info.MinAvailableIP == "" {
		splitaddr = strings.Split(info.IPAddress, ".")
	} else {
		splitaddr = strings.Split(info.MinAvailableIP, ".")
		if val, err := strconv.Atoi(splitaddr[3]); err == nil {
			if val < endIPVal {
				n = val
			}
		}
	}

	contrip = info.MinAvailableIP
	nextip = fmt.Sprintf("%s.%s.%s.%d", splitaddr[0], splitaddr[1], splitaddr[2], n+1)
	netinfo, _ := model.ReadJSON()
	if netinfo == nil {
		log.Debugf("netinfo is nil, info [%v][%v][%v]", info.IPAddress, info.MinAvailableIP, nextip)
		return contrip, nextip
	}

	ips := make(map[string]struct{})
	for _, v := range netinfo.Containers {
		if _, ok := v.ContainerNetworks[ifs]; ok {
			ips[v.ContainerNetworks[ifs].IpAddress] = struct{}{}
		}
	}

	contrip = fmt.Sprintf("%s.%s.%s.%d", splitaddr[0], splitaddr[1], splitaddr[2], n)
	for i := 0; i < len(ips); i++ {
		if _, ok := ips[contrip]; !ok {
			if _, ok := netinfo.NodeNicIPAddr[contrip]; !ok {
				log.Debugf("info:[%v][%v][%+v], find: [%v][%v]", info.IPAddress, info.MinAvailableIP, ips, contrip, nextip)
				return contrip, nextip
			}
		}

		contrip = nextip
		n = n + 1
		nextip = fmt.Sprintf("%s.%s.%s.%d", splitaddr[0], splitaddr[1], splitaddr[2], n+1)

	}

	log.Debugf("end:[%v][%v][%v]", n, contrip, nextip)
	if n >= endIPVal {
		return "", ""
	}
	return contrip, nextip
}
