package internal

import (
	"context"

	"github.com/shirou/gopsutil/disk"
	"github.com/shirou/gopsutil/mem"
	log "github.com/sirupsen/logrus"

	"scmc/model"
	"scmc/rpc"
	pb "scmc/rpc/pb/node"
)

type NodeServer struct {
	pb.UnimplementedNodeServer
}

func nodeStatus() (*pb.NodeStatus, error) {
	cli, err := model.DockerClient()
	if err != nil {
		return &pb.NodeStatus{State: int64(pb.NodeState_Unknown)}, nil
	}

	var ret = pb.NodeStatus{
		State: int64(pb.NodeState_Online),
	}

	if memInfo, err := mem.VirtualMemory(); err != nil {
		log.Infof("VirtualMemory err=%v", err)
	} else {
		ret.MemStat = &pb.MemoryStat{
			Total:          memInfo.Total / megaBytes,
			Used:           memInfo.Used / megaBytes,
			Free:           memInfo.Free / megaBytes,
			UsedPercentage: memInfo.UsedPercent,
		}
	}

	if cpuUsage, err := globalCPUUsage.get(); err != nil {
		log.Infof("cpuUsage err=%v", err)
	} else {
		ret.CpuStat = &pb.CpuStat{
			Used:  cpuUsage,
			Total: float64(numCPU()),
		}
		ret.CpuStat.UsedPercentage = ret.CpuStat.Used / ret.CpuStat.Total
	}

	if diskUsage, err := disk.Usage("/"); err != nil {
		log.Infof("disk.Usage err=%v", err)
	} else {
		ret.DiskStat = &pb.DiskStat{
			Total:          diskUsage.Total,
			Free:           diskUsage.Free,
			Used:           diskUsage.Used,
			UsedPercentage: diskUsage.UsedPercent,
		}
	}

	dockerInfo, err := cli.Info(context.Background())
	if err == nil {
		ret.ContainerStat = &pb.ContainerStat{
			Running: int64(dockerInfo.ContainersRunning),
			Total:   int64(dockerInfo.Containers),
		}
	}

	return &ret, nil
}

func (*NodeServer) Status(ctx context.Context, in *pb.StatusRequest) (*pb.StatusReply, error) {
	s, err := nodeStatus()
	if err != nil {
		log.Infof("nodeStatus err=%v", err)
		return nil, rpc.ErrInternal
	}

	return &pb.StatusReply{
		StatusList: []*pb.NodeStatus{s},
	}, nil
}
