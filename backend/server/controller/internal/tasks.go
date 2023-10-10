package internal

import (
	"context"
	"fmt"
	"net"
	"time"

	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"

	"scmc/common"
	"scmc/model"
	pb "scmc/rpc/pb/container"
	"scmc/rpc/pb/logging"
)

const (
	ImageSourceNone   = 0
	ImageSourceUpload = 1
	ImageSourceBackup = 2
)

func isMaster() bool {
	ifaceName, vip := common.Config.Controller.VirtualIf, common.Config.Controller.VirtualIP
	if ifaceName == "" || vip == "" {
		return true
	}

	iface, err := net.InterfaceByName(ifaceName)
	if err != nil {
		log.Warnf("net.InterfaceByName iface=%s err=%v", ifaceName, err)
		return false
	}

	addrs, err := iface.Addrs()
	if err != nil {
		log.Warnf("get iface=%s address list, err=%v", ifaceName, err)
	}

	for _, i := range addrs {
		addr, _, err := net.ParseCIDR(i.String())
		if err != nil {
			log.Warnf("net.ParseCIDR(%s) err=%v", i.String(), err)
			return false
		}
		if addr.String() == vip {
			return true
		}
	}

	return false
}

func allValidImages() (map[string]int, error) {
	dbImages, err := model.QueryImageByStatus()
	if err != nil {
		log.Warnf("query db images err=%v", err)
		return nil, err
	}

	backups, err := model.ListContainerBackup()
	if err != nil {
		log.Warnf("query backup images err=%v", err)
		return nil, err
	}

	data := make(map[string]int, len(dbImages)+len(backups))
	for _, i := range dbImages {
		data[i.Name+":"+i.Version] = ImageSourceUpload
	}
	for _, b := range backups {
		data[b.ImageRef] = ImageSourceBackup
	}

	return data, nil
}

func GetBackupJob(conn *grpc.ClientConn, id int64) (*pb.GetBackupJobReply, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	cli := pb.NewContainerClient(conn)
	return cli.GetBackupJob(ctx, &pb.GetBackupJobRequest{Id: id})
}

func DelBackupJob(conn *grpc.ClientConn, id int64) (*pb.DelBackupJobReply, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	cli := pb.NewContainerClient(conn)
	return cli.DelBackupJob(ctx, &pb.DelBackupJobRequest{Id: id})
}

func checkContainerBackupJob() {
	jobs, err := model.QueryUndoneContainerBackup()
	if err != nil {
		log.Warnf("model.QueryContainerBackups err=%v", err)
		return
	}

	for _, job := range jobs {
		nodeInfo, err := model.QueryNodeByID(job.NodeID)
		if err != nil {
			log.Warnf("model.QueryNodeByID node_id=%v err=%v", job.NodeID, err)
			continue
		}

		conn, err := getAgentConn(nodeInfo.Address)
		if err != nil {
			log.Warnf("get agent conn addr=%v err=%v", nodeInfo.Address, err)
			continue
		}

		rep, err := GetBackupJob(conn, job.ID)
		if err != nil {
			log.Warnf("GetBackupJob id=%v err=%v", job.ID, err)
			continue
		}

		if rep.Status != 0 || time.Since(time.Unix(rep.UpdatedAt, 0)) > time.Minute*5 {
			job.ImageRef = rep.ImageRef
			job.ImageID = rep.ImageId
			job.ImageSize = rep.ImageSize
			job.Status = int8(rep.Status)
			if job.Status == 0 { // 超时
				job.Status = 2
			}

			if err := model.UpdateContainerBackup(job); err != nil {
				log.Warnf("model.UpdateContainerBackup id=%v err=%v", job.ID, err)
				continue
			}

			_, err = DelBackupJob(conn, job.ID)
			if err != nil {
				log.Warnf("DelBackupJob id=%v err=%v", job.ID, err)
			}
		}
	}
}

func CheckContainerBackupJob() {
	for {
		if isMaster() {
			checkContainerBackupJob()
		}
		time.Sleep(time.Minute)
	}
}

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

type IllegalContainerDetection struct{}

func (IllegalContainerDetection) ScanNodeContainers(images map[string]int, n *model.NodeInfo) {
	conn, err := getAgentConn(n.Address)
	if err != nil {
		log.Warnf("restart container ErrInternal")
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

		if _, ok := images[c.Info.Image]; !ok {
			log.Debugf("stop container=%s", c.Info.Name)
			if _, err := StopContainer(conn, c.Info.Id); err != nil {
				log.Infof("StopContainer %v err=%v", c.Info.Id, err)
			}

			warnLog := []*model.WarnLog{
				{
					NodeId:        n.ID,
					NodeInfo:      fmt.Sprintf("%s (%s)", n.Name, n.Address),
					EventType:     int64(logging.EVENT_TYPE_WARN_ILLEGAL_CONTAINER),
					EventModule:   int64(logging.EVENT_MODULE_CONTAINER),
					ContainerName: c.Info.Name,
					Detail:        "容器镜像非法",
				},
			}

			model.CreateWarnLog(warnLog)
		}
	}
}

func (t IllegalContainerDetection) Run() {
	validImages, err := allValidImages()
	if err != nil {
		log.Infof("get valid images err=%v", err)
		return
	}

	nodes, err := model.ListNodes()
	if err != nil {
		log.Infof("get node list from DB err=%v", err)
		return
	}

	for _, n := range nodes {
		t.ScanNodeContainers(validImages, &n)
	}
}

func DetectIllegalContainer() {
	for {
		if isMaster() {
			IllegalContainerDetection{}.Run()
		}
		time.Sleep(time.Minute)
	}
}
