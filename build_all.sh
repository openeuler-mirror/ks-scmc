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
