## 安全容器后端部署说明

### 1. 介绍

#### 1.1 系统要求

- 服务器版本：3.4-4-PG

- 硬件要求：2核CPU，内存2GB以上

- 分区要求：系统有xfs分区

  - 用于保存docker的数据，实现限制单个docker大小的特性

  - 可以在安装系统时选择xfs分区，或在已有的系统加入xfs分区；此外还需要修改`/etc/fstab`文件，在xfs分区的挂载选项中加入pquota选项。

- 环境要求：安装完成后禁用firewalld服务 `systemctl disable --now firewalld.service`

#### 1.2 后台模块介绍

1. agent服务：
   - 维护容器的服务器节点
   - 容器安全功能
   - 数据收集上报
   - 依赖包 ks-scmc kernel cadvisor influxdb opensnitch docker-ce，配置复杂
2. controller服务：
   - 直接对客户端提供服务，分发用户的请求至各个agent节点
   - 其他功能模块包括用户、数据库、镜像、日志等
   - 依赖包 ks-scmc mysql5-server mysql5，配置较简单
   - 可以实现双节点高可用，额外的配置较复杂
3. 镜像服务：
   - controller服务对此推送镜像
   - agent节点从此获取镜像
   - 只依赖docker-registry服务，配置简单

### 2. 安装依赖

解压安装包文件 ks-scmc.zip，进入解压后目录，目录结构如下：
```
.
├── agent
│   ├── cadvisor-0.33.1-5.20190708git2ccad4b.fc32.x86_64.rpm
│   ├── containerd.io-1.4.3-3.1.kb1.ky3.x86_64.rpm
│   ├── container-selinux-2.138.0-1.kb1.ky3.noarch.rpm
│   ├── docker-ce-20.10.7-3.kb3.ky3.x86_64.rpm
│   ├── docker-ce-cli-20.10.7-3.kb2.ky3.x86_64.rpm
│   ├── docker-ce-rootless-extras-20.10.7-3.kb2.ky3.x86_64.rpm
│   ├── docker-scan-plugin-0.8.0-3.kb2.ky3.x86_64.rpm
│   ├── fuse3-3.9.2-4.kb1.ky3.x86_64.rpm
│   ├── fuse-overlayfs-0.7.2-5.kb2.ky3.x86_64.rpm
│   ├── influxdb-1.8.10.x86_64.rpm
│   ├── kernel-4.19.90-2106.3.0.0095.test.kb5.ky3.x86_64.rpm
│   ├── kernel-devel-4.19.90-2106.3.0.0095.test.kb5.ky3.x86_64.rpm
│   ├── libnetfilter_queue-1.0.5-1.kb1.ky3.x86_64.rpm
│   ├── opensnitch-1.5.0-1.x86_64.rpm
│   └── slirp4netns-0.4.2-3.git21fdece.kb2.ky3.x86_64.rpm
├── common
│   ├── keepalived-2.0.20-18.kb1.ky3.x86_64.rpm
│   ├── keepalived-help-2.0.20-18.kb1.ky3.noarch.rpm
│   ├── lsyncd-2.2.3-2.kb1.ky3.x86_64.rpm
│   ├── socat-1.7.3.2-8.kb1.ky3.x86_64.rpm
│   └── xinetd-2.3.15-31.kb1.ky3.x86_64.rpm
├── controller
│   ├── mariadb-common-10.3.9-9.kb1.ky3.x86_64.rpm
│   ├── mecab-0.996-2.kb1.ky3.x86_64.rpm
│   ├── mysql5-5.7.21-3.kb1.ky3.x86_64.rpm
│   ├── mysql5-common-5.7.21-3.kb1.ky3.x86_64.rpm
│   ├── mysql5-errmsg-5.7.21-3.kb1.ky3.x86_64.rpm
│   └── mysql5-server-5.7.21-3.kb1.ky3.x86_64.rpm
├── install_agent.sh
├── install_controller.sh
├── install_registry.sh
├── ks-scmc-1.0.2-1.ky3.x86_64.rpm
└── registry
    └── docker-registry-2.8.1-2.x86_64.rpm
```

#### 2.1 镜像服务

```
rpm -ivh --nodeps registry/*.rpm
```

#### 2.2 controller服务

```
rpm -ivh --nodeps common/*.rpm
rpm -ivh --nodeps controller/*.rpm
rpm -ivh --nodeps ./ks-scmc*.rpm
```

#### 2.3 agent服务

```
rpm -ivh --nodeps common/*.rpm
rpm -ivh --nodeps agent/*.rpm
rpm -ivh --nodeps ./ks-scmc*.rpm
```

更新了内核，需要重启一次

### 3 配置镜像服务

运行 docker-registry 服务，默认端口5000

```
systemctl enable --now docker-registry.service docker-registry-gc.timer
```

查看镜像仓库列表：`curl http://127.0.0.1:5000/v2/_catalog`

