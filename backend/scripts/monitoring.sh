docker run -d -p 8086:8086 --name influxdb \
    -v influxdb:/var/lib/influxdb \
    -v influxdb2:/var/lib/influxdb2 \
    -e DOCKER_INFLUXDB_INIT_USERNAME=root \
    -e DOCKER_INFLUXDB_INIT_PASSWORD=root \
    -e DOCKER_INFLUXDB_INIT_ORG=my-org \
    -e DOCKER_INFLUXDB_INIT_BUCKET=my-bucket \
    influxdb:2.0

docker run -d \
    --restart=always \
    -p 8083:8083 -p 8086:8086 \
    --expose 8090 --expose 8099 \
    --name influxdb tutum/influxdb

# CREATE DATABASE "cadvisor"

VERSION=v0.36.0 # use the latest release version from https://github.com/google/cadvisor/releases
docker run -d -p 8080:8080 --name=cadvisor \
    --restart=always \
    --volume=/:/rootfs:ro \
    --volume=/var/run:/var/run:ro \
    --volume=/sys:/sys:ro \
    --volume=/var/lib/docker/:/var/lib/docker:ro \
    --volume=/dev/disk/:/dev/disk:ro \
    --privileged \
    --device=/dev/kmsg \
    --link influxdb:influxdb \
    gcr.io/cadvisor/cadvisor:$VERSION -storage_driver=influxdb -storage_driver_db=cadvisor -storage_driver_host=influxdb:8086


docker run -d --name grafana \
    --restart=always \
    -p 3000:3000 \
    -e INFLUXDB_HOST=influxdb \
    -e INFLUXDB_PORT=8086 \
    -e INFLUXDB_NAME=cadvisor \
    -e INFLUXDB_USER=root \
    -e INFLUXDB_PASS=root \
    --link influxdb:influxdb \
    grafana/grafana

SELECT difference(mean("value"))  / 1000000000 FROM "cpu_usage_total" WHERE ("container_name" = '/') AND time >= now() - 3h and time <= now() GROUP BY time(1m) fill(null);
SELECT difference(mean("value"))  / 1000000000 FROM "cpu_usage_total" WHERE ("container_name" = 'cadvisor') AND time >= now() - 3h and time <= now() GROUP BY time(1m) fill(null);
SELECT difference(mean("value"))  / 1000000000 FROM "cpu_usage_total" WHERE ("container_name" = 'influxdb') AND time >= now() - 3h and time <= now() GROUP BY time(1m) fill(null)

select * from cpu_usage_total where container_name = 'cadvisor'  limit 20
select * from memory_usage where container_name = 'cadvisor'  limit 20


SELECT mean(value) FROM cadvisor.autogen.cpu_usage_system WHERE time >= 1640934601135ms and time <= 1640954795040ms GROUP BY time(1m) fill(null)

