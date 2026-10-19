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

使用run包方式安装，首先通过命令`chmod +x ks-scmc-<VERSION>.run` 添加可执行权限。

再运行run包 `./ks-scmc-<VERSION>.run`，当出现以下提示信息时输入2或3将安装后台服务以及依赖

```txt
*****************************************
KS SCMC type
1     ks-scmc-gui
2     ks-scmc
3     ks-scmc-gui && ks-scmc
Please enter the KS SCMC type to install:

```

后台包括了内核更新，安装完成后需要重启一次保证生效。

### 3 配置镜像服务

运行 docker-registry 服务，默认端口5000

```bash
systemctl enable --now docker-registry.service docker-registry-gc.timer
```

查看镜像仓库列表：`curl http://127.0.0.1:5000/v2/_catalog`

查看指定镜像标签列表：`curl http://127.0.0.1:5000/v2/仓库/tags/list`

### 4 配置 Controller 服务

#### 4.1 （可选双节点高可用）mysql双主复制

在一台节点上执行命令，分别在两个节点上安装mysql，创建backup用户，密码为backup，并配置mysql双主模式。

**注意：需要配置防火墙允许mysql节点间的数据同步，或关闭防火墙**

```bash
# 参数用于配置mysql root用户密码
bash /etc/ks-scmc/mysql_double_master.sh "PASSWORD"
```

#### 4.2 （可选双节点高可用）keepalived高可用

在一台节点上执行命令，分别在两个节点上安装keepalived，配置vip（虚拟IP），通过vip实现ks-scmc-controller和mysql的高可用

**注意：需要配置防火墙允许keepalived心跳，或关闭防火墙**

```bash
bash /etc/ks-scmc/keepalived.sh
```

#### 4.3 （可选双节点高可用）镜像文件同步

在两台服务器上都运行`bash /etc/ks-scmc/sync_image.sh`，按提示输入。

#### 4.4 复制镜像签名公钥

密钥生成导出方法请参考[6 镜像签名密钥配置](#jump)

```bash
mkdir -p /var/lib/ks-scmc/images/
cp public-key.txt /var/lib/ks-scmc/images/public-key.txt
```

#### 4.5 mysql数据库配置

```bash
# 启动systemd服务
systemctl enable --now mysqld.service

# 创建数据库表, 数据库授权，数据库密码, 可以依照需要设置
# 如果已配置双节点高可用，只需要在其中一个节点运行这条命令
bash /etc/ks-scmc/setup_db.sh "PASSWORD"
```

为避免数据库密码泄露，配置文件中需要对明文密码进行加密：

首先使用`ks-scmc-encrypt`命令对明文加密

```bash
$ ks-scmc-encrypt "PASSWORD"    # 输入参数明文密码，输出 xxxx 为加密后内容
xxxx
```

修改配置文件 `/etc/ks-scmc/server.toml` 中数据库密码字段

```toml
[mysql]
password = "加密后内容"
```


#### 4.6 启动Controller服务

修改配置文件`/etc/ks-scmc/server.toml`

1. 修改`[registry]`下`addr`字段为在**3 配置镜像服务**配置的镜像服务的地址
2. 修改`[controller]`和`[mysql]`下相关属性，与双节点高可用高配置相关参数一致

修改的内容如下

注意：若配置双节点,需配置virtual-if和virtual-ip

```toml
[controller]
virtual-if = "虚拟IP网卡名"
virtual-ip = "虚拟IP地址"
```

注意：若配置mysql配置双主复制，则mysql IP地址为虚拟IP地址

```toml
[mysql]
addr = "mysql IP地址:3306"

[registry]
addr = "镜像服务IP:PORT"
```

运行systemd服务

```bash
systemctl enable --now ks-scmc-controller.service
```

#### 4.7 创建默认三权用户

打开配置文件`/etc/ks-scmc/server.toml`. 确保`[controller]`下的配置项`super-user`值为`true`，开启超级用户配置

```toml
[controller]
super-user = true
```

如果之前的值是`false`，需要重启controller服务，使配置更改生效

```bash
systemctl restart ks-scmc-controller.service
```

运行以下命令将创建三个管理员用户`sysadm secadm audadm`，密码分别为`sys.1234 sec.1234 aud.1234`。

```bash
bash /etc/ks-scmc/init_users.sh
```

创建完用户之后可以将配置项`super-user`改为`false`，重启controller服务。用于禁止对用户、角色的增删改查操作。

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

```json
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

#### 5.3 启动Agent服务

```bash
systemctl enable --now ks-scmc-agent.service
```

### <span id="jump">6 镜像签名密钥配置</span>

后端（controller服务）保存公钥，服务端导入私钥。

#### 6.1 生成密钥对

```bash
gpg --full-generate-key
```

根据提示输入信息，需要记住ID，在输出结果 `gpg: key C5885179647A3BA4 marked as ultimately trusted` **其中的C5885179647A3BA4** 即是ID。

#### 6.2 导出密钥

```bash
# 导出公钥
gpg --output public-key.txt --export C5885179647A3BA4

# 导出私钥
gpg --output private-key.txt --export-secret-keys C5885179647A3BA4
```

#### 6.3 导入密钥

```bash
gpg --import [密钥文件]
```

#### 6.4 对文件签名

-u 选项用于指定密钥

```bash
gpg -u C5885179647A3BA4 --detach-sign [被签名文件]
```
