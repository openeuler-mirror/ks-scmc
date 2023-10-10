package internal

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	client "github.com/influxdata/influxdb1-client/v2"
	log "github.com/sirupsen/logrus"

	"scmc/common"
	pb "scmc/rpc/pb/container"
)

const megaBytes = 1 << 20

func cpuQuery(interval uint) string {
	division := time.Minute * time.Duration(interval)
	f := ` SELECT non_negative_difference(mean("value"))  / %d FROM "cpu_usage_system"
	WHERE ("container_name" = $container) AND time >= $start and time <= $end
	GROUP BY time($interval) fill(previous)`
	return fmt.Sprintf(f, division)
}

func memQuery() string {
	f := `SELECT mean("value") / %d FROM "memory_working_set"
	WHERE ("container_name" = $container) AND time >= $start and time <= $end
	GROUP BY time($interval) fill(0)`
	return fmt.Sprintf(f, megaBytes)
}

func diskQuery() string {
	f := `SELECT mean("value") / %d FROM "fs_usage"
	WHERE ("container_name" = $container) AND time >= $start and time <= $end
	GROUP BY time($interval) fill(0)`
	return fmt.Sprintf(f, megaBytes)
}

func rxQuery() string {
	f := `SELECT non_negative_difference(mean("value")) / %d FROM "rx_bytes"
	WHERE ("container_name" = $container) AND time >= $start and time <= $end
	GROUP BY time($interval) fill(previous);`
	return fmt.Sprintf(f, megaBytes)
}

func txQuery() string {
	f := `SELECT non_negative_difference(mean("value")) / %d FROM "tx_bytes"
	WHERE ("container_name" = $container) AND time >= $start and time <= $end
	GROUP BY time($interval) fill(previous);`
	return fmt.Sprintf(f, megaBytes)
}

func influxdbQuery(start, end int64, interval uint, containerName string) (*pb.MonitorHistoryReply, error) {
	cli, err := client.NewHTTPClient(client.HTTPConfig{
		Addr: "http://" + common.Config.InfluxDB.Addr,
	})
	if err != nil {
		log.Warnf("get influxdb client error=%v", err)
		return nil, err
	}
	defer cli.Close()

	queries := strings.Join([]string{
		cpuQuery(uint(interval)),
		memQuery(),
		diskQuery(),
		rxQuery(),
		txQuery(),
	}, ";")

	q := client.NewQueryWithParameters(queries, "cadvisor", "s", client.Params{
		"container": client.StringValue(containerName),
		"start":     client.TimeValue(time.Unix(start, 0)),
		"end":       client.TimeValue(time.Unix(end, 0)),
		"interval":  client.DurationValue(time.Minute * time.Duration(interval)),
	})

	response, err := cli.Query(q)
	if err != nil {
		log.Warnf("influxdb query error=%v", err)
		return nil, err
	} else if response.Error() != nil {
		log.Warnf("influxdb query error=%v", response.Error())
		return nil, response.Error()
	}

	var reply pb.MonitorHistoryReply
	hooks := []*[]*pb.MonitorSample{
		&reply.CpuUsage,
		&reply.MemoryUsage,
		&reply.DiskUsage,
		&reply.NetRx,
		&reply.NetTx,
	}

	for i, r := range response.Results {
		if len(hooks) > i+1 {
			var list []*pb.MonitorSample
			for _, s := range r.Series {
				for _, v := range s.Values {
					ts, _ := v[0].(json.Number).Int64()
					val, _ := v[1].(json.Number).Float64()
					list = append(list, &pb.MonitorSample{
						Timestamp: ts,
						Value:     val,
					})
				}
			}
			*hooks[i] = list
		}
	}
	return &reply, nil
}
