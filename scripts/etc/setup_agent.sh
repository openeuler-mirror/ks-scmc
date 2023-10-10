#!/usr/bin/bash

### 保证容器可以正常使用网络
echo "开启ip_foward ..."
echo "net.ipv4.ip_forward=1" >> /etc/sysctl.conf
systemctl restart network


### 访问容器的图形界面需要开启 SSH X11 forwarding
echo "sshd 开启 X11 forwarding"
sed -i 's/^X11Forwarding no/X11Forwarding yes/g' /etc/ssh/sshd_config
systemctl restart sshd.service


### influxdb组件用于存储监控历史数据
echo "配置influxdb组件"
ln -s /usr/lib/influxdb/scripts/influxdb.service /usr/lib/systemd/system/influxdb.service
systemctl enable --now influxdb.service
# 配置influxdb数据最多存储7天
influx -execute 'CREATE DATABASE "cadvisor"; CREATE RETENTION POLICY "cadvisor_retention" ON "cadvisor" DURATION 7d REPLICATION 1 DEFAULT;'


### cadvisor组件用于采集监控数据
echo "配置cadvisor组件"
# 修改服务端口
sed -i 's/CADVISOR_PORT="4194"/CADVISOR_PORT="8080"/g' /etc/sysconfig/cadvisor
# 采集数据写入influxdb
sed -i 's/CADVISOR_STORAGE_DRIVER=""/CADVISOR_STORAGE_DRIVER="influxdb"/g' /etc/sysconfig/cadvisor
systemctl enable --now cadvisor.service


### 容器网络进程白名单功能依赖opensnitch
echo "配置opensnitch组件"
cat > /etc/opensnitchd/rules/0001-allow-by-default.json << EOF
{
    "name": "0001-allow-by-default",
    "enabled": true,
    "precedence": false,
    "action": "allow",
    "duration": "always",
    "operator": {
        "type": "simple",
        "sensitive": false,
        "operand": "true"
    }
}
EOF
cat > /etc/opensnitchd/system-fw.json << EOF
{
    "SystemRules": [
        {
            "Rule": {
                "Description": "Allow icmp",
                "Table": "mangle",
                "Chain": "OUTPUT",
                "Parameters": "-p icmp",
                "Target": "ACCEPT",
                "TargetParameters": ""
            }
        },
        {
            "Rule": {
                "Enabled": true,
                "Description": "",
                "Table": "mangle",
                "Chain": "FORWARD",
                "Parameters": "-m conntrack --ctstate NEW",
                "Target": "NFQUEUE",
                "TargetParameters": "--queue-num 0 --queue-bypass"
            }
        }
    ]
}
EOF
systemctl restart opensnitch.service


### 容器启停控制依赖 authz插件
echo "开启authz插件"
systemctl enable --now ks-scmc-authz.service


### 后续操作
echo "请更新docker配置，并配置虚拟网卡"