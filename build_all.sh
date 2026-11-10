#!/bin/bash

set -e

SHELL_FOLDER=$(cd "$(dirname "$0")";pwd)
cd $SHELL_FOLDER

pkgname=ks-scmc
pkgver=1.1
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
				cp -aR $SHELL_FOLDER/buildrpm/_buildres/${pkgname}-*.$arch.rpm $SHELL_FOLDER/buildrun/ks-run/$pkgname/$pkgver/$os/$arch
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
