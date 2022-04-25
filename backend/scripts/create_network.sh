#!/bin/bash

createDockerNicCmd="#!/bin/bash\n"

parse_CIDR() {
    ipAddr=$@
    ipArr=(${ipAddr//// })
    addr=${ipArr[0]}
    mask=${ipArr[1]}

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
    echo "${subnet}"
}

check_subnet() {
    subnet=$@
    iparr=(${subnet//// })
    ip=${iparr[0]}
    mask=${iparr[1]}
    num=`echo $mask | tr -cd "[0-9]"`
    regex="^\b(25[0-5]|2[0-4][0-9]|1[0-9][0-9]|[1-9][0-9]|[1-9])\.(25[0-5]|2[0-4][0-9]|1[0-9][0-9]|[1-9][0-9]|[0-9])\.(25[0-5]|2[0-4][0-9]|1[0-9][0-9]|[1-9][0-9]|[0-9])\.(25[0-5]|2[0-4][0-9]|1[0-9][0-9]|[1-9][0-9]|[0-9])\b$"
    chk=`echo $ip | grep -E $regex | wc -l`
    if [[ $chk -eq 0 ]];then
        echo "invalid ip($ip)"
        return 1
    fi
    if [[ ${#num} -eq 0 ]] || [[ ${#num} -ne ${#mask} ]] || [[ $mask -lt 0 ]] || [[ $mask -gt 32 ]];then
        echo "invalid mask($mask)"
        return 1
    fi
}

get_bridge_param() {
    echo "请输入容器桥接网卡名后缀"
    while true
    do
        read bridgeNicSuffix
        if [[ "x$bridgeNicSuffix" == "x" ]]; then
            echo "请重新输入"
        else
            break
        fi
    done

    echo "请输入容器桥接网卡网段(建议与物理网卡不同网段)"
    while true
    do
        read bridgeSubnet
        if [[ "x$bridgeSubnet" != "x" ]]; then
            echo "正在检查网段(${bridgeSubnet})"
            check_subnet ${bridgeSubnet}
            if [[ $? -eq 0 ]];then
                break
            fi
        fi
        echo "网段不符合要求请重新输入(ip/mask)"
    done

    bridgeNicName="bridge${bridgeNicSuffix}"
    bridgeName="docker-${bridgeNicName}"
    createDockerNicCmd="${createDockerNicCmd}docker network create --internal --driver=bridge --subnet=${bridgeSubnet} -o com.docker.network.bridge.name=${bridgeName} ${bridgeNicName}\n"
}

get_macvlan_param() {
    echo "请输入macvlan网卡名后缀"
    while true
    do
        read macvlanNicSuffix
        if [[ "x$macvlanNicSuffix" == "x" ]]; then
            echo "请重新输入"
        else
            break
        fi
    done

    physicalNicArr=(`find /sys/class/net -mindepth 1 -maxdepth 1 -lname '*virtual*' -prune -o -printf '%f\n'`)
    for ((i=0; i<${#physicalNicArr[*]}; i++))
    do
        physicalNic=${physicalNicArr[$i]}
        macvlanName="${physicalNic}${macvlanNicSuffix}"
        bridgeName="docker-${macvlanName}"

        physicalIPAddr=`ip addr show ${physicalNic} | grep -w inet | awk '{print $2}'`
        physicalSubnet=`parse_CIDR ${physicalIPAddr}`

        createDockerNicCmd="${createDockerNicCmd}docker network create --driver=macvlan --subnet=${physicalSubnet} -o parent=${physicalNic} -o com.docker.network.bridge.name=${bridgeName} ${macvlanName}\n"
    done
}

create_nic() {
    systemctl status docker.service > /dev/null
    if [[ $? -ne 0 ]]; then
        echo "请先启动docker服务"
        return 1
    fi

    get_macvlan_param
    get_bridge_param

    echo "正在创建网络"

    tmpScript="/tmp/docker_network.sh"
    tmp=`echo -e "${createDockerNicCmd}\n"`
cat > ${tmpScript} <<EOF
${tmp}
EOF

    chmod 755 ${tmpScript}
    bash ${tmpScript}

    echo "配置结束"
}

create_nic