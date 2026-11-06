package internal

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"

	// "scmc/model/authz"
	"scmc/model/iptables"
	"scmc/model/opensnitch"
	"sort"
	"strings"

	"github.com/docker/docker/api/types"

	"github.com/shirou/gopsutil/disk"
	"github.com/shirou/gopsutil/mem"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"

	"scmc/common"
	"scmc/model/docker"
	"scmc/model/kysec"
	"scmc/model/litedb"
	"scmc/rpc"
	pb "scmc/rpc/pb/node"
	"scmc/rpc/pb/security"

	log "github.com/sirupsen/logrus"
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

	if in.SecurityConfig.FileProtection == nil {
		return nil, rpc.ErrInvalidArgument
	}

	cfgs, err := litedb.GetNodeConfig("localhost")
	if err != nil {
		if err = s.nodeConfigDataTableCreate("", ""); err != nil {
			return nil, err
		}
		if cfgs, err = litedb.GetNodeConfig("localhost"); err != nil {
			log.Errorf("GetContainerConfig err=%v", err)
			return nil, rpc.ErrDatabaseFail
		}
	}

	for _, path := range in.SecurityConfig.FileProtection.FileList {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			return nil, status.Errorf(codes.NotFound, "配置节点文件保护失败-文件%s不存在", path)
		}
	}

	var oldFileList []string
	if cfgs.SecurityConfig.FileProtection != nil {
		oldFileList = cfgs.SecurityConfig.FileProtection.FileList
	}

	deletedFiles := make([]string, 0)
	oldFileMap := make(map[string]struct{})
	for _, f := range oldFileList {
		oldFileMap[f] = struct{}{}
	}
	newFileMap := make(map[string]struct{})
	for _, f := range in.SecurityConfig.FileProtection.FileList {
		newFileMap[f] = struct{}{}
	}

	for _, f := range oldFileList {
		if _, exists := newFileMap[f]; !exists {
			deletedFiles = append(deletedFiles, f)
		}
	}

	if err := kysec.UpdateFileAccess(
		"localhost",
		in.SecurityConfig.FileProtection.IsOn,
		in.SecurityConfig.FileProtection.FileList,
		deletedFiles,
	); err != nil {
		log.Errorf("UpdateFileAccess failed: %v", err)
		return nil, rpc.ErrNodeFileProtection
	}

	cfgs.SecurityConfig.FileProtection = in.SecurityConfig.FileProtection
	if err := litedb.UpdateNodeConfig(cfgs); err != nil {
		log.Errorf("UpdateContainerConfig err=%v", err)
		return nil, rpc.ErrDatabaseFail
	}

	return &pb.UpdateFileProtectReply{}, nil
}

func (s *NodeServer) UpdateNetworkRule(ctx context.Context, in *pb.UpdateNetworkRuleRequest) (*pb.UpdateNetworkRuleReply, error) {
	log.Debugf("UpdateNetworkRule called: node_id=%v , enabel:%v  rule:%v at agent", in.NodeId, in.SecurityConfig.NetworkRule.IsOn, in.SecurityConfig.NetworkRule.Rules)

	var mergedRules []*litedb.NetworkRule

	if in.SecurityConfig.NetworkRule != nil {

		for port := range blockedPorts {
			mergedRules = append(mergedRules, &litedb.NetworkRule{
				Protocols: []string{"tcp"},
				Addr:      "0.0.0.0/0",
				Port:      port,
			})
		}

		for _, v := range in.SecurityConfig.NetworkRule.Rules {

			if blockedPorts[v.Port] {
				return nil, rpc.ErrContainerNetworkRulePortOccupancy
			}

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
			err = s.nodeConfigDataTableCreate("", "")
			if err != nil {
				return nil, err
			}
			cfgs, err = litedb.GetNodeConfig("localhost")
			if err != nil {
				log.Errorf("GetContainerConfig err=%v", err)
				return nil, rpc.ErrDatabaseFail
			}
		}

		var isOn bool
		if in.SecurityConfig.NetworkRule.IsOn {
			isOn = true
		} else {
			isOn = false
		}

		cfgs.SecurityConfig.NetworkRule = &security.NetworkRuleList{
			IsOn:  isOn,
			Rules: in.SecurityConfig.NetworkRule.Rules,
		}

		if err := litedb.UpdateNodeConfig(cfgs); err != nil {
			log.Errorf("UpdateContainerConfig err=%v", err)
			return nil, rpc.ErrDatabaseFail
		}
	}

	return &pb.UpdateNetworkRuleReply{}, nil
}

func ResolveSymlink(path string) (string, error) {

	visited := make(map[string]bool)
	return resolveSymlinkRecursive(path, visited)
}

