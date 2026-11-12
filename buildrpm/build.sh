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
[[ "$arch" == "aarch64" || "$arch" == "arm64" ]] && arch="aarch64"
allow_os=("3.3-6" "3.4-4")

mock_build() {
	mock_cfg=mock-${ky_version}-${arch}.cfg
	if [[ ! -e $builddir/$mock_cfg ]];then
		echo "unsupport platform, only x86_64 and aarch64: $arch"
		return
	fi

	cd $SHELL_FOLDER
	mkdir -p $builddir
	mkdir -p $resultdir

	cd ${SHELL_FOLDER}/..
        if [[ $use_git_code == '1' ]];then
		git archive -o "${builddir}/${pkgname}-${version}.tar.gz" --prefix=${pkgname}-${version}/ HEAD
	else
		tar --transform "s,^.,${pkgname}-${version}," -zcf "${builddir}/${pkgname}-${version}.tar.gz" . --exclude=${pkgname}-${version}.tar.gz
        fi
	\cp --preserve=time ${SHELL_FOLDER}/${pkgname}.spec $builddir
	\cp -rf ${builddir}/$mock_cfg /etc/mock

	cd $builddir
	mock -r mock-${ky_version}-${arch} --buildsrpm --spec ./${pkgname}.spec --sources ./${pkgname}-${version}.tar.gz --resultdir "$resultdir"
	if [[ $? -ne 0 ]];then
		echo "mock build srpm failed"
		exit 1
	fi

	srpmpath=$(ls $resultdir/${pkgname}-${version}-${release}*.src.rpm)
	mock -r mock-${ky_version}-${arch} --rebuild "$srpmpath" --resultdir "$resultdir"
	if [[ $? -ne 0 ]];then
		echo "mock build rpm failed"
		exit
	fi

}

local_build() {

	build_base_dir=/root/rpmbuild
	cat /etc/.kyinfo | grep milestone | grep '3.0-' > /dev/null
	if [[ $? -eq 0 ]];then
		build_base_dir=/usr/src/kylin
	fi

	source_dir=$build_base_dir/SOURCES
	spec_dir=$build_base_dir/SPECS
	mkdir -p $source_dir $spec_dir

	cd ${SHELL_FOLDER}/..
        if [[ $use_git_code == "1" ]];then
		# --prefix make all files in ${pkgname}-${version} dir
		git archive --prefix=${pkgname}-${version}/ -o "${source_dir}/${pkgname}-${version}.tar.gz" HEAD
	else
		tar --transform "s,^.,${pkgname}-${version}," -zcf "${source_dir}/${pkgname}-${version}.tar.gz" .
	fi
	cp --preserve=time ${SHELL_FOLDER}/${pkgname}.spec ${spec_dir}

	rpmbuild -ba ${spec_dir}/${pkgname}.spec
	if [[ $? -ne 0 ]];then
		echo "rpmbuild failed"
		exit 1
	fi
	
	cp -aR $build_base_dir/SRPMS/${pkgname}-${version}-${release}*.src.rpm ${resultdir}
	cp -aR $build_base_dir/RPMS/$arch/${pkgname}-${version}-${release}.*.rpm ${resultdir}
}
