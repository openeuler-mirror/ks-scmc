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
        "fmt"
        "net"
        "os"
        "os/exec"
        "path/filepath"
        "strings"
	"time"

	log "github.com/sirupsen/logrus"
)

type UpdateIptables struct {
}

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
        //ipcmd := fmt.Sprintf("iptables -t nat -C POSTROUTING -s %s -o %s -j MASQUERADE", subnet, iface)
        cmd := exec.Command(
                "iptables",
                "-t", "nat",
                "-C", "POSTROUTING",
                "-s", subnet,
                "-o", iface,
                "-j", "MASQUERADE",
        )

        _, err := cmd.CombinedOutput()
        if err != nil {  //报错说明没有记录，添加记录
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

func checkRun() {
	getVirtualInterface()
}

