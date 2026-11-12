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
done  # 以下两段原为 ksimc=1 / grub2-mkconfig，已整段加 # 停用，行数未改

#if [[ "${without_kernel:-0}" != "1" ]]; then
#    paratest=$(grep -E '^GRUB_CMDLINE_LINUX=.*ksimc=1' /etc/default/grub)
#    if [ -z "$paratest" ]; then
#        sed -i '/^GRUB_CMDLINE_LINUX=/ s/"$/ ksimc=1"/' /etc/default/grub
#    fi
#fi

lines=43
tail -n +$lines $0 > /opt/ks-run-tmp.tar.gz
tar -xf /opt/ks-run-tmp.tar.gz -C /opt

KS_PATH=/opt/ks-run-tmp

chmod +x ${KS_PATH}/install.sh && ${KS_PATH}/install.sh ${KS_PATH} KS_PROJECK_NAME $without_kernel

#if [[ "${without_kernel:-0}" != "1" ]]; then
#	for i in grub2 grub2-efi
#	do
#		cfg=/etc/$i.cfg
#		if [ -e $cfg ]; then
#			grub2-mkconfig -o $cfg
#		fi
#	done
#fi

rm -rf ${KS_PATH} /opt/ks-run-tmp.tar.gz
exit 0
