#!/bin/bash

#variable
logpath=/var/log/ks-scmc/network
declare -A macvlanAddr=(["ens3"]="192.168.122.2/24" ["ens8"]="10.111.2.2/24")
macvlanNicSuffix="mac0"
bridgeSubnet="172.20.0.0/16"
bridgeNicName="bridge0"
jsonFile="/var/lib/ks-scmc/networks/container-ipaddr.json"

log() {
    # max log file size: 1MB
    filesize=$(ls -l $logpath 2>/dev/null | awk '{print $5}')
    if [[ $filesize -gt 1000000 ]];then
        echo "" > $logpath
    fi
    echo "$(date): $*" >> $logpath
}

restore_connect() {
    data=`cat /var/lib/ks-scmc/networks/container-ipaddr.json > /dev/null 2>&1`
    if [[ $? -ne 0 ]]; then
        return
    fi

    info=(`echo $data |grep -Po 'ForShell[" :]+\K[^"]+'`)
    for (( i=0; i<${#info[*]}; i++ ))
    do
        str=${info[$i]}
        arr=(${str//// })
        if [[ ${#arr[*]} -eq 4 ]]; then
            docker network disconnect ${arr[1]} ${arr[0]}
            docker network connect --ip=${arr[2]} ${arr[1]} ${arr[0]}
        fi
    done
}

check_ip() {
    ip=$1
    regex="^\b(25[0-5]|2[0-4][0-9]|1[0-9][0-9]|[1-9][0-9]|[1-9])\.(25[0-5]|2[0-4][0-9]|1[0-9][0-9]|[1-9][0-9]|[0-9])\.(25[0-5]|2[0-4][0-9]|1[0-9][0-9]|[1-9][0-9]|[0-9])\.(25[0-5]|2[0-4][0-9]|1[0-9][0-9]|[1-9][0-9]|[1-9])\b$"
    chk=`echo $ip | grep -E $regex | wc -l`
    if [[ $chk -eq 0 ]];then
       log "[check_ip] invalid ip: $ip"
        return 1
    fi
}

check_ipaddr() {
    ipAddr=$1
    iparr=(${ipAddr//// })
    ip=${iparr[0]}
    mask=${iparr[1]}
    log "[check_ipaddr] split the ipaddr $ipAddr: ip=$ip, mask=$mask"

    check_ip $ip
    if [[ $? -ne 0 ]]; then
        return 1
    fi

    num=`echo $mask | tr -cd "[0-9]"`
    if [[ ${#num} -eq 0 ]] || [[ ${#num} -ne ${#mask} ]] || [[ $mask -lt 0 ]] || [[ $mask -gt 32 ]];then
        log "[check_ipaddr] invalid mask: $mask"
        return 1
    fi
}

parse_CIDR() {
    ipAddr=$@
    ipArr=(${ipAddr//// })
    addr=${ipArr[0]}
    mask=${ipArr[1]}
    log "[parse_CIDR] deal $ipAddr: ipaddr=${addr}, mask=${mask}"

    ipv4len=4
    n=$mask
    maskarr=()
    for ((i=0; i<$ipv4len; i++))
    do
        if [[ $n -ge 8 ]];then
            maskarr[$i]=255
            n=`expr $n - 8`
            continue
        fi

        a=$((255>>${n}))
        maskarr[$i]=$((${a}^255))
        n=0
    done

    outarr=()
    addrarr=(${addr//./ })
    ipstr=""
    for ((i=0; i<$ipv4len; i++))
    do
        outarr[$i]=$((${addrarr[$i]}&${maskarr[$i]}))
        ipstr=${ipstr}.${outarr[$i]}
    done

    subnet=`echo $ipstr | awk '{print substr($1,2)}'`
    subnet=$subnet/${mask}
    echo "${subnet} ${addr} ${maskarr[@]}"
}

check_network_segment() {
    subnet=$1
    maskarr=($2 $3 $4 $5)
    ip=$6

    subnetarr=(${subnet//// })
    subnetip=${subnetarr[0]}
    subnetiparr=(${subnetip//./ })
    iparr=(${ip//./ })

    sameflg=1
    for ((j=0; j<${#maskarr[*]}; j++)) #ipv4
    do
        a=$((${subnetiparr[$j]}&${maskarr[$j]}))
        b=$((${iparr[$j]}&${maskarr[$j]}))
        if [[ $a -ne $b ]]; then
            sameflg=0
            break
        fi
    done

    if [[ $sameflg -ne 0 ]]; then
        return 0
    fi

    log "$subnet and $ip are different network segments"
    return 1
}

create_nic() {
    physicalNic=(`find /sys/class/net -mindepth 1 -maxdepth 1 -lname '*virtual*' -prune -o -printf '%f\n'`)
    if [[ ${#physicalNic[*]} -ne ${#macvlanAddr[*]} ]];then
        log "${#physicalNic[*]} -ne ${#macvlanAddr[*]}, param err"
        return 1
    fi

    for ((i=0; i<${#physicalNic[*]}; i++))
    do
        #在宿主机上创建macvlan网卡(网卡ip不能与宿主机物理网卡一样，注意与其他机器网卡冲突)，使容器可以通宿主机
        macvlanName="${physicalNic[$i]}${macvlanNicSuffix}"
        ip link add ${macvlanName} link ${physicalNic[$i]} type macvlan mode bridge
        ip link set ${macvlanName} up
        k=${physicalNic[$i]}
        ip addr add ${macvlanAddr[$k]} dev ${macvlanName}

        #docker network create 创建容器网卡，与宿主机物理网卡同网段
        bridgeName="docker-${macvlanName}"
        ipAddr=`ip addr show ${physicalNic[$i]} | grep -w inet | awk '{print $2}'`
        resarr=(`parse_CIDR ${ipAddr}`)
        subnet=${resarr[0]}
        docker network create --driver=macvlan --subnet=${subnet} -o parent=${physicalNic[$i]} -o com.docker.network.bridge.name=${bridgeName} ${macvlanName}
    done

    # --internal  Restrict external access to the network
    bridgeName="docker-${bridgeNicName}"
    docker network create --internal --driver=bridge --subnet=${bridgeSubnet} -o com.docker.network.bridge.name=${bridgeName} ${bridgeNicName}
}

check_macvlanAddr() {
    physicalNic=(`find /sys/class/net -mindepth 1 -maxdepth 1 -lname '*virtual*' -prune -o -printf '%f\n'`)
    if [[ ${#physicalNic[*]} -ne ${#macvlanAddr[*]} ]];then
        log "${#physicalNic[*]} -ne ${#macvlanAddr[*]}, param err"
        return 1
    fi

    for ((i=0; i<${#physicalNic[*]}; i++))
    do
        k=${physicalNic[$i]}
        if [[ ! -n "${macvlanAddr[$k]}" ]]; then
            log "$k not in macvlanAddr, param err"
            return 1
        fi

        check_ipaddr ${macvlanAddr[$k]}
        if [[ $? -ne 0 ]]; then
            return 1
        fi


        ipAddr=`ip addr show ${physicalNic[$i]} | grep -w inet | awk '{print $2}'`
        resarr=(`parse_CIDR ${ipAddr}`)
        subnet=${resarr[0]}
        maskarr=(${resarr[2]} ${resarr[3]} ${resarr[4]} ${resarr[5]})
        resarr=(`parse_CIDR ${macvlanAddr[$k]}`)
        log "$i ${subnet} ${maskarr[@]} ${resarr[1]}"
        check_network_segment ${subnet} ${maskarr[@]} ${resarr[1]}
        if [[ $? -ne 0 ]];then
            return 1
        fi
    done
}

check_param() {
    check_macvlanAddr
    iparr=(${bridgeSubnet//// })
    ip=${iparr[0]}
    mask=${iparr[1]}
    num=`echo $mask | tr -cd "[0-9]"`
    regex="^\b(25[0-5]|2[0-4][0-9]|1[0-9][0-9]|[1-9][0-9]|[1-9])\.(25[0-5]|2[0-4][0-9]|1[0-9][0-9]|[1-9][0-9]|[0-9])\.(25[0-5]|2[0-4][0-9]|1[0-9][0-9]|[1-9][0-9]|[0-9])\.(25[0-5]|2[0-4][0-9]|1[0-9][0-9]|[1-9][0-9]|[0-9])\b$"
    chk=`echo $ip | grep -E $regex | wc -l`
    if [[ $chk -eq 0 ]];then
       log "${bridgeSubnet}  invalid ip: $ip"
        return 1
    fi
    if [[ ${#num} -eq 0 ]] || [[ ${#num} -ne ${#mask} ]] || [[ $mask -lt 0 ]] || [[ $mask -gt 32 ]];then
        log " ${bridgeSubnet} invalid mask: $mask"
        return 1
    fi
    
    if [[ ${#macvlanNicSuffix} -lt 1 || ${#bridgeNicName} -lt 3 ]]; then 
        log "macvlanNicSuffix or bridgeNicName lens err"
        return 1
    fi

    create_nic
    if [[ $? -ne 0 ]]; then
        echo "create nic failed"
        return 1
    fi

    restore_connect
}


# ip link add ens3mac0 link ens3 type macvlan mode bridge
# ip link set ens3mac0 up
# ip addr add 192.168.122.2/24 dev ens3mac0
# docker network create --driver=macvlan --subnet=192.168.122.0/24 -o parent=ens3 -o com.docker.network.bridge.name=docker-ens3mac0 ens3mac0
# ip link add ens8mac0 link ens8 type macvlan mode bridge
# ip link set ens8mac0 up
# ip addr add 10.111.2.2/24 dev ens8mac0
# docker network create --driver=macvlan --subnet=10.111.2.0/24 -o parent=ens8 -o com.docker.network.bridge.name=docker-ens8mac0 ens8mac0
# docker network create --internal --driver=bridge --subnet=172.20.0.0/16 -o com.docker.network.bridge.name=docker-bridge0 bridge0
check_param