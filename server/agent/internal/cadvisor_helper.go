package internal

import (
	"time"

	"github.com/google/cadvisor/client"
	v1 "github.com/google/cadvisor/info/v1"
	log "github.com/sirupsen/logrus"

	"scmc/common"
	pb "scmc/rpc/pb/container"
)

func cadvisorClient() (*client.Client, error) {
	cli, err := client.NewClient("http://" + common.Config.CAdvisor.Addr)
	if err != nil {
		log.Warnf("create cadvisor client %s: %v", "", err)
	}
	return cli, err
}

func calculateNetworkTraffic(stats *v1.ContainerStats) (rxBytes, txBytes uint64) {

    // 遍历所有网络接口
    for _, iface := range stats.Network.Interfaces {
        rxBytes += iface.RxBytes
        txBytes += iface.TxBytes
    }

    return rxBytes, txBytes
}

func calculateNetworkSpeed(prevStats *v1.ContainerStats, currentStats *v1.ContainerStats) (rxSpeed, txSpeed float64) {

	// 计算两次采样的字节增量
	prevRx, prevTx := calculateNetworkTraffic(prevStats)
	currentRx, currentTx := calculateNetworkTraffic(currentStats)

	rxBytesDelta := currentRx - prevRx
	txBytesDelta := currentTx - prevTx

	// 计算速率（假设时间间隔为秒）
	timeDelta := currentStats.Timestamp.Sub(prevStats.Timestamp).Seconds()
	rxRate := float64(rxBytesDelta) / timeDelta  // 下行速率 (B/s)
	txRate := float64(txBytesDelta) / timeDelta  // 上行速率 (B/s)

	return rxRate, txRate
}


func getContainerStats() (map[string]*pb.ResourceStat, error) {
	cli, err := cadvisorClient()
	if err != nil {
		return nil, err
	}

	now := time.Now()
	request := v1.ContainerInfoRequest{
		Start: now.Add(-time.Second * 10),
		End:   now,
	}

	infos, err := cli.AllDockerContainers(&request)
	if err != nil {
		log.Warnf("get all container stats from cadvisor: %v", err)
		return nil, err
	}

	m := make(map[string]*pb.ResourceStat)
	for _, info := range infos {
		var p *pb.ResourceStat
		var head, tail *v1.ContainerStats

		if len(info.Stats) > 0 {
			head, tail = info.Stats[0], info.Stats[len(info.Stats)-1]

			p = &pb.ResourceStat{
				MemStat: &pb.MemoryStat{
					Used: float64(tail.Memory.WorkingSet) / megaBytes,
				},
			}

			if len(info.Stats) > 1 {
				duration := tail.Timestamp.Sub(head.Timestamp)
				cpuPercent := float64(tail.Cpu.Usage.Total-head.Cpu.Usage.Total) / float64(duration.Nanoseconds())
				p.CpuStat = &pb.CpuStat{
					CoreUsed: cpuPercent,
				}
			}
			m[info.Id] = p
		}
	}

	return m, nil
}
