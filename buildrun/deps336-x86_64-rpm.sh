#!/bin/bash
set -euo pipefail  # 开启严格模式，增强脚本容错性

# ===================== 配置项 =====================
# 下载超时时间（秒）
WGET_TIMEOUT=30
# 下载失败重试次数
WGET_RETRY=3
# 所有需要下载的RPM包URL列表（集中管理，便于维护）
RPM_URLS=(
	"http://10.20.20.17/kojifiles/packages/bullet/2.82/3.ky3.kb1/x86_64/bullet-2.82-3.ky3.kb1.x86_64.rpm"
	"http://10.20.20.17/kojifiles/packages/cadvisor/0.38.8/1.ky3_3.kb2/x86_64/cadvisor-0.38.8-1.ky3_3.kb2.x86_64.rpm"
	"http://10.20.20.17/kojifiles/packages/containerd.io/1.6.4/3.1.el7/x86_64/containerd.io-1.6.4-3.1.el7.x86_64.rpm"
	"http://10.20.20.17/kojifiles/packages/container-selinux/2.76/1.ky3/noarch/container-selinux-2.76-1.ky3.noarch.rpm"
	"http://10.20.20.17/kojifiles/packages/dbus-c++/0.9.0/20.kb5/x86_64/dbus-c++-0.9.0-20.kb5.x86_64.rpm"
	"http://10.20.20.17/kojifiles/packages/docker-buildx-plugin/0.10.5/1.ky3_3.kb1/x86_64/docker-buildx-plugin-0.10.5-1.ky3_3.kb1.x86_64.rpm"
	"http://10.20.20.17/kojifiles/packages/docker-ce/23.0.6/1.ky3_3.kb1/x86_64/docker-ce-23.0.6-1.ky3_3.kb1.x86_64.rpm"
