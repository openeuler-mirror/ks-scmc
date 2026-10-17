// network info
package iptables

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"strconv"
	"strings"

	log "github.com/sirupsen/logrus"

	"scmc/common"
	"scmc/model/litedb"
)

// node: iptables -A OUTPUT -d 172.21.1.5 -j ACCEPT
// container: iptables -A INPUT -s 192.168.122.10 -p all -j ACCEPT
type RuleInfo struct {
	Source       string //源ip -s
	Destination  string //目标ip -d
	Protocol     string //协议类型 TCP、UDP、ICMP和ALL -p
	SrcPort      string //源端口 与protocol的TCP、UDP一起使用 --sport
	DestPort     string //目标端口 与protocol的TCP、UDP一起使用  --dport
	InInterface  string //指定数据包从哪个网络接口进入 -i
	OutInterface string //指定数据包从哪个网络接口输出 -o
	Policy       string //动作 ACCEPT DROP REJECT ... -j
}

func (r *RuleInfo) srcOrDest() (string, error) {
	part := func(ip, prefix string) (string, error) {
		suffix := ""
		addr := net.ParseIP(ip)
		if addr != nil {
			suffix = "/32"
		} else {
			if _, _, err := net.ParseCIDR(ip); err != nil {
				log.Warnf("incorrect ip address: %v", err)
				return "", os.ErrInvalid
			}
		}

		return prefix + ip + suffix, nil
	}

	ret := ""
	s, err := part(r.Source, "-s ")
	if err != nil {
		return "", err
	}
	ret = strings.TrimSpace(ret + " " + s)

	s, err = part(r.Destination, "-d ")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(ret + " " + s), nil
}

func (r *RuleInfo) protoAndPort() string {
	s := ""
	proto := strings.ToLower(r.Protocol)

	if proto != "" {
		s += "-p " + proto
		if proto == "tcp" || proto == "udp" {
			if r.SrcPort != "" {
				s += "--sport " + r.SrcPort
			}
			if r.DestPort != "" {
				s += "--dport " + r.DestPort
			}
		}
	}
	return s
}
func (r *RuleInfo) IPTables() (string, error) {
	var parts []string

	p, err := r.srcOrDest()
	if err != nil {
		return "", err
	}
	parts = append(parts, p, r.protoAndPort())

	if r.InInterface != "" {
		parts = append(parts, "-i "+r.InInterface)
	}

	if r.OutInterface != "" {
		parts = append(parts, "-o "+r.OutInterface)
	}

	if r.Policy != "" {
		parts = append(parts, "-j "+r.Policy)
	}

	return strings.TrimSpace(strings.Join(parts, " ")), nil
}

type ChainRules struct {
	Chain string //链 INPUT FORWARD OUTPUT -A
	Rules []RuleInfo
}

const cmdTable = "iptables -t filter"
const outputRule = "-A OUTPUT -m state --state RELATED,ESTABLISHED -j ACCEPT"
const inputRule = "-A INPUT -m state --state RELATED,ESTABLISHED -j ACCEPT"
const maxJsonFileSize = 1048576 //1M
const (
	OperateContainer = 1
	OperateNode      = 2
)

type EnableInfo map[string]bool

func path() string {
	return common.Config.Network.IPtablesPath
}

func nodeFile() string {
	return common.Config.Network.IPtablesPath + "/node.rule"
}

func nodeInitFile() string {
	return common.Config.Network.IPtablesPath + "/node-default.rule"
}

func jsonFile() string {
	return common.Config.Network.IPtablesJsonFile
}

