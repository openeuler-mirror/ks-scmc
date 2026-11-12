#!/bin/bash

set -e

SHELL_FOLDER=$(cd "$(dirname "$0")";pwd)
cd $SHELL_FOLDER

pkgname=ks-scmc
pkgver=1.2
declare -a supported_arch=(x86_64)
declare -a supported_os=(3.3-6 3.4-4)

dobuild() {
	for os in ${supported_os[*]};do
		for arch in ${supported_arch[*]};do
			rm -f $SHELL_FOLDER/buildrun/ks-run/$pkgname/$pkgver/$os/$arch/${pkgname}*
			if [[ $arch == "$(uname -i)" ]];then
				pushd $SHELL_FOLDER/buildrpm
				bash ./build.sh "$1" $os
				popd
				deps_dir=$SHELL_FOLDER/buildrun/ks-run/$pkgname/$pkgver/$os/$arch
				mkdir -p "$deps_dir"
				cp -aR $SHELL_FOLDER/buildrpm/_buildres/${pkgname}-*.$arch.rpm "$deps_dir"
				# 本地流程：不 wget；依赖须已置于 deps_dir，与 deps*-rpm.sh 中 RPM_URLS 的 basename 一致
				if [[ "$os" == "3.3-6" && "$arch" == "x86_64" ]]; then
					pushd "$deps_dir"
					bash $SHELL_FOLDER/buildrun/deps336-x86_64-rpm.sh check
					popd
				elif [[ "$os" == "3.4-4" && "$arch" == "x86_64" ]]; then
					pushd "$deps_dir"
					bash $SHELL_FOLDER/buildrun/deps344-x86_64-rpm.sh check
					popd
				fi
			fi
		done
	done

	pushd $SHELL_FOLDER/buildrun
	bash ./build_all.sh
	popd

	pushd $SHELL_FOLDER/scripts
	zip $SHELL_FOLDER/buildrun/${pkgname}-deploy-$(date "+%Y%m%d").zip deploy/ -r
	popd

	pushd $SHELL_FOLDER/sdk
	rm -rf __pycache__/ *_pb2.py *_pb2_grpc.py
	popd
	zip $SHELL_FOLDER/buildrun/${pkgname}-sdk-$(date "+%Y%m%d").zip sdk/ -r
}

usage() {
        echo "Usage:"
        echo "    $0 [--help|-h|\$pkgversion]"
}

if [[ $1 == "--help" || $1 == "-h" ]];then
        usage
else
	dobuild $1
fi
