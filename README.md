# 麒麟信安安全容器魔方

提供精简、高效、安全的容器及管理.

## 开源协议

本项目采用[木兰宽松许可证第2版（Mulan PSL v2）](http://license.coscl.org.cn/MulanPSL2)开源。

```
Copyright (c) 2022 KylinSec
ks-scmc is licensed under Mulan PSL v2.
You can use this software according to the terms and conditions of the Mulan PSL v2.
You may obtain a copy of Mulan PSL v2 at:
         http://license.coscl.org.cn/MulanPSL2
THIS SOFTWARE IS PROVIDED ON AN "AS IS" BASIS, WITHOUT WARRANTIES OF ANY KIND,
EITHER EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO NON-INFRINGEMENT,
MERCHANTABILITY OR FIT FOR A PARTICULAR PURPOSE.
See the Mulan PSL v2 for more details.
```

完整协议文本见仓库根目录 [LICENSE](./LICENSE) 文件。

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

