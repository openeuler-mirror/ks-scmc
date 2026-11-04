package internal

import (
	"context"
	"strings"

	"github.com/docker/docker/api/types"
	// "github.com/google/uuid"
	// "github.com/google/uuid"
	"scmc/model/authz"
	"scmc/model/iptables"
	"scmc/model/opensnitch"

	"github.com/shirou/gopsutil/disk"
	"github.com/shirou/gopsutil/mem"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"

	log "github.com/sirupsen/logrus"

	"scmc/model/docker"
	"scmc/model/kysec"
	"scmc/model/litedb"
	"scmc/rpc"
	pb "scmc/rpc/pb/node"
	"scmc/rpc/pb/security"
)

const gigaBytes = 1 << 30

type NodeServer struct {
	pb.UnimplementedNodeServer
}

// 默认端口
var blockedPorts = map[uint32]bool{
	3306:  true, // mysql
	8080:  true, // cadvisor
	8086:  true, // influxdb
	19091: true,
	10051: true, // agent
	10050: true, // controller
}

func nodeStatus() (*pb.NodeStatus, error) {
	cli, err := docker.Client()
	if err != nil {
		return &pb.NodeStatus{State: int64(pb.NodeState_Unknown)}, nil
	}

	var ret = pb.NodeStatus{
		State: int64(pb.NodeState_Online),
	}

	if memInfo, err := mem.VirtualMemory(); err != nil {
		log.Infof("VirtualMemory err=%v", err)
	} else {
		ret.MemStat = &pb.MemoryStat{
			Total:          memInfo.Total / megaBytes,
			Used:           memInfo.Used / megaBytes,
			Free:           memInfo.Free / megaBytes,
			UsedPercentage: memInfo.UsedPercent,
		}
	}

	if cpuUsage, err := globalCPUUsage.get(); err != nil {
		log.Infof("cpuUsage err=%v", err)
	} else {
		ret.CpuStat = &pb.CpuStat{
			Used:  cpuUsage,
			Total: float64(numCPU()),
		}
		ret.CpuStat.UsedPercentage = ret.CpuStat.Used / ret.CpuStat.Total
	}

	if diskUsage, err := disk.Usage("/"); err != nil {
		log.Infof("disk.Usage err=%v", err)
	} else {
		ret.DiskStat = &pb.DiskStat{
			Total:          diskUsage.Total / megaBytes,
			Free:           diskUsage.Free / megaBytes,
			Used:           diskUsage.Used / megaBytes,
			UsedPercentage: diskUsage.UsedPercent,
		}
	}

	containers, err := cli.ContainerList(context.Background(), types.ContainerListOptions{All: true, Size: true})
	if err == nil {
		runnings := 0
		for _, c := range containers {
			if c.State == "running" {
				runnings = runnings + 1
			}
		}
		ret.ContainerStat = &pb.ContainerStat{
			Running: int64(runnings),
			Total:   int64(len(containers)),
		}
	}

	return &ret, nil
}

func (*NodeServer) Status(ctx context.Context, in *pb.StatusRequest) (*pb.StatusReply, error) {
	s, err := nodeStatus()
	if err != nil {
		log.Infof("nodeStatus err=%v", err)
		return nil, rpc.ErrInternal
	}

	return &pb.StatusReply{
		StatusList: []*pb.NodeStatus{s},
	}, nil
}

func (*NodeServer) Usage(ctx context.Context, in *pb.UsageRequest) (*pb.UsageReply, error) {
	_, err := docker.Client()
	if err != nil {
		return nil, rpc.ErrInternal
	}

	data, err := influxdbQueryNodeMax(in.StartTime, in.EndTime)
	if err != nil {
		return nil, rpc.ErrInternal
	}

	reply := &pb.UsageReply{
		Status: &pb.NodeStatus{
			State: int64(pb.NodeState_Online),
			CpuStat: &pb.CpuStat{
				Used: data[0],
			},
			MemStat: &pb.MemoryStat{
				Used: uint64(data[1]),
			},
			DiskStat: &pb.DiskStat{
				Used: uint64(data[2]),
			},
		},
	}

	return reply, nil
}

