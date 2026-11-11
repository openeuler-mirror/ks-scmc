/*
 * 添加 iptables 规则， 保证网络可以相互访问
 * iptables 会因为关机丢失， 所以 开个定时器检查，规则应该是可以写文件的
 * 但是我还不会
 */

// 获取虚拟网卡接口 和 对应的网段
// 获取 iptables 信息
// 查看 网段信息有没有在 iptables 里面。
// 没有就加入， 有就不管了。

package tasks

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	log "github.com/sirupsen/logrus"

	"scmc/model/db"
	"scmc/model/docker"
	"scmc/model/litedb"
	pb "scmc/rpc/pb/container"
	"scmc/server/controller/internal"
)

const updateContainerFlagFile = "/var/lib/ks-scmc/networks/iptables/updateIptables.flag"

type UpdateIptables struct{}

func (s *UpdateIptables) Interval() time.Duration {
	return time.Minute
}

func (s *UpdateIptables) RunAtInit() bool {
	return true
}

func (s *UpdateIptables) Run() {
	checkRun()
}

/*
 * 检查和添加Iptables 规则
 * 如果 没有 对应的规则就添加
 * 如果 有  就不用管了。
 */
func checkIptables(subnet, iface string) {
	// ipcmd := fmt.Sprintf("iptables -t nat -C POSTROUTING -s %s -o %s -j MASQUERADE", subnet, iface)
	cmd := exec.Command(
		"iptables",
		"-t", "nat",
		"-C", "POSTROUTING",
		"-s", subnet,
		"-o", iface,
		"-j", "MASQUERADE",
	)

	_, err := cmd.CombinedOutput()
	if err != nil { // 报错说明没有记录，添加记录
		cmd := exec.Command(
			"iptables",
			"-t", "nat",
			"-A", "POSTROUTING",
			"-s", subnet,
			"-o", iface,
			"-j", "MASQUERADE",
		)
		err = cmd.Run()
		if err != nil {
			log.Warnf("add iptables error: -s %s -o %s err=%v", subnet, iface, err)
		}
	}
}

/*
 * 获取虚拟网卡对应的网段信息
 * 返回 192.168.230.0/24 这样的网段字符串
 */
func getInterfaceInfo(name string) (string, error) {
	// 获取所有网络接口
	interfaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}

	// 遍历接口，按名称匹配目标接口
	for _, iface := range interfaces {
		if iface.Name == name {
			addrs, err := iface.Addrs()
			if err != nil {
				return "", err
			}

			if strings.Contains(iface.Flags.String(), "up") {
				addr := addrs[0]
				_, ipNet, err := net.ParseCIDR(addr.String())
				if err != nil {
					return "", err
				}
				return ipNet.String(), nil
			}

			// 网卡down 状态的不处理
			return "", nil
		}
	}

	return "", fmt.Errorf("interface %s not found", name)
}

/*
 * 获取虚拟网卡对应的 物理网卡名称
 */
func getVirtlowerName(path string) (string, error) {
	sysPath := filepath.Join(path, "lower_*")

	lowerLinks, err := filepath.Glob(sysPath)
	if err != nil {
		return "", err
	}

	if len(lowerLinks) == 0 {
		return "", nil
	}

	linkPath, err := os.Readlink(lowerLinks[0])
	if err != nil {
		return "", nil
	}

	parentName := filepath.Base(linkPath)
	return parentName, nil
}

/*
 * 获取虚拟网卡信息只处理后缀为 -host 的虚拟网卡
 * 必须是 up 的， 有对应物理网卡的
 */
func getVirtualInterface() {
	sysPath := "/sys/devices/virtual/net/*"
	ethName, err := filepath.Glob(sysPath)
	if err != nil {
		log.Warnf("filepath.Glob(%s) err=%v", sysPath, err)
		return
	}

	for _, name := range ethName {
		if strings.HasSuffix(name, "-host") {
			parentName, _ := getVirtlowerName(name)
			if len(parentName) > 0 {
				// 获取 网卡网段信息
				name = filepath.Base(name)
				ipNet, _ := getInterfaceInfo(name)
				if len(ipNet) > 0 {
					checkIptables(ipNet, parentName)
				}
			}
		}
	}
}

/*
 * 获取本机所有IP
 */
func getAllIPs() ([]string, error) {
	var ips []string

	ifaces, err := net.Interfaces()
	if err != nil {
		log.Warnf("getAllIPs failed: %v", err)
		return nil, err
	}

	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 {
			continue // 接口未激活
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}

			if ip == nil || ip.IsLoopback() {
				continue
			}

			ip = ip.To4()
			if ip == nil {
				continue // 不是IPv4地址
			}

			ips = append(ips, ip.String())
		}
	}

	if len(ips) == 0 {
		return nil, fmt.Errorf("getAllIPs failed, no valid ip")
	}

	return ips, nil
}

/*
 * 获取本机 nodeid
 */
func getNodeIdAddress() (int64, string, error) {
	ips, err := getAllIPs()
	if err != nil {
		log.Warnf("getNodeId failed, %v", err)
		return 0, "", err
	}

	nodes, err := db.ListNodes()
	if err != nil {
		log.Warnf("Failed to list nodes: %v", err)
		return 0, "", err
	}

	var nodeid int64 = 0
	var address string = ""
	for _, n := range nodes {
		for _, ip := range ips {
			if n.Address == ip {
				nodeid = n.ID
				address = n.Address
			}
		}
	}

	return nodeid, address, nil
}

/*
 * 有容器启动或停止的时候，需要刷新所有容器的 iptables 规则
 */
func updateContainersIptables() {
	_, err := os.Stat(updateContainerFlagFile)
	if err != nil {
		// 没权限等，也视为不存在
		return
	}

	nodeid, address, err := getNodeIdAddress()
	if err != nil || nodeid < 1 || address == "" {
		log.Warnf("updateContainersIptables get nodeid failed, err: %v", err)
		return
	}

	conn, err := internal.GetAgentConn(address)
	if err != nil {
		log.Warnf("getAgentConn failed: %v", err)
		return
	}

	cli, err := docker.Client()
	if err != nil {
		log.Warnf("updateContainersIptables get docker client failed, %v", err)
		return
	}

	containers, err := cli.ContainerList(context.Background(), types.ContainerListOptions{})
	if err != nil {
		log.Warnf("updateContainersIptables get container list failed, %v", err)
		return
	}

	succeed := false

	for _, container := range containers {
		log.Infof("updateContainersIptables container: %v", container)

		cfgs, err := litedb.GetContainerConfig(container.ID)
		if err != nil {
			log.Warnf("GetContainerConfig err=%v", err)
			continue // 跳过获取不到配置的容器
		}
		_, err = pb.NewContainerClient(conn).Update(context.Background(), &pb.UpdateRequest{
			NodeId:         nodeid,
			ContainerId:    container.ID,
			SecurityConfig: cfgs.SecurityConfig,
		})
		if err != nil {
			log.Warnf("update contain iptables failed, %v", err)
			continue
		}

		succeed = true
	}

	if succeed {
		err = os.Remove(updateContainerFlagFile)
		if err != nil {
			log.Warnf("updateContainersIptables rm flag file:%v failed:%v", updateContainersIptables, err)
		}
	}
}

func checkRun() {
	getVirtualInterface()
	updateContainersIptables()
}
