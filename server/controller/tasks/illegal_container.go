package tasks

import (
	"context"
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"

	"scmc/model/db"
	"scmc/model/warnlog"
	pb "scmc/rpc/pb/container"
	"scmc/rpc/pb/logging"
	"scmc/server/controller/internal"
)

const (
	ImageSourceNone   = 0
	ImageSourceUpload = 1
	ImageSourceBackup = 2
)

func ListContainer(conn *grpc.ClientConn, listAll bool) (*pb.ListReply, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	cli := pb.NewContainerClient(conn)
	return cli.List(ctx, &pb.ListRequest{ListAll: listAll})
}

func StopContainer(conn *grpc.ClientConn, id string) (*pb.StopReply, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	cli := pb.NewContainerClient(conn)
	return cli.Stop(ctx, &pb.StopRequest{
		Ids: []*pb.ContainerIdList{
			{
				ContainerIds: []string{id},
			},
		},
	})
}

type IllegalContainer struct {
}

func (i *IllegalContainer) Interval() time.Duration {
	return time.Minute
}

func (i *IllegalContainer) RunAtInit() bool {
	return true
}

func (i *IllegalContainer) ScanNodeContainers(images map[string]*db.ImageInfo, n *db.NodeInfo) {
	conn, err := internal.GetAgentConn(n.Address)
	if err != nil {
		log.Warnf("restart container ErrInternal")
		return
	}

	backupImages, err := internal.ListBackupImage(conn)
	if err != nil {
		log.Warnf("ListBackupImage err=%v", err)
		return
	}

	r, err := ListContainer(conn, false)
	if err != nil {
		log.Warnf("ListContainer err=%v", err)
		return
	}

	for _, c := range r.Containers {
		if c.Info == nil || c.Info.State != "running" {
			continue
		}

		if images[c.Info.Image] == nil && !backupImages[c.Info.Image] {
			log.Debugf("stop container=%s", c.Info.Name)
			if _, err := StopContainer(conn, c.Info.Id); err != nil {
				log.Infof("StopContainer %v err=%v", c.Info.Id, err)
			}

			warnlog.Create([]*db.WarnLog{
				{
					NodeId:        n.ID,
					NodeInfo:      fmt.Sprintf("%s (%s)", n.Name, n.Address),
					EventType:     int64(logging.EVENT_TYPE_WARN_ILLEGAL_CONTAINER),
					EventModule:   int64(logging.EVENT_MODULE_CONTAINER),
					ContainerName: c.Info.Name,
					Detail:        "容器镜像非法",
				},
			})
		}
	}
}

func (i *IllegalContainer) Run() {
	if !isMaster() {
		return
	}

	images, err := db.GetImageMap()
	if err != nil {
		log.Infof("get valid images err=%v", err)
		return
	}

	nodes, err := db.ListNodes()
	if err != nil {
		log.Errorf("db.ListNodes err=%v", err)
		return
	}

	for _, n := range nodes {
		i.ScanNodeContainers(images, n)
	}
}
