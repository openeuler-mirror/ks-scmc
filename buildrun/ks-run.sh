#!/bin/bash
lines=12
tail -n +$lines $0 > /tmp/ks-run-tmp.tar.gz
tar -xf /tmp/ks-run-tmp.tar.gz -C /tmp

KS_PATH=/tmp/ks-run-tmp

chmod +x ${KS_PATH}/install.sh && ${KS_PATH}/install.sh ${KS_PATH} KS_PROJECK_NAME

rm -rf ${KS_PATH} /tmp/ks-run-tmp.tar.gz
exit 0
