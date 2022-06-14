#/bin/bash
set -e
WORKDIR="/tmp/"
DB_PASSWD="123456"
CONN_STR="/usr/bin/mysql -uroot -p${DB_PASSWD}"
MYSQLCONF=$PWD/mysql5-server.cnf

function check_ip() {
    IP=$1
    VALID_CHECK=$(echo $IP | awk -F. '$1<=255 && $2<=255 && $3<=255 && $4<=255 {print "yes"}')
    if echo $IP | grep -E "^[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}$" >/dev/null; then
        if [[ $VALID_CHECK == "yes" ]]; then
            echo "$IP available."
        else
            echo "$IP not available!"
            exit 1
        fi
    else
        echo "$IP Format error!"
        exit 1
    fi
}

function generat_config() {
    cat >>$MYSQLCONF <<EOF
#
# These groups are read by MySQL server.
# Use it for options that only the server (but not clients) should see
#
# See the examples of server my.cnf files in /usr/share/mysql/
#

# this is read by the standalone daemon and embedded servers
[server]

# this is only for the mysqld standalone daemon
# Settings user and group are ignored when systemd is used.
# If you need to run mysqld under a different user or group,
# customize your systemd unit file for mysqld/mariadb.
[mysqld]
datadir=/var/lib/mysql
socket=/var/lib/mysql/mysql.sock
log-error=/var/log/mysqld.log
pid-file=/run/mysqld/mysqld.pid

server-id = 2
log-bin = mysql-bin
binlog-format=ROW
auto-increment-increment=2
auto-increment-offset=2
slave-skip-errors=all
gtid-mode=ON
enforce-gtid-consistency=ON


# this is only for embedded server
[embedded]
EOF
}

function modifyServerId() {
    echo "modify server-id on $HOSTAIP and restart mysql:"
    #修改server-id为2并重启数据库
    sed -i 's/^server-id.*$/server-id = 1/g' mysql5-server.cnf
    sed -i 's/^auto-increment-offset.*$/auto-increment-offset=1/g' mysql5-server.cnf
    scp mysql5-server.cnf root@$HOSTAIP:/etc/my.cnf.d/
    ssh root@$HOSTAIP "systemctl restart mysqld"
    echo "modify server-id on $HOSTBIP and restart mysql:"
    #修改server-id为2并重启数据库
    sed -i 's/^server-id.*$/server-id = 2/g' mysql5-server.cnf
    sed -i 's/^auto-increment-offset.*$/auto-increment-offset=2/g' mysql5-server.cnf
    scp mysql5-server.cnf root@$HOSTBIP:/etc/my.cnf.d/
    ssh root@$HOSTBIP "systemctl restart mysqld"
    [[ $? -eq 0 ]] || exit 1
}

function addroot() {
    cat >>$WORKDIR/2root-defaul.sql <<EOF
GRANT ALL PRIVILEGES ON *.* TO 'root'@'%' IDENTIFIED BY '${DB_PASSWD}' with grant option;
FLUSH PRIVILEGES;
EOF
    scp $WORKDIR/2root-defaul.sql root@$HOSTAIP:/tmp/
    ssh root@$HOSTAIP "mysql -uroot < /tmp/2root-defaul.sql"
    scp $WORKDIR/2root-defaul.sql root@$HOSTBIP:/tmp
    ssh root@$HOSTBIP "mysql -uroot < /tmp/2root-defaul.sql"

}

function checkDefaultSql() {
    if [ ! -f $WORKDIR/2master-defaul.sql ]; then
        echo "file not exist: $WORKDIR/2master-defaul.sql "
        echo "creating file:$WORKDIR/2master-defaul.sql"
        cat >>$WORKDIR/2master-defaul.sql <<EOF
GRANT REPLICATION SLAVE ON *.* TO backup@'masterip' IDENTIFIED BY 'backup';
CHANGE MASTER TO MASTER_HOST='masterip',MASTER_USER='backup',MASTER_PASSWORD='backup',MASTER_LOG_FILE='log_file',MASTER_LOG_POS=log_pos;
start slave;
flush privileges;
EOF
        echo "created file done"
    fi
}

function configMaster() {
    localip=$1
    masterip=$2
    #获取master status的状态值
    status_a=$($CONN_STR -h $masterip -e "show master status" | sed '1d')
    [[ ! -n "$status_a" ]] && exit 1
    log_file=$(echo ${status_a} | awk '{print $1}') #获取log_file
    log_pos=$(echo ${status_a} | awk '{print $2}')  #获取log_pos
    echo "Generating master config file..."
    cp $WORKDIR/2master-defaul.sql $WORKDIR/2master-$localip.sql
    #替换的sql中masterip,log_file,log_pos为真实值
    sed -i -e "s/masterip/$masterip/g" -e "s/log_file/$log_file/g" -e "s/log_pos/$log_pos/g" $WORKDIR/2master-$localip.sql
    echo "configing master and privileges on $1..."
    #向mysql注入配置命令
    $CONN_STR -h $localip <$WORKDIR/2master-$localip.sql
    if [ $? -eq 0 ]; then
        echo "config on $localip success"
    else
        echo "config on $localip fail"
        exit 1
    fi
}

function checkSlaveStatus() {
    HOSTAIP_s=$($CONN_STR -h $HOSTAIP -e "show slave status\G;" | grep "Slave_IO_Running")
    HOSTBIP_s=$($CONN_STR -h $HOSTBIP -e "show slave status\G;" | grep "Slave_IO_Running")
    echo $HOSTAIP_s
    echo $HOSTBIP_s
    [[ "$HOSTAIP_s" =~ "Yes" ]] && [[ "$HOSTBIP_s" =~ "Yes" ]] && echo "2master between $HOSTAIP and $HOSTBIP is running now"
}

function read_ip() {
    read -p "请输入mysql节点IP: " HOSTAIP
    check_ip $HOSTAIP
    read -p "请输入另一台mysql节点IP: " HOSTBIP
    check_ip $HOSTBIP
}

function install() {
    read_ip
    generat_config

    IPNUM=1
    for HOSTIP in $HOSTAIP $HOSTBIP; do
        if [ ! -f ${HOME}/.ssh/id_rsa ]; then
            ssh-keygen
        fi
        ssh-copy-id root@${HOSTIP}
        echo "install mysql"
        ssh root@${HOSTIP} "yum install -y mysql5-server mysql5"
    done

    modifyServerId
    checkDefaultSql
    addroot
    configMaster $HOSTAIP $HOSTBIP
    configMaster $HOSTBIP $HOSTAIP

    checkSlaveStatus
}

function clean() {
    read_ip
    for HOSTIP in $HOSTAIP $HOSTBIP; do
        if [ ! -f ${HOME}/.ssh/id_rsa ]; then
            ssh-keygen
        fi

        ssh-copy-id root@${HOSTIP}
        echo "remove mysql"
        ssh root@${HOSTIP} "yum remove -y mysql5-server mysql5;rm -fr /etc/my.cnf.d/;rm -fr /var/lib/mysql"
    done
}

echo "----------------------------------"
echo "在两台主机上部署mysql，
        设置mysql双主高可用。"
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
