#!/bin/bash
set -euo pipefail  # 开启严格模式，增强脚本容错性

# ===================== 配置项 =====================
# 下载超时时间（秒）
WGET_TIMEOUT=30
# 下载失败重试次数
WGET_RETRY=3
# 所有需要下载的RPM包URL列表（集中管理，便于维护）
RPM_URLS=(
    "http://10.20.20.71/kojifiles/packages/cadvisor/0.38.8/1.ky3_4.kb2/x86_64/cadvisor-0.38.8-1.ky3_4.kb2.x86_64.rpm"
    "http://10.20.20.71/kojifiles/packages/containerd.io/1.6.15/1.ky3_4.kb1/x86_64/containerd.io-1.6.15-1.ky3_4.kb1.x86_64.rpm"
    "http://10.20.20.71/kojifiles/packages/container-selinux/2.138.0/1.kb1.ky3/noarch/container-selinux-2.138.0-1.kb1.ky3.noarch.rpm"
    "http://10.20.20.71/kojifiles/packages/docker-buildx-plugin/0.10.5/1.ky3_4.kb1/x86_64/docker-buildx-plugin-0.10.5-1.ky3_4.kb1.x86_64.rpm"
    "http://10.20.20.71/kojifiles/packages/docker-ce/23.0.6/1.ky3_4.kb1/x86_64/docker-ce-23.0.6-1.ky3_4.kb1.x86_64.rpm"
    "http://10.20.20.71/kojifiles/packages/docker-ce-cli/23.0.6/1.ky3_4.kb1/x86_64/docker-ce-cli-23.0.6-1.ky3_4.kb1.x86_64.rpm"
    "http://10.20.20.71/kojifiles/packages/docker-ce-rootless-extras/23.0.6/1.ky3_4.kb1/x86_64/docker-ce-rootless-extras-23.0.6-1.ky3_4.kb1.x86_64.rpm"
    "http://10.20.20.71/kojifiles/packages/docker-compose-plugin/2.18.1/1.ky3_4.kb1/x86_64/docker-compose-plugin-2.18.1-1.ky3_4.kb1.x86_64.rpm"
    "http://10.20.20.71/kojifiles/packages/fuse3/3.9.2/4.kb1.ky3/x86_64/fuse3-3.9.2-4.kb1.ky3.x86_64.rpm"
    "http://10.20.20.71/kojifiles/packages/fuse3/3.9.2/4.kb1.ky3/x86_64/fuse3-help-3.9.2-4.kb1.ky3.x86_64.rpm"
    "http://10.20.20.71/kojifiles/packages/fuse-overlayfs/0.7.2/5.kb2.ky3/x86_64/fuse-overlayfs-0.7.2-5.kb2.ky3.x86_64.rpm"
    "http://10.20.20.71/kojifiles/packages/gflags/2.2.2/1.kb1.ky3/x86_64/gflags-2.2.2-1.kb1.ky3.x86_64.rpm"
    "http://10.20.20.71/kojifiles/packages/grpc/1.31.0/4.kb6.ky3_4/x86_64/grpc-1.31.0-4.kb6.ky3_4.x86_64.rpm"
    "http://10.20.20.71/kojifiles/packages/influxdb/1.8.10/1/x86_64/influxdb-1.8.10-1.x86_64.rpm"
    "http://10.20.20.71/kojifiles/packages/keepalived/2.0.20/19.kb2.ky3/x86_64/keepalived-2.0.20-19.kb2.ky3.x86_64.rpm"
    "http://10.20.20.71/kojifiles/packages/keepalived/2.0.20/19.kb2.ky3/noarch/keepalived-help-2.0.20-19.kb2.ky3.noarch.rpm"
    "http://10.20.20.71/kojifiles/packages/kernel/4.19.90/2408.1.0.0288.kb35.ky3_4/x86_64/kernel-4.19.90-2408.1.0.0288.kb35.ky3_4.x86_64.rpm"
    "http://10.20.20.71/kojifiles/packages/kernel/4.19.90/2408.1.0.0288.kb35.ky3_4/x86_64/kernel-devel-4.19.90-2408.1.0.0288.kb35.ky3_4.x86_64.rpm"
    "http://10.20.20.71/kojifiles/packages/kernel/4.19.90/2408.1.0.0288.kb35.ky3_4/x86_64/kernel-tools-4.19.90-2408.1.0.0288.kb35.ky3_4.x86_64.rpm"
    "http://10.20.20.71/kojifiles/packages/kernel/4.19.90/2408.1.0.0288.kb35.ky3_4/x86_64/kernel-tools-devel-4.19.90-2408.1.0.0288.kb35.ky3_4.x86_64.rpm"
    "http://10.20.20.71/kojifiles/packages/libnetfilter_queue/1.0.5/1.kb1.ky3/x86_64/libnetfilter_queue-1.0.5-1.kb1.ky3.x86_64.rpm"
    "http://10.20.20.71/kojifiles/packages/lsyncd/2.2.3/2.kb1.ky3/x86_64/lsyncd-2.2.3-2.kb1.ky3.x86_64.rpm"
    "http://10.20.20.71/kojifiles/packages/mariadb-connector-c/3.0.6/9.kb1.ky3_4/x86_64/mariadb-connector-c-3.0.6-9.kb1.ky3_4.x86_64.rpm"
    "http://10.20.20.71/kojifiles/packages/mecab/0.996/2.kb1.ky3/x86_64/mecab-0.996-2.kb1.ky3.x86_64.rpm"
    "http://10.20.20.71/kojifiles/packages/mysql/8.0.43/1.kb1.ky3_4/x86_64/mysql-8.0.43-1.kb1.ky3_4.x86_64.rpm"
    "http://10.20.20.71/kojifiles/packages/mysql/8.0.43/1.kb1.ky3_4/x86_64/mysql-common-8.0.43-1.kb1.ky3_4.x86_64.rpm"
    "http://10.20.20.71/kojifiles/packages/mysql/8.0.43/1.kb1.ky3_4/x86_64/mysql-config-8.0.43-1.kb1.ky3_4.x86_64.rpm"
    "http://10.20.20.71/kojifiles/packages/mysql/8.0.43/1.kb1.ky3_4/x86_64/mysql-errmsg-8.0.43-1.kb1.ky3_4.x86_64.rpm"
    "http://10.20.20.71/kojifiles/packages/mysql/8.0.43/1.kb1.ky3_4/x86_64/mysql-libs-8.0.43-1.kb1.ky3_4.x86_64.rpm"
    "http://10.20.20.71/kojifiles/packages/mysql/8.0.43/1.kb1.ky3_4/x86_64/mysql-server-8.0.43-1.kb1.ky3_4.x86_64.rpm"
