docker run --name mysql-data --restart=unless-stopped \
    -e MYSQL_ROOT_PASSWORD=123456 -p 13306:3306 -d mysql:8

mycli -h127.0.0.1 -P13306 -uroot -p123456

### cadvisor deploy
CADVISOR_VERSION="v0.36.0"

git clone https://github.com/google/cadvisor && cd cadvisor
git checkout ${CADVISOR_VERSION}
make release

docker run -d -p 8080:8080 --name=cadvisor \
    --restart=always \
    --volume=/:/rootfs:ro \
    --volume=/var/run:/var/run:ro \
    --volume=/sys:/sys:ro \
    --volume=/var/lib/docker/:/var/lib/docker:ro \
    --volume=/dev/disk/:/dev/disk:ro \
    --privileged \
    --device=/dev/kmsg \
    google/cadvisor:${CADVISOR_VERSION} \
        -disable_metrics=udp,advtcp,sched,process,tcp,percpu        
        # -storage_driver=influxdb -storage_driver_db=cadvisor -storage_driver_host=influxdb:8086

yum install -y mysql5-server mysql5

systemctl enable --now mysqld.service

mysql -uroot < ./database.sql

cp ks-scmc-controller.service ks-scmc-agent.service /usr/lib/systemd/system/

systemctl enable --now ks-scmc-agent.service
systemctl enable --now ks-scmc-controller.service
