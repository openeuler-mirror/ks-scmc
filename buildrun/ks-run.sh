#!/bin/bash

without_kernel=0

for arg in $*; do
    if [[ $arg == "-h" || $arg == "--help" ]];then
        echo "$0 [-h|--help|--without-kernel]"
