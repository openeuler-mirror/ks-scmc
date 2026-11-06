#!/bin/bash

without_kernel=0

for arg in $*; do
    if [[ $arg == "-h" || $arg == "--help" ]];then
        echo "$0 [-h|--help|--without-kernel]"
        exit 0
    fi

    if [[ $arg == "--without-kernel" ]];then
        without_kernel=1
    fi
done

lines=26
tail -n +$lines $0 > /opt/ks-run-tmp.tar.gz
tar -xf /opt/ks-run-tmp.tar.gz -C /opt

KS_PATH=/opt/ks-run-tmp

chmod +x ${KS_PATH}/install.sh && ${KS_PATH}/install.sh ${KS_PATH} KS_PROJECK_NAME $without_kernel

rm -rf ${KS_PATH} /opt/ks-run-tmp.tar.gz
exit 0
