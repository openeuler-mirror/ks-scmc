package internal

import (
	"context"
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"

	"scmc/model"
	pb "scmc/rpc/pb/node"
)

var nodeStatus = make(map[int64]*pb.NodeStatus)

func getNodeStatus(node *model.NodeInfo) (*pb.NodeStatus, error) {
	conn, err := getAgentConn(node.Address)
	if err != nil {
		log.Warnf("Failed to connect to agent service, node=%+v", node)
		return nil, err
	}

	cli := pb.NewNodeClient(conn)
	ctx_, cancel := context.WithTimeout(context.Background(), time.Second*2)
	defer cancel()
	r, err := cli.Status(ctx_, &pb.StatusRequest{})
	if err != nil {
		log.Warnf("get node status ID=%v address=%v: %v", node.ID, node.Address, err)
		return nil, err
	}

	var s *pb.NodeStatus
	if len(r.StatusList) > 0 {
		s = r.StatusList[0]
	}
	return s, nil
}

func getAllNodeData(needStatus bool) []*pb.NodeInfo {
	nodes, err := model.ListNodes()
	if err != nil {
		log.Infof("get node list from DB err=%v", err)
		return nil
	}

	var nodeInfos []*pb.NodeInfo
	for _, n := range nodes {
		var s *pb.NodeStatus
		if needStatus {
			s, _ = getNodeStatus(&n)
		}

		nodeInfos = append(nodeInfos, &pb.NodeInfo{
			Id:      n.ID,
			Name:    n.Name,
			Address: n.Address,
			Comment: n.Comment,
			RscLimit: &pb.ResourceLimit{
				CpuLimit:    n.CpuLimit,
				MemoryLimit: n.MemoryLimit,
				DiskLimit:   n.DiskLimit,
			},
			Status: s,
		})
	}

	return nodeInfos
}

func writeNodeWatchLogs(n *pb.NodeInfo) []model.RuntimeLog {
	var r []model.RuntimeLog

	if n.Status == nil {
		r = append(r, model.RuntimeLog{
			Level:     1,
			NodeId:    n.Id,
			NodeInfo:  n.Name + " " + n.Address,
			EventType: "NODE_OFFLINE",
		})
		return r
	}

	if n.RscLimit != nil {
		if n.RscLimit.CpuLimit > 0 && n.Status.CpuStat != nil {
			if n.Status.CpuStat.Used > n.RscLimit.CpuLimit {
				r = append(r, model.RuntimeLog{
					Level:     1,
					NodeId:    n.Id,
					NodeInfo:  n.Name + " " + n.Address,
					EventType: "CPU_USAGE_EXCEED",
					Detail:    fmt.Sprintf("CPU usage %.2f", n.Status.CpuStat.Used),
				})
			}
		}

		if n.RscLimit.MemoryLimit > 0 && n.Status.MemStat != nil {
			if float64(n.Status.MemStat.Used) > n.RscLimit.MemoryLimit {
				r = append(r, model.RuntimeLog{
					Level:     1,
					NodeId:    n.Id,
					NodeInfo:  n.Name + " " + n.Address,
					EventType: "MEMORY_USAGE_EXCEED",
					Detail:    fmt.Sprintf("Memory usage %d", n.Status.MemStat.Used),
				})
			}
		}

		if n.RscLimit.DiskLimit > 0 && n.Status.DiskStat != nil {
			if float64(n.Status.DiskStat.Used) > n.RscLimit.DiskLimit {
				r = append(r, model.RuntimeLog{
					Level:     1,
					NodeId:    n.Id,
					NodeInfo:  n.Name + " " + n.Address,
					EventType: "DISK_USAGE_EXCEED",
					Detail:    fmt.Sprintf("Disk usage %d", n.Status.DiskStat.Used),
				})
			}
		}
	}

	if len(r) > 0 {
		model.CreateLog(r)
	}

	return r
}

func Watchdog() {
	// TODO: check current host is the master
	// TODO: container monitor data
	for {
		time.Sleep(10 * time.Second)
		nodeData := getAllNodeData(true)
		for _, n := range nodeData {
			writeNodeWatchLogs(n)
		}
	}
}
