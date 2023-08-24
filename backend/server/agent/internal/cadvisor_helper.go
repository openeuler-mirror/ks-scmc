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
					Used: tail.Memory.WorkingSet / megaBytes,
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

func cpuUsage() (float64, error) {
	cli, err := cadvisorClient()
	if err != nil {
		return 0, err
	}

	now := time.Now()
	request := v1.ContainerInfoRequest{
		Start: now.Add(-time.Second * 10),
		End:   now,
	}

	info, err := cli.ContainerInfo("/", &request)
	if err != nil {
		log.Warnf("get container info(/) from cadvisor: %v", err)
		return 0, err
	}

	if len(info.Stats) > 1 {
		head := info.Stats[0]
		tail := info.Stats[len(info.Stats)-1]

		d := tail.Timestamp.Sub(head.Timestamp)
		u := float64(tail.Cpu.Usage.Total-head.Cpu.Usage.Total) / float64(d.Nanoseconds())

		return u, nil
	}

	return 0, nil
}
