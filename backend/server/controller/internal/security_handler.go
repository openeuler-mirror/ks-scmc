package internal

import (
	"context"
	"encoding/json"
	"reflect"
	"time"

	log "github.com/sirupsen/logrus"

	"scmc/model"
	"scmc/rpc"
	pb "scmc/rpc/pb/security"
)

type SecurityServer struct {
	pb.UnimplementedSecurityServer
}

func (s *SecurityServer) ListFileProtection(ctx context.Context, in *pb.ListFileProtectionRequest) (*pb.ListFileProtectionReply, error) {
	reply := pb.ListFileProtectionReply{}
	data, err := model.GetContainerConfigs(in.NodeId, in.ContainerId)
	if err != nil {
		log.Warnf("get container config err: %v", err)
		return &reply, err
	}

	if data != nil && data.SecurityConfig != "" {
		var info model.SecurityConfigs
		err = json.Unmarshal([]byte(data.SecurityConfig), &info)
		if err != nil {
			log.Errorf("Unmarshal failed: %v", err)
			return &reply, err
		}

		reply.IsOn = info.FileProtection.IsOn
		reply.FileList = info.FileProtection.FileList
	}

	return &reply, nil
}

func (s *SecurityServer) UpdateFileProtection(ctx context.Context, in *pb.UpdateFileProtectionRequest) (*pb.UpdateFileProtectionReply, error) {
	containerConfigs, err := model.GetContainerConfigs(in.NodeId, in.ContainerId)
	if err != nil {
		if in.ContainerId != "localhost" {
			return nil, err
		} else {
			if containerConfigs == nil {
				containerConfigs = &model.ContainerConfigs{
					NodeID:      in.NodeId,
					ContainerID: in.ContainerId,
				}
			}
		}
	}
	var info model.SecurityConfigs
	if containerConfigs.SecurityConfig != "" {
		err = json.Unmarshal([]byte(containerConfigs.SecurityConfig), &info)
		if err != nil {
			log.Errorf("Unmarshal failed: %v", err)
		}
	}

	info.FileProtection.IsOn = in.IsOn
	m := make(map[string]struct{})
	for _, v := range info.FileProtection.FileList {
		m[v] = struct{}{}
	}
	for _, v := range in.ToRemove {
		if _, ok := m[v]; ok {
			delete(m, v)
		}
	}
	var fileList []string
	for k, _ := range m {
		fileList = append(fileList, k)
	}
	for _, v := range in.ToAppend {
		fileList = append(fileList, v)
	}

	info.FileProtection.FileList = fileList

	data, err := json.Marshal(info)
	containerConfigs.SecurityConfig = string(data)
	if err := model.UpdateContainerConfigs(containerConfigs); err != nil {
		log.Infof("UpdateFileProtection: UpdateContainerConfigs nodeId=%s container_id=%s err=%v", in.NodeId, in.ContainerId, err)
		// still return OK
	}

	nodeInfo, err := model.QueryNodeByID(in.NodeId)
	if err != nil {
		if err == model.ErrRecordNotFound {
			return nil, rpc.ErrNotFound
		}
		return nil, rpc.ErrInternal
	}

	conn, err := getAgentConn(nodeInfo.Address)
	if err != nil {
		return nil, rpc.ErrInternal
	}

	cli := pb.NewSecurityClient(conn)
	ctx_, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	agentReply, err := cli.UpdateFileProtection(ctx_, in)
	if err != nil {
		log.Warnf("UpdateFileProtection: %v", err)
		return nil, err
	}

	return agentReply, nil
}

