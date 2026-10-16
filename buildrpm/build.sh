#!/bin/bash

set -x

use_git_code=1
use_mock_build=1
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
arch=$(uname -i)

mock_build() {
	mock_cfg=mock-ky3.3-${arch}.cfg
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
	mock -r mock-ky3.3-${arch} --buildsrpm --spec ./${pkgname}.spec --sources ./${pkgname}-${version}.tar.gz --resultdir "$resultdir"
	if [[ $? -ne 0 ]];then
		echo "mock build srpm failed"
		exit 1
	fi

	srpmpath=$(ls $resultdir/${pkgname}-${version}-${release}*.src.rpm)
	mock -r mock-ky3.3-${arch} --rebuild "$srpmpath" --resultdir "$resultdir"
	if [[ $? -ne 0 ]];then
		echo "mock build rpm failed"
		exit
	fi

}

normal_build() {

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

do_build() {
	rel=$1

	expr $rel "+" 10 &> /dev/null
	if [[ $? -ne 0 ]];then
		echo "please make sure that release number is valid ${rel}"
		exit 1
	fi

	if [[ "$(id -u)" != 0 ]]; then
		echo "must be root user, mock cmd need it"
		return
	fi
        
	if [[ -n $rel ]];then
        	release=$rel
		sed "s/^%global pkg_rel.*$/%global pkg_rel $rel/g" ${SHELL_FOLDER}/${pkgname}.spec -i
        fi

	git pull

	if [[ $use_mock_build == "1" ]];then
		mock_build $rel
	else
		normal_build $rel
	fi
}

clean() {
	echo "rm -rf $builddir"
	rm -rf $builddir
	
	echo "rm -rf $resultdir"
	rm -rf $resultdir
}

init() {
	echo "mkdir -p $builddir $resultdir"
	mkdir -p $builddir $resultdir

	pushd $SHELL_FOLDER
	\cp mock-temp-ky3.3.cfg $builddir/mock-ky3.3-${arch}.cfg
	sed "s/{arch}/$arch/g" $builddir/mock-ky3.3-${arch}.cfg -i
	popd
}

usage() {
	echo "Usage:"
	echo "    $0 [--clean|--help|-h|\$pkgversion]"
}

if [[ $1 == "--help" || $1 == "-h" ]];then
	usage
elif [[ $1 == "--clean" ]];then
	clean
else
	clean
	init

	do_build $1
fi

