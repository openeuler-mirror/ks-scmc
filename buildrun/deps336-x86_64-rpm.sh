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
	"http://10.20.20.17/kojifiles/packages/keepalived/2.0.20/18.ky3.kb3/x86_64/keepalived-help-2.0.20-18.ky3.kb3.x86_64.rpm"
	"http://10.20.20.17/kojifiles/packages/kernel/4.19.90/2408.1.0.0288.kb35.ky3_3/x86_64/kernel-4.19.90-2408.1.0.0288.kb35.ky3_3.x86_64.rpm"
	"http://10.20.20.17/kojifiles/packages/kernel/4.19.90/2408.1.0.0288.kb35.ky3_3/x86_64/kernel-devel-4.19.90-2408.1.0.0288.kb35.ky3_3.x86_64.rpm"
	"http://10.20.20.17/kojifiles/packages/kernel/4.19.90/2408.1.0.0288.kb35.ky3_3/x86_64/kernel-tools-4.19.90-2408.1.0.0288.kb35.ky3_3.x86_64.rpm"
	"http://10.20.20.17/kojifiles/packages/kernel/4.19.90/2408.1.0.0288.kb35.ky3_3/x86_64/kernel-tools-devel-4.19.90-2408.1.0.0288.kb35.ky3_3.x86_64.rpm"
	"http://10.20.20.17/kojifiles/packages/kiran-log/2.2.4/1/x86_64/kiran-log-gtk3-2.2.4-1.x86_64.rpm"
	"http://10.20.20.17/kojifiles/packages/kiran-log/2.2.4/1/x86_64/kiran-log-qt5-2.2.4-1.x86_64.rpm"
	"http://10.20.20.17/kojifiles/packages/ks-license/2.7.9/8.ky3_3/x86_64/ks-license-client-2.7.9-8.ky3_3.x86_64.rpm"
	"http://10.20.20.17/kojifiles/packages/ks-license-gui/2.7.9/8.ky3_3/x86_64/ks-license-gui-client-2.7.9-8.ky3_3.x86_64.rpm"
	"http://10.20.20.17/kojifiles/packages/libinput/1.10.7/2.ky3.kb3/x86_64/libinput-1.10.7-2.ky3.kb3.x86_64.rpm"
	"http://10.20.20.17/kojifiles/packages/libnetfilter_queue/1.0.2/2.ky3.kb6/x86_64/libnetfilter_queue-1.0.2-2.ky3.kb6.x86_64.rpm"
	"http://10.20.20.17/kojifiles/packages/libnftnl/1.0.8/3.ky3.kb1/x86_64/libnftnl-1.0.8-3.ky3.kb1.x86_64.rpm"
	"http://10.20.20.17/kojifiles/packages/libwacom/0.30/1.ky3.kb3/x86_64/libwacom-0.30-1.ky3.kb3.x86_64.rpm"
	"http://10.20.20.17/kojifiles/packages/libwacom/0.30/1.ky3.kb3/noarch/libwacom-data-0.30-1.ky3.kb3.noarch.rpm"
	"http://10.20.20.17/kojifiles/packages/lsyncd/2.2.2/1.ky3.kb1/x86_64/lsyncd-2.2.2-1.ky3.kb1.x86_64.rpm"
	"http://10.20.20.17/kojifiles/packages/luajit/2.0.4/3.ky3.kb1/x86_64/luajit-2.0.4-3.ky3.kb1.x86_64.rpm"
	"http://10.20.20.17/kojifiles/packages/mysql-community/8.0.44/1.ky3_3/x86_64/mysql-community-client-8.0.44-1.ky3_3.x86_64.rpm"
	"http://10.20.20.17/kojifiles/packages/mysql-community/8.0.44/1.ky3_3/x86_64/mysql-community-client-plugins-8.0.44-1.ky3_3.x86_64.rpm"
	"http://10.20.20.17/kojifiles/packages/mysql-community/8.0.44/1.ky3_3/x86_64/mysql-community-common-8.0.44-1.ky3_3.x86_64.rpm"
	"http://10.20.20.17/kojifiles/packages/mysql-community/8.0.44/1.ky3_3/x86_64/mysql-community-devel-8.0.44-1.ky3_3.x86_64.rpm"
	"http://10.20.20.17/kojifiles/packages/mysql-community/8.0.44/1.ky3_3/x86_64/mysql-community-icu-data-files-8.0.44-1.ky3_3.x86_64.rpm"
	"http://10.20.20.17/kojifiles/packages/mysql-community/8.0.44/1.ky3_3/x86_64/mysql-community-libs-8.0.44-1.ky3_3.x86_64.rpm"
	"http://10.20.20.17/kojifiles/packages/mysql-community/8.0.44/1.ky3_3/x86_64/mysql-community-libs-compat-8.0.44-1.ky3_3.x86_64.rpm"
	"http://10.20.20.17/kojifiles/packages/mysql-community/8.0.44/1.ky3_3/x86_64/mysql-community-server-8.0.44-1.ky3_3.x86_64.rpm"
	"http://10.20.20.17/kojifiles/packages/qrencode/3.4.1/3.ky3.kb6/x86_64/qrencode-3.4.1-3.ky3.kb6.x86_64.rpm"
	"http://10.20.20.17/kojifiles/packages/SDL2/2.0.10/1.ky3.kb3/x86_64/SDL2-2.0.10-1.ky3.kb3.x86_64.rpm"
	"http://10.20.20.17/kojifiles/packages/slirp4netns/0.4.3/4.ky3/x86_64/slirp4netns-0.4.3-4.ky3.x86_64.rpm"
	"http://10.20.20.17/kojifiles/packages/socat/1.7.3.2/2.ky3.kb5/x86_64/socat-1.7.3.2-2.ky3.kb5.x86_64.rpm"
	"http://10.20.20.17/kojifiles/packages/tslib/1.0/10.ky3.kb1/x86_64/tslib-1.0-10.ky3.kb1.x86_64.rpm"
	"http://10.20.20.17/kojifiles/packages/xinetd/2.3.15/14.ky3.kb1/x86_64/xinetd-2.3.15-14.ky3.kb1.x86_64.rpm"
	"http://10.20.20.17/kojifiles/packages/ngbe/1.2.6/1dkms.ky3_3/noarch/ngbe-1.2.6-1dkms.ky3_3.noarch.rpm"
)

