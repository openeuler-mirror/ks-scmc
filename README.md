# 麒麟信安安全容器魔方

提供精简、高效、安全的容器及管理.

## 编译安装

```
# yum install git golang make protobuf-compiler docker-engine mysql5-server mysql5 socat
# cd backend/
# make
# make install
```

## 运行

后端运行

```
systemctl enable --now docker.service
systemctl enable --now mysqld.service

bash /etc/ks-scmc/setup_config.sh /etc/ks-scmc/server.toml

systemctl start ks-scmc-agent.service
systemctl start ks-scmc-controller.service
```

## 日志

日志在/var/log/ks-scmc/目录