func resolveSymlinkRecursive(path string, visited map[string]bool) (string, error) {

	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("获取绝对路径失败: %v", err)
	}

	if visited[absPath] {
		return "", fmt.Errorf("检测到循环符号链接: %s", strings.Join(getCyclePath(visited, absPath), " → "))
	}
	visited[absPath] = true

	fileInfo, err := os.Lstat(absPath)
	if err != nil {
		return "", err
	}

	if fileInfo.Mode()&os.ModeSymlink != 0 {
		target, err := os.Readlink(absPath)
		if err != nil {
			return "", err
		}

		if !filepath.IsAbs(target) {
			target = filepath.Join(filepath.Dir(absPath), target)
		}
		target = filepath.Clean(target)

		return resolveSymlinkRecursive(target, visited)
	}

	return absPath, nil
}

func getCyclePath(visited map[string]bool, current string) []string {
	var path []string
	for k := range visited {
		path = append(path, k)
	}
	path = append(path, current)
	return path
}

func getFileMD5(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}

	defer file.Close()

	hasher := md5.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return "", err
	}

	return hex.EncodeToString(hasher.Sum(nil)), nil
}

func mergeAndDeduplicatePaths(defaultPaths, exeList []string) []string {
	pathSet := make(map[string]struct{})

	addPath := func(rawPath string) {
		target, err := ResolveSymlink(rawPath)
		if err != nil {
			log.Warnf("路径解析失败 [%s]: %v", rawPath, err)
			return
		}

		cleanPath := filepath.Clean(target)
		pathSet[cleanPath] = struct{}{}
	}

	for _, p := range defaultPaths {
		addPath(p)
	}

	for _, p := range exeList {
		addPath(p)
	}

	return mapKeysToSlice(pathSet)
}

func mapKeysToSlice(m map[string]struct{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}

	sort.Strings(keys)
	return keys
}

func (s *NodeServer) UpdateProcessWhiteList(ctx context.Context, in *pb.UpdateProcessWhiteListRequest) (*pb.UpdateProcessWhiteListReply, error) {

	if in.SecurityConfig.ProcProtection == nil {
		return nil, rpc.ErrNodeProcProtection
	}

	mergedPaths := mergeAndDeduplicatePaths(
		common.GetNodeProWhiteListPaths(),
		in.SecurityConfig.ProcProtection.ExeList,
	)

	var hashes []string
	for _, rawPath := range mergedPaths {

		target, err := ResolveSymlink(rawPath)
		if err != nil {
			log.Errorf("[%s] 符号链接解析失败: %v", rawPath, err)
			continue
		}

		fileHash, err := getFileMD5(target)
		if err != nil {
			log.Warnf("获取文件哈希失败 path=%v err=%v", target, err)
			continue
		}

		log.Debugf("添加白名单条目 path=%s hash=%s", target, fileHash)
		hashes = append(hashes, fileHash)
	}

	if len(hashes) == 0 {
		log.Warn("有效白名单条目为空")
		return nil, rpc.ErrNodeProcProtection
	}

	if err := kysec.UpdateWhiteList(
		"localhost",
		in.SecurityConfig.ProcProtection.IsOn,
		hashes,
		[]string{},
	); err != nil {
		log.Errorf("更新内核白名单失败: %v", err)
		return nil, rpc.ErrNodeProcProtection
	}

	cfgs, err := litedb.GetNodeConfig("localhost")
	if err != nil {
		if err := s.nodeConfigDataTableCreate("", ""); err != nil {
			return nil, err
		}
		if cfgs, err = litedb.GetNodeConfig("localhost"); err != nil {
			log.Errorf("获取节点配置失败: %v", err)
			return nil, rpc.ErrDatabaseFail
		}
	}

	cfgs.SecurityConfig.ProcProtection = in.SecurityConfig.ProcProtection
	cfgs.SecurityConfig.ProcProtection.ExeList = mergedPaths

	if err := litedb.UpdateNodeConfig(cfgs); err != nil {
		log.Errorf("更新节点配置失败: %v", err)
		return nil, rpc.ErrDatabaseFail
	}

	log.Info("进程白名单更新成功")
	return &pb.UpdateProcessWhiteListReply{}, nil
}

