// network info
package model

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/vishvananda/netlink"
	"github.com/vishvananda/netlink/nl"

	"scmc/common"
)

const maxJsonFileSize = 1048576 //1M

type VirNicInfo struct {
	IPAddress      string
	IPMaskLen      int
	MinAvailableIP string
}

type ContainerNetwork struct {
	IpAddress string
	MaskLen   int
	ForShell  string
}

type ContainerNic struct {
	ContainerNetworks map[string]*ContainerNetwork //[LinkIfs][ContainerNetwork]
}

type NetworkInfo struct {
	NodeNicIPAddr map[string]struct{}
	VirtualNic    map[string]*VirNicInfo   //[LinkIfs][VirNicInfo]
	Containers    map[string]*ContainerNic //[ContainerId][ContainerNic]
}

func ReadJSON() (*NetworkInfo, error) {
	JsonFile := common.Config.Network.JsonFile
	log.Debugf("read file [%v]", JsonFile)
	file, err := os.Open(JsonFile)
	if err != nil {
		log.Errorf("open file err: %v", err)
		return nil, err
	}
	defer file.Close()
	fileInfo, err := file.Stat()
	fileSize := fileInfo.Size()
	if fileSize > maxJsonFileSize {
		log.Errorf("file %v size(%v) too large", JsonFile, fileSize)
		return nil, errors.New("file size too large")
	}

	reader := bufio.NewReader(file)
	buffer := make([]byte, fileSize)
	n, err := reader.Read(buffer)
	if err != nil {
		log.Errorf("read file %v err: %v", JsonFile, err)
		return nil, err
	}
	log.Debugf("read file %v size: %v", JsonFile, n)

	network := &NetworkInfo{}
	err = json.Unmarshal(buffer, network)
	if err != nil {
		log.Errorf("Unmarshal failed: %v", err)
		return nil, err
	}

	return network, nil
}

func writeJSON(network *NetworkInfo) error {
	JsonFile := common.Config.Network.JsonFile
	log.Debugf("write to file [%v]", JsonFile)
	data, err := json.MarshalIndent(network, "", "\t")
	if err != nil {
		log.Errorf("json err: %v", err)
		return err
	}

	file, err := os.Create(JsonFile)
	if err != nil {
		log.Errorf("cannot create file: %v", JsonFile)
		return err
	}
	defer file.Close()

	_, err = file.Write(data)
	if err != nil {
		log.Errorf("cannot write json data to file: %v", err)
		return err
	}
	return nil
}

func AddJSON(containerId string, info *NetworkInfo) error {
	netinfo, _ := ReadJSON()
	if netinfo == nil {
		return writeJSON(info)
	}
	if netinfo.VirtualNic == nil {
		netinfo.VirtualNic = make(map[string]*VirNicInfo)
	}
	if netinfo.Containers == nil {
		netinfo.Containers = make(map[string]*ContainerNic)
	}
	for k, v := range info.VirtualNic {
		netinfo.VirtualNic[k] = v
	}

	netinfo.Containers[containerId] = info.Containers[containerId]
	return writeJSON(netinfo)
}

func DelContainerNetInfo(containerId string) error {
	netinfo, err := ReadJSON()
	if err != nil {
		return err
	}

	if _, ok := netinfo.Containers[containerId]; ok {
		for k, v := range netinfo.Containers[containerId].ContainerNetworks {
			if _, ok := netinfo.VirtualNic[k]; ok {
				if len(v.IpAddress) < len(netinfo.VirtualNic[k].MinAvailableIP) {
					netinfo.VirtualNic[k].MinAvailableIP = v.IpAddress
				} else if len(v.IpAddress) == len(netinfo.VirtualNic[k].MinAvailableIP) && v.IpAddress < netinfo.VirtualNic[k].MinAvailableIP {
					netinfo.VirtualNic[k].MinAvailableIP = v.IpAddress
				}
			}
		}
		for k, v := range netinfo.VirtualNic {
			log.Debugf("update VirtualNic:[%+v][%+v]", k, v)
		}

		delete(netinfo.Containers, containerId)
	}

	return writeJSON(netinfo)
}

func TransMask(mask string) int {
	var masklen int
	if mask != "" {
		maskarr := strings.Split(mask, ".")
		if len(maskarr) == 4 {
			maskmap := make([]byte, 4)
			for i, value := range maskarr {
				intValue, err := strconv.Atoi(value)
				if err != nil || intValue > 255 {
					break
				}
				maskmap[i] = byte(intValue)
			}

			if len(maskmap) == 4 {
				masklen, _ = net.IPv4Mask(maskmap[0], maskmap[1], maskmap[2], maskmap[3]).Size()
			}
		}
	}

	return masklen
}

func GetNodeIP() map[string]struct{} {
	links, err := netlink.LinkList()
	if err != nil {
		log.Errorf("call LinkList : %v", err)
		return nil
	}

	nodeIPs := make(map[string]struct{})
	for _, l := range links {
		attrs := l.Attrs()
		if attrs.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, err := netlink.AddrList(l, nl.FAMILY_V4)
		if err != nil {
			log.Warnf("name=%v get address list error=%v", attrs.Name, err)
			continue
		}

		for i := 0; i < len(addrs); i++ {
			nodeIPs[addrs[i].IP.String()] = struct{}{}
		}
	}

	netinfo, err := ReadJSON()
	if netinfo == nil {
		netinfo = &NetworkInfo{}
	}
	netinfo.NodeNicIPAddr = nodeIPs
	writeJSON(netinfo)
	return nodeIPs
}

func isSameNetworkSegment(s, d string) bool {
	_, subnet, _ := net.ParseCIDR(s)
	ip, _, _ := net.ParseCIDR(d)
	if !subnet.Contains(ip) {
		log.Warnf("%v is not within the range of %v", s, d)
		return false
	}

	return true
}

func IsConflict(linkifs, ipaddr string, masklen int) bool {
	netinfo, _ := ReadJSON()
	if netinfo != nil {
		if netinfo.NodeNicIPAddr != nil {
			if _, ok := netinfo.NodeNicIPAddr[ipaddr]; ok {
				return true
			}
		}

		if netinfo.VirtualNic != nil {
			if _, ok := netinfo.VirtualNic[linkifs]; ok {
				subnet := fmt.Sprintf("%v/%d", netinfo.VirtualNic[linkifs].IPAddress, netinfo.VirtualNic[linkifs].IPMaskLen)
				ip := fmt.Sprintf("%v/%d", ipaddr, masklen)

				if ok := isSameNetworkSegment(subnet, ip); !ok {
					return true
				}
			}
		}

		for _, v := range netinfo.Containers {
			if _, ok := v.ContainerNetworks[linkifs]; ok {
				if ipaddr == v.ContainerNetworks[linkifs].IpAddress {
					log.Warnf("%v already used", ipaddr)
					return true
				}
			}
		}
	}

	return false
}