func readJSON() (EnableInfo, error) {
	JsonFile := jsonFile()
	log.Debugf("read file [%v]", JsonFile)
	file, err := os.Open(JsonFile)
	if err != nil {
		log.Warnf("open file err: %v", err)
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

	info := make(EnableInfo)
	err = json.Unmarshal(buffer, &info)
	if err != nil {
		log.Errorf("Unmarshal failed: %v", err)
		return nil, err
	}

	return info, nil
}

func writeJSON(info EnableInfo) error {
	JsonFile := jsonFile()
	log.Debugf("write to file [%v]", JsonFile)
	data, err := json.MarshalIndent(info, "", "\t")
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

func updateJSON(who string, enbale bool) error {
	info, _ := readJSON()
	if info == nil {
		info = make(EnableInfo)
	}

	info[who] = enbale
	return writeJSON(info)
}

func dealJSON(who string) error {
	info, _ := readJSON()
	if info != nil {
		if _, ok := info[who]; ok {
			delete(info, who)
			return writeJSON(info)
		}
	}

	return nil
}

func getEnableStatus(who string) bool {
	info, _ := readJSON()
	if info != nil {
		if _, ok := info[who]; ok {
			return info[who]
		}
	}

	return false
}

func ContainerIPtablesFile(containerId string) (string, string) {
	fileInfoList, err := ioutil.ReadDir(path())
	if err != nil {
		log.Warnf("readdir err: %v", err)
		return "", ""
	}

	for i := range fileInfoList {
		fileName := fileInfoList[i].Name()
		array := strings.Split(fileName, "-")
		if len(array) == 2 {
			if array[0] == containerId || array[1] == containerId {
				log.Debugf("fileName:%v, containerId:%v", fileName, containerId)
				return path() + "/" + fileName, fileName
			}
		}
	}

	return "", ""
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

func callShell(arg string) ([]byte, error) {
	log.Debugf("shell param: %v", arg)
	cmd := exec.Command("/bin/bash", "-c", arg)
	stdout, _ := cmd.StdoutPipe()
	if err := cmd.Start(); err != nil {
		log.Errorf("Execute failed when Start %v", err)
		return nil, err
	}

	out_bytes, _ := ioutil.ReadAll(stdout)
	stdout.Close()

	if err := cmd.Wait(); err != nil {
		log.Errorf("Execute failed when Wait: %v", err)
		return nil, err
	}

	log.Debugf("Execute finished: %v", string(out_bytes))

	return out_bytes, nil
}

func parseRule(rule string) RuleInfo {
	split := strings.Split(rule, " ")
	var info RuleInfo
	for j := 0; j < len(split); j++ {
		match := true
		switch split[j] {
		case "-s":
			info.Source = split[j+1]
		case "-d":
			info.Destination = split[j+1]
		case "-p":
			info.Protocol = split[j+1]
		case "--sport":
			info.SrcPort = split[j+1]
		case "--dport":
			info.DestPort = split[j+1]
		case "-i":
			info.InInterface = split[j+1]
		case "-o":
			info.OutInterface = split[j+1]
		case "-j":
			info.Policy = split[j+1]
		default:
			match = false
			break
		}
		if match {
			j++
		}
	}

	return info
}

func listRule(cmd string) ([]string, error) {
	out, err := callShell(cmd)
	if err != nil {
		return nil, err
	}

	list := strings.Trim(string(out), " \n\t")
	rules := strings.Split(list, "\n")
	return rules, nil
}

func ListChains(who int, pid int) ([]string, error) {
	var cmd string
	if who == OperateNode {
		cmd = fmt.Sprintf("%s -S", cmdTable)
	} else {
		cmd = fmt.Sprintf("nsenter -t %d -n %s -S", pid, cmdTable)
	}

	rules, err := listRule(cmd)
	if err != nil {
		return nil, err
	}

	var chains []string
	for _, val := range rules {
		if strings.HasPrefix(val, "-P") || strings.HasPrefix(val, "-N") {
			chains = append(chains, strings.Fields(val)[1])
		} else {
			break
		}
	}

	return chains, nil
}

func ListChainRules(who int, pid int, chain string) ([]RuleInfo, error) {
	var cmd string
	if who == OperateNode {
		cmd = fmt.Sprintf("%s -S %s", cmdTable, chain)
	} else {
		cmd = fmt.Sprintf("nsenter -t %d -n %s -S %s", pid, cmdTable, chain)
	}

	rules, err := listRule(cmd)
	if err != nil {
		return nil, err
	}

	var infos []RuleInfo
	for _, rule := range rules {
		if strings.Contains(rule, "DOCKER") {
			continue
		}
		info := parseRule(rule)
		if info != (RuleInfo{}) {
			infos = append(infos, info)
		}
	}

	return infos, nil
}

func ListRules(who int, pid int) ([]ChainRules, error) {
	chains, err := ListChains(who, pid)
	if err != nil {
		return nil, err
	}

	var chainRules []ChainRules
	for _, chain := range chains {
		info, _ := ListChainRules(who, pid, chain)
		rules := ChainRules{
			Chain: chain,
			Rules: info,
		}

		chainRules = append(chainRules, rules)
	}

	return chainRules, nil
}

func AddRule(who int, containerId string, pid int, chain string, info RuleInfo) error {
	rule, err := info.IPTables()
	if err != nil || rule == "" {
		log.Warnf("invalid parameter:[%+v]", info)
		return os.ErrInvalid
	}

	fullRule := fmt.Sprintf("-A %s %s", chain, rule)
	fileName := nodeFile()
	if who == OperateContainer {
		fileName, _ = ContainerIPtablesFile(containerId)
	}
	var cmd string
	if who == OperateNode {
		cmd = fmt.Sprintf("grep -xe \"%s\" %s || sed -i '/%s/a\\%s' %s && iptables-restore %s", fullRule, fileName, outputRule, fullRule, fileName, fileName)
	} else {
		cmd = fmt.Sprintf("grep -xe \"%s\" %s || sed -i '/COMMIT/i \\%s' %s && nsenter -t %d -n iptables-restore %s", fullRule, fileName, fullRule, fileName, pid, fileName)
	}

	if _, err := callShell(cmd); err != nil {
		return err
	}

	return nil
}

func getRuleLineNumber(fullRule, fileName string) (int, error) {
	cmd := fmt.Sprintf("grep -nxe \"%s\" %s | cut -f1 -d:", fullRule, fileName)
	out, err := callShell(cmd)
	if err != nil || len(out) == 0 {
		log.Warnf("Failed to get the line number where the rule is located: %v", err)
		return 0, os.ErrInvalid
	}
	linNum, err := strconv.Atoi(strings.Trim(string(out), " \n\t"))
	if err != nil {
		log.Warnf("Failed to convert line number to int: %v", err)
		return 0, os.ErrInvalid
	}

	return linNum, nil
}

func DelRule(who int, containerId string, pid int, chain string, info RuleInfo) error {
	rule, err := info.IPTables()
	if err != nil || rule == "" {
		log.Warnf("invalid parameter:[%+v]", info)
		return os.ErrInvalid
	}

	fullRule := fmt.Sprintf("-A %s %s", chain, rule)
	fileName := nodeFile()
	if who == OperateContainer {
		fileName, _ = ContainerIPtablesFile(containerId)
	}
	linNum, err := getRuleLineNumber(fullRule, fileName)
	if err != nil {
		return err
	}

	var cmd string
	if who == OperateNode {
		cmd = fmt.Sprintf("sed -i '%dd' %s && iptables-restore %s", linNum, fileName, fileName)
	} else {
		cmd = fmt.Sprintf("sed -i '%dd' %s && nsenter -t %d -n iptables-restore %s", linNum, fileName, pid, fileName)
	}

	if _, err := callShell(cmd); err != nil {
		return err
	}

	return nil
}

func ModifyRule(who int, containerId string, pid int, oldchain string, oldRule RuleInfo, newchain string, newRule RuleInfo) error {
	old, err := oldRule.IPTables()
	if err != nil || old == "" {
		log.Warnf("invalid parameter:[%+v]", old)
		return os.ErrInvalid
	}
	new, err := newRule.IPTables()
	if err != nil || new == "" {
		log.Warnf("invalid parameter:[%+v]", new)
		return os.ErrInvalid
	}

	fileName := nodeFile()
	if who == OperateContainer {
		fileName, _ = ContainerIPtablesFile(containerId)
	}
	oldFullRule := fmt.Sprintf("-A %s %s", oldchain, old)
	linNum, err := getRuleLineNumber(oldFullRule, fileName)
	if err != nil {
		return err
	}

	newFullRule := fmt.Sprintf("-A %s %s", newchain, new)
	var cmd string
	if who == OperateNode {
		cmd = fmt.Sprintf("sed -i '%dd' %s && sed -i '%di %s' %s  && iptables-restore %s", linNum, fileName, linNum, newFullRule, fileName, fileName)
	} else {
		cmd = fmt.Sprintf("sed -i '%dd' %s && sed -i '%di %s' %s  && nsenter -t %d -n iptables-restore %s", linNum, fileName, linNum, newFullRule, fileName, pid, fileName)
	}

	if _, err := callShell(cmd); err != nil {
		return err
	}

	return nil
}

func ContainerRemveIPtables(file string) {
	fileName := path() + "/" + file
	err := os.Remove(fileName)
	log.Infof("remove filename(%v): %v", fileName, err)
	dealJSON(file)
}

func ContainerWhitelistInitialization(containerIdName string, pid int) error {
	enable := getEnableStatus(containerIdName)
	fileName := path() + "/" + containerIdName
	var chainCmd string
	if enable {
		chainCmd = fmt.Sprintf("echo '*filter\n:INPUT DROP [0:0]\n:FORWARD ACCEPT [0:0]\n:OUTPUT ACCEPT [0:0]\n%s\nCOMMIT' > %s", inputRule, fileName)
	} else {
		chainCmd = fmt.Sprintf("echo '*filter\n:INPUT ACCEPT [0:0]\n:FORWARD ACCEPT [0:0]\n:OUTPUT ACCEPT [0:0]\n%s\nCOMMIT' > %s", inputRule, fileName)
	}

	cmd := fmt.Sprintf("ls -l %s > /dev/null 2>&1", fileName)
	_, err := callShell(cmd)
	if err != nil {
		if enable {
			cmd = fmt.Sprintf("%s; nsenter -t %d -n iptables-restore %s", chainCmd, pid, fileName)
		} else {
			return nil
		}
	} else {
		cmd = fmt.Sprintf("grep -xe \"%s\" %s || %s && nsenter -t %d -n iptables-restore %s", inputRule, fileName, chainCmd, pid, fileName)
	}

	if _, err := callShell(cmd); err != nil {
		return err
	}

	return nil
}

func DisableContainer(containerIdName string, pid int) error {
	cmd := fmt.Sprintf("nsenter -t %d -n iptables -F; nsenter -t %d -n iptables -P INPUT ACCEPT", pid, pid)
	if _, err := callShell(cmd); err != nil {
		return err
	}

	return updateJSON(containerIdName, false)
}

func EnableContainer(containerIdName string, pid int) error {
	fileName := path() + "/" + containerIdName
	cmd := fmt.Sprintf("ls -l %s > /dev/null 2>&1", fileName)
	_, err := callShell(cmd)
	if err != nil {
		cmd = fmt.Sprintf("echo '*filter\n:INPUT DROP [0:0]\n:FORWARD ACCEPT [0:0]\n:OUTPUT ACCEPT [0:0]\n%s\nCOMMIT' > %s; nsenter -t %d -n iptables-restore %s", inputRule, fileName, pid, fileName)
	} else {
		cmd = fmt.Sprintf("grep -xe \"%s\" %s || echo '*filter\n:INPUT DROP [0:0]\n:FORWARD ACCEPT [0:0]\n:OUTPUT ACCEPT [0:0]\n%s\nCOMMIT' > %s && nsenter -t %d -n iptables-restore %s", inputRule, fileName, inputRule, fileName, pid, fileName)
	}

	if _, err := callShell(cmd); err != nil {
		return err
	}

	return updateJSON(containerIdName, true)
}

func NodeWhitelistInit(linkifs string) error {
	initFile := nodeInitFile()
	cmd := fmt.Sprintf("ls -l %s > /dev/null 2>&1 || iptables-save > %s", initFile, initFile)
	if _, err := callShell(cmd); err != nil {
		return err
	}

	enable := getEnableStatus("Node")
	if enable {
		rejectRule := fmt.Sprintf("-A OUTPUT -o %s -j REJECT", linkifs)
		fileName := nodeFile()
		cmd = fmt.Sprintf("ls -l %s > /dev/null 2>&1", fileName)
		_, err := callShell(cmd)
		if err != nil {
			cmd = fmt.Sprintf("%s %s; %s %s; iptables-save > %s", cmdTable, outputRule, cmdTable, rejectRule, fileName)
		} else {
			cmd = fmt.Sprintf("iptables-restore %s; (grep -xe \"%s\" %s || %s %s); (grep -we \"%s\" %s || %s %s); iptables-save > %s",
				fileName, outputRule, fileName, cmdTable, outputRule, rejectRule, fileName, cmdTable, rejectRule, fileName)
		}

		if _, err := callShell(cmd); err != nil {
			return err
		}
	} /*else {
		cmd = fmt.Sprintf("iptables-restore %s", initFile)
		if _, err := callShell(cmd); err != nil {
			return err
		}
	}*/

	return nil
}

func DisableNode() error {
	initFile := nodeInitFile()
	cmd := fmt.Sprintf("ls -l %s > /dev/null 2>&1 && iptables-restore %s", initFile, initFile)
	if _, err := callShell(cmd); err != nil {
		return err
	}

	return updateJSON("Node", false)
}

func EnableNode(linkifs string) error {
	rejectRule := fmt.Sprintf("-A OUTPUT -o %s -j REJECT", linkifs)
	fileName := nodeFile()
	cmd := fmt.Sprintf("ls -l %s > /dev/null 2>&1", fileName)
	_, err := callShell(cmd)
	if err != nil {
		cmd = fmt.Sprintf("%s %s; %s %s; iptables-save > %s", cmdTable, outputRule, cmdTable, rejectRule, fileName)
	} else {
		cmd = fmt.Sprintf("iptables-restore %s; (grep -xe \"%s\" %s || %s %s); (grep -we \"%s\" %s || %s %s); iptables-save > %s",
			fileName, outputRule, fileName, cmdTable, outputRule, rejectRule, fileName, cmdTable, rejectRule, fileName)
	}

	if _, err := callShell(cmd); err != nil {
		return err
	}

	return updateJSON("Node", true)
}

func ruleToCmd(filePath, fileName string, isOn bool, rules []*litedb.NetworkRule) string {
	var chainCmd string
	if isOn {
		chainCmd = fmt.Sprintf("echo '*filter\n:INPUT DROP [0:0]\n:FORWARD ACCEPT [0:0]\n:OUTPUT ACCEPT [0:0]\n")
		updateJSON(fileName, true)
	} else {
		chainCmd = fmt.Sprintf("echo '*filter\n:INPUT ACCEPT [0:0]\n:FORWARD ACCEPT [0:0]\n:OUTPUT ACCEPT [0:0]\n")
		updateJSON(fileName, false)
	}

	var allRule string
	// TODO refactor to type function
	for _, rule := range rules {
		var curRule string
		var prestr string = "-A INPUT "
		var sufstr string = " -j ACCEPT"
		var addrstr string
		if rule.Addr != "" {
			addrstr = "-s " + rule.Addr
			if net.ParseIP(rule.Addr) != nil {
				addrstr += "/32"
			}
		}

		//多个协议情况处理
		for _, protocol := range rule.Protocols {
			protocol = strings.ToLower(protocol)
			if protocol == "" {
				continue
			}
			var arr []string
			arr = append(arr, prestr)
			arr = append(arr, addrstr)
			str := "-p " + protocol
			arr = append(arr, str)
			if (protocol == "tcp" || protocol == "udp") && rule.Port != 0 {
				str = fmt.Sprintf("--sport %d", rule.Port)
				arr = append(arr, str)
			}
			arr = append(arr, sufstr)
			res := strings.Join(arr, " ") + "\n"
			curRule = curRule + res
		}

		//只有ip没有协议情况处理
		if curRule == "" && addrstr != "" {
			curRule = prestr + " " + addrstr + sufstr + "\n"
		}

		allRule = allRule + curRule
	}

	cmd := fmt.Sprintf("%s%s\n%sCOMMIT' > %s", chainCmd, inputRule, allRule, filePath)

	return cmd
}

func UpdateContainerIPtablesFile(fileName string, isOn bool, rules []*litedb.NetworkRule, pid int) error {
	filePath := path() + "/" + fileName
	cmd := ruleToCmd(filePath, fileName, isOn, rules)
	if pid != 0 {
		cmd = fmt.Sprintf("%s; nsenter -t %d -n iptables-restore %s", cmd, pid, filePath)
	}

	if _, err := callShell(cmd); err != nil {
		return err
	}

	return nil
}
