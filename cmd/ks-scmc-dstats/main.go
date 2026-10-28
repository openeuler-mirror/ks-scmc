package main

import (
	"log"
        "context"
        //"fmt"
	"os"
	//"encoding/json"
	//"io"
	//"bytes"
        "github.com/apache/thrift/lib/go/thrift"
	//"github.com/docker/docker/api/types"
	"github.com/sirupsen/logrus"
	//"github.com/KyleBanks/dockerstats"
        "scmc/thrift/gen-go/node"
	_ "scmc/model/docker"
	"scmc/common"
	"scmc/server/controller/thriftapi"
)

type DockerStatsServer struct {
}

/*
func nodeStatus() {
	cli, err := docker.Client()
	if err != nil {
		fmt.Println(err)
		return
	}

	containers, err := cli.ContainerList(context.Background(), types.ContainerListOptions{All: true, Size: true})
	if err == nil {
		runnings := 0
		for _, c := range containers {
			fmt.Println("containers: ", containers)
			if c.State == "running" {
				runnings = runnings + 1
			}

			stats, err := cli.ContainerStats(context.Background(), c.ID, false) 
			if (err == nil) {
				dec := json.NewDecoder(stats.Body)
				var v types.StatsJSON
				if err := dec.Decode(&v); err != nil {
					fmt.Printf("Error decoding stats for container %s: %v\n", c.ID, err)
					continue
				}

				// 打印 CPU 和内存使用情况
				fmt.Printf("CPU Usage: %v\n", v.CPUStats.CPUUsage.TotalUsage)
				fmt.Printf("Memory Usage: %v\n", v.MemoryStats.Usage)
				fmt.Println("v: ", v)

			} else {
				fmt.Println("err :", err)
			}

		}
		fmt.Println("running: ", runnings, " all:", len(containers))
	}
	return
}
*/

func (e *DockerStatsServer) GetAllContainers(ctx context.Context, node_name string) (r string, err error) {
	logrus.Infof("GetAllContainers node_name: %s", node_name)

	json_str, err := thriftapi.GetContainerStats(node_name)
	if err != nil {
		json_str = "{\"status\":-1,\"message\":\"" + err.Error() + "\",\"containers\":[]}"
	} else {
		if (len(json_str) == 0) {
			json_str = "{\"status\":0,\"message\":\"success\",\"containers\":[]}"
		}
	}
	return json_str, nil
}

func main() {
	if len(os.Args) != 2 {
		usage()
	}

	configFile := os.Args[1];

	if err := common.LoadConfig(configFile); err != nil {
		log.Fatalf("load config file=%s error=%v", configFile, err)
	}

	common.InitLogger("dockerMonitor")

	logrus.Infof("CA: %s", common.Config.TLS.CA)
	logrus.Infof("serverCert: %s", common.Config.TLS.ServerCert)
	logrus.Infof("serverKey: %s", common.Config.TLS.ServerKey)

	cfg, err:= common.LoadTLSConfig()
	if (err != nil) {
		logrus.Errorf("loadTLSConfig failed! %v", err)
		log.Fatalf("loadTLSConfig failed! %v", err)

	}

	addr := common.Config.Controller.Addr()
	logrus.Infof("addr: %s", addr)

	transport, err := thrift.NewTSSLServerSocket(addr, cfg)
        if err != nil {
                panic(err)
        }

	//fmt.Printf("%T\n", transport);

        handler := &DockerStatsServer{}
        processor := node.NewDockerStatsDataProcessor(handler)

        transportFactory := thrift.NewTBufferedTransportFactory(8192)
        //也可以选择Framed传输协议，但是必须确保服务端和客户端的传输协议一致
        //transportFactory := thrift.NewTBufferedTransportFactory(8192)
        //framedtransportFactory := thrift.NewTFramedTransportFactory(transportFactory)

        //可以选择任意一种编码协议，但是必须确保服务端和客户端的编码协议一致
        protocolFactory := thrift.NewTBinaryProtocolFactory(true, true) //布尔参数strictRead, strictWrite，读和写时是否加入版本校验。
        //protocolFactory := thrift.NewTCompactProtocolFactory()
        //protocolFactory := thrift.NewTJSONProtocolFactory()

        server := thrift.NewTSimpleServer4(
                processor,
                transport,
                transportFactory,
                protocolFactory,
        )

        if err := server.Serve(); err != nil {
                panic(err)
        }
}

func usage() {
	log.Fatalf("Usage: %s CONFIG_FILE", os.Args[0])
}
