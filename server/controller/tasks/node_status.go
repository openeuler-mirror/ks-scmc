package tasks

import (
	"context"
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"scmc/model/db"
	"scmc/model/warnlog"
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

func getNodeUsage(n *db.NodeInfo) (*pb.NodeStatus, error) {
	conn, err := internal.GetAgentConn(n.Address)
	if err != nil {
		log.Warnf("Failed to connect to agent service, node=%+v", n)
		return nil, err
	}

	cli := pb.NewNodeClient(conn)
	ctx_, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	require := &pb.UsageRequest{
		StartTime: int64(time.Now().Unix() - 60),
		EndTime:   int64(time.Now().Unix()),
	}
	r, err := cli.Usage(ctx_, require)
	if err != nil {
		log.Warnf("get node Usage ID=%v address=%v: %v", n.ID, n.Address, err)
		s, ok := status.FromError(err)
		if ok && s.Code() == codes.Unavailable {
			return &pb.NodeStatus{
				State: int64(pb.NodeState_Offline),
			}, nil
		}
		return &pb.NodeStatus{
			State: int64(pb.NodeState_Unknown),
		}, nil
	}
	return r.Status, err
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
			s, _ = getNodeUsage(n)
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

func writeNodeWatchLogs(n *pb.NodeInfo) {
	var r []*db.WarnLog

	newWarnLog := func(n *pb.NodeInfo, et logging.EVENT_TYPE, detail string) {
		r = append(r, &db.WarnLog{
			NodeId:      n.Id,
			NodeInfo:    fmt.Sprintf("%s (%s)", n.Name, n.Address),
			EventType:   int64(et),
			EventModule: int64(logging.EVENT_MODULE_NODE),
			Detail:      detail,
		})
	}

	newResourceUsageWarn := func(n *pb.NodeInfo, detail string) {
		newWarnLog(n, logging.EVENT_TYPE_WARN_RESOURCE_USAGE, detail)
	}

	st := n.Status
	if st == nil || st.State == int64(pb.NodeState_Offline) {
		newWarnLog(n, logging.EVENT_TYPE_WARN_NODE_OFFLINE, "节点离线")
	}

	if st.State == int64(pb.NodeState_Unknown) {
		newWarnLog(n, logging.EVENT_TYPE_WARN_NODE_ABNORMAL, "节点状态异常")
	}

	lim := n.RscLimit
	if lim != nil {
		if lim.CpuLimit > 0 && st.CpuStat != nil && (st.CpuStat.Used >= lim.CpuLimit) {
			newResourceUsageWarn(n, fmt.Sprintf("CPU使用率 %.2f%%", st.CpuStat.Used*100))
		}

		if lim.MemoryLimit > 0 && st.MemStat != nil && (float64(st.MemStat.Used) >= lim.MemoryLimit) {
			newResourceUsageWarn(n, fmt.Sprintf("内存使用 %dMB", st.MemStat.Used))
		}

		if lim.DiskLimit > 0 && st.DiskStat != nil && (float64(st.DiskStat.Used) >= lim.DiskLimit) {
			newResourceUsageWarn(n, fmt.Sprintf("磁盘使用 %dMB", st.DiskStat.Used))
		}
	}

	warnlog.Create(r)
}
