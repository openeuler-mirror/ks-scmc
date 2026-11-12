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

# Container build: run rpmbuild inside build:v1 container with project code mounted
container_build() {
	if ! docker image inspect "$container_build_image" &>/dev/null; then
		echo "Build image $container_build_image not found. Run: $0 --build-image"
		exit 1
	fi

	mkdir -p "$builddir" "$resultdir"
	project_root="$(cd ${SHELL_FOLDER}/.. && pwd)"
	resultdir_abs="$(cd "$resultdir" && pwd)"

	docker run --rm \
		-v "${project_root}:/workspace:rw" \
		-v "${resultdir_abs}:/result:rw" \
		-e "pkgname=$pkgname" \
		-e "version=$version" \
		-e "release=$release" \
		-e "use_git_code=$use_git_code" \
		-e "arch=$arch" \
		-w /workspace \
		"$container_build_image" \
		/bin/bash -c '
			set -x
			build_base_dir=/root/rpmbuild
			source_dir=$build_base_dir/SOURCES
			spec_dir=$build_base_dir/SPECS
			mkdir -p $source_dir $spec_dir
			cd /workspace
			if [[ "$use_git_code" == "1" ]]; then
				git archive --prefix=${pkgname}-${version}/ -o "${source_dir}/${pkgname}-${version}.tar.gz" HEAD
			else
				tar --transform "s,^.,${pkgname}-${version}," -zcf "${source_dir}/${pkgname}-${version}.tar.gz" .
			fi
			cp --preserve=time /workspace/buildrpm/${pkgname}.spec ${spec_dir}
			rpmbuild -ba ${spec_dir}/${pkgname}.spec
			[[ $? -ne 0 ]] && { echo "Rpmbuild failed"; exit 1; }
			cp -aR $build_base_dir/SRPMS/${pkgname}-${version}-${release}*.src.rpm /result/
			cp -aR $build_base_dir/RPMS/$arch/${pkgname}-${version}-${release}.*.rpm /result/
		'
}

do_build() {
	rel=$1

	expr $rel "+" 10 &> /dev/null
	if [[ $? -ne 0 ]];then
		echo "please make sure that release number is valid ${rel}"
		exit 1
	fi

	# Root required for local and mock; docker build runs as root inside container
	if [[ "$build_type" != "container" && "$(id -u)" != 0 ]]; then
		echo "Must be root user for local/mock build, or use build_type=docker for container build"
		return
	fi

	if [[ -n $rel ]];then
		release=$rel
		sed "s/^%global pkg_rel.*$/%global pkg_rel $rel/g" ${SHELL_FOLDER}/${pkgname}.spec -i
	fi

	case "$build_type" in
		local) local_build $rel ;;
		mock)   mock_build $rel ;;
		container)  container_build $rel ;;
		*)
			echo "Unknown build_type: $build_type, must be local|mock|container"
			exit 1
			;;
	esac
}

clean() {
	echo "rm -rf $builddir"
	rm -rf $builddir
	
	echo "rm -rf $resultdir"
	rm -rf $resultdir
}

init() {
	os=$1
	if [[ -z "$os" ]]; then
		if [[ -f /etc/.kyinfo ]]; then
			os=$(cat /etc/.kyinfo | grep milestone | awk -F= '{print $2}' | tr -d ' ' | cut -c1-6)
			if [[ "${os:0:2}" == "3-" ]]; then
				TMP_VERSION=$(cat /etc/.kyinfo | sed 's/ //g' | grep dist_id | tr '-' '\n' | tail -n 2 | head -n 1)
				if [[ "$TMP_VERSION" == "44"* ]]; then
					os="3.4-4"
				elif [[ "$TMP_VERSION" == "36"* ]]; then
					os="3.3-6"
				fi
			else
				[[ ${os: -1} =~ [0-9] ]] || os=${os:0:-1}
			fi
		elif [[ "$build_type" == "docker" ]]; then
			echo "When using build_type=docker on non-Kylin host, please specify Kylin version: $0 --build-type=docker <pkgversion> <3.3-6|3.4-4>"
			exit 1
		fi
	fi

	if [[ -z "$os" || ! " ${allow_os[@]} " =~ " ${os} " ]]; then