func (s *NodeServer) UpdateFileProtect(ctx context.Context, in *pb.UpdateFileProtectRequest) (*pb.UpdateFileProtectReply, error) {
	log.Debugf("UpdateFileProtect called: node_id=%v ,paths:%s enabel:%v at agent",
		in.NodeId, in.SecurityConfig.FileProtection.FileList, in.SecurityConfig.FileProtection.IsOn)
	if in.SecurityConfig != nil {	
		cfgs, err := litedb.GetNodeConfig("localhost")
		if err != nil {
			err = s.nodeConfigDataTableCreate("","")
			if err != nil{
				return nil,err
			}
			cfgs, err = litedb.GetNodeConfig("localhost")
			if err != nil {
				log.Errorf("GetContainerConfig err=%v", err)
				return nil, rpc.ErrDatabaseFail
			}
		}

		if err := kysec.UpdateFileAccess("localhost", in.SecurityConfig.FileProtection.IsOn, in.SecurityConfig.FileProtection.FileList, nil); err != nil {
			log.Errorf("UpdateFileAccess failed: %v", err)
			return nil, status.Errorf(codes.Internal, "failed to update file access")
		}

		cfgs.SecurityConfig.FileProtection = in.SecurityConfig.FileProtection

		if err := litedb.UpdateNodeConfig(cfgs); err != nil {
			log.Errorf("UpdateContainerConfig err=%v", err)
			return nil, rpc.ErrDatabaseFail
		}
	}

	return &pb.UpdateFileProtectReply{}, nil
}

func (s *NodeServer) UpdateNetworkRule(ctx context.Context, in *pb.UpdateNetworkRuleRequest) (*pb.UpdateNetworkRuleReply, error) {
	log.Debugf("UpdateNetworkRule called: node_id=%v , enabel:%v  rule:%v at agent", in.NodeId, in.SecurityConfig.NetworkRule.IsOn, in.SecurityConfig.NetworkRule.Rules)

	var mergedRules []*litedb.NetworkRule

	if in.SecurityConfig.NetworkRule != nil {
		// 如果开关开启，添加默认阻断端口规则
		if in.SecurityConfig.NetworkRule.IsOn {
			for port := range blockedPorts {
				mergedRules = append(mergedRules, &litedb.NetworkRule{
					Protocols: []string{"tcp"},
					Addr:      "0.0.0.0/0",
					Port:      port,
				})
			}
		}

		for _, v := range in.SecurityConfig.NetworkRule.Rules {
			mergedRules = append(mergedRules, &litedb.NetworkRule{
				Protocols: v.Protocols,
				Addr:      v.Addr,
				Port:      v.Port,
			})
		}

		if err := iptables.UpdateNodeIPtablesFile(in.SecurityConfig.NetworkRule.IsOn, mergedRules); err != nil {
			log.Warnf("UpdateContainerIPtablesFile err: %v", err)
			return nil, rpc.ErrContainerNetworkRule
		}

		cfgs, err := litedb.GetNodeConfig("localhost")
		if err != nil {
			err = s.nodeConfigDataTableCreate("","")
			if err != nil{
				return nil,err
			}
			cfgs, err = litedb.GetNodeConfig("localhost")
			if err != nil {
				log.Errorf("GetContainerConfig err=%v", err)
				return nil, rpc.ErrDatabaseFail
			}
		}

		var securityRules []*security.NetworkRule
		for _, rule := range mergedRules {
			securityRules = append(securityRules, &security.NetworkRule{
				Protocols: rule.Protocols,
				Addr:      rule.Addr,
				Port:      rule.Port,
			})
		}

		var isOn bool
		if in.SecurityConfig.NetworkRule.IsOn {
			isOn = true
		} else {
			isOn = false
		}

		cfgs.SecurityConfig.NetworkRule = &security.NetworkRuleList{
			IsOn:  isOn,
			Rules: securityRules,
		}

		if err := litedb.UpdateNodeConfig(cfgs); err != nil {
			log.Errorf("UpdateContainerConfig err=%v", err)
			return nil, rpc.ErrDatabaseFail
		}
	}

	return &pb.UpdateNetworkRuleReply{}, nil
}

func (s *NodeServer) GetFileProtect(ctx context.Context, in *pb.GetFileProtectRequest) (*pb.GetFileProtectReply, error) {
	log.Debugf("GetFileProtect called: node_id=%v  at agent ", in.NodeId)

	cfgs, err := litedb.GetNodeConfig("localhost")
	if err != nil {
		err = s.nodeConfigDataTableCreate("","")
		if err != nil{
			return nil,err
		}
		cfgs, err = litedb.GetNodeConfig("localhost")
		if err != nil {
			log.Errorf("GetContainerConfig err=%v", err)
			return nil, rpc.ErrDatabaseFail
		}
	}

	var reply pb.GetFileProtectReply

	if cfgs != nil && cfgs.SecurityConfig != nil && cfgs.SecurityConfig.FileProtection != nil {

		reply.FileProtection = &security.FileProtection{
			IsOn:     cfgs.SecurityConfig.FileProtection.IsOn,
			FileList: cfgs.SecurityConfig.FileProtection.FileList,
		}
	} else {
		reply.FileProtection = &security.FileProtection{
			IsOn:     false,
			FileList: []string{},
		}
		log.Warnf("FileProtection data not found for node_id=%v", in.NodeId)
	}

	log.Debugf("GetFileProtect called: node_id=%v  at agent reply:%v", in.NodeId, reply.FileProtection)

	return &reply, nil
}

