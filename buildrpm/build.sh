#!/bin/bash

set -x

use_git_code=0
build_type="container" # local, mock, container
container_build_image="scmc-compiler:"
pkgname=ks-scmc

SHELL_FOLDER=$(cd "$(dirname "$0")";pwd)
cd $SHELL_FOLDER

version=$(cat ${SHELL_FOLDER}/${pkgname}.spec | grep "pkg_ver" | awk '{print $3}')
release=$(cat ${SHELL_FOLDER}/${pkgname}.spec | grep "pkg_rel" | awk '{print $3}')

expr $release "+" 10 &> /dev/null
if [[ $? -ne 0 || -z $version ]];then
	echo "please make sure that spec file exist and has right config: ${SHELL_FOLDER}/${pkgname}.spec"
	exit 1
fi

builddir="${SHELL_FOLDER}/_buildtmp"
resultdir="${SHELL_FOLDER}/_buildres"
arch=$(uname -m 2>/dev/null || uname -i)
# normalize arch: x86_64, aarch64
[[ "$arch" == "x86_64" || "$arch" == "amd64" ]] && arch="x86_64"
