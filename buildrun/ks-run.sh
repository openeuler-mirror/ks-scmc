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
