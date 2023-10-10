
#####################
INFLUX_DATA="/var/lib/influxdb"
mkdir -p ${INFLUX_DATA}
docker run -d -p 8086:8086 \
      -v ${INFLUX_DATA}:${INFLUX_DATA} \
      --name=influxdb \
      --restart=unless-stopped \
      influxdb:1.8.10

docker exec -it influxdb /bin/bash
influx
> CREATE DATABASE "cadvisor";
> CREATE RETENTION POLICY "cadvisor_3d" ON cadvisor DURATION 3d REPLICATION 1 DEFAULT;
#####################

docker run -d -p 8080:8080 --name=cadvisor \
    --restart=unless-stopped \
    --volume=/:/rootfs:ro \
    --volume=/var/run:/var/run:ro \
    --volume=/sys:/sys:ro \
    --volume=/var/lib/docker/:/var/lib/docker:ro \
    --volume=/dev/disk/:/dev/disk:ro \
    --privileged \
    --device=/dev/kmsg \
    --link influxdb:influxdb \
    google/cadvisor:v0.36.0 \
        -storage_driver=influxdb -storage_driver_db=cadvisor -storage_driver_host=influxdb:8086 \
        -disable_metrics=udp,advtcp,sched,process,tcp,percpu


docker run -d --name grafana \
    --restart=unless-stopped \
    -p 3000:3000 \
    -e INFLUXDB_HOST=influxdb \
    -e INFLUXDB_PORT=8086 \
    -e INFLUXDB_NAME=cadvisor \
    -e INFLUXDB_USER=root \
    -e INFLUXDB_PASS=root \
    --link influxdb:influxdb \
    grafana/grafana

SELECT difference(mean("value"))  / 1000000000 FROM "cpu_usage_total" WHERE ("container_name" = '/') AND time >= now() - 3h and time <= now() GROUP BY time(1m) fill(null);
SELECT difference(mean("value"))  / 60000000000 FROM "cpu_usage_total" WHERE ("container_name" = 'cadvisor') AND time >= now() - 3h and time <= now() GROUP BY time(1m) fill(null);
SELECT difference(mean("value"))  / 1000000000 FROM "cpu_usage_total" WHERE ("container_name" = 'influxdb') AND time >= now() - 3h and time <= now() GROUP BY time(1m) fill(null)

select * from cpu_usage_total where container_name = 'cadvisor'  limit 20
select * from memory_usage where container_name = 'cadvisor'  limit 20


SELECT mean(value) FROM cadvisor.autogen.cpu_usage_system WHERE time >= 1640934601135ms and time <= 1640954795040ms GROUP BY time(1m) fill(null)

SELECT mean("value")  / 1000000 FROM "fs_usage" WHERE ("device" = 'overlay') AND time >= now() - 24h and time <= now() GROUP BY time(1m) fill(null)