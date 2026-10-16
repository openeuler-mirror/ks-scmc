package tasks

import (
	"context"
	"time"

	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"

	"scmc/model/db"
	pb "scmc/rpc/pb/image"
	"scmc/server/controller/internal"
)

type SyncImages struct {
}

func (s *SyncImages) Interval() time.Duration {
	return time.Minute
}

func (s *SyncImages) RunAtInit() bool {
	return true
}
func (s *SyncImages) Run() {
	if isMaster() {
		SyncNodeImages()
	}
}

func listImage(conn *grpc.ClientConn) (map[string]bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	cli := pb.NewImageClient(conn)
	reply, err := cli.List(ctx, &pb.ListRequest{})
	if err != nil {
		log.Warnf("image.List err=%v", err)
		return nil, err
	}

	r := make(map[string]bool, len(reply.Images))
	for _, i := range reply.Images {
		r[i.Name] = true
	}

	return r, nil
}

func agentSync(conn *grpc.ClientConn, req *pb.AgentSyncRequest) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	cli := pb.NewImageClient(conn)
	_, err := cli.AgentSync(ctx, req)
	if err != nil {
		log.Warnf("image.AgentSync err=%v", err)
		return err
	}

	return nil
}

// SyncNodeImages 读取数据库中镜像列表，删除节点上多余的（注意不能删除备份产生的镜像），
// 拉取缺失的
func SyncNodeImages() {
	images, err := dbImages()
	if err != nil {
		log.Infof("get valid images err=%v", err)
		return
	}

	nodes, err := db.ListNodes()
	if err != nil {
		log.Warnf("Failed to list node images: %v", err)
		return
	}

	for _, n := range nodes {
		go syncNodeImages(images, n.Address)
	}
}

func syncNodeImages(dbImages map[string]bool, addr string) {
	conn, err := internal.GetAgentConn(addr)
	if err != nil {
		log.Warnf("Failed to connect to agent service, addr=%v", addr)
		return
	}

	backupImages, err := ListBackupImage(conn)
	if err != nil {
		log.Warnf("ListBackupImage err=%v", err)
		return
	}

	nodeImages, err := listImage(conn)
	if err != nil {
		log.Warnf("listImage err=%v", err)
		return
	}

	var req pb.AgentSyncRequest

	// 优化备份创建数据维护，防止备份镜像不会被错误地删除后取消下面的注释
	// 节点存在的镜像，上传和备份中都没有，需要删除
	for i := range nodeImages {
		if !dbImages[i] && !backupImages[i] {
			req.ToRemove = append(req.ToRemove, i)
		}
	}

	// 上传且审批通过的镜像，节点中不存在，需要同步
	for i := range dbImages {
		if !nodeImages[i] {
			req.ToPull = append(req.ToPull, i)
		}
	}

	if len(req.ToRemove) == 0 && len(req.ToPull) == 0 {
		return // no changes
	}

	agentSync(conn, &req)
}