func (s *NodeServer) UpdateNetworkProcessWhiteList(ctx context.Context, in *pb.UpdateNetworkProcessWhiteListRequest) (*pb.UpdateNetworkProcessWhiteListReply, error) {
	log.Debugf("UpdateNetworkProcessWhiteList called: node_id=%v  at agent ", in.NodeId)

	if in.SecurityConfig.NprocProtection == nil {
		return nil, rpc.ErrNodeNetworkProcProtection
	}

	cfgs, err := litedb.GetNodeConfig("localhost")
	if err != nil {
		err = s.nodeConfigDataTableCreate("", "")
		if err != nil {
			return nil, err
		}
		cfgs, err = litedb.GetNodeConfig("localhost")
		if err != nil {
			log.Errorf("GetContainerConfig err=%v", err)
			return nil, rpc.ErrDatabaseFail
		}
	}

	nodeInfo, err := litedb.GetNodeInfo(1)
	if err != nil {
		return nil, rpc.ErrDatabaseFail
	}

	uuid := nodeInfo.UUID

	defaultNetProList := common.GetNetworkProcWhitelist()

	// 合并用户列表和默认列表（保留顺序：用户列表在前，默认列表在后）
	mergedPaths := make([]string, 0, len(in.SecurityConfig.NprocProtection.ExeList)+len(defaultNetProList))
	mergedPaths = append(mergedPaths, in.SecurityConfig.NprocProtection.ExeList...)
	mergedPaths = append(mergedPaths, defaultNetProList...)

	// 去重逻辑（保留首次出现的元素）
	seen := make(map[string]struct{})
	uniquePaths := make([]string, 0, len(mergedPaths))
	for _, path := range mergedPaths {
		if _, exists := seen[path]; !exists {
			seen[path] = struct{}{}
			uniquePaths = append(uniquePaths, path)
		}
	}
	mergedPaths = uniquePaths

	log.Debugf("UpdateNetworkProcessWhiteList mergedPaths = %v ", mergedPaths)

	var inRule = &litedb.ProcProtection{
		IsOn:    in.SecurityConfig.NprocProtection.IsOn,
		ExeList: mergedPaths,
	}

	if err := opensnitch.SaveNodeRule(inRule, "localhost", uuid); err != nil {
		log.Warnf("SaveRule id=%v err=%v", "localhost", err)
		return nil, rpc.ErrNodeNetworkProcProtection
	}

	cfgs.SecurityConfig.NprocProtection = in.SecurityConfig.NprocProtection

	if err := litedb.UpdateNodeConfig(cfgs); err != nil {
		log.Errorf("UpdateContainerConfig err=%v", err)
		return nil, rpc.ErrDatabaseFail
	}

	return &pb.UpdateNetworkProcessWhiteListReply{}, nil
}

func (s *NodeServer) GetNetworkProcessWhiteList(ctx context.Context, in *pb.GetNetworkProcessWhiteListRequest) (*pb.GetNetworkProcessWhiteListReply, error) {
	log.Debugf("GetNetworkProcessWhiteList called: node_id=%v  at agetn ", in.NodeId)

	cfgs, err := litedb.GetNodeConfig("localhost")
	if err != nil {
		err = s.nodeConfigDataTableCreate("", "")
		if err != nil {
			return nil, err
		}
		cfgs, err = litedb.GetNodeConfig("localhost")
		if err != nil {
			log.Errorf("GetContainerConfig err=%v", err)
			return nil, rpc.ErrDatabaseFail
		}
	}

	var reply pb.GetNetworkProcessWhiteListReply

	if cfgs != nil && cfgs.SecurityConfig != nil && cfgs.SecurityConfig.NprocProtection != nil {

		reply.NprocProtection = &security.ProcProtection{
			IsOn:    cfgs.SecurityConfig.NprocProtection.IsOn,
			ExeList: cfgs.SecurityConfig.NprocProtection.ExeList,
		}
	} else {
		reply.NprocProtection = &security.ProcProtection{
			IsOn:    false,
			ExeList: []string{},
		}
		log.Warnf("GetNetworkProcessWhiteList data not found for node_id=%v", in.NodeId)
	}

	log.Debugf("GetNetworkProcessWhiteList called: node_id=%v  at agetn reply:%v", in.NodeId, reply)

	return &reply, nil
}

func (s *NodeServer) GetNodeDefaultNetProWhiteList(ctx context.Context, in *pb.GetNodeDefaultNetProWhiteListRequest) (*pb.GetNodeDefaultNetProWhiteListReply, error) {
	log.Debugf("GetNodeDefaultNetProWhiteList called: node_id=%v  at agent ", in.NodeId)

	rawPaths := common.GetNetworkProcWhitelist()
	reply := &pb.GetNodeDefaultNetProWhiteListReply{
		ProcessNames: make([]string, 0, len(rawPaths)),
	}

	for _, path := range rawPaths {
		reply.ProcessNames = append(reply.ProcessNames, path)
	}

	log.Debugf("GetNodeDefaultNetProWhiteList called: node_id=%v  at agent reply:%v", in.NodeId, reply)

	return reply, nil
}

