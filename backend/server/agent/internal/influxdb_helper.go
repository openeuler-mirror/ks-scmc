package internal

import (
	"encoding/json"
	"time"

	client "github.com/influxdata/influxdb1-client/v2"
	log "github.com/sirupsen/logrus"

	"ksc-mcube/common"
	"ksc-mcube/rpc"
	pb "ksc-mcube/rpc/pb/container"
)

/*
func influxdbQuery(query client.Query, parse func([]interface{}) *pb.MonitorSample) (*pb.MonitorHistoryReply, error) {
	cli, err := client.NewHTTPClient(client.HTTPConfig{
		Addr: "http://" + common.Config.InfluxDB.Addr,
	})
	if err != nil {
		log.Warnf("get influxdb client error=%v", err)
		return nil, rpc.ErrInternal
	}
	defer cli.Close()

	response, err := cli.Query(query)
	if err != nil {
		log.Warnf("influxdb query error=%v", err)
		return nil, rpc.ErrInternal
	} else if response.Error() != nil {
		log.Warnf("influxdb query error=%v", response.Error())
		return nil, rpc.ErrInternal
	}

	reply := pb.MonitorHistoryReply{}
	for _, r := range response.Results {
		for _, s := range r.Series {
			for _, v := range s.Values {
				reply.Data = append(reply.Data, parse(v))
				ts, _ := v[0].(json.Number).Int64()
				usage, _ := v[1].(json.Number).Float64()
				reply.Data = append(reply.Data, &pb.MonitorSample{
					Time:     ts,
					CpuUsage: usage,
				})
			}
		}
	}

	return &reply, nil
}
*/

func influxdbQueryCPU(containerName string, startTime, endTime int64) (*pb.MonitorHistoryReply, error) {
	cli, err := client.NewHTTPClient(client.HTTPConfig{
		Addr: "http://" + common.Config.InfluxDB.Addr,
	})
	if err != nil {
		log.Warnf("get influxdb client error=%v", err)
		return nil, rpc.ErrInternal
	}
	defer cli.Close()

	q := `SELECT difference(mean("value"))  / 60000000000 FROM "cpu_usage_system"
	WHERE ("container_name" = $container) AND time >= $start and time <= $end
	GROUP BY time(1m) fill(previous)`

	query := client.NewQueryWithParameters(q, "cadvisor", "s", client.Params{
		"container": client.StringValue(containerName),
		"start":     client.TimeValue(time.Unix(startTime, 0)),
		"end":       client.TimeValue(time.Unix(endTime, 0)),
	})

	response, err := cli.Query(query)
	if err != nil {
		log.Warnf("influxdb query error=%v", err)
		return nil, rpc.ErrInternal
	} else if response.Error() != nil {
		log.Warnf("influxdb query error=%v", response.Error())
		return nil, rpc.ErrInternal
	}

	reply := pb.MonitorHistoryReply{}
	for _, r := range response.Results {
		for _, s := range r.Series {
			for _, v := range s.Values {
				ts, _ := v[0].(json.Number).Int64()
				usage, _ := v[1].(json.Number).Float64()
				reply.Data = append(reply.Data, &pb.MonitorSample{
					Time:     ts,
					CpuUsage: usage,
				})
				// log.Infof("CPU usage %d %f", ts, usage)
			}
		}
	}

	return &reply, nil
}

func influxdbQueryMemory(containerName string, startTime, endTime int64) (*pb.MonitorHistoryReply, error) {
	cli, err := client.NewHTTPClient(client.HTTPConfig{
		Addr: "http://" + common.Config.InfluxDB.Addr,
	})
	if err != nil {
		log.Warnf("get influxdb client error=%v", err)
		return nil, rpc.ErrInternal
	}
	defer cli.Close()

	q := `SELECT mean("value")  / $division FROM "memory_working_set"
	WHERE ("container_name" = $container) AND time >= $start and time <= $end
	GROUP BY time(1m) fill(0)`

	query := client.NewQueryWithParameters(q, "cadvisor", "s", client.Params{
		"division":  client.IntegerValue(1 << 20),
		"container": client.StringValue(containerName),
		"start":     client.TimeValue(time.Unix(startTime, 0)),
		"end":       client.TimeValue(time.Unix(endTime, 0)),
	})

	response, err := cli.Query(query)
	if err != nil {
		log.Warnf("influxdb query error=%v", err)
		return nil, rpc.ErrInternal
	} else if response.Error() != nil {
		log.Warnf("influxdb query error=%v", response.Error())
		return nil, rpc.ErrInternal
	}

	reply := pb.MonitorHistoryReply{}
	for _, r := range response.Results {
		for _, s := range r.Series {
			for _, v := range s.Values {
				ts, _ := v[0].(json.Number).Int64()
				usage, _ := v[1].(json.Number).Float64()
				reply.Data = append(reply.Data, &pb.MonitorSample{
					Time:        ts,
					MemoryUsage: usage,
				})
			}
		}
	}

	return &reply, nil
}

func influxdbQueryDisk(containerName string, startTime, endTime int64) (*pb.MonitorHistoryReply, error) {
	cli, err := client.NewHTTPClient(client.HTTPConfig{
		Addr: "http://" + common.Config.InfluxDB.Addr,
	})
	if err != nil {
		log.Warnf("get influxdb client error=%v", err)
		return nil, rpc.ErrInternal
	}
	defer cli.Close()

	q := `SELECT mean("value")  / $division FROM "fs_usage"
	WHERE ("container_name" = $container) AND time >= $start and time <= $end
	GROUP BY time(1m) fill(0)`

	query := client.NewQueryWithParameters(q, "cadvisor", "s", client.Params{
		"division":  client.IntegerValue(1 << 20),
		"container": client.StringValue(containerName),
		"start":     client.TimeValue(time.Unix(startTime, 0)),
		"end":       client.TimeValue(time.Unix(endTime, 0)),
	})

	response, err := cli.Query(query)
	if err != nil {
		log.Warnf("influxdb query error=%v", err)
		return nil, rpc.ErrInternal
	} else if response.Error() != nil {
		log.Warnf("influxdb query error=%v", response.Error())
		return nil, rpc.ErrInternal
	}

	reply := pb.MonitorHistoryReply{}
	for _, r := range response.Results {
		for _, s := range r.Series {
			for _, v := range s.Values {
				ts, _ := v[0].(json.Number).Int64()
				usage, _ := v[1].(json.Number).Float64()
				reply.Data = append(reply.Data, &pb.MonitorSample{
					Time:      ts,
					DiskUsage: usage,
				})
			}
		}
	}

	return &reply, nil
}
