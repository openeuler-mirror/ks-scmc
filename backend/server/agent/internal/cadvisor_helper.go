package internal

import (
	"time"

	"github.com/google/cadvisor/client"
	v1 "github.com/google/cadvisor/info/v1"
	log "github.com/sirupsen/logrus"

	"ksc-mcube/common"
	pb "ksc-mcube/rpc/pb/container"
)

func containerStats(m map[string]*pb.ContainerStatus) error {
	client, err := client.NewClient("http://" + common.CAdvisorAddr)
	if err != nil {
		log.Warnf("create cadvisor client %s: %v", "", err)
		return err
	}

	now := time.Now()
	request := v1.ContainerInfoRequest{
		Start: now.Add(-time.Second * 10),
		End:   now,
	}

	infos, err := client.AllDockerContainers(&request)
	if err != nil {
		log.Warnf("get all container stats from cadvisor: %v", err)
		return err
	}

	for _, info := range infos {
		if len(info.Stats) > 1 {
			head, tail := info.Stats[0], info.Stats[len(info.Stats)-1]

			duration := tail.Timestamp.Sub(head.Timestamp)
			cpuPercent := float64(tail.Cpu.Usage.Total-head.Cpu.Usage.Total) / float64(duration.Nanoseconds())
			if p, ok := m[info.Id]; ok {
				p.CpuStat = &pb.CpuStat{
					CoreUsed: cpuPercent * 100,
				}
				p.MemStat = &pb.MemoryStat{
					Used: tail.Memory.WorkingSet,
				}
			}
		}
	}

	return nil
}
