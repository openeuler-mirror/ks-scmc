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
	"http://10.20.20.17/kojifiles/packages/docker-ce-cli/23.0.6/1.ky3_3.kb1/x86_64/docker-ce-cli-23.0.6-1.ky3_3.kb1.x86_64.rpm"
	"http://10.20.20.17/kojifiles/packages/docker-ce-rootless-extras/23.0.6/1.ky3_3.kb1/x86_64/docker-ce-rootless-extras-23.0.6-1.ky3_3.kb1.x86_64.rpm"
	"http://10.20.20.17/kojifiles/packages/docker-compose-plugin/2.18.1/1.ky3_3.kb1/x86_64/docker-compose-plugin-2.18.1-1.ky3_3.kb1.x86_64.rpm"
	"http://10.20.20.17/kojifiles/packages/efl/1.17.1/1.ky3.1.kb2/x86_64/efl-1.17.1-1.ky3.1.kb2.x86_64.rpm"
	"http://10.20.20.17/kojifiles/packages/fmt/6.2.1/2.ky3.kb1/x86_64/fmt-6.2.1-2.ky3.kb1.x86_64.rpm"
	"http://10.20.20.17/kojifiles/packages/fuse3/3.6.1/2.ky3.kb1/x86_64/fuse3-libs-3.6.1-2.ky3.kb1.x86_64.rpm"
	"http://10.20.20.17/kojifiles/packages/fuse-overlayfs/0.7.2/6.ky3/x86_64/fuse-overlayfs-0.7.2-6.ky3.x86_64.rpm"
	"http://10.20.20.17/kojifiles/packages/gperftools/2.6.1/1.ky3.kb4/x86_64/gperftools-libs-2.6.1-1.ky3.kb4.x86_64.rpm"
	"http://10.20.20.17/kojifiles/packages/grpc/1.17.1/3.ky3.kb1/x86_64/grpc-1.17.1-3.ky3.kb1.x86_64.rpm"
	"http://10.20.20.17/kojifiles/packages/ibus/1.5.17/12.ky3_3/x86_64/ibus-libs-1.5.17-12.ky3_3.x86_64.rpm"
	"http://10.20.20.17/kojifiles/packages/influxdb/1.8.10/1/x86_64/influxdb-1.8.10-1.x86_64.rpm"
	"http://10.20.20.17/kojifiles/packages/keepalived/2.0.20/18.ky3.kb3/x86_64/keepalived-2.0.20-18.ky3.kb3.x86_64.rpm"
