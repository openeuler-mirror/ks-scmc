package internal

import (
	"context"
	"runtime"
	"time"

	pb "ksc-mcube/rpc/pb/node"

	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/mem"
)

type NodeServer struct {
	pb.UnimplementedNodeServer
}

func (s *NodeServer) Status(ctx context.Context, in *pb.StatusRequest) (*pb.StatusReply, error) {
	reply := pb.StatusReply{}

	cli, err := dockerCli()
	if err != nil {
		reply.StatusList = append(reply.StatusList, &pb.NodeStatus{State: int64(pb.NodeState_Unknown)})
	} else {
		var MemTotal, memUsed, memFree uint64
		var memUsedPct, cpuUsedPct float64
		memInfo, err := mem.VirtualMemory()
		if err == nil {
			MemTotal = memInfo.Total
			memUsed = memInfo.Used
			memFree = memInfo.Free
			memUsedPct = memInfo.UsedPercent
		}

		cpunum := float64(runtime.NumCPU())
		percent, err := cpu.Percent(time.Second, false)
		if err == nil {
			cpuUsedPct = percent[0]
		}
		used := cpunum * cpuUsedPct / 100

		var cntrTotal, cntrRunning int64
		cntrInfo, err := cli.Info(context.Background())
		if err == nil {
			cntrTotal = int64(cntrInfo.Containers)
			cntrRunning = int64(cntrInfo.ContainersRunning)
		}

		reply.StatusList = append(reply.StatusList, &pb.NodeStatus{
			State:         int64(pb.NodeState_Online),
			ContainerStat: &pb.ContainerStat{Total: cntrTotal, Running: cntrRunning},
			CpuStat:       &pb.CpuStat{Total: cpunum, Used: used, UsedPercentage: cpuUsedPct},
			MemStat:       &pb.MemoryStat{Total: MemTotal, Used: memUsed, Free: memFree, UsedPercentage: memUsedPct},
		})
	}

	return &reply, nil
}
