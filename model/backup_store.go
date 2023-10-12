package model

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"scmc/common"
	"strings"
	"sync"
	"time"

	"github.com/docker/docker/api/types"
	log "github.com/sirupsen/logrus"
)

var backup = &backupJobManager{}

type backupJobManager struct {
	sync.Mutex

	m map[int64]*ContainerBackupJob
}

func (b *backupJobManager) readFile() error {
	data, err := ioutil.ReadFile(common.Config.Agent.BackupJob)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			b.m = make(map[int64]*ContainerBackupJob)
			return nil
		}
		log.Warnf("read backup job file err=%v", err)
		return err
	}

	if err := json.Unmarshal(data, &b.m); err != nil {
		log.Warnf("unmarshal backup job from json err=%v", err)
		return err
	}

	return nil
}

func (b *backupJobManager) writeFile() error {
	data, err := json.MarshalIndent(b.m, "", "\t")
	if err != nil {
		log.Warnf("marshal backup job to json err=%v", err)
		return err
	}

	if err := ioutil.WriteFile(common.Config.Agent.BackupJob, data, 0644); err != nil {
		log.Warnf("write backup job file err=%v", err)
		return err
	}

	return nil
}

func (b *backupJobManager) withLock(f func() error) error {
	b.Lock()
	defer b.Unlock()

	if err := b.readFile(); err != nil {
		return err
	}

	if err := f(); err != nil {
		return err
	}

	return b.writeFile()
}

func (b *backupJobManager) add(id int64, containerID, name string) (*ContainerBackupJob, error) {
	j := ContainerBackupJob{
		ID:          id,
		ContainerID: containerID,
		BackupName:  name,
		Status:      0,
		UpdatedAt:   time.Now().Unix(),
	}

	err := b.withLock(func() error {
		if _, ok := b.m[id]; ok {
			return fmt.Errorf("backup job id=%v already exist", id)
		}
		b.m[id] = &j
		return nil
	})

	if err != nil {
		return nil, err
	}
	return &j, nil
}

func (b *backupJobManager) update(j *ContainerBackupJob) error {
	if j == nil {
		return nil
	}

	return b.withLock(func() error {
		b.m[j.ID] = j
		return nil
	})
}

func (b *backupJobManager) get(id int64) (*ContainerBackupJob, error) {
	var j *ContainerBackupJob
	err := b.withLock(func() error {
		j, _ = b.m[id]
		return nil
	})

	return j, err
}

func (b *backupJobManager) del(id int64) error {
	return b.withLock(func() error {
		delete(b.m, id)
		return nil
	})
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

func addBackupJob(j *ContainerBackupJob) {
	cli, err := DockerClient()
	if err != nil {
		log.Warnf("create docker cli err=%v", err)
		j.Status = 2
		return
	}

	info, err := cli.ContainerInspect(context.Background(), j.ContainerID)
	if err != nil {
		log.Warnf("inspect container=%v err=%v", j.ContainerID, err)
		j.Status = 2
		return
	}

	if info.Config == nil {
		log.Warnf("inspect container=%v .Config=nil", j.ContainerID)
		j.Status = 2
		return
	}

	j.ImageRef = strings.Split(info.Config.Image, ":")[0] + ":" + j.BackupName
	newImage, err := cli.ContainerCommit(context.Background(), j.ContainerID, types.ContainerCommitOptions{
		Reference: j.ImageRef,
		Pause:     true,
	})
	if err != nil {
		log.Warnf("commit container=%v err=%v", j.ContainerID, err)
		j.Status = 2
		return
	}

	imageInfo, _, err := cli.ImageInspectWithRaw(context.Background(), newImage.ID)
	if err != nil {
		log.Warnf("inspect image=%v err=%v", newImage.ID, err)
	}

	log.Debugf("backup id=%v finished", j.ID)
	j.ImageID = strings.TrimPrefix(newImage.ID, "sha256:")
	j.ImageSize = imageInfo.Size
	j.Status = 1

	backup.update(j)
}

func AddContainerBackupJob(id int64, containerID, name string) error {
	job, err := backup.add(id, containerID, name)
	if err != nil {
		log.Warnf("add backup job err=%v", err)
		return err
	}

	go addBackupJob(job)

	return nil
}

func GetContainerBackupJob(id int64) (*ContainerBackupJob, error) {
	return backup.get(id)
}

func DelContainerBackupJob(id int64) error {
	return backup.del(id)
}
