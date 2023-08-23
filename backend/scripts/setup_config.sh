#!/usr/bin/bash

cd $(dirname $0)

DB_PASSWD=$(< /dev/urandom tr -dc _A-Z-a-z-0-9 | head -c16)

# create database
mysql -uroot < ./database.sql

# grant user permission
mysql -uroot << EOF
GRANT ALL PRIVILEGES ON \`ks-scmc\`.*
TO 'ks-scmc'@'localhost'
IDENTIFIED BY '${DB_PASSWD}';
FLUSH PRIVILEGES;
EOF

# server config file
cat << EOF > $1
[log]
basedir = "/var/log/ks-scmc"
level   = "info"
stdout  = false

[tls]
enable = false
ca = ""
server_cert = ""
server_key = ""

[agent]
host = "0.0.0.0"
port = 10051
container-extra-data-basedir = "/var/lib/ks-scmc/containers"

[controller]
host = "0.0.0.0"
port = 10050

[mysql]
addr = "localhost:3306"
user = "ks-scmc"
password = "${DB_PASSWD}"
db = "ks-scmc"

[cadvisor]
addr = "127.0.0.1:8080"

[influxdb]
addr = "127.0.0.1:8086"
EOF