# ===================== 工具函数 =====================
# 检查wget是否安装
check_wget() {
    if ! command -v wget &> /dev/null; then
        echo "[ERROR] wget未安装，请先安装wget后再执行脚本！"
        exit 1
    fi
}

# 下载单个RPM包（存在则跳过）
download_rpm() {
    local url="$1"
    local filename=$(basename "$url")
    
    # 检查文件是否存在且非空
    if [ -f "$filename" ] && [ -s "$filename" ]; then
        echo "[SKIP] $filename 已存在且非空，跳过下载"
        return 0
    fi

    # 执行下载（带超时、重试）
    echo "[DOWNLOAD] 开始下载: $filename"
    if wget --timeout="$WGET_TIMEOUT" --tries="$WGET_RETRY" --no-clobber "$url" -O "$filename"; then
        echo "[SUCCESS] $filename 下载完成"
    else
        echo "[FAILED] $filename 下载失败！URL: $url"
        # 可选：删除下载失败的空文件
        [ -f "$filename" ] && rm -f "$filename"
        return 1
    fi
}

# 本地/离线：不下载，仅检查当前目录下依赖 RPM 是否与 RPM_URLS 列表一致且非空
check_deps_local() {
    local missing=()
    for url in "${RPM_URLS[@]}"; do
        local fn
        fn=$(basename "$url")
        if [[ ! -f "$fn" || ! -s "$fn" ]]; then
            missing+=("$fn")
        fi
    done
    if [[ ${#missing[@]} -gt 0 ]]; then
        echo "[ERROR] 依赖目录缺少或文件为空: $(pwd)"
        echo "请将以下 RPM 放入该目录（共需 ${#RPM_URLS[@]} 个，缺 ${#missing[@]} 个）:"
        for m in "${missing[@]}"; do
            echo "  - $m"
        done
        exit 1
    fi
    echo "[OK] 依赖包齐全（${#RPM_URLS[@]} 个）"
}

