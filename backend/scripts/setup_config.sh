#!/usr/bin/bash

cd $(dirname $0)

DB_PASSWD=$(< /dev/urandom tr -dc _A-Z-a-z-0-9 | head -c16)

# create database
mysql -uroot < ./database.sql

# grant user permission
mysql -uroot << EOF
GRANT ALL PRIVILEGES ON \`ksc-mcube\`.*
TO 'ksc-mcube'@'localhost'
IDENTIFIED BY '${DB_PASSWD}';
FLUSH PRIVILEGES;
EOF

# server config file
cat << EOF > $1
-agent-port=10051
-controller-port=10050
-cadvisor-addr=127.0.0.1:8080
-graphic-conf-base=/var/lib/ksc-mcube/containers
-logdir=/var/log/ksc-mcube
-host=0.0.0.0
-mysql-dsn=ksc-mcube:${DB_PASSWD}@tcp(127.0.0.1:3306)/ksc-mcube?charset=utf8mb4&timeout=10s
-stdout=0
-verbose=4
EOF

