package common

import (
	"fmt"
	"io/ioutil"
	"net"
	"strings"
)

/*
 * 获取系统所有的IP地址
 * 包括了 IPv6 地址
 */
func GetAllIPs() ([]string, error) {
	var ips []string
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, fmt.Errorf("获取网络接口失败: %v", err)
	}

	for _, iface := range interfaces {
		if iface.Flags&net.FlagUp == 0 {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			return nil, fmt.Errorf("获取网络地址失败: %v", err)
		}
		for _, addr := range addrs {
			ipNet, ok := addr.(*net.IPNet)
			if !ok {
				continue
			}
			ip := ipNet.IP.String()
			ips = append(ips, ip)
		}
	}
	return ips, nil
}

// 获取 Linux 系统的 product_uuid
func GetProductUUID() (string, error) {
	uuidPath := "/sys/class/dmi/id/product_uuid"
	data, err := ioutil.ReadFile(uuidPath)
	if err != nil {
		return "", fmt.Errorf("读取UUID文件失败:%v", err)
	}
	uuid := strings.TrimSpace(string(data))
	if uuid == "" {
		return "", fmt.Errorf("UUID内容为空")
	}
	return uuid, nil
}
