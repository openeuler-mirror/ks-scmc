#/bin/bash
set -e

function generat_script() {
    if [[ ! -d $PWD/ks-scmc ]]; then
        mkdir $PWD/ks-scmc -p
    fi
    cat >$PWD/ks-scmc/check_ks-scmc.sh <<EOF
#!/bin/bash
systemctl is-active -q mysqld.service || exit 1
systemctl is-active -q ks-scmc-controller.service || exit 1
EOF
    chmod +x $PWD/ks-scmc/check_ks-scmc.sh
}

function generat_config() {
    global_defs
    IFACE=$KSHAIPNET
    ks-scmc_ha_config
}

function generat_vvrp() {
    cat >>$TMPCONF <<EOF

vrrp_script ${SCRIPT_NAME} {
    script ${SCRIPT_PATH}
    interval 2
    weight -55
    fall 2
    rise 1
}
EOF

    cat >>$TMPCONF <<EOF

vrrp_instance ${INSTANCE_NAME} {
    state ${STATE}
    interface ${IFACE}
    virtual_router_id 51
    priority ${PRIORITY}
    advert_int 1
    authentication {
        auth_type PASS
        auth_pass ax8d93xr4
    }
    virtual_ipaddress {
        ${VIRTUAL_IP}
    }
    track_script {
        ${SCRIPT_NAME}
    }
}
EOF
}

function global_defs() {
    cat >$TMPCONF <<EOF
global_defs {
    router_id $ROUTERID
    script_user root
    enable_script_security
}
EOF
}

function ks-scmc_ha_config() {
    SCRIPT_NAME=check_ks-scmc
    SCRIPT_PATH=/etc/keepalived/ks-scmc/check_ks-scmc.sh
    INSTANCE_NAME=ks-scmc-ha
    VIRTUAL_IP=$KSHAIP
    generat_vvrp
}

function check_ip() {
    IP=$1
    VALID_CHECK=$(echo $IP | awk -F. '$1<=255 && $2<=255 && $3<=255 && $4<=255 {print "yes"}')
    if echo $IP | grep -E "^[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}$" >/dev/null; then
        if [[ $VALID_CHECK != "yes" ]]; then
            echo "$IP not available!"
            exit 1
        fi
    else
        echo "$IP Format error!"
        exit 1
    fi
}

function read_ip() {
    read -p "请输入keepalived节点IP: " HOSTAIP
    check_ip $HOSTAIP
    read -p "请输入另一台keepalived节点IP: " HOSTBIP
    check_ip $HOSTBIP

}

function read_config() {
    read -p "请输入ks-scmc高可用IP: " KSHAIP
    check_ip $KSHAIP
    read -p "请输入ks-scmc高可用IP绑定网卡名[ens3]: " KSHAIPNET
    KSHAIPNET=${KSHAIPNET:-ens3}
}

function install() {
    read_ip
    read_config
    generat_script

    ROUTERID=lb01
    PRIORITY=150
    STATE=MASTER
    TMPCONF=$PWD/keepalivedA.conf

    generat_config

    ROUTERID=lb02
    PRIORITY=100
    STATE=BACKUP
    TMPCONF=$PWD/keepalivedB.conf

    generat_config

    IPNUM=1
    for HOSTIP in $HOSTAIP $HOSTBIP; do
        echo $HOSTIP
        if [ ${IPNUM} -eq 1 ]; then
            echo "添加ssh免密"
            if [ ! -f ${HOME}/.ssh/id_rsa ]; then
                ssh-keygen
            fi
            ssh-copy-id root@${HOSTIP}
            ssh root@${HOSTIP} "yum install -y keepalived net-tools"
            scp keepalivedA.conf root@${HOSTIP}:/etc/keepalived/keepalived.conf
            scp -r ks-scmc root@${HOSTIP}:/etc/keepalived/
            # scp -r mysql root@${HOSTIP}:/etc/keepalived/
            ssh root@${HOSTIP} "systemctl restart keepalived"
            IPNUM=2
        else
            echo "添加ssh免密"
            if [ ! -f ${HOME}/.ssh/id_rsa ]; then
                ssh-keygen
            fi
            ssh-copy-id root@${HOSTIP}
            ssh root@${HOSTIP} "yum install -y keepalived net-tools"
            scp keepalivedB.conf root@${HOSTIP}:/etc/keepalived/keepalived.conf
            scp -r ks-scmc root@${HOSTIP}:/etc/keepalived/
            # scp -r mysql root@${HOSTIP}:/etc/keepalived/
            ssh root@${HOSTIP} "systemctl restart keepalived"
        fi
    done
}

function clean() {
    read_ip
    IPNUM=1
    for HOSTIP in $HOSTAIP $HOSTBIP; do
        echo $HOSTIP
        if [ ${IPNUM} -eq 1 ]; then
            echo "添加ssh免密"
            if [ ! -f ${HOME}/.ssh/id_rsa ]; then
                ssh-keygen
            fi
            ssh-copy-id root@${HOSTIP}
            ssh root@${HOSTIP} "yum remove -y keepalived;rm -fr /etc/keepalived"
            IPNUM=2
        else
            echo "添加ssh免密"
            if [ ! -f ${HOME}/.ssh/id_rsa ]; then
                ssh-keygen
            fi
            ssh-copy-id root@${HOSTIP}
            ssh root@${HOSTIP} "yum remove -y keepalived;rm -fr /etc/keepalived"
        fi
    done
}

echo "----------------------------------"
echo "在两台主机上部署keepalivd，
        设置ks-scmc高可用和mysql高可用。"
echo "please enter your choise:"
echo "(1) install"
echo "(2) clean"
echo "(0) Exit Menu"
echo "----------------------------------"
read -t 5 input
input=${input:-0}

case $input in
1)
    echo "install"
    install
    ;;
2)
    echo "clean"
    clean
    ;;
0)
    exit
    ;;
esac
