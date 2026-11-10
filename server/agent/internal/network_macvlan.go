package internal

import (
	// log "github.com/sirupsen/logrus"
	"errors"
	// "flag"
	"fmt"
	"net"
	// "os"
)

// 解析CIDR格式的IP地址和子网掩码
func macvlanparseCIDR(cidr string) (net.IP, net.IPNet, error) {
	ip, ipNet, err := net.ParseCIDR(cidr)
	if err != nil {
		return nil, net.IPNet{}, fmt.Errorf("无效的CIDR格式: %v", err)
	}
	return ip, *ipNet, nil
}

// 获取指定网络接口的所有IP地址（IPv4）
func getInterfaceIPs(ifaceName string) ([]net.IPNet, error) {
	iface, err := net.InterfaceByName(ifaceName)
	if err != nil {
		return nil, fmt.Errorf("无法找到接口 %s: %v", ifaceName, err)
	}

	addrs, err := iface.Addrs()
	if err != nil {
		return nil, fmt.Errorf("无法获取接口 %s 的地址: %v", ifaceName, err)
	}

	var ipNets []net.IPNet
	for _, addr := range addrs {
		// 只处理IPv4地址
		if ipNet, ok := addr.(*net.IPNet); ok && !ipNet.IP.IsLoopback() && ipNet.IP.To4() != nil {
			ipNets = append(ipNets, *ipNet)
		}
	}

	if len(ipNets) == 0 {
		return nil, errors.New("接口没有配置IPv4地址")
	}

	return ipNets, nil
}

// 检查两个IP是否在同一网段
func isSameSubnet(ip1 net.IP, ipNet1 net.IPNet, ip2 net.IP, ipNet2 net.IPNet) bool {
	// 确保两个IP版本一致
	if ip1.To4() != nil && ip2.To4() == nil {
		return false
	}
	if ip1.To16() != nil && ip2.To16() == nil && ip2.To4() != nil {
		return false
	}

	// 计算网络地址
	network1 := ipNet1.IP.Mask(ipNet1.Mask)
	network2 := ip2.Mask(ipNet1.Mask) // 使用第一个IP的子网掩码计算

	return network1.Equal(network2)
}

//判断给定的 ip/掩码段  与  网卡是否为同一个网段
func CheckNicSameSubnet(targetCIDR, ifaceName string) bool {
        targetIP, targetNet, err := macvlanparseCIDR(targetCIDR)
        if err != nil {
            return false;
        }
        ifaceIPs, err := getInterfaceIPs(ifaceName)
        if err != nil {
            return false
        }

        for _, ifaceIPNet := range ifaceIPs {
            if isSameSubnet(ifaceIPNet.IP, ifaceIPNet, targetIP, targetNet) {
                return true
            }
        }
        return false
}
