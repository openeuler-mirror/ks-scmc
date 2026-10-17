#!/bin/bash

set -e

SHELL_FOLDER=$(cd "$(dirname "$0")";pwd)
cd $SHELL_FOLDER

pkg_name=ks-scmc
pkg_ver=1.1

dobuild() {
	pushd $SHELL_FOLDER/buildrpm
	bash ./build.sh $1
	popd

	rm -f $SHELL_FOLDER/buildrun/ks-run/${pkg_name}/${pkg_ver}/$(uname -i)/${pkg_name}*
	cp -aR $SHELL_FOLDER/buildrpm/_buildres/${pkg_name}-*.$(uname -i).rpm $SHELL_FOLDER/buildrun/ks-run/${pkg_name}/${pkg_ver}/$(uname -i)

	pushd $SHELL_FOLDER/buildrun
	bash ./build_all.sh
	popd

	pushd $SHELL_FOLDER/scripts
	zip $SHELL_FOLDER/buildrun/${pkg_name}-deploy-$(date "+%Y%m%d").zip deploy/ -r
	popd

	pushd $SHELL_FOLDER/sdk
	rm -rf __pycache__/ *_pb2.py *_pb2_grpc.py
	popd
	zip $SHELL_FOLDER/buildrun/${pkg_name}-sdk-$(date "+%Y%m%d").zip sdk/ -r
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
