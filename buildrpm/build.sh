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
