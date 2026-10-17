
## 1 编程环境配置

1. 安装 python pip

```bash
yum install python3 python3-pip
```

2. 安装 grpc 相关包

```bash
pip3 install deps/*
```

3. 生成协议代码

```bash
python3 -m grpc_tools.protoc -I./ --python_out=. --grpc_python_out=. *.proto
```

## 2 后台生成 API TOKEN

在后台服务的mysql数据库内向表 `api_tokens` 插入一条记录，`name` 和 `token` 会在后面的代码中用到。

```sql
USE ks-scmc;
INSERT INTO api_tokens (`name`, `token`) VALUES ('test', uuid());

mysql> SELECT * FROM api_tokens;
+----+------+--------------------------------------+
| id | name | token                                |
+----+------+--------------------------------------+
|  1 | test | f4a2bcb9-5c0f-11ee-98de-080027d46de0 |
+----+------+--------------------------------------+
```

## 3 编写客户端代码

`client.py` 中的 `RpcClient` 类封装了四个日志相关的接口调用

- `query_runtime_log` 查询审计事件日志
- `query_warn_log` 查询告警事件日志
- `set_warn_log_read` 设置告警事件日志已读
- `subscribe_warn_log` 订阅告警事件日志


示例代码如下

```python
from sdk import RpcClient

# name/token 记录在后台mysql数据库 api_tokens表中
cli = RpcClient(addr='127.0.0.1:10050',
                name='test',
                token='5c64097e-5cc1-11ee-8e17-08002753d522')

import time
now = int(time.time())

# query_runtime_log 查询审计事件日志
reply = cli.query_runtime_log(start_time=0, end_time=now, page_size=10, page_no=1)
print('========== runtime log ==========')
print(reply)

# query_warn_log 查询告警事件日志
reply = cli.query_warn_log(page_size=10, page_no=1)
print('========== warn log ==========')
print(reply)

# set_warn_log_read 设置告警事件日志已读
reply = cli.set_warn_log_read([1])
print('========== set warn log read ==========')
print(reply)

# subscribe_warn_log 订阅告警事件日志
# 服务每生成新的告警日志会推送到客户端
result_generator = cli.subscribe_warn_log()
print('========== subscribe warn log ==========')
for data in result_generator:
    print(data)
    break

print('cancel server stream...')
result_generator.cancel()   # 退出订阅，断开连接
```

## MISC

下载 grpc 相关包

```bash
VERSION="1.43.0"
pip3 download -d deps grpcio==${VERSION} grpcio-tools==${VERSION}
```