func (s *NodeServer) GetNetworkRule(ctx context.Context, in *pb.GetNetworkRuleRequest) (*pb.GetNetworkRuleReply, error) {
	log.Debugf("GetNetworkRule called: node_id=%v  at agent ", in.NodeId)

	cfgs, err := litedb.GetNodeConfig("localhost")
	if err != nil {
		err = s.nodeConfigDataTableCreate("","")
		if err != nil{
			return nil,err
		}
		cfgs, err = litedb.GetNodeConfig("localhost")
		if err != nil {
			log.Errorf("GetContainerConfig err=%v", err)
			return nil, rpc.ErrDatabaseFail
		}
	}

	var reply pb.GetNetworkRuleReply

	if cfgs != nil && cfgs.SecurityConfig != nil && cfgs.SecurityConfig.NetworkRule != nil {
		reply.NetworkRule = &security.NetworkRuleList{
			IsOn:  cfgs.SecurityConfig.NetworkRule.IsOn,
			Rules: cfgs.SecurityConfig.NetworkRule.Rules,
		}
	} else {
		reply.NetworkRule = &security.NetworkRuleList{
			IsOn:  false,
			Rules: nil,
		}
		log.Warnf("GetNetworkRule data not found for node_id=%v", in.NodeId)
	}

	log.Debugf("GetNetworkRule called: node_id=%v reply:%v at agent ", in.NodeId, reply)

	return &reply, nil
}

func (s *NodeServer) GetNodeSecurityConfig(ctx context.Context, in *pb.GetNodeSecurityConfigRequest) (*pb.GetNodeSecurityConfigReply, error) {
	log.Debugf("UpdateFileProtect called: node_id=%v  at agent ", in.NodeId)

	cfgs, err := litedb.GetNodeConfig("localhost")
	if err != nil {
		log.Errorf("GetContainerConfig err=%v", err)
		return nil, rpc.ErrDatabaseFail
	}

	var reply pb.GetNodeSecurityConfigReply

	if cfgs != nil && cfgs.SecurityConfig != nil {
		reply.SecurityConfig = cfgs.SecurityConfig
	} else {
		return nil, rpc.ErrNotFound
	}

	log.Debugf("GetNetworkRule called: node_id=%v reply:%v at agent ", in.NodeId, reply)

	return &reply, nil

}

func (s *NodeServer) Update(ctx context.Context, in *pb.UpdateRequest) (*pb.UpdateReply, error) {
	if in.RscLimit == nil {
		return nil, rpc.ErrInvalidArgument
	}

	if memInfo, err := mem.VirtualMemory(); err != nil {
		log.Warnf("VirtualMemory err=%v", err)
		return nil, rpc.ErrInternal
	} else {
		memTotal := float64(memInfo.Total / megaBytes)
		if memTotal < in.RscLimit.MemoryLimit {
			return nil, status.Errorf(codes.InvalidArgument, "内存参数超过上限%.0f", memTotal)
		}
	}

	cpuNum := float64(numCPU())
	if cpuNum < in.RscLimit.CpuLimit {
		return nil, status.Errorf(codes.InvalidArgument, "CPU参数超过上限%.0f", cpuNum)
	}

	if diskUsage, err := disk.Usage("/"); err != nil {
		log.Warnf("disk.Usage err=%v", err)
		return nil, rpc.ErrInternal
	} else {
		label := "GB"
		diskTotal := float64(diskUsage.Total / gigaBytes)
		if diskUsage.Total < gigaBytes {
			label = "MB"
			diskTotal = float64(diskUsage.Total / megaBytes)
		}

		if float64(diskUsage.Total/megaBytes) < in.RscLimit.DiskLimit {
			return nil, status.Errorf(codes.InvalidArgument, "磁盘参数超过上限%.0f%s", diskTotal, label)
		}
	}

	return &pb.UpdateReply{}, nil
}

