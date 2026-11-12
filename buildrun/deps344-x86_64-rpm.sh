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
    "http://10.20.20.71/kojifiles/packages/protobuf/3.14.0/6.kb1.ky3_4/x86_64/protobuf-3.14.0-6.kb1.ky3_4.x86_64.rpm"
    "http://10.20.20.71/kojifiles/packages/protobuf/3.14.0/6.kb1.ky3_4/x86_64/protobuf-compiler-3.14.0-6.kb1.ky3_4.x86_64.rpm"
    "http://10.20.20.71/kojifiles/packages/protobuf/3.14.0/8.kb1.ky3_4/x86_64/protobuf-lite-3.14.0-8.kb1.ky3_4.x86_64.rpm"
    "http://10.20.20.71/kojifiles/packages/protobuf-c/1.3.2/5.kb1.ky3_4/x86_64/protobuf-c-1.3.2-5.kb1.ky3_4.x86_64.rpm"
    "http://10.20.20.71/kojifiles/packages/qt5-qtbase/5.11.1/18.kb1.ky3_4/noarch/qt5-qtbase-common-5.11.1-18.kb1.ky3_4.noarch.rpm"
    "http://10.20.20.71/kojifiles/packages/qt5-qtbase/5.11.1/18.kb1.ky3_4/x86_64/qt5-qtbase-5.11.1-18.kb1.ky3_4.x86_64.rpm"
    "http://10.20.20.71/kojifiles/packages/qt5-qtbase/5.11.1/18.kb1.ky3_4/x86_64/qt5-qtbase-devel-5.11.1-18.kb1.ky3_4.x86_64.rpm"
    "http://10.20.20.71/kojifiles/packages/qt5-qtbase/5.11.1/18.kb1.ky3_4/x86_64/qt5-qtbase-gui-5.11.1-18.kb1.ky3_4.x86_64.rpm"
    "http://10.20.20.71/kojifiles/packages/qt5/5.14.2/1.kb1.ky3/noarch/qt5-rpm-macros-5.14.2-1.kb1.ky3.noarch.rpm"
    "http://10.20.20.71/kojifiles/packages/re2/20200801/1.kb1.ky3/x86_64/re2-20200801-1.kb1.ky3.x86_64.rpm"
    "http://10.20.20.71/kojifiles/packages/slirp4netns/0.4.2/3.git21fdece.kb2.ky3/x86_64/slirp4netns-0.4.2-3.git21fdece.kb2.ky3.x86_64.rpm"
    "http://10.20.20.71/kojifiles/packages/socat/1.7.3.2/8.kb1.ky3/x86_64/socat-1.7.3.2-8.kb1.ky3.x86_64.rpm"
    "http://10.20.20.71/kojifiles/packages/xinetd/2.3.15/31.kb1.ky3/x86_64/xinetd-2.3.15-31.kb1.ky3.x86_64.rpm"
    "http://10.20.20.71/kojifiles/repos/KY3.4-4-PG-ks-scmc-1.1-build/latest/x86_64/toplink/packages/ks-license/2.7.9/35.ky3_4/x86_64/ks-license-client-2.7.9-35.ky3_4.x86_64.rpm"
    "http://10.20.20.71/kojifiles/packages/ks-license-gui/2.7.8/8.ky3_4/x86_64/ks-license-gui-client-2.7.8-8.ky3_4.x86_64.rpm"
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

# ===================== 主逻辑 =====================
main() {
    # 前置检查
    check_wget
    echo "===== 开始批量下载RPM包（共${#RPM_URLS[@]}个） ====="

    # 循环下载所有包
    local fail_count=0
    local fail_list=()
    for url in "${RPM_URLS[@]}"; do
        if ! download_rpm "$url"; then
            fail_count=$((fail_count + 1))
            fail_list+=("$url")
        fi
        echo "----------------------------------------"
    done

    # 输出汇总信息
    echo -e "\n===== 下载完成汇总 ====="
    echo "总数量: ${#RPM_URLS[@]}"
    echo "成功数量: $(( ${#RPM_URLS[@]} - fail_count ))"
    echo "失败数量: $fail_count"