查看指定镜像标签列表：`curl http://127.0.0.1:5000/v2/仓库/tags/list`

### 4 配置 Controller 服务

#### 4.1 （可选双节点高可用）mysql双主复制

在一台节点上执行命令，分别在两个节点上安装mysql，创建backup用户，密码为backup，并配置mysql双主模式。

```bash
bash /etc/ks-scmc/mysql_double_master.sh
```

#### 4.2 （可选双节点高可用）keepalived高可用

在一台节点上执行命令，分别在两个节点上安装keepalived，配置vip，通过vip实现ks-scmc-controller和mysql的高可用

```bash
bash /etc/ks-scmc/keepalived.sh
```

#### 4.3 （可选双节点高可用）镜像文件同步

在两台服务器上都运行`bash /etc/ks-scmc/sync_image.sh`，按提示输入。


#### 4.4 复制镜像签名公钥

密钥生成导出方法请参考 **6 镜像签名密钥配置**

```
mkdir -p /var/lib/ks-scmc/images/
cp public-key.txt /var/lib/ks-scmc/images/public-key.txt
```

#### 4.5 mysql数据库配置

```
# 启动systemd服务
systemctl enable --now mysqld.service

# 创建数据库表, 数据库授权 "123456"是数据库密码, 可以依照需要设置
# 如果已配置双节点高可用，只需要在其中一个节点运行这条命令
bash /etc/ks-scmc/setup_db.sh "123456"
```

#### 4.6 启动Controller服务

修改配置文件`/etc/ks-scmc/server.toml`

1. 修改`[registry]`下`addr`字段为在**3 配置镜像服务**配置的镜像服务的地址
2. 修改`[controller]`和`[mysql]`下相关属性，与双节点高可用高配置相关参数一致

修改的内容如下

```
[controller]
virtual-if = "虚拟IP网卡名"
virtual-ip = "虚拟IP地址"

[mysql]
addr = "虚拟IP地址:3306"
password = "4.5中所设置的密码"

[registry]
addr = "镜像服务IP:PORT"
```

运行systemd服务

```
systemctl enable --now ks-scmc-controller.service
```

#### 4.7 创建默认三权用户

首先需要修改配置文件`/etc/ks-scmc/server.toml`. 将`[controller]`下的`check-auth`和`check-perm`值为`false`，避免用户和权限检查导致操作失败

```
[controller]
check-auth = false
check-perm = false
```

重启controller服务，使配置更改生效

```
systemctl restart ks-scmc-controller.service
```

运行以下命令将创建三个管理员用户`sysadm secadm audadm`，密码都是`12345678`。

```
bash /etc/ks-scmc/init_users.sh
```

创建完用户之后可以将修改的配置项`check-auth`和`check-perm`恢复为true，重启controller服务。


### 5 配置 Agent 服务

#### 5.1 运行自动化配置脚本

部分固定的配置已整理到脚本，直接运行即可。

```bash
bash /etc/ks-scmc/setup_agent.sh
```

将自动进行以下配置：

- 开启ip_forward
- 开启SSH X11 forwarding
- 配置influxdb服务
- 配置cadvisor服务
- 配置opensnitch服务
- 开启authz插件


#### 5.2 配置docker

修改docker配置文件`/etc/docker/daemon.json`，内容如下：

```
{
	"live-restore": true,
	"bridge": "none",
	"authorization-plugins": ["authz-plugin"],
	"host": [
		"unix:///var/run/docker.sock"
	],
	"insecure-registries": [
		"127.0.0.1:5000"
	],
	"registry-mirrors": [
		"http://127.0.0.1:5000"
	],
	"data-root": "/var/lib/docker",
	"selinux-enabled": false
}
```

注意：
1. `insecure-registries`和`registry-mirrors`需要对应**3 配置镜像服务**配置的镜像服务的地址
2. `data-root`对应路径所在分区需要保证是xfs分区


保存文件重启docker服务：`systemctl restart docker.service`

#### 5.3 虚拟网卡配置

运行`bash /etc/ks-scmc/create_network.sh`，按提示输入。

#### 5.4 启动Agent服务

```
systemctl enable --now ks-scmc-agent.service
```

### 6 镜像签名密钥配置

后端（controller服务）保存公钥，服务端导入私钥。

#### 6.1 生成密钥对

```
gpg --full-generate-key
```

根据提示输入信息，需要记住ID，在输出结果 `gpg: key C5885179647A3BA4 marked as ultimately trusted` **其中的C5885179647A3BA4** 即是ID。

#### 6.2 导出密钥

```
# 导出公钥
gpg --output public-key.txt --export C5885179647A3BA4

# 导出私钥
gpg --output private-key.txt --export-secret-keys C5885179647A3BA4
```


#### 6.3 导入密钥

```
gpg --import [密钥文件]
```

#### 6.4 对文件签名

-u 选项用于指定密钥

```
gpg -u C5885179647A3BA4 --detach-sign [被签名文件]
```