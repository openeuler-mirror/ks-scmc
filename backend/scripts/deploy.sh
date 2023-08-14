docker run --name mysql-data --restart=unless-stopped \
    -e MYSQL_ROOT_PASSWORD=123456 -p 13306:3306 -d mysql:8

mycli -h127.0.0.1 -P13306 -uroot -p123456

