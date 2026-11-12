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
