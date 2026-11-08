package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/docker/docker/pkg/jsonmessage"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"sync"
	"syscall"
	"time"
    "strconv"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/strslice"
	"github.com/docker/docker/client"
	"github.com/google/uuid"
	"github.com/shirou/gopsutil/mem"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/metadata"

	"scmc/common"
	"scmc/model/authz"
	"scmc/model/db"
	"scmc/model/docker"
	"scmc/model/iptables"
	"scmc/model/kysec"
	"scmc/model/litedb"
	"scmc/model/opensnitch"
	"scmc/rpc"
	pb "scmc/rpc/pb/container"
	secpb "scmc/rpc/pb/security"

	"scmc/rpc/pb/logging"
	"scmc/server/args"
)

const (
	// defaultCPUShares   = 1024
	maxCPUShares       = 3072
	maxBlkioWeight     = 1000
	defaultBlkioWeight = 0
	bufSize            = 4 * 1024 * 1024
	dockerTimeout      = 10 * time.Minute
	hostsPath          = "/var/lib/docker/containers"
)

//--cpushares不设置的话，docker默认是1024
//优先级为1,设置为默认值的1.25倍
//优先级为2,设置为默认值的1.5倍
//优先级为3,设置为默认值的2倍
//优先级为4,设置为默认值的2.5倍
//优先级为5,设置为默认值的3倍
var cpuShareMap = map[int64]int64{
	1: 1280,
	2: 1536,
	3: 2048,
	4: 2560,
	5: maxCPUShares,
}

var blkioWeightMap = map[int64]uint16{
	1: 600,
	2: 700,
	3: 800,
	4: 900,
	5: maxBlkioWeight,
}

//磁盘优先级=>blkioWeigh
func toBlkioWeight(diskPrio int64) uint16 {
	if diskPrio <= 0 {
		return defaultBlkioWeight
	}
	if v, ok := blkioWeightMap[diskPrio]; ok {
		return uint16(v)
	}
	return defaultBlkioWeight
}

//blkioWeight => 磁盘优先级
func fromBlkioWeight(blkioWeight uint16) int64 {

	if blkioWeight <= defaultBlkioWeight {
		return 0
	}
	for k, v := range blkioWeightMap {
		if v == blkioWeight {
			return k
		}
	}
	return 0
}

//const defaultCPUShares = 1024
//const updateContainerFlagFile = "/var/lib/ks-scmc/networks/iptables/updateIptables.flag"
var (
	defaultCPUShares int64 = 1024
	updateContainerFlagFile string = filepath.Join(common.Config.Network.IPtablesPath, "updateIptables.flag")
)

func toCPUShares(cpuPrio int64) int64 {
	//小于等于0,或大于5都返回defaultCPUShares
	if cpuPrio <= 0 {
		return defaultCPUShares
	}
	if v, ok := cpuShareMap[cpuPrio]; ok {
		return v
	}
	return defaultCPUShares
}

func fromCPUShares(cpuShares int64) int64 {

	if cpuShares <= defaultCPUShares {
		return 0
	}
	for k, v := range cpuShareMap {
		if v == cpuShares {
			return k
		}
	}
	return 0
}

func dockerResourceConfig(in *pb.ResourceLimit) container.Resources {
	var r container.Resources

	if in != nil {
		r.NanoCPUs = int64(in.CpuLimit * 1e9)
		if r.NanoCPUs == 0 {
			r.NanoCPUs = int64(numCPU()) * 1e9
		}
		r.CPUShares = toCPUShares(in.CpuPrio)
		r.BlkioWeight = toBlkioWeight(in.DiskPrio)
		r.Memory = int64(in.MemoryLimit * megaBytes)
		r.MemoryReservation = int64(in.MemorySoftLimit * megaBytes)
		r.MemorySwap = -1 // prevent contaienr start error
	}

	return r
}

func ensureImage(imageRef string) error {
	_, err := docker.IsImageExist(imageRef)
	if err != nil {
		log.Infof("docker.IsImageExist err=%v", err)
		return rpc.ErrInternal
	}

	return nil
}

type ContainerServer struct {
	pb.UnimplementedContainerServer
}

// 确认是否为 IPC 容器, 如果为 IPC 容器就不展示出来
func (s *ContainerServer) isIpcContainer(id string) bool {
	_, err := litedb.GetIpcConfigByID(id)
	if err != nil {
		return false
	}
	return true
}

func (s *ContainerServer) List(ctx context.Context, in *pb.ListRequest) (*pb.ListReply, error) {
	cli, err := docker.Client()
	if err != nil {
		return nil, rpc.ErrInternal
	}

	stats, err := getContainerStats()
	if err != nil {
		log.Warnf("getContainerStats: %v", err)
	}

	opts := types.ContainerListOptions{All: in.GetListAll(), Size: true}
	containers, err := cli.ContainerList(context.Background(), opts)
	if err != nil {
		log.Warnf("ContainerList: %v", err)
		return nil, rpc.ErrInternal
	}

	var reply pb.ListReply
	for _, c := range containers {
		if s.isIpcContainer(c.ID) { // 过滤掉 IPC 容器
			continue
		}

		info := pb.ContainerInfo{
			Id:           c.ID,
			Image:        c.Image,
			ImageId:      c.ImageID,
			Command:      c.Command,
			State:        c.State,
			Created:      c.Created,
			Started:      containerStartedTime(ctx, cli, &c),
			Name:         containerName(c.Names),
			ResourceStat: containerResourceStat(stats, &c),
		}

		reply.Containers = append(reply.Containers, &pb.NodeContainer{Info: &info})
	}

	return &reply, nil
}


func (s *ContainerServer) getHostIps(nodeid int64) ([]string, error) {
	var result []string

	//vip, err := extractVIPFromConfig("/etc/keepalived/keepalived.conf")
	vip, err := getVIPAddr()
	if err != nil {
		log.Warnf("get platform vip failed, err: %v", err)
	} else {
		result = append(result, vip)
	}

	n, err := db.QueryNodeByID(nodeid)
	if err != nil {
		log.Warnf("get platform nodeip failed, err: %v", err)
		return result, err
	} else {
		result = append(result, n.Address)
	}

	return result, nil
}

// 获取所有运行中容器的 IP 地址
func (s *ContainerServer) getContainerIps(curContainerId string) ([]string, error) {
	cli, err := docker.Client()
	if err != nil {
		return nil, rpc.ErrInternal
	}

	containers, err := cli.ContainerList(context.Background(), types.ContainerListOptions{})
	if err != nil {
		return nil, fmt.Errorf("get container list: %w", err)
	}

	var result []string
	for _, container := range containers {

		//如果为当前容器，则跳过该容器
		if container.ID == curContainerId {
			continue
		}

		// 获取容器详细信息
		inspect, err := cli.ContainerInspect(context.Background(), container.ID)
		if err != nil {
			continue // 跳过无法获取详情的容器
		}

		// 提取主网络 IP
		networkSettings := inspect.NetworkSettings
		if networkSettings != nil {
			for _, network := range networkSettings.Networks {
				if network.IPAddress != "" {
					result = append(result, network.IPAddress)
					break
				}
			}
		}
	}

	return result, nil
}

