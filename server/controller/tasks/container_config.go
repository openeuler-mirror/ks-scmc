package tasks

import (
	"context"
	"encoding/json"
	"reflect"
	"scmc/model/db"
	pb "scmc/rpc/pb/security"
	"scmc/server/controller/internal"
	"time"

	log "github.com/sirupsen/logrus"
)

type ContainerConfig struct{}

func (c *ContainerConfig) Interval() time.Duration {
	return time.Minute
}

func (c *ContainerConfig) RunAtInit() bool {
	return true
}

func (c *ContainerConfig) Run() {
	if isMaster() {
		go resumeContainerConfigs()
	}
}

func resumeContainerConfigs() {
	datas, err := dealDbData()
	if err != nil {
		return
	}

	for k, v := range datas {
		// log.Debugf("datas: [%v][%v]", k, v.Configs)
		nodeInfo, err := db.QueryNodeByID(k)
		if err != nil {
			return
		}

		conn, err := internal.GetAgentConn(nodeInfo.Address)
		if err != nil {
			return
		}
		cli := pb.NewSecurityClient(conn)
		ctx_, cancel := context.WithTimeout(context.Background(), time.Second*10)
		defer cancel()

		cli.LoadSecurityConfig(ctx_, v)
	}
}

func dealDbData() (map[int64]*pb.LoadSecurityConfigRequset, error) {
	datas, err := db.ListContainerConfigs()
	if err != nil {
		return nil, err
	}

	mapDatas := make(map[int64]*pb.LoadSecurityConfigRequset)
	for _, data := range datas {
		var info db.SecurityConfigs
		if data.SecurityConfig != "" {
			err = json.Unmarshal([]byte(data.SecurityConfig), &info)
			if err != nil {
				log.Errorf("Unmarshal failed: %v", err)
				continue
			}
		} else {
			continue
		}

		var (
			proc    *pb.ProcProtection
			file    *pb.FileProtection
			configs []*pb.FullSeucirytConfig
		)

		if info.ProcProtection != nil {
			if !reflect.DeepEqual(info.ProcProtection, db.ProcProtection{}) {
				proc = &pb.ProcProtection{
					ProtectionType: int64(info.ProcProtection.Type),
					IsOn:           info.ProcProtection.IsOn,
					ExeList:        info.ProcProtection.ExeList,
				}
			}
		}

		if info.FileProtection != nil {
			if !reflect.DeepEqual(info.FileProtection, db.FileProtection{}) {
				file = &pb.FileProtection{
					IsOn:     info.FileProtection.IsOn,
					FileList: info.FileProtection.FileList,
				}
			}
		}

		config := &pb.FullSeucirytConfig{
			ContainerId:     data.ContainerID,
			ProcProtections: proc,
			FileProtections: file,
		}

		if _, exist := mapDatas[data.NodeID]; exist {
			configs = mapDatas[data.NodeID].Configs
		}

		configs = append(configs, config)
		in := &pb.LoadSecurityConfigRequset{
			Configs: configs,
		}
		mapDatas[data.NodeID] = in
	}

	return mapDatas, nil
}
