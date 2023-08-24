#/bin/bash
WORKDIR="/tmp/"
DB_PASSWD="123456"
CONN_STR="/usr/bin/mysql -uroot -p${DB_PASSWD}"
IP1=$1
IP2=$2
IS_LOCALIP=false

function modifyServerId() {
	echo "modify server-id on $IP1 and restart mysql:"
	#修改server-id为2并重启数据库
	sed -i 's/^server-id.*$/server-id = 1/g' mysql5-server.cnf
	sed -i 's/^auto-increment-offset.*$/auto-increment-offset=1/g' mysql5-server.cnf
	scp mysql5-server.cnf root@$IP1:/etc/my.cnf.d/
	ssh root@$IP1 "systemctl restart mysqld"
	echo "modify server-id on $IP2 and restart mysql:"
	#修改server-id为2并重启数据库
	sed -i 's/^server-id.*$/server-id = 2/g' mysql5-server.cnf
	sed -i 's/^auto-increment-offset.*$/auto-increment-offset=2/g' mysql5-server.cnf
	scp mysql5-server.cnf root@$IP2:/etc/my.cnf.d/
	ssh root@$IP2 "systemctl restart mysqld"
	[[ $? -eq 0 ]] || exit 1
}

function addroot() {
	cat >>$WORKDIR/2root-defaul.sql <<EOF
GRANT ALL PRIVILEGES ON *.* TO 'root'@'%' IDENTIFIED BY '${DB_PASSWD}' with grant option;
FLUSH PRIVILEGES;
EOF
	scp $WORKDIR/2root-defaul.sql root@$IP1:/tmp/
	ssh root@$IP1 "mysql -uroot < /tmp/2root-defaul.sql"
	scp $WORKDIR/2root-defaul.sql root@$IP2:/tmp
	ssh root@$IP2 "mysql -uroot < /tmp/2root-defaul.sql"

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
	ip1_s=$($CONN_STR -h $IP1 -e "show slave status\G;" | grep "Slave_IO_Running")
	ip2_s=$($CONN_STR -h $IP2 -e "show slave status\G;" | grep "Slave_IO_Running")
	echo $ip1_s
	echo $ip2_s
	[[ "$ip1_s" =~ "Yes" ]] && [[ "$ip2_s" =~ "Yes" ]] && echo "2master between $IP1 and $IP2 is running now"
}

IPNUM=1
for HOSTIP in $*; do
	IS_LOCALHOST ${HOSTIP}
	if ${IS_LOCALIP}; then
		echo "install mysql"
		yum install -y mysql5-server mysql5
	else
		if [ ! -f ${HOME}/.ssh/id_rsa ]; then
			ssh-keygen
		fi
		ssh-copy-id root@${HOSTIP}
		echo "install mysql"
		ssh root@${HOSTIP} "yum install -y mysql5-server mysql5"
	fi
done

modifyServerId
checkDefaultSql
addroot
configMaster $1 $2
configMaster $2 $1

checkSlaveStatus