func (s *NodeServer) GetNodeProcessWhiteList(ctx context.Context, in *pb.GetProcessWhiteListRequest) (*pb.GetProcessWhiteListReply, error) {
	log.Debugf("UpdateFileProtect called: node_id=%v  at agent ", in.NodeId)

	cfgs, err := litedb.GetNodeConfig("localhost")
	if err != nil {
		err = s.nodeConfigDataTableCreate("", "")
		if err != nil {
			return nil, err
		}
		cfgs, err = litedb.GetNodeConfig("localhost")
		if err != nil {
			log.Errorf("GetContainerConfig err=%v", err)
			return nil, rpc.ErrDatabaseFail
		}
	}

	var reply pb.GetProcessWhiteListReply

	if cfgs != nil && cfgs.SecurityConfig != nil && cfgs.SecurityConfig.ProcProtection != nil {

		reply.ProcProtection = &security.ProcProtection{
			IsOn:    cfgs.SecurityConfig.ProcProtection.IsOn,
			ExeList: cfgs.SecurityConfig.ProcProtection.ExeList,
		}
	} else {
		reply.ProcProtection = &security.ProcProtection{
			IsOn:    false,
			ExeList: []string{},
		}
		log.Warnf("FileProtection data not found for node_id=%v", in.NodeId)
	}

	log.Debugf("GetFileProtect called: node_id=%v  at agent reply:%v", in.NodeId, reply.ProcProtection)

	return &reply, nil
}

func (s *NodeServer) GetNodeDefaultProcessWhiteList(ctx context.Context, in *pb.GetNodeDefaultProcessWhiteListRequest) (*pb.GetNodeDefaultProcessWhiteListReply, error) {
	log.Debugf("UpdateFileProtect called: node_id=%v  at agent ", in.NodeId)

	rawPaths := common.GetNodeProWhiteListPaths()
	reply := &pb.GetNodeDefaultProcessWhiteListReply{
		ProcessNames: make([]string, 0, len(rawPaths)),
	}

	for _, path := range rawPaths {
		reply.ProcessNames = append(reply.ProcessNames, path)
	}

	return reply, nil

}

func (s *NodeServer) GetFileProtect(ctx context.Context, in *pb.GetFileProtectRequest) (*pb.GetFileProtectReply, error) {
	log.Debugf("GetFileProtect called: node_id=%v  at agent ", in.NodeId)

	cfgs, err := litedb.GetNodeConfig("localhost")
	if err != nil {
		err = s.nodeConfigDataTableCreate("", "")
		if err != nil {
			return nil, err
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
		err = s.nodeConfigDataTableCreate("", "")
		if err != nil {
			return nil, err
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

func (s *NodeServer) GetNodeDefaultPort(ctx context.Context, in *pb.GetNodeDefaultPortRequest) (*pb.GetNodeDefaultPortReply, error) {

	var portStatusList []*pb.PortStatus
	for port, blocked := range blockedPorts {
		portStatusList = append(portStatusList, &pb.PortStatus{
			Port:    port,
			Blocked: blocked,
		})
	}

	return &pb.GetNodeDefaultPortReply{
		BlockedPorts: portStatusList,
	}, nil
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

func (s *NodeServer) nodeConfigDataTableCreate(uuid string, nodeName string) error {

	if uuid == "" {
		nodeInfo, err := litedb.GetNodeInfo(1)

		if err != nil {
			return err
		}

		uuid = nodeInfo.UUID
	}

	cfgs := litedb.NodeConfig{
		ID:             1, // 强制设置id=1，表只能有一条记录
		UUID:           uuid,
		NodeID:         "localhost",
		NodeName:       nodeName,
		SecurityConfig: &pb.SecurityConfig{},
	}

	if err := litedb.CreateNodeConfig(&cfgs); err != nil {
		log.Infof("CreateNodeConfig err=%v", err)
		return rpc.ErrDatabaseFail
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

	err = s.nodeConfigDataTableCreate(in.Uuid, in.Name)
	if err != nil {
		return nil, err
	}
	return &pb.CreateReply{}, nil
}

func (s *NodeServer) Remove(ctx context.Context, in *pb.RemoveRequest) (*pb.RemoveReply, error) {
	err := litedb.RemoveNodeInfo(1, in.Uuid) // 强制设置id=1，表只能有一条记录

	res := litedb.RemoveNodeConfigByID(1) // 强制设置id=1，表只能有一条记录

	if err := opensnitch.RemoveRule("localhost"); err != nil {
		log.Warnf("RemoveRule container=%v err=%v", "localhost", err)
	}

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
