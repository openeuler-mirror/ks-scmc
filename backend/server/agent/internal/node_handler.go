package internal

import (
	"context"
	"runtime"

	"github.com/shirou/gopsutil/mem"

	pb "ksc-mcube/rpc/pb/node"
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
		var memUsedPct float64
		memInfo, err := mem.VirtualMemory()
		if err == nil {
			MemTotal = memInfo.Total
			memUsed = memInfo.Used
			memFree = memInfo.Free
			memUsedPct = memInfo.UsedPercent
		}

		cpuNum := float64(runtime.NumCPU())
		cpuCoreUsed, _ := cpuUsage()
		cpuUsagePercent := cpuCoreUsed / cpuNum

		var cntrTotal, cntrRunning int64
		cntrInfo, err := cli.Info(context.Background())
		if err == nil {
			cntrTotal = int64(cntrInfo.Containers)
			cntrRunning = int64(cntrInfo.ContainersRunning)
		}

		reply.StatusList = append(reply.StatusList, &pb.NodeStatus{
			State:         int64(pb.NodeState_Online),
			ContainerStat: &pb.ContainerStat{Total: cntrTotal, Running: cntrRunning},
			CpuStat:       &pb.CpuStat{Total: cpuNum, Used: cpuCoreUsed, UsedPercentage: cpuUsagePercent},
			MemStat:       &pb.MemoryStat{Total: MemTotal, Used: memUsed, Free: memFree, UsedPercentage: memUsedPct},
		})
	}

	return &reply, nil
}
