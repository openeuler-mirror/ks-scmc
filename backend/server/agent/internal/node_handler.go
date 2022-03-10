package internal

import (
	"context"
	"runtime"

	"github.com/shirou/gopsutil/mem"

	pb "scmc/rpc/pb/node"
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
		var memStat *pb.MemoryStat
		memInfo, err := mem.VirtualMemory()
		if err == nil {
			memStat = &pb.MemoryStat{
				Total:          memInfo.Total / megaBytes,
				Used:           memInfo.Used / megaBytes,
				Free:           memInfo.Free / megaBytes,
				UsedPercentage: memInfo.UsedPercent,
			}
		}

		cpuStat := &pb.CpuStat{
			Total: float64(runtime.NumCPU()),
		}
		cpuStat.Used, _ = cpuUsage()
		cpuStat.UsedPercentage = cpuStat.Used / cpuStat.Total

		var containerStat *pb.ContainerStat
		dockerInfo, err := cli.Info(context.Background())
		if err == nil {
			containerStat = &pb.ContainerStat{
				Running: int64(dockerInfo.ContainersRunning),
				Total:   int64(dockerInfo.Containers),
			}
		}

		// TODO disk usage

		reply.StatusList = append(reply.StatusList, &pb.NodeStatus{
			State:         int64(pb.NodeState_Online),
			ContainerStat: containerStat,
			CpuStat:       cpuStat,
			MemStat:       memStat,
		})
	}

	return &reply, nil
}