func dealDbData() (map[int64]pb.LoadSecurityConfigRequset, error) {
	datas, err := model.ListContainerConfigs()
	if err != nil {
		return nil, err
	}

	mapDatas := make(map[int64]pb.LoadSecurityConfigRequset)
	for _, data := range datas {
		var info model.SecurityConfigs
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
			if !reflect.DeepEqual(info.ProcProtection, model.ProcProtection{}) {
				proc = &pb.ProcProtection{
					ProtectionType: int64(info.ProcProtection.Type),
					IsOn:           info.ProcProtection.IsOn,
					ExeList:        info.ProcProtection.ExeList,
				}
			}
		}

		if info.FileProtection != nil {
			if !reflect.DeepEqual(info.FileProtection, model.FileProtection{}) {
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
		in := pb.LoadSecurityConfigRequset{
			Configs: configs,
		}
		mapDatas[data.NodeID] = in
	}

	return mapDatas, nil
}

func resumeContainerConfigs() {
	datas, err := dealDbData()
	if err != nil {
		return
	}

	for k, v := range datas {
		// log.Debugf("datas: [%v][%v]", k, v.Configs)
		nodeInfo, err := model.QueryNodeByID(k)
		if err != nil {
			return
		}

		conn, err := getAgentConn(nodeInfo.Address)
		if err != nil {
			return
		}
		cli := pb.NewSecurityClient(conn)
		ctx_, cancel := context.WithTimeout(context.Background(), time.Second*10)
		defer cancel()

		cli.LoadSecurityConfig(ctx_, &v)
	}
}

func ResumeContainerConfigs() {
	for {
		if isMaster() {
			resumeContainerConfigs()
		}
		time.Sleep(time.Minute)
	}
}

func (sp *SecurityServer) ListProcProtection(ctx context.Context, in *pb.ListProcProtectionRequest) (*pb.ListProcProtectionReply, error) {
	if in.NodeId <= 0 || in.ContainerId == "" {
		return nil, rpc.ErrInvalidArgument
	}

	cc, err := model.GetContainerConfigs(in.NodeId, in.ContainerId)
	if err != nil {
		log.Warn(err)
		return nil, rpc.ErrInternal
	}
	var sc model.SecurityConfigs
	if cc.SecurityConfig == "" {
		return &pb.ListProcProtectionReply{
			IsOn:    false,
			ExeList: nil,
		}, nil
	}
	if err = json.Unmarshal([]byte(cc.SecurityConfig), &sc); err != nil {
		log.Warn(err)
		return nil, rpc.ErrInternal
	}

	switch in.ProtectionType {
	case int64(pb.PROC_PROTECTION_EXEC_WHITELIST):
		if sc.ProcProtection == nil {
			return nil, rpc.ErrNotFound
		}
		return &pb.ListProcProtectionReply{
			IsOn:    sc.ProcProtection.IsOn,
			ExeList: sc.ProcProtection.ExeList,
		}, nil
	case int64(pb.PROC_PROTECTION_NET_WHITELIST):
		if sc.NprocProtection == nil {
			return nil, rpc.ErrNotFound
		}
		return &pb.ListProcProtectionReply{
			IsOn:    sc.NprocProtection.IsOn,
			ExeList: sc.NprocProtection.ExeList,
		}, nil

	default:
		return nil, rpc.ErrInvalidArgument
	}

}

func (sp *SecurityServer) UpdateProcProtection(ctx context.Context, in *pb.UpdateProcProtectionRequest) (*pb.UpdateProcProtectionReply, error) {
	if in.NodeId <= 0 || in.ContainerId == "" {
		return nil, rpc.ErrInvalidArgument
	}

	cc, err := model.GetContainerConfigs(in.NodeId, in.ContainerId)
	if err != nil {
		log.Warn(err)
		return nil, rpc.ErrInternal
	}
	var sc model.SecurityConfigs
	if cc.SecurityConfig != "" {
		if err = json.Unmarshal([]byte(cc.SecurityConfig), &sc); err != nil {
			log.Warn(err)
			return nil, rpc.ErrInternal
		}
	}

	switch in.ProtectionType {
	case int64(pb.PROC_PROTECTION_EXEC_WHITELIST):
		if sc.ProcProtection == nil {
			sc.ProcProtection = new(model.ProcProtection)
		}
		sc.ProcProtection.IsOn = in.GetIsOn()
		exeList := sc.ProcProtection.ExeList

		// 获取最终保存在db中的数据
		sc.ProcProtection.ExeList = union(in.GetToAppend(), sc.ProcProtection.ExeList)
		sc.ProcProtection.ExeList = difference(sc.ProcProtection.ExeList, in.GetToRemove())

		// 获取需要append数据
		in.ToAppend = difference(in.GetToAppend(), exeList)
		in.ToAppend = difference(in.ToAppend, in.GetToRemove())

	case int64(pb.PROC_PROTECTION_NET_WHITELIST):
		if sc.NprocProtection == nil {
			sc.NprocProtection = new(model.ProcProtection)
		}
		sc.NprocProtection.IsOn = in.GetIsOn()
		sc.NprocProtection.ExeList = union(in.GetToAppend(), sc.NprocProtection.ExeList)
		sc.NprocProtection.ExeList = difference(sc.NprocProtection.ExeList, in.GetToRemove())
		in.ToAppend = sc.NprocProtection.ExeList
		in.ToRemove = []string{}
	default:
		return nil, rpc.ErrInvalidArgument
	}

	reply := new(pb.UpdateProcProtectionReply)

	scByte, err := json.Marshal(sc)
	if err != nil {
		log.Warn(err)
		return nil, rpc.ErrInternal
	}

	securityConfig := string(scByte)
	if cc.SecurityConfig != securityConfig {

		cc.SecurityConfig = securityConfig
		//更新节点上规则
		log.Debug("update node security policy")
		nodeInfo, err := model.QueryNodeByID(in.NodeId)
		if err != nil {
			if err == model.ErrRecordNotFound {
				return nil, rpc.ErrNotFound
			}
			return nil, rpc.ErrInternal
		}

		conn, err := getAgentConn(nodeInfo.Address)
		if err != nil {
			return nil, rpc.ErrInternal
		}

		cli := pb.NewSecurityClient(conn)
		ctx_, cancel := context.WithTimeout(context.Background(), time.Second*10)
		defer cancel()
		reply, err = cli.UpdateProcProtection(ctx_, in)
		if err != nil {
			log.Warnf("update %s network process protection file faild,err is :%s", in.ContainerId, err)
			return nil, err
		}

		//更新数据库数据
		log.Debug("update db security policy")
		err = model.UpdateContainerConfigs(cc)
		if err != nil {
			log.Warnf("update %s  network process protection security policy err: %s", in.ContainerId, err)
			return nil, rpc.ErrInternal
		}
	}

	return reply, nil

}

// 求并集
func union(A, B []string) []string {
	result := make([]string, 0)

	flagMap := make(map[string]bool, 0)
	A = append(A, B...)
	for _, a := range A {
		if _, ok := flagMap[a]; ok {
			continue
		}
		flagMap[a] = true
		result = append(result, a)
	}
	return result
}

// 求差集
func difference(A, B []string) []string {
	if len(A) < 1 || len(B) < 1 {
		return A
	}
	result := make([]string, 0)

	flagMap := make(map[string]bool, 0)
	for _, a := range A {
		if _, ok := flagMap[a]; ok {
			continue
		}
		flagMap[a] = true
		flag := true
		for _, b := range B {
			if b == a {
				flag = false
				break
			}
		}
		if flag {
			result = append(result, a)
		}
	}
	return result
}
