#!/bin/sh

syncDir=/var/lib/ks-scmc/images/
syncIP=127.0.0.1
syncModule=scmc-images
xinetdCfg=/etc/xinetd.d/rsync
rsyncdCfg=/etc/rsyncd.conf
lsyncdCfg=/etc/lsyncd.conf

config_lsyncd() {
ls -l ${lsyncdCfg}.bak > /dev/null 2>&1 || cp ${lsyncdCfg} ${lsyncdCfg}.bak > /dev/null 2>&1

cat > ${lsyncdCfg} <<EOF
----
-- User configuration file for lsyncd.
--
-- Simple example for default rsync, but executing moves through on the target.
--
-- For more examples, see /usr/share/doc/lsyncd*/examples/
--
-- sync{default.rsyncssh, source="/var/www/html", host="localhost", targetdir="/tmp/htmlcopy/"}

settings{
    logfile = "/var/log/lsyncd/lsyncd.log",
    statusFile = "/var/log/lsyncd/lsyncd.stat",
    statusInterval = 15,
    maxProcesses = 5,
}

sync{
    default.rsync,
    delay = 5,
    source="${syncDir}",
    target="${syncIP}::${syncModule}",
    exclude='.*',

    rsync = {
        archive = true,
        compress = true,
        update = true,
    }
}

EOF

    echo "写数据到${lsyncdCfg}，返回 $?"
    echo "请确认两台机器都已执行到此步骤，否则lsyncd服务将启动失败，确认请输入：yes"
    while true
    do
        read tips
        if [[ "x$tips" != "x" ]]; then
            tmp=$(echo $tips | tr [a-z] [A-Z])
            if [[ $tmp == "YES" ]]; then
                break
            fi
        fi
    done

    systemctl enable lsyncd.service > /dev/null
    systemctl restart lsyncd.service > /dev/null
    systemctl status lsyncd.service > /dev/null
    if [[ $? -ne 0 ]]; then
        echo "启动lsyncd服务失败，请检查"
        return 1
    fi
}

config_rsyncd() {
ls -l ${rsyncdCfg}.bak > /dev/null 2>&1 || cp ${rsyncdCfg} ${rsyncdCfg}.bak > /dev/null 2>&1

cat > ${rsyncdCfg} <<EOF
# /etc/rsyncd: configuration file for rsync daemon mode

# See rsyncd.conf man page for more options.

# configuration example:

# uid = nobody
# gid = nobody
# use chroot = yes
# max connections = 4
# pid file = /var/run/rsyncd.pid
# exclude = lost+found/
# transfer logging = yes
# timeout = 900
# ignore nonreadable = yes
# dont compress   = *.gz *.tgz *.zip *.z *.Z *.rpm *.deb *.bz2

# [ftp]
#        path = /home/ftp
#        comment = ftp export area

[${syncModule}]
path        = ${syncDir}
hosts allow = ${syncIP}
hosts deny  = *
list        = true
uid         = root
gid         = root
read only   = false
EOF

    echo "写数据到${rsyncdCfg}，返回 $?"
    systemctl enable rsyncd.service > /dev/null
    systemctl restart rsyncd.service > /dev/null
    systemctl status rsyncd.service > /dev/null
    if [[ $? -ne 0 ]]; then
        echo "启动rsyncd服务失败，请检查"
        return 1
    fi

    rsync -auvz ${syncDir} ${syncIP}::${syncModule} > /dev/null 2>&1
    rsync -auvz ${syncIP}::${syncModule} ${syncDir} > /dev/null 2>&1

    return 0
}

config_xinetd() {
ls -l ${xinetdCfg}.bak > /dev/null 2>&1 || cp ${xinetdCfg} ${xinetdCfg}.bak > /dev/null 2>&1

cat > ${xinetdCfg} <<EOF
#default:off
#description: The rsync server is a good addition to an ftp server,as it \
#allows crc checksumming etc
service rsync
{
   disable           = no
   flags             = IPv4
   socker_type       = stream
   wait              = no
   user              = root
   server            = /usr/bin/rsync
   server_args       = --daemon
   log_on_failure    += USERID
}
EOF
    echo "写数据到${xinetdCfg}，返回 $?"
    systemctl enable xinetd > /dev/null
    systemctl restart xinetd > /dev/null
    systemctl status xinetd > /dev/null
    if [[ $? -ne 0 ]]; then
        echo "启动xinetd服务失败，请检查"
        return 1
    fi
}

config_envs() {
    rpm -qa | grep xinetd > /dev/null || yum -y install xinetd
    if [[ $? -ne 0 ]]; then
        echo "安装xinetd失败"
        return 1
    fi

    rpm -qa | grep lsyncd > /dev/null || yum -y install lsyncd
    if [[ $? -ne 0 ]]; then
        echo "安装lsyncd失败"
        return 1
    fi

    mkdir -p $syncDir
    if [[ $? -ne 0 ]]; then
        echo "创建目录$syncDir失败"
        return 1
    fi
}

check_ip() {
    ip=$@
    regex="^\b(25[0-5]|2[0-4][0-9]|1[0-9][0-9]|[1-9][0-9]|[1-9])\.(25[0-5]|2[0-4][0-9]|1[0-9][0-9]|[1-9][0-9]|[0-9])\.(25[0-5]|2[0-4][0-9]|1[0-9][0-9]|[1-9][0-9]|[0-9])\.(25[0-5]|2[0-4][0-9]|1[0-9][0-9]|[1-9][0-9]|[1-9])\b$"
    chk=`echo $ip | grep -E $regex | wc -l`
    if [[ $chk -eq 0 ]];then
       log "invalid ip address: $ip"
        return 1
    fi
}

entry() {
    echo "请输入允许同步的机器的ip (另外一台服务器的ip，确认机器已启动，例如：192.168.1.1)"
    while true
    do
        read hostIP
        if [[ "x$hostIP" != "x" ]]; then
            check_ip $hostIP
            if [[ $? -eq 0 ]]; then
                break
            fi
        fi
        echo "请重新输入"
    done

    config_envs
    if [[ $? -ne 0 ]]; then
        echo "搭建环境出错"
        return 1
    fi

    config_xinetd
    if [[ $? -ne 0 ]]; then
        echo "配置xinetd服务出错"
        return 1
    fi

    syncIP=$hostIP
    echo "同步到主机：$syncIP"

    config_rsyncd
    if [[ $? -ne 0 ]]; then
        echo "配置rsyncd服务出错"
        return 1
    fi

    config_lsyncd
    if [[ $? -ne 0 ]]; then
        echo "配置lsyncd服务出错"
        return 1
    fi

    echo "配置完成"
    echo "注意：1.两台机器都配置完成后，请检查lsyncd服务状态是否正在运行，如出错请重启lsyncd服务"
    echo "注意：2.同步有问题，请分别查看两台机器的lsyncd服务是否报错，如出错请重启lsyncd服务"
}

entry