func (s *NodeServer) setSecurityConfig(id string, uuid, name string, pid int, fromUpdate bool, sec *pb.SecurityConfig) error {
	log.Debugf("NodeServer setSecurityConfig")

	if sec == nil {
		return nil
	}

	if fromUpdate {
		if err := kysec.CleanAll(id, false); err != nil {
			log.Warnf("kysec.CleanAll: %v", err)
			return rpc.ErrContainerFileProtection
		}
	}

	if sec.ProcProtection != nil {
		var hashes []string
		for _, p := range sec.ProcProtection.ExeList {
			s, err := docker.ContainerFileMD5(id, p)
			if err != nil {
				log.Warnf("ContainerFileMD5 path=%v err=%v", p, err)
				return rpc.ErrContainerProcProtection
			}
			hashes = append(hashes, s)
		}

		if err := kysec.UpdateWhiteList(id, sec.ProcProtection.IsOn, hashes, []string{}); err != nil {
			log.Warnf("UpdateWhiteList %v err: %v", id, err)
			return rpc.ErrContainerProcProtection
		}
	}

	if sec.NprocProtection != nil {
		var inRule = &litedb.ProcProtection{
			IsOn:    sec.NprocProtection.IsOn,
			ExeList: sec.NprocProtection.ExeList,
		}

		if err := opensnitch.SaveRule(inRule, id, uuid); err != nil {
			log.Warnf("SaveRule id=%v err=%v", id, err)
			return rpc.ErrContainerNprocProtection
		}
	}

	if sec.FileProtection != nil {
		if err := kysec.UpdateFileAccess(id, sec.FileProtection.IsOn, sec.FileProtection.FileList, []string{}); err != nil {
			log.Warnf("UpdateFileAccess %v err: %v", id, err)
			return rpc.ErrContainerFileProtection
		}
	}

	if err := authz.HandleSensitiveContainer(sec.DisableCmdOperation, fromUpdate, name, id); err != nil {
		log.Warnf("HandleSensitiveContainer err=%v", err)
		return rpc.ErrContainerCmdOperation
	}

	if sec.NetworkRule != nil {
		var rules []*litedb.NetworkRule
		for _, v := range sec.NetworkRule.Rules {
			rules = append(rules, &litedb.NetworkRule{
				Protocols: v.Protocols,
				Addr:      v.Addr,
				Port:      v.Port,
			})
		}

		fileName := id + "-" + name
		if err := iptables.UpdateContainerIPtablesFile(fileName, sec.NetworkRule.IsOn, rules, pid); err != nil {
			log.Warnf("UpdateContainerIPtablesFile %v err: %v", fileName, err)
			return rpc.ErrContainerNetworkRule
		}
	}

	return nil
}

func (s *NodeServer) nodeConfigDataTableCreate(uuid string,nodeName string) ( error ) {
	
	if uuid == "" {
		nodeInfo,err := litedb.GetNodeInfo(1)
		
		if err != nil{
			return err;
		}
		
		uuid = nodeInfo.UUID
	}
	
	securityConfig := &pb.SecurityConfig{
		NetworkRule: &security.NetworkRuleList{
			IsOn:  false,
			Rules: []*security.NetworkRule{},
		},
	}

	for port := range blockedPorts {
		securityConfig.NetworkRule.Rules = append(
			securityConfig.NetworkRule.Rules,
			&security.NetworkRule{
				Protocols: []string{"tcp"},
				Addr:      "0.0.0.0/0",
				Port:      port,
			},
		)
	}

	cfgs := litedb.NodeConfig{
		ID:             1, // 强制设置id=1，表只能有一条记录
		UUID:           uuid,
		NodeID:         "localhost",
		NodeName:       nodeName,
		SecurityConfig: securityConfig,
	}

	if err := litedb.CreateNodeConfig(&cfgs); err != nil {
		log.Infof("CreateNodeConfig err=%v", err)
		return  rpc.ErrDatabaseFail
	}
	
	return nil
}

func (s *NodeServer) Create(ctx context.Context, in *pb.CreateRequest) (*pb.CreateReply, error) {
	if len(in.Uuid) == 0 {
		return nil, rpc.ErrInvalidArgument
	}

	err := litedb.CreateNodeInfo(1, in.Uuid) // 强制设置id=1，表只能有一条记录
	if err != nil {
		log.Warnf("Node.Create uuid=%s err=%v", in.Uuid, err)
		if strings.HasPrefix(err.Error(), "UNIQUE constraint failed") {
			return nil, status.Error(codes.Internal, "节点已加入其他集群")
		}
		return nil, rpc.ErrDatabaseFail
	}

	err = s.nodeConfigDataTableCreate(in.Uuid,in.Name)
	if err != nil{
		return nil,err
	}	
	return &pb.CreateReply{}, nil
}

func (s *NodeServer) Remove(ctx context.Context, in *pb.RemoveRequest) (*pb.RemoveReply, error) {
	err := litedb.RemoveNodeInfo(1, in.Uuid) // 强制设置id=1，表只能有一条记录

	res := litedb.RemoveNodeConfigByID(1) // 强制设置id=1，表只能有一条记录

	if err != nil && res == nil {
		log.Warnf("Node.Remove uuid=%s err=%v", in.Uuid, err)
		if err == gorm.ErrRecordNotFound {
			goto success
		}
		return nil, rpc.ErrDatabaseFail
	}

success:
	return &pb.RemoveReply{}, nil
}
