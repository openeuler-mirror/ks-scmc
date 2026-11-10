package internal

import (
	log "github.com/sirupsen/logrus"
	"os"
	"sync"
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
