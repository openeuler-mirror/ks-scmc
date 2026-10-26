#!/bin/bash
set -ex

CURRENT_DIR=$(cd `dirname $0`; pwd)
cd $CURRENT_DIR

pkg_name=$(cat $CURRENT_DIR/buildrpm/ks-scmc.spec | grep " pkg_name " | awk '{print $3}')
pkg_ver=$(cat $CURRENT_DIR/buildrpm/ks-scmc.spec | grep " pkg_ver " | awk '{print $3}')
pkg_rel=$(cat $CURRENT_DIR/buildrpm/ks-scmc.spec | grep " pkg_rel " | awk '{print $3}')

export PACKAGE_NAME=$pkg_name
export VERSION=$pkg_ver
export RELEASE=$pkg_rel

TAG=${VERSION}-${RELEASE}
DOCKERFILE="Dockerfile.x86"
ARCH="x86_64"

create_output_dir() {
        export SULT_DIR=docker/`date '+%Y%m%d'`
        export RESULT_DIR=`date '+%Y%m%d'`
        mkdir -p ${SULT_DIR}/dockering ${SULT_DIR}/rpms
}

build_rpm_file() {
        CURRENT_DIR=$(cd `dirname $0`; pwd)
        if [ ! -f buildrpm/_buildres/${PACKAGE_NAME}-${VERSION}-${RELEASE}.*.${ARCH}.rpm ] && [ ! -f buildrpm/_buildres/${PACKAGE_NAME}-${VERSION}-${RELEASE}.*.src.rpm ]; then
                buildrpm/build.sh
        fi
        
        cp buildrpm/_buildres/${PACKAGE_NAME}-${VERSION}-${RELEASE}.*.${ARCH}.rpm ${SULT_DIR}/rpms
        
}

build_docker_image() {
        BUILD_DIR=build_${RESULT_DIR}
        FTP_DIR=${RESULT_DIR}
        mkdir -p ${BUILD_DIR}
        pushd .
        cp -r docker/.kyinfo ${BUILD_DIR}
        cp -r docker/* ${BUILD_DIR}
	cp -r scripts/deploy ${BUILD_DIR}
        cp buildrpm/_buildres/${PACKAGE_NAME}-${VERSION}-${RELEASE}.*.${ARCH}.rpm ${BUILD_DIR}
        cd ${BUILD_DIR}
        	
        echo "构建image"
        docker build -f ${DOCKERFILE}  --build-arg TIME=${FTP_DIR} --build-arg VERSION=${VERSION}-${RELEASE}  -t ${PACKAGE_NAME}:${TAG} .
        echo "输出tar包"
        DOCKER_OUTPUT=${SULT_DIR}/dockering
        docker save ${PACKAGE_NAME}:${TAG} | gzip > ../${DOCKER_OUTPUT}/${PACKAGE_NAME}-${VERSION}-${RELEASE}-${RESULT_DIR}.tar.gz
        cd $CURRENT_DIR/$DOCKER_OUTPUT
        md5sum ${PACKAGE_NAME}-${VERSION}-${RELEASE}-${RESULT_DIR}.tar.gz > ${PACKAGE_NAME}-${VERSION}-${RELEASE}-${RESULT_DIR}.tar.gz.md5sum
        popd
        rm -fr ${BUILD_DIR}
        exit
}

case $1 in
"clean")
        rm buildrpm/buildres/*
        docker images | grep -E "$pkg_name|<none>" | awk '{print $3}' | xargs docker rmi || true
        rm -fr ${PACKAGE_NAME}.tar.gz
        exit 0
        ;;
*)
        create_output_dir
        build_rpm_file
        build_docker_image
        ;;
esac


