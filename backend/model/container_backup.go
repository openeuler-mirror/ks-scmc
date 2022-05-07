package model

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	log "github.com/sirupsen/logrus"
)

var (
	containerBackupJobs      = make(map[int64]*ContainerBackupJob, 10)
	containerBackupJobsGuard = sync.RWMutex{}
)

type ContainerBackup struct {
	ID          int64 `gorm:"primaryKey"`
	NodeID      int64
	UUID        string
	ContainerID string
	BackupName  string
	BackupDesc  string
	ImageRef    string
	ImageID     string
	ImageSize   int64
	Status      int8  // 0:备份中 1:成功 2:失败
	CreatedAt   int64 `gorm:"autoCreateTime"`
	UpdatedAt   int64 `gorm:"autoUpdateTime"`
}

type ContainerBackupJob struct {
	ID          int64
	ContainerID string
	BackupName  string
	ImageRef    string
	ImageID     string
	ImageSize   int64
	Status      int64
	UpdatedAt   int64
}

func CreateContainerBackup(nodeID int64, uuid string, backupName, backupDesc string) (*ContainerBackup, error) {
	db, err := getConn()
	if err != nil {
		return nil, err
	}

	data := ContainerBackup{
		NodeID:     nodeID,
		UUID:       uuid,
		BackupName: backupName,
		BackupDesc: backupDesc,
	}

	if err := db.Create(&data).Error; err != nil {
		log.Errorf("db create container backup %v", err)
		return nil, translateError(err)
	}

	return &data, nil
}

func UpdateContainerBackup(data *ContainerBackup) error {
	db, err := getConn()
	if err != nil {
		return err
	}

	if err := db.Save(data).Error; err != nil {
		log.Errorf("db update container backup %v", err)
		return translateError(err)
	}

	return nil
}

func RemoveContainerBackup(data *ContainerBackup) error {
	db, err := getConn()
	if err != nil {
		return err
	}

	if err := db.Delete(data).Error; err != nil {
		log.Errorf("db remove container backup %v", err)
		return translateError(err)
	}

	return nil
}

func RemoveContainerBackupByUUID(uuid []string) error {
	db, err := getConn()
	if err != nil {
		return err
	}

	result := db.Where("uuid IN ?", uuid).Delete(ContainerBackup{})
	if result.Error != nil {
		log.Errorf("db remove container backup, uuid=%v: %v", uuid, result.Error)
		return result.Error
	}

	return nil
}

func QueryContainerBackupByID(id int64) (*ContainerBackup, error) {
	db, err := getConn()
	if err != nil {
		return nil, err
	}

	var data ContainerBackup
	if err := db.First(&data, "id = ?", id).Error; err != nil {
		log.Warnf("QueryContainerBackupByID err=%v", err)
		return nil, translateError(err)
	}

	return &data, nil
}

func QueryContainerBackupByUUID(uuid string) ([]*ContainerBackup, error) {
	db, err := getConn()
	if err != nil {
		return nil, err
	}

	var data []*ContainerBackup
	if err := db.Find(&data, "uuid = ?", uuid).Error; err != nil {
		log.Warnf("QueryContainerBackupByUUID err=%v", err)
		return nil, translateError(err)
	}

	return data, nil
}

func QueryUndoneContainerBackup() ([]*ContainerBackup, error) {
	db, err := getConn()
	if err != nil {
		return nil, err
	}

	var data []*ContainerBackup
	if err := db.Find(&data, "status = 0").Error; err != nil {
		log.Warnf("QueryUndoneContainerBackup err=%v", err)
		return nil, translateError(err)
	}

	return data, nil
}

func ListContainerBackup() ([]*ContainerBackup, error) {
	db, err := getConn()
	if err != nil {
		return nil, err
	}

	var data []*ContainerBackup
	if err := db.Find(&data).Error; err != nil {
		log.Warnf("ListContainerBackup err=%v", err)
		return nil, err
	}

	return data, nil
}

func AddContainerBackupJob(id int64, containerID, name string) error {
	job := &ContainerBackupJob{
		ID:          id,
		ContainerID: containerID,
		BackupName:  name,
		Status:      0,
		UpdatedAt:   time.Now().Unix(),
	}

	go func(job *ContainerBackupJob) {
		containerBackupJobsGuard.Lock()
		defer containerBackupJobsGuard.Unlock()

		if _, ok := containerBackupJobs[job.ID]; ok {
			log.Infof("backup job id=%v already exists", job.ID)
			return
		}

		containerBackupJobs[job.ID] = job

		cli, err := client.NewClientWithOpts(
			client.FromEnv,
			client.WithAPIVersionNegotiation(),
			client.WithHTTPHeaders(map[string]string{"AuthZ-User": "KS-SCMC-SERVICE"}),
		)
		if err != nil {
			log.Warnf("create docker cli err=%v", err)
			job.Status = 2
			return
		}

		info, err := cli.ContainerInspect(context.Background(), job.ContainerID)
		if err != nil {
			log.Warnf("inspect container=%v err=%v", job.ContainerID, err)
			job.Status = 2
			return
		}

		if info.Config == nil {
			log.Warnf("inspect container=%v .Config=nil", job.ContainerID)
			job.Status = 2
			return
		}

		job.ImageRef = strings.Split(info.Config.Image, ":")[0] + ":" + job.BackupName
		newImage, err := cli.ContainerCommit(context.Background(), job.ContainerID, types.ContainerCommitOptions{
			Reference: job.ImageRef,
			Pause:     true,
		})
		if err != nil {
			log.Warnf("commit container=%v err=%v", job.ContainerID, err)
			job.Status = 2
			return
		}

		imageInfo, _, err := cli.ImageInspectWithRaw(context.Background(), newImage.ID)
		if err != nil {
			log.Warnf("inspect image=%v err=%v", newImage.ID, err)
		}

		log.Debugf("backup id=%v finished", job.ID)
		job.ImageID = strings.TrimPrefix(newImage.ID, "sha256:")
		job.ImageSize = imageInfo.Size
		job.Status = 1

	}(job)

	return nil
}

func GetContainerBackupJob(id int64) *ContainerBackupJob {
	containerBackupJobsGuard.RLock()
	defer containerBackupJobsGuard.RUnlock()

	r, ok := containerBackupJobs[id]
	if ok {
		return r
	}

	return nil
}

func DelContainerBackupJob(id int64) {
	containerBackupJobsGuard.Lock()
	defer containerBackupJobsGuard.Unlock()

	_, ok := containerBackupJobs[id]
	if ok {
		delete(containerBackupJobs, id)
		log.Debugf("backup id=%v deleted", id)
	}
}
