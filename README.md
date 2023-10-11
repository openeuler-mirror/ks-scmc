# 麒麟信安安全容器魔方

提供精简、高效、安全的容器及管理.

# 后端代码

- cmd: 二进程程序main入口
- common: 日志等公共模块
- model: mysql数据库访问
- rpc: rpc接口协议定义
- server: controller agent 服务实现
- setup: 开发部署用到的脚本
- vendor: 项目依赖包


## 编译安装

```
# yum install git golang make protobuf-compiler docker-engine mysql5-server mysql5 socat
# make
# make install
```

## 配置运行

参考 [INSTALLING.md](./INSTALLING.md)

## 日志

日志在/var/log/ks-scmc/目录

