#/bin/bash
HOSTAIP=$1
HOSTBIP=$2
IS_LOCALIP=false

function IS_LOCALHOST() {
  machine_ips=$(ip addr | grep 'inet' | grep -v 'inet6\|127.0.0.1' | grep -v grep | awk -F '/' '{print $1}' | awk '{print $2}')

  for machine_ip in ${machine_ips}; do
    if [[ "X${machine_ip}" == "X$1" ]]; then
      IS_LOCALIP=true
    fi
  done

}

IPNUM=1
for HOSTIP in $*; do
  echo $HOSTIP
  IS_LOCALHOST ${HOSTIP}
  if ${IS_LOCALIP}; then
    yum install -y keepalived
    cp keepalivedA.conf /etc/keepalived/keepalived.conf
    systemctl restart keepalived
  else
    if [ ${IPNUM} -eq 1 ]; then
      if [ ! -f ${HOME}/.ssh/id_rsa ]; then
        ssh-keygen
      fi
      ssh-copy-id root@${HOSTIP}
      ssh root@${HOSTIP} "yum install -y keepalived"
      scp keepalivedA.conf root@${HOSTIP}:/etc/keepalived/keepalived.conf
      ssh root@${HOSTIP} "systemctl restart keepalived"
      IPNUM=2
    else
      if [ ! -f ${HOME}/.ssh/id_rsa ]; then
        ssh-keygen
      fi
      ssh-copy-id root@${HOSTIP}
      ssh root@${HOSTIP} "yum install -y keepalived"
      scp keepalivedB.conf root@${HOSTIP}:/etc/keepalived/keepalived.conf
      ssh root@${HOSTIP} "systemctl restart keepalived"
    fi
  fi
done
