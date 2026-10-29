#!/bin/bash
lines=12
tail -n +$lines $0 > /opt/ks-run-tmp.tar.gz
tar -xf /opt/ks-run-tmp.tar.gz -C /opt

KS_PATH=/opt/ks-run-tmp

chmod +x ${KS_PATH}/install.sh && ${KS_PATH}/install.sh ${KS_PATH} KS_PROJECK_NAME KS_ALLOW_OS

rm -rf ${KS_PATH} /opt/ks-run-tmp.tar.gz
exit 0
