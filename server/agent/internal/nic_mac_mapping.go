// 记录 虚拟网卡和 mac 地址 的映射关系

package internal

import (
        "bufio"
        "encoding/json"
        "io/ioutil"
        "net"
        "os"
        "os/exec"
	"path/filepath"
        "strings"
	"scmc/common"
)

type EthMac map[string]string

func jsonFile() string {
	return common.Config.Network.IPtablesPath + "eth-mac.json"
}

func readJSON() (EthMac, error) {
	JsonFile := jsonFile()
	file, err := os.Open(JsonFile)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	fileSize := fileInfo.Size()

	reader := bufio.NewReader(file)
	buffer := make([]byte, fileSize)
	_, err = reader.Read(buffer)
	if err != nil {
		return nil, err
	}

	info := make(EthMac)
	err = json.Unmarshal(buffer, &info)
	if err != nil {
		return nil, err
	}

	return info, nil
}

func writeJSON(info EthMac) error {
	JsonFile := jsonFile()
	data, err := json.MarshalIndent(info, "", "\t")
	if err != nil {
		return err
	}
	file, err := os.Create(JsonFile)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.Write(data)
	if err != nil {
		return err
	}
	return nil
}

func updateJSON(who string, address string) error {
	info, _ := readJSON()
	if info == nil {
		info = make(EthMac)
	}

	info[who] = address
	return writeJSON(info)
}

func delJSON(who string) error {
	info, _ := readJSON()
	if info != nil {
		if _, ok := info[who]; ok {
			delete(info, who)
			return writeJSON(info)
		}
	}
	return nil
}

func getMacJSON(who string) string {
	info, _ := readJSON()
	if info != nil {
		if _, ok := info[who]; ok {
			return info[who]
		}
	}
	return ""
}

func getMacAddr(nicName string) string {
        address := filepath.Join(VirtualNicBase, nicName, "address")
        content, err := ioutil.ReadFile(address)
        if err != nil {
                return ""
        }
	macaddr := string(content)
        return strings.TrimSpace(macaddr)
}

// 遍历 更改 MAC 地址
// 安全容器创建的 macvlan 是用 scmc- 开头
// 创建的 bridge 是以 -host 结尾
func UpdataVirtualNicMAC() {
	nicList, err := net.Interfaces()
	if err != nil {
		return
	}
	for _, iface := range nicList {
		if strings.HasPrefix(iface.Name, "scmc-") || strings.HasSuffix(iface.Name, "-host") {
			nicMAC1 := getMacJSON(iface.Name)
			nicMAC2 := getMacAddr(iface.Name)
			if nicMAC1 != nicMAC2 {
				cmd := exec.Command("ifconfig", iface.Name, "hw", "ether", nicMAC1)
				cmd.Run()
			}
		}
	}
}
