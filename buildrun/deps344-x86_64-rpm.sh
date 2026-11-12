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
