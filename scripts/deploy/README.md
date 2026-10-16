
# 安全容器后台配置脚本

通过 `ansible-playbook` 完成自动化部署，支持两种场景后台服务部署

- 单节点后台服务
- 双节点高可用集群

## 0 安装依赖

需要安装两个python依赖包：

- ansible >= 2.9
- paramiko

以下三种方法任选一种可行的即可。

1. 安装当前目录下的python依赖包：`pip3 install deps/*`
2. 安装RPM包：`yum install ansible python3-paramiko` （确保ansible最低版本2.9）
3. 在线安装pip包：`pip3 install ansible>=2.9 paramiko`

## 1 更新配置文件 `hosts`

### 1.1 主机列表

1. `HOST_A` 和 `HOST_B` 替换为自己环境的IP地址
2. 单节点场景下 `HOST_A` 和 `HOST_B` 只需要保留一个
3. `ansible_password` 是root用户密码
4. 双节点集群场景下必须配置 `other_host` (另一个节点的地址) 和 `server_id`

```yml
hosts:
  HOST_A:
    ansible_password: kylin.2022
    other_host: HOST_B
    server_id: 1
  HOST_B:
    ansible_password: kylin.2022
    other_host: HOST_A
    server_id: 2
```

### 1.2 配置选项

默认配置如下

```yml
vars:
  ansible_user: root
  ansible_connection: paramiko
  cluster:
    enabled: false
    virtual_ip:
    virtual_ip_nic:
  install:
    download_url: ''
    username: ''
    password: ''
    delete: false
    default_kernel: ''
    dest: /root/ks-scmc-packages.run
  mysql:
    root_password: kylin.2023
    scmc_password: kylin@2023
  docker_dir: /var/lib/docker
  image_dir: /var/lib/ks-scmc/images
  image_verify_pubkey: ''
  script_dir: /etc/ks-scmc
```

1. 集群高可用配置项 `cluster`
  - 开关 `enabled`: true 进行双节点高可用集群配置
  - 虚拟IP即网卡 `virtual_ip` `virtual_ip_nic`: 开启集群时必须配置
2. 安装包配置项 `install`
  - 下载链接 `download_url`: 不为空会自动下载
    - 下载链接需要认证信息时可补充 `username` `password`
  - `default_kernel`: 替换默认内核版本
  - `delete`: 安装完成后自动删除安装包文件开关
  - `dest`: 安装包文件位置
3. 数据库配置项 `mysql`
  - `root_password`: 数据库root用户密码
  - `scmc_password`: 后台服务访问数据库的密码
4. `docker_dir`: docker数据存放目录
5. `image_dir`: 上传镜像文件保存的目录
6. `image_verify_pubkey`: 镜像校验公钥文件的路径
7. 其他未标注的配置项不建议修改

## 2 运行自动化配置流程

```bash
ansible-playbook deploy.yml
```

### 节约时间的技巧

有时候一部分配置已经完成，比如已经依赖包已经安装，而且数据库也配好了。可以通过 `--skip-tags` 选项来跳过这两个步骤

```shell
ansible-playbook deploy.yml --skip-tags "install,mysql"
```

可跳过的步骤如下:

- install: 安装包
- mysql: 数据库配置
- image: 镜像服务配置
- vip: 虚拟IP配置

## 3 注意事项

### 3.1 需要执行自动安装重启步骤，需要在服务器之外的机器运行

自动安装重启过程会重启服务器，因此 `ansible-playbook` 命令要在服务器之外的机器上运行

如果是在服务器上运行 `ansible-playbook`, 下载安装包、执行安装、替换默认内核、重启这些步骤都需要手动执行

### 3.2 部署步骤一直卡在重启服务器，实际服务器已完成重启

按 `Ctrl-C` 中止等待

再执行命令 `ansible-playbook deploy.yml --skip-tags "install"` 跳过安装环节继续剩下的流程

### 3.3 系统启用 ks-agent, ansible 运行会卡住

编辑文件 `/etc/bashrc`, 找到 ks-agent 相关的代码行将其注释保存，待安装完成后再恢复

### 3.4 镜像签名GPG密钥

生成密钥

```shell
# 根据提示输入信息，需要记住ID，在输出结果 gpg: key C5885179647A3BA4 marked as ultimately trusted
# 其中的C5885179647A3BA4 即是ID。
gpg --full-generate-key
```

导出公钥

```shell
gpg --output public-key.txt --export C5885179647A3BA4
```

导出私钥

```shell
gpg --output private-key.txt --export-secret-keys C5885179647A3BA4
```

导入密钥

```shell
gpg --import private-key.txt
```

对文件签名

```shell
# -u 指定密钥ID
gpg -u C5885179647A3BA4 --detach-sign [被签名文件]
```
