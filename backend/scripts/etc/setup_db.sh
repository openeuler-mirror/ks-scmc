#!/usr/bin/bash

cd $(dirname $0)

set -e

if [[ $# < 1 ]]; then
    echo "Usage: $0 [DB_PASSWORD]"
    exit 1
fi

# default mysql password
DB_PASSWD="$1"

# create database
mysql -uroot < ./database.sql

# grant user permission
mysql -uroot << EOF
GRANT ALL PRIVILEGES ON \`ks-scmc\`.*
TO 'ks-scmc'@'%'
IDENTIFIED BY '${DB_PASSWD}';
FLUSH PRIVILEGES;
EOF
