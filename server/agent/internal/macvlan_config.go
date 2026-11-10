package internal

import (
	log "github.com/sirupsen/logrus"
	"os"
	"sync"
        "fmt"
)

// 文件路径，根据实际情况修改
const upfilePath = "/etc/sysconfig/network-scripts/ifup-macvlan"

// 文件内容
const upfileContent = `#!/bin/bash
# Network Interface Configuration System
# Copyright (c) 1996-2014 Red Hat, Inc. all rights reserved.
#
# This program is free software; you can redistribute it and/or modify
# it under the terms of the GNU General Public License, version 2,
# as published by the Free Software Foundation.
#
# This program is distributed in the hope that it will be useful,
# but WITHOUT ANY WARRANTY; without even the implied warranty of
# MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
# GNU General Public License for more details.
#
# You should have received a copy of the GNU General Public License
# along with this program; if not, write to the Free Software
# Foundation, Inc., 51 Franklin Street, Fifth Floor, Boston, MA 02110-1301, USA.

. /etc/init.d/functions

cd /etc/sysconfig/network-scripts
. ./network-functions

[ -f ../network ] && . ../network

CONFIG=${1}

[ -f "$CONFIG" ] || exit 1

need_config "${CONFIG}"
source_config

if [ -z "${DEVICE}" ]; then
	echo "Device name not specified"
	exit 1
fi

if [ -z "${PHYSDEV}" ]; then
	echo "Physical device not specified"
	exit 1
fi

MODE=${MODE:-bridge}

ip link set "${PHYSDEV}" promisc on
ip link add "${DEVICE}" link "${PHYSDEV}" type macvlan mode "${MODE}"

if [ -n "${IPADDR}" ]; then
	if [ -n "${NETMASK}" ]; then
		ip addr add ${IPADDR}/${NETMASK} dev "${DEVICE}"
	else
		ip addr add "${IPADDR}" dev "${DEVICE}"
	fi
fi

ip link set "${DEVICE}" up

if [ -n "${GATEWAY}" ]; then
	ip route add default via "${GATEWAY}" dev "${DEVICE}"
fi`

// 定义文件路径
const downfilePath = "/etc/sysconfig/network-scripts/ifdown-macvlan"

// 定义文件内容
const downfileContent = `#!/bin/bash
# Network Interface Configuration System
# Copyright (c) 1996-2014 Red Hat, Inc. all rights reserved.
#
# This program is free software; you can redistribute it and/or modify
# it under the terms of the GNU General Public License, version 2,
# as published by the Free Software Foundation.
#
# This program is distributed in the hope that it will be useful,
# but WITHOUT ANY WARRANTY; without even the implied warranty of
# MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
# GNU General Public License for more details.
#
# You should have received a copy of the GNU General Public License
# along with this program; if not, write to the Free Software
# Foundation, Inc., 51 Franklin Street, Fifth Floor, Boston, MA 02110-1301, USA.

. /etc/init.d/functions

cd /etc/sysconfig/network-scripts
. ./network-functions

[ -f ../network ] && . ../network

CONFIG=${1}

[ -f "$CONFIG" ] || exit 1

need_config "${CONFIG}"
source_config

if [ -z "${DEVICE}" ]; then
	echo "Device name not specified"
	exit 1
fi

ip link delete "${DEVICE}"

exit 0
`

func createMacvlanConf(filePath, fileContent string) {
	// 检查文件是否存在
	_, err := os.Stat(filePath)
	if os.IsNotExist(err) {
		// 文件不存在，创建文件
		file, err := os.Create(filePath)
		if err != nil {
			log.Warnf("无法创建文件: %v", err)
			return
		}
		defer file.Close()

		// 写入内容
		_, err = file.WriteString(fileContent)
		if err != nil {
			log.Warnf("写入文件内容失败: %v", err)
			return
		}

		// 设置文件权限为可执行
		err = os.Chmod(filePath, 0755)
		if err != nil {
			log.Warnf("设置文件权限失败: %v", err)
			return
		}
		log.Infof("文件创建成功")
	} else if err != nil {
		// 其他错误
		log.Warnf("检查文件时发生错误: %v", err)
	} else {
		// 文件已存在
		log.Debugf("文件已存在")
	}
}

var runOnce sync.Once

func CreateMacvlanConf() {
	runOnce.Do(func() {
		createMacvlanConf(upfilePath, upfileContent)
		createMacvlanConf(downfilePath, downfileContent)
	})
}


// Config 存储macvlan接口的配置参数
type Config struct {
	Device       string
	Name         string
	Type         string
	PhysDev      string
	MacvlanMode  string
	BootProto    string
	IPAddr       string
	Netmask      string
	Gateway      string
	DefRoute     string
	OnBoot       string
}

// DefaultConfig 返回默认配置
func DefaultConfig() Config {
	return Config{
		Device:       "macvlan_host",
		Name:         "macvlan_host",
		Type:         "macvlan",
		PhysDev:      "ens33",
		MacvlanMode:  "bridge",
		BootProto:    "static",
		IPAddr:       "192.168.1.11",
		Netmask:      "255.255.255.0",
		Gateway:      "192.168.1.1",
		DefRoute:     "no",
		OnBoot:       "yes",
	}
}


// GenerateConfigContent 生成配置文件内容
func GenerateConfigContent(cfg Config) string {
	content := fmt.Sprintf("DEVICE=%s\n", cfg.Device)
	content += fmt.Sprintf("NAME=%s\n", cfg.Name)
	content += fmt.Sprintf("TYPE=%s\n", cfg.Type)
	content += fmt.Sprintf("PHYSDEV=%s      # 替换为你的物理接口名 (如 ens192, eno1)\n", cfg.PhysDev)
	content += fmt.Sprintf("MACVLAN_MODE=%s # 必须与 Docker Macvlan 网络模式一致\n", cfg.MacvlanMode)
	content += fmt.Sprintf("BOOTPROTO=%s    # 使用静态 IP\n", cfg.BootProto)
	content += fmt.Sprintf("IPADDR=%s       # 替换为宿主机在 Macvlan 子网中的 IP\n", cfg.IPAddr)
	content += fmt.Sprintf("NETMASK=%s      # 子网掩码 (或使用 PREFIX=24)\n", cfg.Netmask)
	content += fmt.Sprintf("GATEWAY=%s      # 可选 - 仅当宿主机需通过此接口上网时添加\n", cfg.Gateway)
	content += fmt.Sprintf("DEFROUTE=%s     # 重要！避免覆盖默认路由\n", cfg.DefRoute)
	content += fmt.Sprintf("ONBOOT=%s       # 开机自动启用\n", cfg.OnBoot)
	return content
}

// SaveToFile 将配置内容保存到文件
func SaveToFile(content, filename string) error {
	// Go 1.15中os.WriteFile是新增函数，为了兼容，这里使用传统方式实现
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.WriteString(content)
	return err
}
