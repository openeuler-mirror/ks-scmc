package tasks

import (
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"

	"scmc/model/db"
	"scmc/rpc/pb/logging"
	pb "scmc/rpc/pb/node"
	"scmc/server/controller/internal"
)

type NodeStatus struct{}

func (n *NodeStatus) Interval() time.Duration {
	return time.Minute
}

func (n *NodeStatus) RunAtInit() bool {
	return true
}

func (n *NodeStatus) Run() {
	if isMaster() {
		nodeData := getAllNodeData(true)
		for _, n := range nodeData {
			go writeNodeWatchLogs(n)
		}
	}
}

func getAllNodeData(needStatus bool) (r []*pb.NodeInfo) {
	nodes, err := db.ListNodes()
	if err != nil {
		log.Infof("db.ListNodes err=%v", err)
		return nil
	}

	for _, n := range nodes {
		var s *pb.NodeStatus
		if needStatus {
			s, _ = internal.GetNodeStatus(n)
		}

		r = append(r, &pb.NodeInfo{
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

	return
}

func writeNodeWatchLogs(n *pb.NodeInfo) []*db.WarnLog {
	var r []*db.WarnLog

	if n.Status == nil || n.Status.State == int64(pb.NodeState_Offline) {
		r = append(r, &db.WarnLog{
			NodeId:      n.Id,
			NodeInfo:    fmt.Sprintf("%s (%s)", n.Name, n.Address),
			EventType:   int64(logging.EVENT_TYPE_WARN_NODE_OFFLINE),
			EventModule: int64(logging.EVENT_MODULE_NODE),
			Detail:      "节点离线",
		})
	}

	if n.Status.State == int64(pb.NodeState_Unknown) {
		r = append(r, &db.WarnLog{
			NodeId:      n.Id,
			NodeInfo:    fmt.Sprintf("%s (%s)", n.Name, n.Address),
			EventType:   int64(logging.EVENT_TYPE_WARN_NODE_ABNORMAL),
			EventModule: int64(logging.EVENT_MODULE_NODE),
			Detail:      "节点状态异常",
		})
	}

	if n.RscLimit != nil {
		if n.RscLimit.CpuLimit > 0 && n.Status.CpuStat != nil {
			if n.Status.CpuStat.Used > n.RscLimit.CpuLimit {
				r = append(r, &db.WarnLog{
					NodeId:      n.Id,
					NodeInfo:    fmt.Sprintf("%s (%s)", n.Name, n.Address),
					EventType:   int64(logging.EVENT_TYPE_WARN_RESOURCE_USAGE),
					EventModule: int64(logging.EVENT_MODULE_NODE),
					Detail:      fmt.Sprintf("CPU使用率 %.2f%%", n.Status.CpuStat.Used*100),
				})
			}
		}

		if n.RscLimit.MemoryLimit > 0 && n.Status.MemStat != nil {
			if float64(n.Status.MemStat.Used) > n.RscLimit.MemoryLimit {
				r = append(r, &db.WarnLog{
					NodeId:      n.Id,
					NodeInfo:    fmt.Sprintf("%s (%s)", n.Name, n.Address),
					EventType:   int64(logging.EVENT_TYPE_WARN_RESOURCE_USAGE),
					EventModule: int64(logging.EVENT_MODULE_NODE),
					Detail:      fmt.Sprintf("内存使用 %dMB", n.Status.MemStat.Used),
				})
			}
		}

		if n.RscLimit.DiskLimit > 0 && n.Status.DiskStat != nil {
			if float64(n.Status.DiskStat.Used) > n.RscLimit.DiskLimit {
				r = append(r, &db.WarnLog{
					NodeId:      n.Id,
					NodeInfo:    fmt.Sprintf("%s (%s)", n.Name, n.Address),
					EventType:   int64(logging.EVENT_TYPE_WARN_RESOURCE_USAGE),
					EventModule: int64(logging.EVENT_MODULE_NODE),
					Detail:      fmt.Sprintf("磁盘使用 %dMB", n.Status.DiskStat.Used),
				})
			}
		}
	}

	if len(r) > 0 {
		db.CreateWarnLog(r)
	}

	return r
}