func (s *ContainerServer) setSecurityConfig(nodeid int64, id, uuid, name string, pid int, fromUpdate bool, sec *pb.SecurityConfig, oldsec *pb.SecurityConfig) error {
	if sec == nil {
		return nil
	}

	if fromUpdate {
		if err := kysec.CleanAll(id, false); err != nil {
			log.Warnf("kysec.CleanAll: %v", err)
			return rpc.ErrContainerFileProtection
		}
	}

	if sec.FileProtection != nil {
		if err := kysec.UpdateFileAccess(id, sec.FileProtection.IsOn, sec.FileProtection.FileList, []string{}); err != nil {
			log.Warnf("UpdateFileAccess %v err: %v", id, err)
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

	// 支持安全分级模板
	if sec.SecurityLevelTemplate < 1 || sec.SecurityLevelTemplate > 5 {
		// 默认设置为 5 级模板
		sec.SecurityLevelTemplate = 5
	}

	// 网络进程白名单比较独立，各个级别的模板都可以设置
	if err := s.SetNprocProtection(id, uuid, sec.NprocProtection); err != nil {
		log.Warnf("SaveRule id=%v err=%v", id, err)
		return rpc.ErrContainerNprocProtection
	}

	if sec.NetworkRule == nil {
		sec.NetworkRule = oldsec.NetworkRule
	}

	hostips, err := s.getHostIps(nodeid)
	if err != nil {
		log.Warnf("get host ips failed, %v", err)
		return rpc.ErrContainerSecurityLevel
	}

	containerips, err := s.getContainerIps(id)
	if err != nil {
		log.Warnf("get container ips failed, %v", err)
		return rpc.ErrContainerSecurityLevel
	}

	// 0:白名单 1:黑名单 2:黑白名单，白名单优先级高
	listType := 0
	var downloadRules []*litedb.NetworkRule

	switch sec.SecurityLevelTemplate {
	case 5:
		// 5 级模板策略，全部可访问，配置了白名单，就按照白名单的规则走，无需处理
		downloadRules = sec.NetworkRule.Rules
	case 4:
		// 4 级模板策略，全部不可访问，除非在白名单中，等效于强制开启白名单
		sec.NetworkRule.IsOn = true
		sec.DisableCmdOperation = true
		downloadRules = sec.NetworkRule.Rules
	case 3:
		// 三级模板策略，能访问平台，平台外的通过白名单也可以访问，等效于强制开启白名单，且平台IP都在白名单内
		sec.DisableCmdOperation = true
		downloadRules = sec.NetworkRule.Rules
		fallthrough
	case 2:
		// 三级模板策略网络规则等同于二级模板，只是少了启停控制
		sec.NetworkRule.IsOn = true
		for _, ip := range hostips {
			rule := &litedb.NetworkRule{
				Protocols: []string{"tcp", "udp", "icmp"},
				Addr:      ip,
				Port:      0,
				Drop:      false,
			}
			if !containsRule(sec.NetworkRule.Rules, rule) {
				downloadRules = append(downloadRules, rule)
			}
		}
		downloadRules = append(downloadRules, sec.NetworkRule.Rules...)
	case 1:
		sec.NetworkRule.IsOn = true
		var platformrules []*litedb.NetworkRule

		if sec.NetworkAccessControll {
			for _, ip := range append(hostips, containerips...) {
				rule := &litedb.NetworkRule{
					Protocols: []string{"tcp", "udp", "icmp"},
					Addr:      ip,
					Port:      0,
					Drop:      false,
				}
				platformrules = append(platformrules, rule)
			}
			// 白名单打开，对外访问控制打开情况下，只有在白名单内的平台和容器IP可以访问
			downloadRules, err = UnionNetworkRules(platformrules, sec.NetworkRule.Rules)
			if err != nil {
				log.Warnf("UnionNetworkRules failed, %v", err)
				return rpc.ErrContainerSecurityLevel
			}

		} else {
			// 白名单打开，对外访问控制关闭情况下，平台和容器IP且不在白名单内，禁止，加黑名单，其它流量，允许
			// 换种方式实现，同时加白名单和黑名单，白名单的优先级高
			for _, ip := range append(hostips, containerips...) {
				rule := &litedb.NetworkRule{
					Protocols: []string{"tcp", "udp", "icmp"},
					Addr:      ip,
					Port:      0,
					Drop:      true,
				}
				platformrules = append(platformrules, rule)
			}

			downloadRules = append(sec.NetworkRule.Rules, platformrules...)
			listType = 2
		}
	default:
		log.Warnf("invalid parameter SecurityLevelTemplate: %v", sec.SecurityLevelTemplate)
		return rpc.ErrContainerSecurityLevel
	}

	if err := s.SetNetworkRule(id, name, pid, sec.NetworkRule.IsOn, listType, downloadRules); err != nil {
		log.Warnf("UpdateContainerIPtablesFile %v err: %v", name, err)
		return rpc.ErrContainerNetworkRule
	}

	// 启停控制比较独立，各个级别的模板设置方式都一样
	if err := authz.HandleSensitiveContainer(sec.DisableCmdOperation, fromUpdate, name, id); err != nil {
		log.Warnf("HandleSensitiveContainer err=%v", err)
		return rpc.ErrContainerCmdOperation
	}

	return nil
}

func (s *ContainerServer) SetNetworkRule(id, name string, pid int, on bool, blacklist int, rules []*litedb.NetworkRule) error {
	fileName := id + "-" + name
	if err := iptables.UpdateContainerIPtablesFile(fileName, pid, on, blacklist, rules); err != nil {
		return rpc.ErrContainerNetworkRule
	}

	return nil
}

func (s *ContainerServer) SetNprocProtection(id, uuid string, cfg *secpb.ProcProtection) error {
	if cfg == nil {
		return nil
	}

	var inRule = &litedb.ProcProtection{
		IsOn:    cfg.IsOn,
		ExeList: cfg.ExeList,
	}

	if err := opensnitch.SaveRule(inRule, id, uuid); err != nil {
		return rpc.ErrContainerNprocProtection
	}

	return nil
}

func (s *ContainerServer) create(nodeid int64, configs *pb.ContainerConfigs, netlog *db.RuntimeLog) (string, error) {
	if configs == nil || configs.Image == "" {
		return "", rpc.ErrInvalidArgument
	} else if !regexp.MustCompile(containerNamePattern).MatchString(configs.Name) {
		return "", status.Errorf(codes.InvalidArgument, "容器名参数错误")
	}

	cli, err := docker.Client()
	if err != nil {
		return "", rpc.ErrInternal
	}

	if err := ensureImage(configs.Image); err != nil {
		return "", status.Errorf(codes.Internal, "检查镜像信息失败")
	}

	config := container.Config{
		Image: configs.Image,
		Labels: map[string]string{
			"KS_SCMC_DESC": configs.Desc,
			"KS_SCMC_UUID": configs.Uuid,
		},
		Env:         createContainerEnvs(configs.Envs, configs.Uuid),
		AttachStdin: true, // AttachStdin, Tty: make sure container like OS(ubuntu, centos) will not exit after start
		Tty:         true,
	}

	if configs.Cmd != "" {
		var cmdSlice strslice.StrSlice = strings.Fields(configs.Cmd)
		config.Cmd = cmdSlice
	}

	/* 设置容器主机名称 */
	if configs.Hostname != "" {
		config.Hostname = configs.Hostname
	}

	hostConfig := container.HostConfig{
		Privileged: false, // force non-privileged
	}

	// 设置 IPC 信息
	if configs.Ipcname != "" {
		modeName := "container:ipc." + configs.Ipcname
		hostConfig.IpcMode = container.IpcMode(modeName)
	}

	for _, m := range configs.Mounts {
		hostConfig.Mounts = append(hostConfig.Mounts, mount.Mount{
			Type:     mount.Type(m.Type),
			Source:   m.Source,
			Target:   m.Target,
			ReadOnly: m.ReadOnly,
		})
	}

	// configs.EnableGraphic 默认开启，后面应用里有图形化选项
	if err := containerGraphicSetup(configs.Name, &config, &hostConfig); err != nil {
		log.Infof("containerGraphicSetup err=%v", err)
		return "", rpc.ErrInternal
	}

	if configs.RestartPolicy != nil {
		hostConfig.RestartPolicy.Name = configs.RestartPolicy.Name
		hostConfig.RestartPolicy.MaximumRetryCount = int(configs.RestartPolicy.MaxRetry)
	}

	if configs.ResouceLimit != nil {
		hostConfig.Resources = dockerResourceConfig(configs.ResouceLimit)

		if configs.ResouceLimit.DiskLimit > 0.0 {
			config.Labels["KS_SCMC_DISK_LIMIT"] = fmt.Sprintf("%f", configs.ResouceLimit.DiskLimit)
			hostConfig.StorageOpt = map[string]string{
				"size": fmt.Sprintf("%fM", configs.ResouceLimit.DiskLimit),
			}
		}
	}

	networkConfig := &network.NetworkingConfig{
		EndpointsConfig: make(map[string]*network.EndpointSettings, len(configs.Networks)),
	}
    
    var loginfo []string

	for _, v := range configs.Networks {
		if err := checkNetworkInfo(v.Interface, "", &v.IpAddress, int(v.IpPrefixLen)); err != nil {
			return "", err
		}

		networkConfig.EndpointsConfig[v.Interface] = &network.EndpointSettings{
			IPAMConfig: &network.EndpointIPAMConfig{
				IPv4Address: v.IpAddress,
			},
			IPAddress:   v.IpAddress,
			IPPrefixLen: int(v.IpPrefixLen),
			MacAddress:  v.MacAddress,
			Gateway:     v.Gateway,
		}

        if v.IpAddress != "" {
            loginfo = append(loginfo, fmt.Sprintf("%s", v.IpAddress))
        }
	}

    if netlog != nil && len(loginfo) != 0 {
        netlog.Detail = "创建网络: [" + strings.Join(loginfo, ",") + "]"
    }

	var networkConfigCreate *network.NetworkingConfig
	if len(configs.Networks) > 0 {
		ifs := configs.Networks[0].Interface
		networkConfigCreate = &network.NetworkingConfig{
			EndpointsConfig: map[string]*network.EndpointSettings{
				ifs: networkConfig.EndpointsConfig[ifs],
			},
		}
	}

	body, err := cli.ContainerCreate(context.Background(), &config, &hostConfig, networkConfigCreate, nil, configs.Name)
	if err != nil {
		log.Warnf("ContainerCreate: %v", err)
		return "", transDockerError(err)
	}

	if len(configs.Networks) > 1 {
		for i := 1; i < len(configs.Networks); i++ {
			ifs := configs.Networks[i].Interface
			network := networkConfig.EndpointsConfig[ifs]
			err = cli.NetworkConnect(context.Background(), ifs, body.ID, network)
			if err != nil {
				log.Warnf("NetworkConnect: %v", err)
			}
		}
	}

	if err := s.setSecurityConfig(nodeid, body.ID, configs.Uuid, configs.Name, 0,
		false, configs.SecurityConfig, nil); err != nil {
		log.Warnf("setSecurityConfig failed err=%v", err)
		if e := s.remove(cli, body.ID, true); e != nil {
			log.Warnf("after setSecurityConfig failed remove container=%v err=%v", body.ID, e)
			return body.ID, err
		}
		return "", err
	}

	return body.ID, nil
}

// 要从 controller 传递过来才有
func getUIDFromCtx(ctx context.Context) int64 {
    var uid int64 = 0
    md, ok := metadata.FromIncomingContext(ctx)
    if ok {
        values := md["user_id"]
        if len(values) > 0 {
            id, err := strconv.ParseInt(values[0], 10, 64)
            if err == nil {
                uid = id
            }
        }
    }
    return uid
}

func initRuntimeLog(netlog *db.RuntimeLog, ctx context.Context, NodeId int64, ContainerName string, eventType int64) {
    netlog.UserID = getUIDFromCtx(ctx)
    netlog.NodeId = NodeId
    netlog.EventType = eventType 
    netlog.EventModule = int64(logging.EVENT_MODULE_CONTAINER)
    if ContainerName != "" {
        netlog.Target = fmt.Sprintf("容器名=%v", ContainerName)
    }
}

func (s *ContainerServer) Create(ctx context.Context, in *pb.CreateRequest) (*pb.CreateReply, error) {
	if in.Configs == nil {
		return nil, rpc.ErrInvalidArgument
	}
    
    var netlog db.RuntimeLog
    initRuntimeLog(&netlog, ctx, in.NodeId, in.Configs.Name, int64(logging.EVENT_TYPE_CREATE_CONTAINER))

	//in.Configs.Ipcname = "testipc"
	uuid := uuid.New().String()
	cfgs := litedb.ContainerConfig{
		UUID:           uuid,
		ContainerName:  in.Configs.Name,
		SecurityConfig: in.Configs.SecurityConfig,
	}

	if err := litedb.CreateContainerConfig(&cfgs); err != nil {
		return nil, rpc.ErrInternal
	}

	in.Configs.Uuid = uuid
	id, err := s.create(in.NodeId, in.Configs, &netlog)

	// 通过 ipcname 获取 ipc容器的容器 id
	ipcID := ""
	if in.Configs.Ipcname != "" {
		ipcInfo, err := litedb.GetIpcConfigByName(in.Configs.Ipcname)
		if err != nil {
			log.Warnf("GetIpcConfigByName name: %v, err: %v", in.Configs.Ipcname, err)
		} else {
			ipcID = ipcInfo.ContainerID
		}
	}

	if err != nil {
		litedb.RemoveContainerConfigByID(cfgs.ID)
	} else {
		cfgs.ContainerID = id
		litedb.UpdateContainerConfig(&cfgs)

		// 容器与IPC 容器映射表
		ipcMapCfg := litedb.IpcmapConfig{
			ContainerName: in.Configs.Name,    //容器名称
			ContainerID:   id,                 // 容器ID
			IpcName:       in.Configs.Ipcname, // IPC 容器名称，去掉头部的 ipc. 前缀
			IpcID:         ipcID,              // IPC 容器对应的 容器ID
		}
		litedb.CreateIpcmapConfig(&ipcMapCfg)

        // 容器磁盘限制相关数据
        var w, r int64 = int64(0), int64(0)
        if in.Configs.ResouceLimit != nil {
            w = in.Configs.ResouceLimit.DiskWbw
            r = in.Configs.ResouceLimit.DiskRbw
        }
        disklimitCfg := litedb.ContainerLimit {
            ContainerID: id,
            ContainerName: in.Configs.Name,
            DiskRbw: r,
            DiskWbw: w,
        }
        litedb.CreateContainerLimit(&disklimitCfg)
	}

    if err == nil && netlog.Detail != "" {
        runtimeLogChan <- &netlog
    }

	return &pb.CreateReply{ContainerId: id}, err
}

func (s *ContainerServer) Start(ctx context.Context, in *pb.StartRequest) (*pb.StartReply, error) {
	if (len(in.Ids)) <= 0 || len(in.Ids[0].ContainerIds) <= 0 {
		return nil, rpc.ErrInvalidArgument
	}

	cli, err := docker.Client()
	if err != nil {
		return nil, rpc.ErrInternal
	}

	var (
		reply pb.StartReply
		opts  types.ContainerStartOptions
	)

	for _, id := range in.Ids[0].ContainerIds {
		if err := cli.ContainerStart(context.Background(), id, opts); err != nil {
			log.Warnf("ContainerStart: id=%v %v", id, err)
			var faileReason string
			if strings.HasSuffix(err.Error(), "Address already in use") {
				faileReason = "IP已被占用"
			} else if s, _ := status.FromError(err); s != nil {
				faileReason = s.Message()
			}
			reply.FailInfos = append(reply.FailInfos, &pb.ContainerFailInfo{
				ContainerId: id,
				FailReason:  faileReason,
			})
			continue
		}

		reply.OkIds = append(reply.OkIds, id)
	}

	if len(reply.OkIds) > 0 {
		// 容器启动成功，要通知所有容器更新 iptables 规则
		file, err := os.OpenFile(updateContainerFlagFile, os.O_RDONLY|os.O_CREATE, 0644)
		if err != nil {
			log.Warnf("create flag failed when start contain: %v", updateContainerFlagFile)
		}
		defer file.Close()
	}

	return &reply, nil
}

func (s *ContainerServer) parallelProcessing(operate string, contaienrIds []string) ([]string, []*pb.ContainerFailInfo, error) {
	cli, err := docker.Client()
	if err != nil {
		return nil, nil, rpc.ErrInternal
	}

	var (
		timeout   = time.Second
		wg        sync.WaitGroup
		lock      sync.Mutex
		okIDs     []string
		failInfos []*pb.ContainerFailInfo
	)

	for _, id := range contaienrIds {
		wg.Add(1)
		go func(containerID string) {
			defer wg.Done()
			var err error
			if operate == "ContainerStop" {
				err = cli.ContainerStop(context.Background(), containerID, &timeout)
			} else if operate == "ContainerRestart" {
				err = cli.ContainerRestart(context.Background(), containerID, &timeout)
			} else {
				log.Warnf("%v: id=%v ", operate, containerID)
				return
			}

			if err != nil {
				log.Warnf("%v: id=%v %v", operate, containerID, err)
				var faileReason string
				if s, _ := status.FromError(err); s != nil {
					faileReason = s.Message()
				}
				lock.Lock()
				failInfos = append(failInfos, &pb.ContainerFailInfo{
					ContainerId: containerID,
					FailReason:  faileReason,
				})
				lock.Unlock()
				return
			}
			okIDs = append(okIDs, containerID)
		}(id)
	}

	wg.Wait()
	return okIDs, failInfos, nil
}

func (s *ContainerServer) Stop(ctx context.Context, in *pb.StopRequest) (*pb.StopReply, error) {
	if (len(in.Ids)) <= 0 || len(in.Ids[0].ContainerIds) <= 0 {
		return nil, rpc.ErrInvalidArgument
	}

	okIDs, failInfos, err := s.parallelProcessing("ContainerStop", in.Ids[0].ContainerIds)
	if err != nil {
		return nil, err
	}

	if len(okIDs) > 0 {
		// 容器停止成功，要通知所有容器更新 iptables 规则
		file, err := os.OpenFile(updateContainerFlagFile, os.O_RDONLY|os.O_CREATE, 0644)
		if err != nil {
			log.Warnf("create flag failed when stop contain: %v", updateContainerFlagFile)
		}
		defer file.Close()
	}

	return &pb.StopReply{
		OkIds:     okIDs,
		FailInfos: failInfos,
	}, nil
}

func (s *ContainerServer) Kill(ctx context.Context, in *pb.KillRequest) (*pb.KillReply, error) {
	if (len(in.Ids)) <= 0 || len(in.Ids[0].ContainerIds) <= 0 {
		return nil, rpc.ErrInvalidArgument
	}

	cli, err := docker.Client()
	if err != nil {
		return nil, rpc.ErrInternal
	}

	var reply pb.KillReply
	for _, id := range in.Ids[0].ContainerIds {
		if err := cli.ContainerKill(context.Background(), id, ""); err != nil {
			log.Warnf("ContainerKill: id=%v %v", id, err)
			var faileReason string
			if s, _ := status.FromError(err); s != nil {
				faileReason = s.Message()
			}
			reply.FailInfos = append(reply.FailInfos, &pb.ContainerFailInfo{
				ContainerId: id,
				FailReason:  faileReason,
			})
			continue
		}
		reply.OkIds = append(reply.OkIds, id)
	}

	return &reply, nil
}

func (s *ContainerServer) Restart(ctx context.Context, in *pb.RestartRequest) (*pb.RestartReply, error) {
	if (len(in.Ids)) <= 0 || len(in.Ids[0].ContainerIds) <= 0 {
		return nil, rpc.ErrInvalidArgument
	}

	okIDs, failInfos, err := s.parallelProcessing("ContainerRestart", in.Ids[0].ContainerIds)
	if err != nil {
		return nil, err
	}

	return &pb.RestartReply{
		OkIds:     okIDs,
		FailInfos: failInfos,
	}, nil
}


func (s *ContainerServer) disklimit(containerName string, in *pb.UpdateRequest) {
    // 磁盘限速，如果没有记录，就创建一条新记录
    disklimitCfg, err := litedb.GetContainerLimitByCID(in.ContainerId);
    if err != nil {
        var w, r int64 = 0, 0
        if in.ResourceLimit != nil {
            w = in.ResourceLimit.DiskWbw
            r = in.ResourceLimit.DiskRbw
        }
        disklimitCfg = &litedb.ContainerLimit {
            ContainerID: in.ContainerId,
            ContainerName:containerName,
            DiskRbw: r,
            DiskWbw: w,
        }
        litedb.CreateContainerLimit(disklimitCfg)
        rStr := fmt.Sprintf("%d", r)
        wStr := fmt.Sprintf("%d", w)
        docker.SetContainerDiskLimit("disk_rbw", in.ContainerId, rStr)
        docker.SetContainerDiskLimit("disk_wbw", in.ContainerId, wStr)
    } else {
        chg := false
        if in.ResourceLimit != nil {
            if disklimitCfg.DiskRbw != in.ResourceLimit.DiskRbw {
                disklimitCfg.DiskRbw = in.ResourceLimit.DiskRbw
                r := in.ResourceLimit.DiskRbw
                rStr := fmt.Sprintf("%d", r)
                docker.SetContainerDiskLimit("disk_rbw", in.ContainerId, rStr)
                chg = true
            }
            if disklimitCfg.DiskWbw != in.ResourceLimit.DiskWbw {
                disklimitCfg.DiskWbw = in.ResourceLimit.DiskWbw
                w := in.ResourceLimit.DiskWbw
                wStr := fmt.Sprintf("%d", w)
                docker.SetContainerDiskLimit("disk_wbw", in.ContainerId, wStr)
                chg = true
            }
            if chg {
                litedb.UpdateContainerLimit(disklimitCfg) // 更新数据库
            }
        }
    }
}

func (s *ContainerServer) Update(ctx context.Context, in *pb.UpdateRequest) (*pb.UpdateReply, error) {
	var reply pb.UpdateReply
	if in.ContainerId == "" || (in.ResourceLimit == nil && in.RestartPolicy == nil && in.Networks == nil && in.SecurityConfig == nil) {
		return nil, rpc.ErrInvalidArgument
	}

	cfgs, err := litedb.GetContainerConfig(in.ContainerId)
	if err != nil {
		log.Infof("GetContainerConfig err=%v", err)
		return nil, rpc.ErrDatabaseFail
	}
    
    var netlog db.RuntimeLog
    initRuntimeLog(&netlog, ctx, in.NodeId, cfgs.ContainerName, int64(logging.EVENT_TYPE_UPDATE_CONTAINER))

	for _, v := range in.Networks {
		if err := checkNetworkInfo(v.Interface, in.ContainerId, &v.IpAddress, int(v.IpPrefixLen)); err != nil {
			return nil, err
		}
	}

	inspectConfigs, pid, err := s.inspect(in.ContainerId, false)
	if err != nil {
		log.Warnf("inspect container=%v err=%v", in.ContainerId, err)
		return nil, rpc.ErrInternal
	}

	cli, err := docker.Client()
	if err != nil {
		return nil, rpc.ErrInternal
	}

	config := container.UpdateConfig{
		Resources: dockerResourceConfig(in.ResourceLimit),
	}

	if in.RestartPolicy != nil {
		config.RestartPolicy = container.RestartPolicy{
			Name:              in.RestartPolicy.Name,
			MaximumRetryCount: int(in.RestartPolicy.MaxRetry),
		}
	}

	body, err := cli.ContainerUpdate(context.Background(), in.ContainerId, config)
	if err != nil {
		log.Warnf("ContainerUpdate: %v", err)
		return nil, transDockerError(err)
	}

	if len(body.Warnings) > 0 {
		log.Infof("ContainerUpdate result warnings: %v", body.Warnings)
	}

	updateNetwork(ctx, cli, in.ContainerId, in.Networks, &netlog)

	if err := s.setSecurityConfig(in.NodeId, in.ContainerId, inspectConfigs.Uuid,
		inspectConfigs.Name, pid, true, in.SecurityConfig, cfgs.SecurityConfig); err != nil {
		log.Warnf("Update setSecurityConfig err=%v", err)
		return nil, err
	}

	cfgs.SecurityConfig = in.SecurityConfig
	if err := litedb.UpdateContainerConfig(cfgs); err != nil {
		log.Infof("UpdateContainerConfig err=%v", err)
		return nil, rpc.ErrDatabaseFail
	}

	var change int
	var envs []string

	if !reflect.DeepEqual(inspectConfigs.Envs, in.Envs) {
		change |= docker.EnvBit
		for k, v := range in.Envs {
			envs = append(envs, k+"="+v)
			//磁盘读写速度、磁盘读写频率
			if k == "disk_wio" || k == "disk_rio" || k == "disk_rbw" || k == "disk_wbw" {
				if err := docker.SetContainerDiskLimit(k, in.ContainerId, v); err != nil {
					return nil, err
				}
			}
		}
		log.Debugf("%v envs change, %+v, %+v", inspectConfigs.ContainerId, in.Envs, envs)
	}

	mountMap, equal := docker.CompareMounts(inspectConfigs.Mounts, in.Mounts)
	if !equal {
		change |= docker.MountBit
		log.Debugf("%v mounts change", inspectConfigs.ContainerId)
	}

	// 获取容器与 ipcBase 的映射关系
	ipcmapCfg, err := litedb.GetIpcmapConfigByCID(in.ContainerId)
	if err != nil {
		// 在创建的时候写入的数据，这里不应该失败
		log.Warnf("Update GetIpcmapConfigByCID id: %v, err: %v", in.ContainerId, err)
		// 创建一条新记录
		ipcmapCfg = &litedb.IpcmapConfig {
			ContainerName: cfgs.ContainerName, //容器名称
			ContainerID : in.ContainerId, // 容器ID
			IpcName: "", // IPC 容器名称，去掉头部的 ipc. 前缀
			IpcID: "",  // IPC 容器对应的容器ID 
		}
		litedb.CreateIpcmapConfig(ipcmapCfg)
	}

	{
		//如果当前传递过来的和以前的不一样
		if strings.TrimSpace(ipcmapCfg.IpcName) != strings.TrimSpace(in.Ipcname) {
			// 获取 ipcBase 容器的容器ID
			if strings.TrimSpace(in.Ipcname) != "" {
				ipcInfo, err := litedb.GetIpcConfigByName(strings.TrimSpace(in.Ipcname))
				if err != nil {
					log.Warnf("Update litedb.GetIpcConfigByName %v, err: %v", in.Ipcname, err)
				} else {
					// 更新数据库
					ipcmapCfg.IpcName = strings.TrimSpace(in.Ipcname)
					ipcmapCfg.IpcID = ipcInfo.ContainerID
					litedb.UpdateIpcmanConfig(ipcmapCfg)
					change |= docker.IpcBit
				}
			} else {
				ipcmapCfg.IpcName = ""
				ipcmapCfg.IpcID = ""
				litedb.UpdateIpcmanConfig(ipcmapCfg)
				change |= docker.IpcBit
			}
		}
	}

	log.Warnf("change: %v docker.NoneBit: %v", change, docker.NoneBit)
	if change != docker.NoneBit {
		if err = docker.ModifyContainerConfig(inspectConfigs.ContainerId, change, envs, mountMap); err != nil {
			log.Infof("UpdateContainerConfig err=%v", err)
			return nil, rpc.ErrInternal
		}
		log.Infof("status:%v", inspectConfigs.Status)
		var timeout = time.Second
		if inspectConfigs.Status == "running" {
			if err = cli.ContainerRestart(context.Background(), in.ContainerId, &timeout); err != nil {
				log.Warnf("restart id=%v %v", in.ContainerId, err)
			}
		} else {
			if err := cli.ContainerStart(context.Background(), in.ContainerId, types.ContainerStartOptions{}); err != nil {
				log.Warnf("start id=%v %v", in.ContainerId, err)
			}
			if err := cli.ContainerStop(context.Background(), in.ContainerId, &timeout); err != nil {
				log.Warnf("stop id=%v %v", in.ContainerId, err)
			}
		}
		log.Infof("modify %v end, change:%v", in.ContainerId, change)
	}

    s.disklimit(cfgs.ContainerName, in)
    
    if netlog.Detail != "" {
        runtimeLogChan <- &netlog
    }

	return &reply, nil
}

func (s *ContainerServer) UpdateContainerVersion(ctx context.Context, in *pb.UpdateContainerVersionRequest) (*pb.UpdateContainerVersionReply, error) {
	cli, err := docker.Client()
	if err != nil {
		return nil, rpc.ErrInternal
	}

	containerId := in.ContainerId

	cfgs, err := litedb.GetContainerConfig(containerId)
	if err != nil {
		log.Warnf("db get container configs id=%v err=%v", containerId, err)
		return nil, rpc.ErrInternal
	}

	// inspect container
	configs, _, err := s.inspect(containerId, true)
	if err != nil {
		log.Warnf("inspect container=%v err=%v", containerId, err)
		return nil, rpc.ErrInternal
	}

	// remove old container
	if err := s.remove(cli, containerId, true); err != nil {
		log.Warnf("remove container=%v err=%v", containerId, err)
		return nil, rpc.ErrInternal
	}

	// create new contaienr
	configs.Image = in.ImgName + ":" + in.ImgVersion

	configs.SecurityConfig = cfgs.SecurityConfig
	id, err := s.create(in.NodeId, configs, nil) //netlog
	if err != nil {
		log.Warnf("create container err=%v", err)
		return nil, rpc.ErrInternal
	}

	cfgs.ContainerID = id
	litedb.UpdateContainerConfig(cfgs)

	return &pb.UpdateContainerVersionReply{}, nil
}

func updateNetwork(ctx context.Context, cli *client.Client, containerID string, networks []*pb.NetworkConfig, netlog *db.RuntimeLog) error {
	if len(networks) == 0 {
		return nil
	}

	info, err := cli.ContainerInspect(context.Background(), containerID)
	if err != nil {
		log.Warnf("ContainerInspect: %v", err)
		return transDockerError(err)
	}

    var addNetworkLog []string // 添加网络日志
    var delNetworkLog []string // 删除网络日志
    var modNetworkLog []string // 修改网络日志

	if info.NetworkSettings == nil {
		for _, v := range networks {
			dockerNetworkConnect(ctx, cli, containerID, v)
            if v.IpAddress != "" {
                addNetworkLog = append(addNetworkLog, v.IpAddress)
            }
		}
		return nil
	}

	m := make(map[string]bool, len(networks))
	for _, v := range networks {
		m[v.Interface] = true
	}
	for k := range info.NetworkSettings.Networks {
		if _, ok := m[k]; ok {
			continue
		}
		dockerNetworkDisconnect(ctx, cli, containerID, k) // 被删除的连接
        ipstr := info.NetworkSettings.Networks[k].IPAddress
        if ipstr != "" {
            delNetworkLog = append(delNetworkLog, ipstr)
        }
	}

	for _, v := range networks {
		_, ok := info.NetworkSettings.Networks[v.Interface]
		if !ok {
			dockerNetworkConnect(ctx, cli, containerID, v) // 新连接
            if v.IpAddress != "" {
                addNetworkLog = append(addNetworkLog, v.IpAddress)
            }
			continue
		}

		// 修改IP
		if v.IpAddress != info.NetworkSettings.Networks[v.Interface].IPAddress {
			dockerNetworkDisconnect(ctx, cli, containerID, v.Interface)
			dockerNetworkConnect(ctx, cli, containerID, v)
            dstStr := "dhcp"
            if v.IpAddress != "" {
                dstStr = v.IpAddress
            }
            modNetworkLog = append(modNetworkLog, fmt.Sprintf("%s->%s", info.NetworkSettings.Networks[v.Interface].IPAddress, dstStr))
		}
	}
    
    if netlog != nil && len(modNetworkLog) > 0 {
        netlog.Detail = netlog.Detail + "修改网络: ["  + strings.Join(modNetworkLog, ",") + "] "
    }
    if netlog != nil && len(delNetworkLog) > 0 {
        netlog.Detail = netlog.Detail + "删除网络: [" + strings.Join(delNetworkLog, ",") + "] "
    }
    if netlog != nil && len(addNetworkLog) > 0 {
        netlog.Detail = netlog.Detail + "添加网络: [" + strings.Join(addNetworkLog, ",") + "]"
    }

	return nil
}

func (s *ContainerServer) remove(cli *client.Client, containerID string, force bool) error {
	if cli == nil {
		c, err := docker.Client()
		if err != nil {
			return rpc.ErrInternal
		}
		cli = c
	}

	configs, _, err := s.inspect(containerID, false)
	if err != nil {
		log.Warnf("inspect container=%v err=%v", containerID, err)
		return err
	}

	if !force && configs.Status == "running" {
		return rpc.ErrRemoveContainerWhenRunning
	}

	opts := types.ContainerRemoveOptions{RemoveVolumes: true, Force: force}
	if err := cli.ContainerRemove(context.Background(), containerID, opts); err != nil {
		log.Warnf("ContainerRemove: id=%v %v", containerID, err)
		return rpc.ErrInternal
	}

	if err := kysec.CleanFileAccess(containerID); err != nil {
		log.Warnf("CleanFileAccess container=%v err=%v", containerID, err)
	}
	if err := kysec.CleanWhiteList(containerID); err != nil {
		log.Warnf("CleanWhiteList container=%v err=%v", containerID, err)
	}

	if err := opensnitch.RemoveRule(containerID); err != nil {
		log.Warnf("RemoveRule container=%v err=%v", containerID, err)
	}

	if err := removeContainerGraphicSetup(containerID); err != nil {
		log.Warnf("removeContainerGraphicSetup container=%v err=%v", containerID, err)
	}

	fileName := containerID + "-" + configs.Name
	iptables.ContainerRemveIPtables(fileName)

	if err := authz.DelSensitiveContainers(configs.Name, containerID); err != nil {
		log.Warnf("DelSensitiveContainers err=%v", err)
	}

	// 移除残余文件夹数据
	hostPath := filepath.Join(common.Config.Agent.ContainerExtraDataBasedir, configs.Name)
	log.Warnf("remove all : %v", hostPath)
	os.RemoveAll(hostPath)

	return nil
}

func (s *ContainerServer) Remove(ctx context.Context, in *pb.RemoveRequest) (*pb.RemoveReply, error) {
	if (len(in.Ids)) <= 0 || len(in.Ids[0].ContainerIds) <= 0 {
		return nil, rpc.ErrInvalidArgument
	}

	cli, err := docker.Client()
	if err != nil {
		return nil, rpc.ErrInternal
	}

	var reply pb.RemoveReply
	for _, id := range in.Ids[0].ContainerIds {
		err := s.remove(cli, id, false)
		if err != nil {
			var faileReason string
			if s, _ := status.FromError(err); s != nil {
				faileReason = s.Message()
			}
			reply.FailInfos = append(reply.FailInfos, &pb.ContainerFailInfo{
				ContainerId: id,
				FailReason:  faileReason,
			})
			reply.FailIds = append(reply.FailIds, id)
			continue
		}
		litedb.RemoveContainerConfig([]string{id})
		litedb.IpcmapConfigRemoveByCID([]string{id}) // 删除容器与ipc 容器的映射数据

		reply.OkIds = append(reply.OkIds, id)
	}

	litedb.RemoveContainerConfig(reply.OkIds)
	litedb.IpcmapConfigRemoveByCID(reply.OkIds) // 删除容器与ipc 容器的映射数据
	go s.removeContainerBackups(reply.OkIds)

	return &reply, nil
}

func (s *ContainerServer) removeContainerBackups(ids []string) error {
	backups, err := litedb.QueryContainerBackupByContainer(ids)
	if err != nil {
		log.Warnf("QueryContainerBackupByContainer err=%v", err)
		return err
	}

	for _, b := range backups {
		docker.RemoveImage(context.Background(), b.ImageRef, false)
	}

	litedb.RemoveContainerBackup(backups)
	return nil
}

func (s *ContainerServer) inspect(id string, backup bool) (*pb.ContainerConfigs, int, error) {
	cli, err := docker.Client()
	if err != nil {
		return nil, 0, rpc.ErrInternal
	}

	info, _, err := cli.ContainerInspectWithRaw(context.Background(), id, true)
	if err != nil {
		log.Warnf("ContainerInspectWithRaw: %v", err)
		return nil, 0, transDockerError(err)
	}

	configs := &pb.ContainerConfigs{
		ContainerId:  info.ID,
		Name:         strings.TrimPrefix(info.Name, "/"),
		Image:        info.Image,
		ResouceLimit: &pb.ResourceLimit{},
	}

	var pid int
	if info.State != nil {
		configs.Status = info.State.Status
		pid = info.State.Pid
	}

	if info.Config != nil {
		s.inspectConfig(cli, info.Config, configs)
	}

	for _, m := range info.Mounts {
		configs.Mounts = append(configs.Mounts, &pb.Mount{
			Type:     string(m.Type),
			Source:   m.Source,
			Target:   m.Destination,
			ReadOnly: !m.RW,
		})
	}
   
    var w, r int64 = 0, 0
    disklimitCfg, err := litedb.GetContainerLimitByCID(info.ID) 
    if err != nil {
        w = 0
        r = 0
    } else {
        w = disklimitCfg.DiskWbw
        r = disklimitCfg.DiskRbw
    }

	if info.HostConfig != nil {
		configs.RestartPolicy = &pb.RestartPolicy{
			Name:     info.HostConfig.RestartPolicy.Name,
			MaxRetry: int32(info.HostConfig.RestartPolicy.MaximumRetryCount),
		}

		configs.ResouceLimit = &pb.ResourceLimit{
			CpuLimit:        float64(info.HostConfig.Resources.NanoCPUs) / 1e9,
			CpuPrio:         fromCPUShares(info.HostConfig.Resources.CPUShares),
			DiskPrio:        fromBlkioWeight(info.HostConfig.Resources.BlkioWeight),
			MemoryLimit:     float64(info.HostConfig.Resources.Memory) / megaBytes,
			MemorySoftLimit: float64(info.HostConfig.Resources.MemoryReservation) / megaBytes,
            DiskWbw: w,
            DiskRbw: r,
		}

		if s, ok := info.HostConfig.StorageOpt["size"]; ok {
			fmt.Sscanf(s, "%fM", &configs.ResouceLimit.DiskLimit)
		}

		if info.HostConfig.IpcMode.IsContainer() {
			parts := strings.SplitN(string(info.HostConfig.IpcMode), ":", 2) // parts[1] 为 IPC 容器的容器ID
			// 容器ID转换为 ipc 名称
			cfg, err := litedb.GetIpcConfigByID(strings.TrimSpace(parts[1]))
			if err != nil {
				configs.Ipcname = ""
			} else {
				configs.Ipcname = cfg.ContainerName
			}
		}

	}

	if info.NetworkSettings != nil {
		s.inspectNetwork(info.ID, backup, info.NetworkSettings, configs)
	}

	if info.Config.Cmd != nil {
		configs.Cmd = strings.Join(info.Config.Cmd, " ")
	}

	return configs, pid, nil
}

func (s *ContainerServer) inspectConfig(cli *client.Client, config *container.Config, ret *pb.ContainerConfigs) {
	// if image tag exists, use the tag, if not use image id
	if _, _, err := cli.ImageInspectWithRaw(context.Background(), config.Image); err == nil {
		ret.Image = config.Image
	}

	if v, ok := config.Labels["KS_SCMC_DESC"]; ok {
		ret.Desc = v
	}

	if v, ok := config.Labels["KS_SCMC_UUID"]; ok {
		ret.Uuid = v
	}

	if len(config.Env) > 0 {
		ret.Envs = make(map[string]string, len(config.Env))
		for _, e := range config.Env {
			parts := strings.SplitN(e, "=", 2)
			if len(parts) >= 2 {
				ret.Envs[parts[0]] = parts[1]
			}
		}
	}

	// 返回容器主机名
	ret.Hostname = config.Hostname
}

func (s *ContainerServer) inspectNetwork(id string, backup bool, settings *types.NetworkSettings, ret *pb.ContainerConfigs) {
	for iface, m := range settings.Networks {
		if isPredefinedNetwork(iface) {
			continue // 需要忽略默认创建的网络连接
		}

		var ipAddr string
		if m.IPAMConfig != nil {
			ipAddr = m.IPAMConfig.IPv4Address // 返回用户填的IP地址
		}

		if !backup && m.IPAddress != "" {
			ipAddr = m.IPAddress // 非备份情况下，当用户没填IP地址，返回运行时IP
		}

		ret.Networks = append(ret.Networks, &pb.NetworkConfig{
			Interface:   iface,
			ContainerId: id,
			IpAddress:   ipAddr,
			IpPrefixLen: int32(m.IPPrefixLen),
			MacAddress:  m.MacAddress,
			Gateway:     m.Gateway,
		})
	}
}

func (s *ContainerServer) Inspect(ctx context.Context, in *pb.InspectRequest) (*pb.InspectReply, error) {
	data, err := litedb.GetContainerConfig(in.ContainerId)
	if err != nil {
		log.Infof("db get container configs id=%v err=%v", in.ContainerId, err)
		return nil, rpc.ErrDatabaseFail
	}

	configs, _, err := s.inspect(in.ContainerId, false)
	if err != nil {
		return nil, err
	}
	configs.SecurityConfig = data.SecurityConfig

	return &pb.InspectReply{Configs: configs}, nil
}

func (s *ContainerServer) MonitorHistory(ctx context.Context, in *pb.MonitorHistoryRequest) (*pb.MonitorHistoryReply, error) {
	now := time.Now()
	if in.StartTime >= in.EndTime || in.Interval < 1 || in.StartTime < now.Add(-time.Hour*24*10).Unix() {
		log.Info("MonitorHistory invalid time args")
		return nil, rpc.ErrInvalidArgument
	}

	containerName := "/" // query influxdb need container name, "/" for query the host

	memInfo, err := mem.VirtualMemory()
	if err != nil {
		log.Infof("VirtualMemory err=%v", err)
		return nil, err
	}

	rscLimit := pb.ResourceLimit{
		CpuLimit:    float64(numCPU()),
		MemoryLimit: float64(memInfo.Total) / megaBytes,
	}

	if in.ContainerId != "" {
		cli, err := docker.Client()
		if err != nil {
			return nil, rpc.ErrInternal
		}
		info, err := cli.ContainerInspect(context.Background(), in.ContainerId)
		if err != nil {
			log.Infof("ContainerInspect:  %v", err)
			return nil, transDockerError(err)
		}

		containerName = strings.TrimPrefix(info.Name, "/")
		if info.HostConfig != nil {
			if info.HostConfig.NanoCPUs > 0 {
				rscLimit.CpuLimit = float64(info.HostConfig.NanoCPUs) / 1e9
			}
			if info.HostConfig.Memory > 0 {
				rscLimit.MemoryLimit = float64(info.HostConfig.Memory) / megaBytes
			}
			if info.HostConfig.MemoryReservation > 0 {
				rscLimit.MemorySoftLimit = float64(info.HostConfig.MemoryReservation) / megaBytes
			}
		}
	}

	r, err := influxdbQuery(in.StartTime, in.EndTime, uint(in.Interval), containerName)
	if err != nil {
		log.Infof("query influxdb error=%v", err)
		return nil, rpc.ErrInternal
	}

	r.RscLimit = &rscLimit
	return r, nil
}

func (*ContainerServer) CreateBackup(ctx context.Context, in *pb.CreateBackupRequest) (*pb.CreateBackupReply, error) {
	if in.ContainerId == "" {
		return nil, rpc.ErrInvalidArgument
	}
	if in.BackupName == "" {
		now := time.Now()
		in.BackupName = now.Format("20060102150405") + fmt.Sprintf("%d", now.Nanosecond()/1000000)
	}

	if !args.BackupDesc.Check(in.BackupDesc) {
		return nil, status.Errorf(codes.InvalidArgument, "参数错误：备份描述无效")
	} else if !args.ImageVersion.Check(in.BackupName) {
		return nil, status.Errorf(codes.InvalidArgument, "参数错误：备份版本名无效")
	}

	cfgs, err := litedb.GetContainerConfig(in.ContainerId)
	if err != nil {
		log.Infof("GetContainerConfigs err=%v", err)
		return nil, rpc.ErrInternal
	}

	b, err := litedb.CreateContainerBackup(cfgs.UUID, in.BackupName, in.BackupDesc)
	if err != nil {
		if strings.HasPrefix(err.Error(), "UNIQUE constraint failed") {
			return nil, status.Errorf(codes.InvalidArgument, "已存在同名备份")
		}

		return nil, rpc.ErrInternal
	}

	b.ContainerID = in.ContainerId
	go addBackupJob(b)

	return &pb.CreateBackupReply{}, nil
}

func (*ContainerServer) UpdateBackup(ctx context.Context, in *pb.UpdateBackupRequest) (*pb.UpdateBackupReply, error) {
	if !args.BackupDesc.Check(in.BackupDesc) {
		return nil, status.Errorf(codes.InvalidArgument, "参数错误：备份描述无效")
	}

	b, err := litedb.QueryContainerBackupByID(in.Id)
	if err != nil {
		log.Infof("db.QueryContainerBackupByID err=%v", err)
		if err == db.ErrRecordNotFound {
			return nil, rpc.ErrNotFound
		}
		return nil, rpc.ErrInternal
	}

	b.BackupDesc = in.BackupDesc
	if err := litedb.UpdateContainerBackup(b); err != nil {
		log.Infof("db.UpdateContainerBackup err=%v", err)
		return nil, rpc.ErrInternal
	}

	return &pb.UpdateBackupReply{}, nil
}

func (*ContainerServer) RemoveBackup(ctx context.Context, in *pb.RemoveBackupRequest) (*pb.RemoveBackupReply, error) {
	b, err := litedb.QueryContainerBackupByID(in.Id)
	if err != nil {
		log.Infof("db.QueryContainerBackupByID err=%v", err)
		if err == db.ErrRecordNotFound {
			return nil, rpc.ErrNotFound
		}
		return nil, rpc.ErrInternal
	}

	cfgs, err := litedb.GetContainerConfigByUUID(b.UUID)
	if err != nil {
		log.Infof("db.GetContainerConfigsByUUID err=%v", err)
		if err == db.ErrRecordNotFound {
			return nil, rpc.ErrNotFound
		}
		return nil, rpc.ErrInternal
	}

	cli, err := docker.Client()
	if err != nil {
		return nil, rpc.ErrInternal
	}

	ci, err := cli.ContainerInspect(ctx, cfgs.ContainerID)
	if err != nil {
		log.Infof("ContainerInspect err=%v", err)
		return nil, rpc.ErrInternal
	}

	if ci.Config.Image == b.ImageRef {
		return nil, rpc.ErrRemoveContainerBackupWhenRunning
	}

	if err := litedb.RemoveContainerBackup(b); err != nil {
		log.Warnf("db remove backup record err=%v", err)
		return nil, rpc.ErrInternal
	}

	docker.RemoveImage(ctx, b.ImageRef, false)

	return &pb.RemoveBackupReply{}, nil
}

func (s *ContainerServer) ResumeBackup(ctx context.Context, in *pb.ResumeBackupRequest) (*pb.ResumeBackupReply, error) {
	b, err := litedb.QueryContainerBackupByID(in.BackupId)
	if err != nil {
		log.Infof("db.QueryContainerBackupByID err=%v", err)
		if err == db.ErrRecordNotFound {
			return nil, rpc.ErrNotFound
		}
		return nil, rpc.ErrInternal
	}

	if b.Status != int8(pb.BACKUP_STATUS_SUCCEED) && b.ContainerID != in.ContainerId {
		log.Infof("invalid backup=%+v target container=%v", b, in.ContainerId)
		return nil, rpc.ErrInvalidArgument
	}

	backups, err := litedb.QueryContainerBackupByUUID(b.UUID)
	if err != nil {
		log.Infof("QueryContainerBackupByUUID uuid=%v err=%v", b.UUID, err)
		return nil, rpc.ErrInternal
	}

	cfgs, err := litedb.GetContainerConfigByUUID(b.UUID)
	if err != nil {
		log.Warnf("db get container configs uuid=%v err=%v", b.UUID, err)
		return nil, rpc.ErrInternal
	}

	cli, err := docker.Client()
	if err != nil {
		return nil, rpc.ErrInternal
	}

	// inspect container
	configs, _, err := s.inspect(in.ContainerId, true)
	if err != nil {
		log.Warnf("inspect container=%v err=%v", in.ContainerId, err)
		return nil, rpc.ErrInternal
	}

	// remove old container
	if err := s.remove(cli, in.ContainerId, true); err != nil {
		log.Warnf("remove container=%v err=%v", in.ContainerId, err)
		return nil, rpc.ErrInternal
	}

	// create new contaienr
	configs.Image = b.ImageRef
	configs.SecurityConfig = cfgs.SecurityConfig
	id, err := s.create(in.NodeId, configs, nil) //netlog
	if err != nil {
		log.Warnf("create container err=%v", err)
		return nil, rpc.ErrInternal
	}

	cfgs.ContainerID = id
	litedb.UpdateContainerConfig(cfgs)

	for _, b := range backups {
		b.ContainerID = id
	}
	if err := litedb.UpdateContainerBackup(backups); err != nil {
		log.Infof("db.UpdateContainerBackup err=%v", err)
		return nil, rpc.ErrInternal
	}

	return &pb.ResumeBackupReply{ContainerId: id}, nil
}

func (*ContainerServer) ListBackup(ctx context.Context, in *pb.ListBackupRequest) (*pb.ListBackupReply, error) {
	cfgs, err := litedb.GetContainerConfig(in.ContainerId)
	if err != nil {
		log.Infof("GetContainerConfigs err=%v", err)
		return nil, rpc.ErrInternal

	}
	data, err := litedb.QueryContainerBackupByUUID(cfgs.UUID)
	if err != nil {
		log.Infof("db.QueryContainerBackupByUUID err=%v", err)
		return nil, rpc.ErrInternal
	}

	var reply pb.ListBackupReply
	for _, b := range data {
		reply.Data = append(reply.Data, &pb.ContainerBackup{
			Id:         b.ID,
			Uuid:       b.UUID,
			BackupName: b.BackupName,
			BackupDesc: b.BackupDesc,
			ImageRef:   b.ImageRef,
			ImageSize:  b.ImageSize,
			Status:     int64(b.Status),
			CreatedAt:  b.CreatedAt,
		})
	}

	return &reply, nil
}

func (*ContainerServer) ListBackupImage(ctx context.Context, in *pb.ListBackupImageRequest) (*pb.ListBackupImageReply, error) {
	backups, err := litedb.ListContainerBackup()
	if err != nil {
		return nil, rpc.ErrDatabaseFail
	}

	var r pb.ListBackupImageReply
	for _, b := range backups {
		r.Images = append(r.Images, b.ImageRef)
	}

	return &r, nil
}

func addBackupJob(b *litedb.ContainerBackup) {
	cli, err := docker.Client()
	if err != nil {
		log.Warnf("create docker cli err=%v", err)
		b.Status = 2
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), common.Config.Agent.BackupTimeoutDuration())
	defer cancel()

	info, err := cli.ContainerInspect(ctx, b.ContainerID)
	if err != nil {
		log.Warnf("inspect container=%v err=%v", b.ContainerID, err)
		b.Status = 2
		if err := litedb.UpdateContainerBackup(b); err != nil {
			log.Errorf("db.UpdateContainerBackup err=%v", err)
		}
		return
	}

	if info.Config == nil {
		log.Warnf("inspect container=%v .Config=nil", b.ContainerID)
		b.Status = 2
		if err := litedb.UpdateContainerBackup(b); err != nil {
			log.Errorf("db.UpdateContainerBackup err=%v", err)
		}
		return
	}

	b.ImageRef = strings.Split(info.Config.Image, ":")[0] + ":" + b.BackupName
	newImage, err := cli.ContainerCommit(ctx, b.ContainerID, types.ContainerCommitOptions{
		Reference: b.ImageRef,
		Pause:     true,
	})
	if err != nil {
		log.Warnf("commit container=%v err=%v", b.ContainerID, err)
		b.Status = 2
		if err := litedb.UpdateContainerBackup(b); err != nil {
			log.Errorf("db.UpdateContainerBackup err=%v", err)
		}
		return
	}

	imageInfo, _, err := cli.ImageInspectWithRaw(ctx, newImage.ID)
	if err != nil {
		log.Warnf("inspect image=%v err=%v", newImage.ID, err)
		b.Status = 2
	} else {
		log.Debugf("backup id=%v finished", b.ID)
		b.ImageID = strings.TrimPrefix(newImage.ID, "sha256:")
		b.ImageSize = imageInfo.Size
		b.Status = 1
	}

	if err := litedb.UpdateContainerBackup(b); err != nil {
		log.Errorf("db.UpdateContainerBackup err=%v", err)
	}
}

func (*ContainerServer) ExportBackup(in *pb.ExportBackupRequest, stream pb.Container_ExportBackupServer) error {
	b, err := litedb.QueryContainerBackupByID(in.BackupId)
	if err != nil {
		log.Warnf("QueryContainerBackupByID err=%v", err)
		return rpc.ErrInternal

	}

	if b.Status != int8(pb.BACKUP_STATUS_SUCCEED) {
		return status.Error(codes.Internal, "备份失败，无法导出")
	}

	cli, err := docker.Client()
	if err != nil {
		return rpc.ErrInternal
	}

	img, _, err := cli.ImageInspectWithRaw(context.Background(), b.ImageRef)
	if err != nil {
		if docker.IsErrNotFound(err) {
			return rpc.ErrNotFound
		}
		log.Warnf("ImageInspectWithRaw err=%v", err)
		return rpc.ErrInternal
	}

	r, err := cli.ImageSave(context.Background(), []string{b.ImageRef})
	if err != nil {
		log.Warnf("ImageSave err=%v", err)
		return rpc.ErrInternal
	}
	defer r.Close()

	chunk := pb.ExportBackupReply{
		ImgSize:    img.Size,
		ImgName:    strings.Split(b.ImageRef, ":")[0],
		ImgVersion: in.ImgVersion,
	}
	if err := stream.Send(&chunk); err != nil {
		log.Warnf("ExportBackup send image info err=%v", err)
		return rpc.ErrInternal
	}

	chunk.ImgSize = 0
	chunk.ImgName = ""
	chunk.ImgVersion = ""
	buf := make([]byte, 16*1024)
	for {
		n, err := r.Read(buf)
		if err != nil {
			if err == io.EOF {
				break
			}
			log.Warnf("read image content err=%v", err)
			return rpc.ErrInternal
		}
		chunk.Data = buf[:n]
		if err := stream.Send(&chunk); err != nil {
			log.Warnf("ExportBackup send stream err=%v", err)
			return rpc.ErrInternal
		}
	}
	return nil
}

func (*ContainerServer) InspectBackup(ctx context.Context, in *pb.InspectBackupRequest) (*pb.InspectBackupReply, error) {
	b, err := litedb.QueryContainerBackupByID(in.BackupId)
	if err != nil {
		log.Warnf("GetContainerConfigs err=%v", err)
		return nil, rpc.ErrInternal

	}

	var reply pb.InspectBackupReply
	reply.Backup = &pb.ContainerBackup{
		Id:         b.ID,
		Uuid:       b.UUID,
		BackupName: b.BackupName,
		BackupDesc: b.BackupDesc,
		ImageRef:   b.ImageRef,
		ImageSize:  b.ImageSize,
		Status:     int64(b.Status),
		CreatedAt:  b.CreatedAt,
	}

	return &reply, nil
}

func (*ContainerServer) AddAppEntry(ctx context.Context, in *pb.AddAppEntryRequest) (*pb.AddAppEntryReply, error) {
	if in.App == nil {
		return nil, rpc.ErrInvalidArgument
	}

	if err := litedb.AddAppEntry(&litedb.AppEntry{
		ContainerID: in.App.ContainerId,
		Name:        in.App.Name,
		ExePath:     in.App.ExePath,
		IsGUI:       in.App.IsGui,
	}); err != nil {
		log.Warnf("AddAppEntry err=%v", err)
		return nil, rpc.ErrDatabaseFail
	}

	return &pb.AddAppEntryReply{}, nil
}

func (*ContainerServer) RemoveAppEntry(ctx context.Context, in *pb.RemoveAppEntryRequest) (*pb.RemoveAppEntryReply, error) {
	if len(in.AppIds) == 0 {
		return nil, rpc.ErrInvalidArgument
	}

	for _, appId := range in.AppIds {
		e, err := litedb.GetAppEntryByID(appId)
		if err != nil {
			log.Warnf("litedb.GetAppEntryByID err=%v", err)
			return nil, rpc.ErrDatabaseFail
		}

		if appIsRunning(ctx, e.IDResponse) {
			return nil, status.Errorf(codes.Internal, "%v正在运行,无法删除", e.Name)
		}
	}

	if err := litedb.RemoveAppEntry(in.AppIds); err != nil {
		log.Warnf("litedb.RemoveAppEntry err=%v", err)
		return nil, rpc.ErrDatabaseFail
	}

	return &pb.RemoveAppEntryReply{}, nil
}

func (*ContainerServer) UpdateAppEntry(ctx context.Context, in *pb.UpdateAppEntryRequest) (*pb.UpdateAppEntryReply, error) {
	if in.App == nil {
		return nil, rpc.ErrInvalidArgument
	}

	e, err := litedb.GetAppEntryByID(in.App.Id)
	if err != nil {
		log.Warnf("litedb.GetAppEntryByID err=%v", err)
		return nil, rpc.ErrDatabaseFail
	}

	if appIsRunning(ctx, e.IDResponse) {
		return nil, status.Error(codes.Internal, "应用正在运行")
	}

	e.ExePath = in.App.ExePath
	e.Name = in.App.Name
	e.IsGUI = in.App.IsGui
	if err := litedb.UpdateAppEntry(e); err != nil {
		log.Warnf("litedb.UpdateAppEntry err=%v", err)
		return nil, rpc.ErrDatabaseFail
	}

	return &pb.UpdateAppEntryReply{}, nil
}

func (*ContainerServer) ListAppEntry(ctx context.Context, in *pb.ListAppEntryRequest) (*pb.ListAppEntryReply, error) {
	if in.ContainerId == "" {
		return nil, rpc.ErrInvalidArgument
	}

	entries, err := litedb.ListAppEntry(in.ContainerId)
	if err != nil {
		log.Warnf("litedb.ListAppEntry err=%v", err)
		return nil, rpc.ErrDatabaseFail
	}

	var reply pb.ListAppEntryReply
	for _, e := range entries {
		reply.Apps = append(reply.Apps, &pb.AppEntry{
			Id:          e.ID,
			ContainerId: e.ContainerID,
			Name:        e.Name,
			ExePath:     e.ExePath,
			IsGui:       e.IsGUI,
			IsRunning:   appIsRunning(ctx, e.IDResponse),
		})
	}

	return &reply, nil
}

func (*ContainerServer) RunAppEntry(ctx context.Context, in *pb.RunAppEntryRequest) (*pb.RunAppEntryReply, error) {
	app, err := litedb.GetAppEntryByID(in.AppId)
	if err != nil {
		log.Warnf("litedb.GetAppEntryByID err=%v", err)
		return nil, rpc.ErrDatabaseFail
	}

	if app.IsGUI {
		return nil, status.Error(codes.InvalidArgument, "无法运行GUI应用")
	}

	cli, err := docker.Client()
	if err != nil {
		log.Warnf("docker.Client err:%s", err)
		return nil, rpc.ErrInternal
	}

	if !containerIsRunning(ctx, cli, app.ContainerID) {
		return nil, status.Error(codes.InvalidArgument, "容器未运行")
	}

	if appIsRunningWithCli(ctx, cli, app.IDResponse) {
		return nil, status.Error(codes.Internal, "应用正在运行")
	}

	idRes, err := cli.ContainerExecCreate(ctx, app.ContainerID, types.ExecConfig{
		Cmd: strings.Split(app.ExePath, " "),
	})
	if err != nil {
		log.Warnf("ContainerExecCreate err=%v", err)
		return nil, rpc.ErrInternal
	}
	if idRes.ID == "" {
		log.Warnln("ContainerExecCreate exec ID empty")
		return nil, rpc.ErrInternal
	}

	if err := cli.ContainerExecStart(ctx, idRes.ID, types.ExecStartCheck{
		Detach: true,
		Tty:    false,
	}); err != nil {
		log.Warnf("ContainerExecStart err=%v", err)
		return nil, rpc.ErrInternal
	}

	app.IDResponse = idRes.ID
	if err := litedb.UpdateAppEntry(app); err != nil {
		log.Warnf("litedb.UpdateAppEntry err=%v", err)
		return nil, rpc.ErrDatabaseFail
	}

	return &pb.RunAppEntryReply{}, nil
}

func (*ContainerServer) KillAppEntry(ctx context.Context, in *pb.KillAppEntryRequest) (*pb.KillAppEntryReply, error) {
	app, err := litedb.GetAppEntryByID(in.AppId)
	if err != nil {
		log.Warnf("litedb.GetAppEntryByID err=%v", err)
		return nil, rpc.ErrDatabaseFail
	}

	if app.IsGUI {
		return nil, status.Error(codes.InvalidArgument, "无法停止GUI应用")
	}

	cli, err := docker.Client()
	if err != nil {
		log.Warnf("docker.Client err:%s", err)
		return nil, rpc.ErrInternal
	}

	if !containerIsRunning(ctx, cli, app.ContainerID) {
		return nil, status.Error(codes.InvalidArgument, "容器未运行")
	}

	execIns, err := cli.ContainerExecInspect(ctx, app.IDResponse)
	if err != nil {
		log.Warnf("cli.ContainerExecInspect err:%s", err)
		return nil, rpc.ErrInternal
	}

	if !execIns.Running {
		return nil, status.Error(codes.Internal, "应用未运行")
	}

	err = syscall.Kill(execIns.Pid, syscall.SIGKILL)
	if err != nil {
		log.Warnf("cli.ContainerExecInspect err:%s", err)
		return nil, status.Error(codes.Internal, "停止应用失败")
	}

	return &pb.KillAppEntryReply{}, nil
}

func (*ContainerServer) Pause(ctx context.Context, in *pb.PauseRequest) (*pb.PauseReply, error) {
	if in.ContainerId == "" {
		return nil, rpc.ErrInvalidArgument
	}
	cli, err := docker.Client()
	if err != nil {
		return nil, rpc.ErrInternal
	}
	if err = cli.ContainerPause(ctx, in.ContainerId); err != nil {
		log.Errorf("ContainerPause: id=%v %v", in.ContainerId, err)
		return nil, transDockerError(err)
	}
	return &pb.PauseReply{}, nil
}

func (*ContainerServer) Unpause(ctx context.Context, in *pb.UnpauseRequest) (*pb.UnpauseReply, error) {
	if in.ContainerId == "" {
		return nil, rpc.ErrInvalidArgument
	}
	cli, err := docker.Client()
	if err != nil {
		return nil, rpc.ErrInternal
	}
	if err = cli.ContainerUnpause(ctx, in.ContainerId); err != nil {
		log.Errorf("ContainerPause: id=%v %v", in.ContainerId, err)
		return nil, transDockerError(err)
	}
	return &pb.UnpauseReply{}, nil
}

func (*ContainerServer) LoadImage(in *pb.LoadRequest, stream pb.Container_LoadImageServer) error {
	if _, err := os.Stat(in.TarPath); err != nil {
		log.Errorf("file not found err: %v", err)
		return rpc.ErrInternal
	}
	pr, pw := io.Pipe()
	defer pr.Close()
	go func() {
		defer pw.Close()
		file, err := os.Open(in.TarPath)
		if err != nil {
			log.Errorf("open file failed: %v", err)
			pw.CloseWithError(rpc.ErrNotFound)
			return
		}
		defer file.Close()
		buf := make([]byte, bufSize)
		for {
			n, err := file.Read(buf)
			if err != nil && err != io.EOF {
				log.Errorf("read tar file error: %v", err)
				pw.CloseWithError(err)
				return
			}
			if n > 0 {
				if _, err := pw.Write(buf[:n]); err != nil {
					log.Errorf("write to buf error: %v", err)
					return
				}
				if err == io.EOF {
					log.Infof("write to buf finished.")
					return
				}
			}
		}
	}()
	dockerCli, err := docker.ClientWithTimeout(dockerTimeout)
	if err != nil {
		log.Errorf("docker cli init error: %v", err)
	}
	defer dockerCli.Close()
	ctx, cancel := context.WithTimeout(stream.Context(), dockerTimeout)
	defer cancel()
	resp, err := dockerCli.ImageLoad(ctx, pr, false)
	if err != nil {
		log.Errorf("image loaded error: %v", err)
	}
	defer resp.Body.Close()
	decoder := json.NewDecoder(resp.Body)
	var lastPercent float32
	for {
		var msg jsonmessage.JSONMessage
		if err = decoder.Decode(&msg); err != nil {
			if err == io.EOF {
				return stream.Send(&pb.LoadProgress{
					Update: &pb.LoadProgress_Result{
						Result: &pb.LoadResult{
							Success: true,
							Detail:  "image load success",
						},
					},
				})
			}
			log.Errorf("response resolve failed: %v", err)
			return rpc.ErrInternal
		}
		if msg.Error != nil {
			return stream.Send(&pb.LoadProgress{
				Update: &pb.LoadProgress_Result{
					Result: &pb.LoadResult{
						Success: false,
						Detail:  msg.Error.Message,
					},
				},
			})
		}
		// 处理进度更新
		if msg.Progress != nil {
			currentPercent := calculatePercent(msg.Progress)
			if currentPercent != lastPercent {
				lastPercent = currentPercent
				if err := stream.Send(&pb.LoadProgress{
					Update: &pb.LoadProgress_Percent{Percent: currentPercent},
				}); err != nil {
					log.Error("image load progress update failed: %v", err)
					return rpc.ErrInternal
				}
			}
		}
		if msg.Stream != "" {
			if err := stream.Send(&pb.LoadProgress{
				Update: &pb.LoadProgress_Message{Message: msg.Stream},
			}); err != nil {
				log.Error("image load message update failed: %v", err)
				return rpc.ErrInternal
			}
		}
	}
}

func calculatePercent(p *jsonmessage.JSONProgress) float32 {
	if p.Total == 0 {
		return 0
	}
	return float32(p.Current) / float32(p.Total) * 100
}

func (s *ContainerServer) SetContainerExtraHosts(ctx context.Context, in *pb.SetHostsRequest) (reply *pb.SetHostsReply, err error) {
	hostsFile := path.Join(hostsPath, in.ContainerId, "hosts")
	content, err := ioutil.ReadFile(hostsFile)
	if err != nil {
		log.Errorf("read container hosts file failed: %v", err)
		return nil, rpc.ErrInternal
	}
	trimmedContent := strings.TrimRight(string(content), "\n")
	var processedLines []string
	for _, line := range in.ContainerExtrahosts {
		processedLine := strings.ReplaceAll(line, ":", " ")
		processedLines = append(processedLines, processedLine)
	}
	fullContent := trimmedContent + "\n" + strings.Join(processedLines, "\n") + "\n"
	newHostsFilePath := path.Join(hostsPath, in.ContainerId, "new_hosts")
	if err := ioutil.WriteFile(newHostsFilePath, []byte(fullContent), 0644); err != nil {
		log.Errorf("write to new container hosts failed: %v", err)
		return nil, rpc.ErrInternal
	}
	ctx_, cancel := context.WithCancel(context.Background())
	defer cancel()
	inspectReq := &pb.InspectRequest{
		NodeId:      in.NodeId,
		ContainerId: in.ContainerId,
	}
	inspectResp, err := s.Inspect(ctx_, inspectReq)
	if err != nil {
		log.Errorf("inspect container id=%v  error: %v", in.ContainerId, err)
	}
	containerConfigs := inspectResp.Configs
	hostsMount := &pb.Mount{
		Type:        "bind",
		Source:      newHostsFilePath,
		Target:      "/etc/hosts",
		ReadOnly:    false,
		Consistency: "consistent",
	}
	containerConfigs.Mounts = append(containerConfigs.Mounts, hostsMount)
	updateRequest := &pb.UpdateRequest{
		NodeId:        in.NodeId,
		ContainerId:   in.ContainerId,
		ResourceLimit: containerConfigs.ResouceLimit,
		RestartPolicy: containerConfigs.RestartPolicy,
		Networks:      containerConfigs.Networks,
		Mounts:        containerConfigs.Mounts,
	}
	_, err = s.Update(ctx_, updateRequest)
	if err != nil {
		log.Errorf("update container id=%v hosts failed: %v", in.ContainerId, err)
		return nil, rpc.ErrInternal
	}
	return reply, nil
}

func (s *ContainerServer) SetContainerQuota(ctx context.Context, in *pb.SetQuotaRequest) (reply *pb.SetQuotaReply, err error) {
	//先将配置写入数据库
	data := &db.ContainerLimits{
		NodeId:      in.NodeId,
		ContainerId: in.ContainerId,
		UploadSpeed: in.ContainerLimit.NetWbw,
		DownSpeed:   in.ContainerLimit.NetRbw,
	}
	if err := db.CreateContainerLimits(data); err != nil {
		log.Errorf("create  container limits failed: %v", err)
		return nil, rpc.ErrInternal
	}
	//开启协程监听docker事件，监听到start restart事件后，处理容器的限速功能
	cli, err := docker.Client()
	if err != nil {
		return nil, rpc.ErrInternal
	}
	dm := GetDockerManager(cli)
	go dm.Start()
	return reply, nil
}

func (s *ContainerServer) GetContainerPid(containerId string) (int, error) {
	_, pid, err := s.inspect(containerId, false)
	if err != nil {
		return 0, fmt.Errorf("Get container Pid error: %v", err)
	}
	return pid, nil
}

func (*ContainerServer) GetContainerStats(ctx context.Context, in *pb.StatsRequest) (*pb.StatsReply, error) {
	// 这里并不关心 StatsRequest.HostId
	json_str, _ := docker.HandleStatsRequest()
	return &pb.StatsReply{StatsData: json_str}, nil
}
