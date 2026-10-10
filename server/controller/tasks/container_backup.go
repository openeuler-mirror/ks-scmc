package tasks

import (
	"context"
	"scmc/model/db"
	pb "scmc/rpc/pb/container"
	"scmc/server/controller/internal"
	"time"

	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

const (
	ImageSourceNone   = 0
	ImageSourceUpload = 1
	ImageSourceBackup = 2
)

type ContainerBackup struct{}

func (c *ContainerBackup) Interval() time.Duration {
	return time.Minute
}

func (c *ContainerBackup) RunAtInit() bool {
	return true
}

func (c *ContainerBackup) Run() {
	if isMaster() {
		go checkContainerBackupJob()
	}
}

func allValidImages() (map[string]int, error) {
	dbImages, err := db.QueryImageByStatus()
	if err != nil {
		log.Warnf("query db images err=%v", err)
		return nil, err
	}

	backups, err := db.ListContainerBackup()
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
	jobs, err := db.QueryUndoneContainerBackup()
	if err != nil {
		log.Warnf("db.QueryContainerBackups err=%v", err)
		return
	}

	for _, job := range jobs {
		nodeInfo, err := db.QueryNodeByID(job.NodeID)
		if err != nil {
			log.Warnf("db.QueryNodeByID node_id=%v err=%v", job.NodeID, err)
			continue
		}

		conn, err := internal.GetAgentConn(nodeInfo.Address)
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

			if err := db.UpdateContainerBackup(job); err != nil {
				log.Warnf("db.UpdateContainerBackup id=%v err=%v", job.ID, err)
				continue
			}

			_, err = DelBackupJob(conn, job.ID)
			if err != nil {
				log.Warnf("DelBackupJob id=%v err=%v", job.ID, err)
			}
		}
	}
}
