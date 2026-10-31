package thriftapi

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"scmc/server/args"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"scmc/common"
	"scmc/model/db"
	containerPB "scmc/rpc/pb/container"
	imagePB "scmc/rpc/pb/image"
	networkPB "scmc/rpc/pb/network"
	nodePB "scmc/rpc/pb/node"
	securityPB "scmc/rpc/pb/security"
	"scmc/server/controller/internal"
	"scmc/thrift/gen-go/ksscmc"

	"github.com/apache/thrift/lib/go/thrift"
	"github.com/sirupsen/logrus"
)

type KsScmcHandler struct {
}

func (h *KsScmcHandler) InstallContainer(ctx context.Context, node_nameorip string, image_name string, image_ver string, container_name string) (int32, error) {
	//通过node_nameorip查找node的ID
	nodeInfo, err := GetNodeId(node_nameorip)
	if err != nil {
		logrus.Error(err)
		return 1, nil
	}

	conn, err := internal.GetAgentConn(nodeInfo.Address)
	if err != nil {
		logrus.Error(err)
		return 2, nil
	}
	ctx_, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	r, err := networkPB.NewNetworkClient(conn).List(ctx_, &networkPB.ListRequest{})
	if err != nil {
		logrus.Error(err)
		return 3, nil
	}
	networkInterfaceName := r.VirtualIfs[0].Name

	//组装PB数据
	containerConfig := &containerPB.ContainerConfigs{
		Name:  container_name,
		Image: image_name + ":" + image_ver,
		Networks: []*containerPB.NetworkConfig{
			{Interface: networkInterfaceName},
		},
		RestartPolicy: &containerPB.RestartPolicy{
			Name: "no",
		},
		ResouceLimit: &containerPB.ResourceLimit{
			CpuPrio:         0,
			MemoryLimit:     31528,
			MemorySoftLimit: 31528,
		},
		SecurityConfig: &containerPB.SecurityConfig{
			ProcProtection: &securityPB.ProcProtection{
				ProtectionType: 1,
			},
			NprocProtection: &securityPB.ProcProtection{
				ProtectionType: 2,
			},
			NetworkRule: &securityPB.NetworkRuleList{
				Rules: []*securityPB.NetworkRule{
					{Protocols: []string{"tcp"}},
				},
			},
		},
	}

	createRequest := &containerPB.CreateRequest{
		NodeId:  nodeInfo.ID,
		Configs: containerConfig,
	}

	containerServer := &internal.ContainerServer{}
	_, err = containerServer.Create(ctx, createRequest)
	if err != nil {
		logrus.Error(err)
		return 4, nil
	}

	return 0, nil
}

func (h *KsScmcHandler) StartContainer(ctx context.Context, node_nameorip string, container_name string) (int32, error) {

	nodeInfo, err := GetNodeId(node_nameorip)
	if err != nil {
		logrus.Error(err)
		return 1, nil
	}

	conn, err := internal.GetAgentConn(nodeInfo.Address)
	if err != nil {
		logrus.Error(err)
		return 2, nil
	}
	cli := containerPB.NewContainerClient(conn)

	ctx_, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	containerId, err := GetContainerId(cli, container_name, ctx_)
	if err != nil {
		logrus.Error(err)
		return 3, nil
	}

	//启动容器
	request := containerPB.StartRequest{
		Ids: []*containerPB.ContainerIdList{
			{
				ContainerIds: []string{containerId},
			},
		},
	}

	_, err = cli.Start(ctx_, &request)
	if err != nil {
		logrus.Error(err)
		return 4, nil
	}

	return 0, nil
}

func (h *KsScmcHandler) StopContainer(ctx context.Context, node_nameorip string, container_name string) (int32, error) {

	nodeInfo, err := GetNodeId(node_nameorip)
	if err != nil {
		logrus.Error(err)
		return 1, nil
	}

	ctx_, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	conn, err := internal.GetAgentConn(nodeInfo.Address)
	if err != nil {
		logrus.Error(err)
		return 2, nil
	}

	cli := containerPB.NewContainerClient(conn)
	containerId, err := GetContainerId(cli, container_name, ctx_)
	if err != nil {
		logrus.Error(err)
		return 3, nil
	}

	//停止容器
	request := containerPB.StopRequest{
		Ids: []*containerPB.ContainerIdList{
			{
				ContainerIds: []string{containerId},
			},
		},
	}

	_, err = cli.Stop(ctx_, &request)
	if err != nil {
		logrus.Error(err)
		return 4, nil
	}

	return 0, nil
}

func (h *KsScmcHandler) UninstallContainer(ctx context.Context, node_nameorip string, container_name string) (int32, error) {

	nodeInfo, err := GetNodeId(node_nameorip)
	if err != nil {
		logrus.Error(err)
		return 1, nil
	}

	ctx_, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	conn, err := internal.GetAgentConn(nodeInfo.Address)
	if err != nil {
		logrus.Error(err)
		return 2, nil
	}

	cli := containerPB.NewContainerClient(conn)

	containerId, err := GetContainerId(cli, container_name, ctx_)
	if err != nil {
		logrus.Error(err)
		return 3, nil
	}

	//停止容器
	stopRequest := &containerPB.StopRequest{
		Ids: []*containerPB.ContainerIdList{
			{
				ContainerIds: []string{containerId},
			},
		},
	}

	_, err = cli.Stop(ctx_, stopRequest)
	if err != nil {
		logrus.Error(err)
		return 4, nil
	}

	removeRequest := &containerPB.RemoveRequest{
		Ids: []*containerPB.ContainerIdList{
			{
				ContainerIds: []string{containerId},
			},
		},
	}

	_, err = cli.Remove(ctx_, removeRequest)
	if err != nil {
		logrus.Error(err)
		return 5, nil
	}

	return 0, nil
}

func (h *KsScmcHandler) GetContainerStatus(ctx context.Context, node_nameorip string, container_name string) (int32, error) {

	nodeInfo, err := GetNodeId(node_nameorip)
	if err != nil {
		logrus.Error(err)
		return 1, nil
	}

	ctx_, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	conn, err := internal.GetAgentConn(nodeInfo.Address)
	if err != nil {
		logrus.Error(err)
		return 2, nil
	}

	cli := containerPB.NewContainerClient(conn)
	r, err := cli.List(ctx_, &containerPB.ListRequest{})
	if err != nil {
		logrus.Error(err)
		return 3, nil
	}

	var containerInfo *containerPB.ContainerInfo
	for i := range r.Containers {
		if r.Containers[i].Info.Name == container_name {
			containerInfo = r.Containers[i].Info
		}
	}

	if containerInfo == nil {
		logrus.Error(err)
		return 4, nil
	}

	if containerInfo.State != "running" {
		logrus.Error(err)
		return 5, nil
	}

	return 0, nil
}

func (h *KsScmcHandler) GetContainerMetrics(ctx context.Context, node_nameorip string, container_name string) (*ksscmc.GetContainerMetricsResponse, error) {

	containerMetricResponse := &ksscmc.ContainerMetric{}
	response := &ksscmc.GetContainerMetricsResponse{
		ContainerMetric: containerMetricResponse,
		RetCode:         1, // 成功码
	}

	nodeInfo, err := GetNodeId(node_nameorip)
	if err != nil {
		logrus.Error(err)
		return response, nil
	}

	ctx_, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	conn, err := internal.GetAgentConn(nodeInfo.Address)
	if err != nil {
		logrus.Error(err)
		return response, nil
	}

	cli := containerPB.NewContainerClient(conn)
	r, err := cli.List(ctx_, &containerPB.ListRequest{})
	if err != nil {
		logrus.Error(err)
		return response, nil
	}

	var containerInfo *containerPB.ContainerInfo
	for i := range r.Containers {
		if r.Containers[i].Info.Name == container_name {
			containerInfo = r.Containers[i].Info
		}
	}

	if containerInfo == nil {
		logrus.Error(err)
		return response, nil
	}

	ContainerMetric := &ksscmc.ContainerMetric{
		ContainerName:   containerInfo.Name,
		ContainerID:     containerInfo.Id,
		NodeName:        nodeInfo.Name,
		ContainerStatus: 1,
	}

	if containerInfo.State == "running" {
		ContainerMetric.ContainerStatus = 0
	}

	if containerInfo.ResourceStat.CpuStat != nil {
		ContainerMetric.CPUUsage = containerInfo.ResourceStat.CpuStat.CoreUsed
	}

	if containerInfo.ResourceStat.MemStat != nil {
		ContainerMetric.MemUsage = int32(containerInfo.ResourceStat.MemStat.Used)
	}

	if containerInfo.ResourceStat.DiskStat != nil {
		ContainerMetric.DiskUsage = containerInfo.ResourceStat.DiskStat.Used
	}

	if containerInfo.ResourceStat.NetStat != nil {
		ContainerMetric.NetUpload = containerInfo.ResourceStat.NetStat.Tx
		ContainerMetric.NetDownload = containerInfo.ResourceStat.NetStat.Rx
	}

	response.ContainerMetric = ContainerMetric
	response.RetCode = 0
	return response, nil
}

func (h *KsScmcHandler) GetImageVer4Container(ctx context.Context, node_nameorip string, container_name string) (*ksscmc.GetImageVer4ContainerResponse, error) {

	response := &ksscmc.GetImageVer4ContainerResponse{
		ContainerImage: &ksscmc.ContainerImage{},
		RetCode:        1, // 成功码
	}

	nodeInfo, err := GetNodeId(node_nameorip)
	if err != nil {
		logrus.Error(err)
		return response, nil
	}

	ctx_, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	conn, err := internal.GetAgentConn(nodeInfo.Address)
	if err != nil {
		logrus.Error(err)
		return response, nil
	}

	cli := containerPB.NewContainerClient(conn)
	r, err := cli.List(ctx_, &containerPB.ListRequest{})
	if err != nil {
		logrus.Error(err)
		return response, nil
	}

	var containerInfo *containerPB.ContainerInfo
	for i := range r.Containers {
		if r.Containers[i].Info.Name == container_name {
			containerInfo = r.Containers[i].Info
		}
	}

	if containerInfo == nil {
		logrus.Error(err)
		return response, nil
	}

	imageInfo := strings.Split(containerInfo.Image, ":")
	containerImage := &ksscmc.ContainerImage{
		ImageName:    imageInfo[0],
		InuseVersion: imageInfo[1],
	}

	response.ContainerImage = containerImage
	response.RetCode = 0
	return response, nil
}

func (h *KsScmcHandler) GetContainer(ctx context.Context, node_name string, container_name string) (int32, error) {

	nodeInfo, err := GetNodeId(node_name)
	if err != nil {
		logrus.Error(err)
		return 1, nil
	}

	ctx_, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	conn, err := internal.GetAgentConn(nodeInfo.Address)
	if err != nil {
		logrus.Error(err)
		return 2, nil
	}

	cli := containerPB.NewContainerClient(conn)
	r, err := cli.List(ctx_, &containerPB.ListRequest{})
	if err != nil {
		logrus.Error(err)
		return 3, nil
	}

	var containerInfo *containerPB.ContainerInfo
	for i := range r.Containers {
		if r.Containers[i].Info.Name == container_name {
			containerInfo = r.Containers[i].Info
		}
	}

	if containerInfo == nil {
		logrus.Error(err)
		return 4, nil
	}

	return 0, nil
}

func (h *KsScmcHandler) UpdateContainer(ctx context.Context, node_name string, container_name string, new_ver string) (int32, error) {

	nodeInfo, err := GetNodeId(node_name)
	if err != nil {
		logrus.Error(err)
		return 1, nil
	}

	ctx_, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	conn, err := internal.GetAgentConn(nodeInfo.Address)
	if err != nil {
		logrus.Error(err)
		return 2, nil
	}

	cli := containerPB.NewContainerClient(conn)
	r, err := cli.List(ctx_, &containerPB.ListRequest{})
	if err != nil {
		logrus.Error(err)
		return 3, nil
	}

	var containerInfo *containerPB.ContainerInfo
	for i := range r.Containers {
		if r.Containers[i].Info.Name == container_name {
			containerInfo = r.Containers[i].Info
		}
	}

	if containerInfo == nil {
		logrus.Error(err)
		return 4, nil
	}

	//inspect
	inspectReply, err := cli.Inspect(ctx_, &containerPB.InspectRequest{
		NodeId:      nodeInfo.ID,
		ContainerId: containerInfo.Id,
	})

	if err != nil {
		logrus.Error(err)
		return 5, nil
	}

	//stop container
	stopRequest := &containerPB.StopRequest{
		Ids: []*containerPB.ContainerIdList{
			{
				ContainerIds: []string{containerInfo.Id},
			},
		},
	}

	_, err = cli.Stop(ctx_, stopRequest)
	if err != nil {
		logrus.Error(err)
		return 6, nil
	}

	//remove container
	removeRequest := &containerPB.RemoveRequest{
		Ids: []*containerPB.ContainerIdList{
			{
				ContainerIds: []string{containerInfo.Id},
			},
		},
	}
	_, err = cli.Remove(ctx_, removeRequest)
	if err != nil {
		logrus.Error(err)
		return 7, nil
	}

	//create container
	oldImage := inspectReply.Configs.Image
	imageArr := strings.Split(oldImage, ":")
	inspectReply.Configs.Image = imageArr[0] + ":" + new_ver
	createRequest := &containerPB.CreateRequest{
		NodeId:  nodeInfo.ID,
		Configs: inspectReply.Configs,
	}

	containerServer := &internal.ContainerServer{}
	_, err = containerServer.Create(ctx, createRequest)
	if err != nil {
		logrus.Error(err)
		return 8, nil
	}

	return 0, nil
}

func (h *KsScmcHandler) GetAlarmList(ctx context.Context, node_name string) (*ksscmc.GetAlarmListResponse, error) {

	response := &ksscmc.GetAlarmListResponse{
		Alarmlist: []*ksscmc.ContainerAlarm{},
		RetCode:   1, // 成功码
	}

	nodeInfo, err := GetNodeId(node_name)
	if err != nil {
		logrus.Error(err)
		return response, nil
	}
	_, data, err := db.ListWarnLogAll(1000, 1, nodeInfo.ID, 2)
	if err != nil {
		logrus.Warnf("ListRuntime db err=%v", err)
		return response, nil
	}

	for _, w := range data {
		item := &ksscmc.ContainerAlarm{
			ID:            int32(w.ID),
			Message:       w.Detail,
			ContainerName: w.ContainerName,
			Status:        0,
		}
		t := time.Unix(w.CreatedAt, 0)
		item.AlarmTime = t.Format("2006-01-02 15:04:05")
		if w.HaveRead {
			item.Status = 1
		}
		response.Alarmlist = append(response.Alarmlist, item)
	}

	response.RetCode = 0
	return response, nil
}

func (h *KsScmcHandler) GetAllNodeList(ctx context.Context) (*ksscmc.GetAllNodeListResponse, error) {

	response := &ksscmc.GetAllNodeListResponse{
		Nodelist: []*ksscmc.NodeStatus{},
		RetCode:  1, // 成功码
	}

	nodes, err := db.ListNodes()
	if err != nil {
		logrus.Error(err)
		return response, nil
	}

	for _, n := range nodes {
		nodestatus := ksscmc.NodeStatusType_NODE_OFFLINE
		item := &ksscmc.NodeStatus{
			NodeName:         n.Name,
			NodeID:           strconv.FormatInt(n.ID, 10),
			IPAddr:           n.Address,
			Status:           nodestatus,
			CPUUsage:         0,
			CPUTotal:         0,
			MemUsage:         0,
			MemTotal:         0,
			DiskUsage:        0,
			ContainerNums:    0,
			ContainerMaxNums: 0,
		}

		conn, err := internal.GetAgentConn(n.Address)
		if err == nil {
			cli := nodePB.NewNodeClient(conn)
			ctx_, cancel := context.WithTimeout(context.Background(), time.Second*5)
			defer cancel()
			r, err := cli.Status(ctx_, &nodePB.StatusRequest{})
			logrus.Debugf("%v", r)
			// logrus.Debugf("%v  %v ", r, r.StatusList)
			if err == nil && int32(r.StatusList[0].State) == int32(nodePB.NodeState_Online) {
				item.Status = ksscmc.NodeStatusType_NODE_ONLINE
				item.CPUUsage = float64(r.StatusList[0].CpuStat.UsedPercentage * 100)
				item.CPUTotal = float64(r.StatusList[0].CpuStat.Total)
				item.MemUsage = float64(r.StatusList[0].MemStat.Used)
				item.MemTotal = float64(r.StatusList[0].MemStat.Total)
				item.DiskUsage = float64(r.StatusList[0].DiskStat.Used / 1024)
				item.DiskTotal = float64(r.StatusList[0].DiskStat.Total / 1024)
				item.ContainerNums = int32(r.StatusList[0].ContainerStat.Total)
				item.ContainerMaxNums = 9999
			}
		}

		response.Nodelist = append(response.Nodelist, item)
	}

	logrus.Debugf("get node list: %v", response.Nodelist)
	response.RetCode = 0
	return response, nil
}

func (h *KsScmcHandler) GetAllContainerList(ctx context.Context) (*ksscmc.GetAllContainerListResponse, error) {

	response := &ksscmc.GetAllContainerListResponse{
		Containerlist: []*ksscmc.ContainerMetric{},
		RetCode:       1, // 成功码
	}

	nodes, err := db.ListNodes()
	if err != nil {
		logrus.Error(err)
		return response, nil
	}

	idNodeMap := make(map[int64]*db.NodeInfo)
	for _, node := range nodes {
		idNodeMap[node.ID] = node
	}

	var (
		mtx   sync.Mutex
		wg    sync.WaitGroup
		reply containerPB.ListReply
	)

	for _, n := range nodes {

		wg.Add(1)
		go func(n *db.NodeInfo) {
			defer wg.Done()
			conn, err := internal.GetAgentConn(n.Address)
			logrus.Infof("agent conn: %v", conn)
			if err != nil {
				mtx.Lock()
				defer mtx.Unlock()
				// reply.FailNodes = append(reply.FailNodes, n.Address)
				return
			}

			r, err := containerPB.NewContainerClient(conn).List(ctx, &containerPB.ListRequest{
				NodeIds: []int64{n.ID},
				ListAll: true,
			})
			logrus.Infof("container reply: %v", r)
			if err != nil {
				logrus.Warnf("container.List node=%+v, err=%v", n, err)
				mtx.Lock()
				defer mtx.Unlock()
				// reply.FailNodes = append(reply.FailNodes, n.Address)
				return
			}

			for i := range r.Containers {
				r.Containers[i].NodeId = n.ID
				r.Containers[i].NodeAddress = n.Address
			}
			mtx.Lock()
			defer mtx.Unlock()
			reply.Containers = append(reply.Containers, r.Containers...)
		}(n)
	}

	wg.Wait()
	sort.Slice(reply.Containers, func(i, j int) bool {
		c0 := reply.Containers[i]
		c1 := reply.Containers[j]
		if c0.NodeId != c1.NodeId {
			return c0.NodeId < c1.NodeId
		}

		var n0, n1 string
		if c0.Info != nil {
			n0 = c0.Info.Name
		}
		if c1.Info != nil {
			n1 = c1.Info.Name
		}
		return strings.Compare(n0, n1) < 0
	})

	logrus.Infof("get all containers: %v", reply.Containers)

	for _, tmpcontainer := range reply.Containers {
		item := &ksscmc.ContainerMetric{
			ContainerName:   tmpcontainer.Info.Name,
			ContainerID:     tmpcontainer.Info.Id,
			NodeName:        idNodeMap[tmpcontainer.NodeId].Name,
			ContainerStatus: ksscmc.ContainerStatus_OFFLINE,
			CPUUsage:        0,
			MemUsage:        0,
			DiskUsage:       0,
			NetUpload:       0,
			NetDownload:     0,
		}
		if tmpcontainer.Info.State == "running" {
			logrus.Infof("tmpcontainer.Info.ResourceStat: %+v", tmpcontainer.Info.ResourceStat)
			item.ContainerStatus = ksscmc.ContainerStatus_ONLINE
			item.CPUUsage = tmpcontainer.Info.ResourceStat.CpuStat.CoreUsed * 100
			item.MemUsage = int32(tmpcontainer.Info.ResourceStat.MemStat.Used)
			item.DiskUsage = float64(tmpcontainer.Info.ResourceStat.DiskStat.Used / 1024)
			item.NetUpload = float64(tmpcontainer.Info.ResourceStat.NetStat.Tx / 1024 / 1024)
			item.NetDownload = float64(tmpcontainer.Info.ResourceStat.NetStat.Rx / 1024 / 1024)
		}

		response.Containerlist = append(response.Containerlist, item)
	}

	logrus.Debugf("get container list: %v", response.Containerlist)
	response.RetCode = 0
	return response, nil
}

func (h *KsScmcHandler) GetContainerNetwork(ctx context.Context, node_nameorip string, container_name string) (*ksscmc.GetContainerNetworkResponse, error) {

	response := &ksscmc.GetContainerNetworkResponse{
		ContainerNetInfo: &ksscmc.ContainerNetInfo{},
		RetCode:          1, // 成功码
	}

	nodeInfo, err := GetNodeId(node_nameorip)
	if err != nil {
		logrus.Error(err)
		return response, nil
	}

	conn, err := internal.GetAgentConn(nodeInfo.Address)
	if err != nil {
		logrus.Error(err)
		return response, nil
	}

	cli := containerPB.NewContainerClient(conn)
	ctx_, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	containerId, err := GetContainerId(cli, container_name, ctx_)
	if err != nil {
		logrus.Error(err)
		return response, nil
	}

	containerServer := internal.ContainerServer{}
	inspectReq := containerPB.InspectRequest{
		NodeId:      nodeInfo.ID,
		ContainerId: containerId,
	}

	inspect, err := containerServer.Inspect(ctx, &inspectReq)
	if err != nil {
		logrus.Error(err)
		return response, nil
	}

	response.ContainerNetInfo.IP = inspect.Configs.Networks[0].IpAddress
	mask := net.CIDRMask(int(inspect.Configs.Networks[0].IpPrefixLen), 32)
	if mask == nil {
		logrus.Error(fmt.Sprintf("Failed to conv IPPrefixLen: %d to string", inspect.Configs.Networks[0].IpPrefixLen))
	}

	response.ContainerNetInfo.Mask = net.IP(mask).String()
	response.ContainerNetInfo.Gateway = inspect.Configs.Networks[0].Gateway
	// todo:
	response.ContainerNetInfo.Port = 0
	response.RetCode = 0
	return response, nil
}

func (h *KsScmcHandler) GetContainerNetRules(ctx context.Context, node_name string, container_name string) (*ksscmc.GetContainerNetRulesResponse, error) {

	response := &ksscmc.GetContainerNetRulesResponse{
		ContainerNetRules: []*ksscmc.ContainerNetRule{},
		RetCode:           1, // 成功码
	}

	nodeInfo, err := GetNodeId(node_name)
	if err != nil {
		logrus.Error(err)
		return response, nil
	}

	conn, err := internal.GetAgentConn(nodeInfo.Address)
	if err != nil {
		logrus.Error(err)
		return response, nil
	}

	cli := containerPB.NewContainerClient(conn)
	ctx_, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	containerId, err := GetContainerId(cli, container_name, ctx_)
	if err != nil {
		logrus.Error(err)
		return response, nil
	}

	containerServer := internal.ContainerServer{}
	inspectReq := containerPB.InspectRequest{
		NodeId:      nodeInfo.ID,
		ContainerId: containerId,
	}

	inspect, err := containerServer.Inspect(ctx, &inspectReq)
	if err != nil {
		logrus.Error(err)
		return response, nil
	}

	for _, rule := range inspect.Configs.SecurityConfig.NetworkRule.Rules {
		response.ContainerNetRules = append(response.ContainerNetRules, &ksscmc.ContainerNetRule{
			Protocol: strings.Join(rule.Protocols, " "),
			IP:       rule.Addr,
			Port:     int32(rule.Port),
		})
	}

	response.RetCode = 0
	return response, nil
}

func (h *KsScmcHandler) AddContainerIpPort(ctx context.Context, node_nameorip string, container_name string, ip_addr string, port int32, net_proto string) (int32, error) {
	nodeInfo, err := GetNodeId(node_nameorip)
	if err != nil {
		logrus.Error(err)
		return 1, nil
	}

	conn, err := internal.GetAgentConn(nodeInfo.Address)
	if err != nil {
		logrus.Error(err)
		return 1, nil
	}

	cli := containerPB.NewContainerClient(conn)

	ctx_, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	containerId, err := GetContainerId(cli, container_name, ctx_)
	if err != nil {
		logrus.Error(err)
		return 3, nil
	}

	containerServer := internal.ContainerServer{}
	inspectReq := containerPB.InspectRequest{
		NodeId:      nodeInfo.ID,
		ContainerId: containerId,
	}

	inspect, err := containerServer.Inspect(ctx, &inspectReq)
	if err != nil {
		logrus.Error(err)
		return 4, nil
	}
	updateReq := containerPB.UpdateRequest{
		NodeId:         nodeInfo.ID,
		ContainerId:    containerId,
		ResourceLimit:  inspect.Configs.ResouceLimit,
		RestartPolicy:  inspect.Configs.RestartPolicy,
		Networks:       inspect.Configs.Networks,
		SecurityConfig: inspect.Configs.SecurityConfig,
		Mounts:         inspect.Configs.Mounts,
		Envs:           inspect.Configs.Envs,
		Cmd:            inspect.Configs.Cmd,
	}
	networkSettings := inspect.Configs.Networks
	networkSettings[0].IpAddress = ip_addr
	updateReq.Networks = append(updateReq.Networks, networkSettings...)
	_, err = cli.Update(ctx, &updateReq)
	if err != nil {
		logrus.Error(err)
		return 5, nil
	}
	return 0, nil
}

func (h *KsScmcHandler) AddContainerReqIpPort(ctx context.Context, node_name string, container_name string, ip_addr string, port int32, net_proto string) (int32, error) {

	nodeInfo, err := GetNodeId(node_name)
	if err != nil {
		logrus.Error(err)
		return 1, nil
	}
	conn, err := internal.GetAgentConn(nodeInfo.Address)
	if err != nil {
		logrus.Error(err)
		return 2, nil
	}
	cli := containerPB.NewContainerClient(conn)

	ctx_, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	containerId, err := GetContainerId(cli, container_name, ctx_)
	if err != nil {
		logrus.Error(err)
		return 3, nil
	}
	containerServer := internal.ContainerServer{}
	inspectReq := containerPB.InspectRequest{
		NodeId:      nodeInfo.ID,
		ContainerId: containerId,
	}
	inspect, err := containerServer.Inspect(ctx, &inspectReq)
	if err != nil {
		logrus.Error(err)
		return 4, nil
	}
	securityCfg := inspect.Configs.SecurityConfig
	if securityCfg == nil {
		securityCfg = &containerPB.SecurityConfig{
			NetworkRule: &securityPB.NetworkRuleList{},
		}
	}
	securityCfg.NetworkRule.IsOn = true
	isExist := false
	isContainProtocol := false
	for _, rule := range securityCfg.NetworkRule.Rules {
		if rule.Addr == ip_addr && rule.Port == uint32(port) {
			isExist = true
			for _, prot := range rule.Protocols {
				if prot == net_proto {
					isContainProtocol = true
				}
			}
			if !isContainProtocol {
				rule.Protocols = append(rule.Protocols, net_proto)
			}
		}
	}
	if !isExist {
		securityCfg.NetworkRule.Rules = append(securityCfg.NetworkRule.Rules, &securityPB.NetworkRule{
			Protocols: []string{net_proto},
			Addr:      ip_addr,
			Port:      uint32(port),
		})
	}

	updateReq := containerPB.UpdateRequest{
		NodeId:         nodeInfo.ID,
		ContainerId:    containerId,
		ResourceLimit:  inspect.Configs.ResouceLimit,
		RestartPolicy:  inspect.Configs.RestartPolicy,
		Networks:       inspect.Configs.Networks,
		SecurityConfig: securityCfg,
		Mounts:         inspect.Configs.Mounts,
		Envs:           inspect.Configs.Envs,
		Cmd:            inspect.Configs.Cmd,
	}
	_, err = containerServer.Update(ctx, &updateReq)
	if err != nil {
		logrus.Error(err)
		return 5, nil
	}
	return 0, nil
}

func (h *KsScmcHandler) AddContainerEnv(ctx context.Context, node_nameorip string, container_name string, env_name string, env_value string) (int32, error) {
	nodeInfo, err := GetNodeId(node_nameorip)
	if err != nil {
		logrus.Error(err)
		return 1, nil
	}
	conn, err := internal.GetAgentConn(nodeInfo.Address)
	if err != nil {
		logrus.Error(err)
		return 2, nil
	}
	cli := containerPB.NewContainerClient(conn)

	ctx_, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	containerId, err := GetContainerId(cli, container_name, ctx_)
	if err != nil {
		logrus.Error(err)
		return 3, nil
	}
	envs := map[string]string{
		env_name: env_value,
	}
	containerServer := internal.ContainerServer{}
	inspectReq := containerPB.InspectRequest{
		NodeId:      nodeInfo.ID,
		ContainerId: containerId,
	}
	inspect, err := containerServer.Inspect(ctx, &inspectReq)
	if err != nil {
		logrus.Error(err)
		return 4, nil
	}
	old_envs := inspect.Configs.Envs
	if old_envs == nil {
		old_envs = map[string]string{}
	}
	for k, v := range envs {
		old_envs[k] = v
	}
	updateReq := containerPB.UpdateRequest{
		NodeId:         nodeInfo.ID,
		ContainerId:    containerId,
		ResourceLimit:  inspect.Configs.ResouceLimit,
		RestartPolicy:  inspect.Configs.RestartPolicy,
		Networks:       inspect.Configs.Networks,
		SecurityConfig: inspect.Configs.SecurityConfig,
		Mounts:         inspect.Configs.Mounts,
		Envs:           old_envs,
		Cmd:            inspect.Configs.Cmd,
	}
	_, err = cli.Update(ctx, &updateReq)
	if err != nil {
		logrus.Error(err)
		return 5, nil
	}
	return 0, nil
}

func (h *KsScmcHandler) AddContainerLocalVolume(ctx context.Context, node_nameorip string, container_name string, in_dir string, out_dir string, access_auth ksscmc.AccessAuth) (int32, error) {

	nodeInfo, err := GetNodeId(node_nameorip)
	if err != nil {
		logrus.Error(err)
		return 1, nil
	}

	conn, err := internal.GetAgentConn(nodeInfo.Address)
	if err != nil {
		logrus.Error(err)
		return 2, nil
	}
	cli := containerPB.NewContainerClient(conn)

	ctx_, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	containerId, err := GetContainerId(cli, container_name, ctx_)
	if err != nil {
		logrus.Error(err)
		return 3, nil
	}

	containerServer := internal.ContainerServer{}
	inspectReq := containerPB.InspectRequest{
		NodeId:      nodeInfo.ID,
		ContainerId: containerId,
	}

	inspect, err := containerServer.Inspect(ctx, &inspectReq)
	if err != nil {
		logrus.Error(err)
		return 4, nil
	}

	logrus.Warnf("container inspect: %v", inspect)

	fileInfo, err := os.Stat(out_dir)
	if os.IsNotExist(err) {
		err := os.MkdirAll(out_dir, 0755)
		if err != nil {
			logrus.Warnf("create dir: %v faied", out_dir)
			return 5, nil
		}
	} else if !fileInfo.IsDir() {
		logrus.Warnf("%v exist and is not dir", out_dir)
		return 6, nil
	}

	updateReq := containerPB.UpdateRequest{
		NodeId:         nodeInfo.ID,
		ContainerId:    containerId,
		ResourceLimit:  inspect.Configs.ResouceLimit,
		RestartPolicy:  inspect.Configs.RestartPolicy,
		Networks:       inspect.Configs.Networks,
		SecurityConfig: inspect.Configs.SecurityConfig,
		Mounts:         inspect.Configs.Mounts,
		Envs:           inspect.Configs.Envs,
		Cmd:            inspect.Configs.Cmd,
	}
	mountSettings := inspect.Configs.Mounts
	mountSettings = append(mountSettings, &containerPB.Mount{
		Type:     "bind",
		Source:   out_dir,
		Target:   in_dir,
		ReadOnly: access_auth == ksscmc.AccessAuth_READ_ONLY,
	})

	updateReq.Mounts = append(updateReq.Mounts, mountSettings...)
	logrus.Warnf("updateReq: %v", updateReq)

	_, err = cli.Update(ctx, &updateReq)
	if err != nil {
		logrus.Error(err)
		return 7, nil
	}

	return 0, nil
}

func (h *KsScmcHandler) SetContainerDisplayV2(ctx context.Context, node_name string, container_name string, enabled bool, display string) (int32, error) {
	return 0, nil
}

func (h *KsScmcHandler) SetContainerQuota(ctx context.Context, node_name string, container_name string, containerLimit *ksscmc.ContainerLimit) (int32, error) {

	nodeInfo, err := GetNodeId(node_name)
	if err != nil {
		logrus.Error(err)
		return 1, nil
	}

	ctx_, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	conn, err := internal.GetAgentConn(nodeInfo.Address)
	if err != nil {
		logrus.Error(err)
		return 2, nil
	}

	cli := containerPB.NewContainerClient(conn)
	containerId, err := GetContainerId(cli, container_name, ctx_)
	if err != nil {
		logrus.Error(err)
		return 3, nil
	}
	containerServer := internal.ContainerServer{}

	// 通过 nodeid 和 容器 ID 获取容器的配置信息
	in := &containerPB.InspectRequest{
		NodeId:      nodeInfo.ID,
		ContainerId: containerId,
	}

	// 获取配置信息
	rep, err := containerServer.Inspect(ctx_, in)
	if err != nil {
		logrus.Error(err)
		return 4, nil
	}

	//logrus.Error(rep.Configs)
	if !args.Float64Resource.Check(containerLimit.DiskRbw, containerLimit.DiskWbw) {
		logrus.Error("disk_rbw or disk_wbw arg error")
		return 6, nil
	}
	if !args.Int32Resource.Check(containerLimit.DiskRio, containerLimit.DiskWio) {
		logrus.Error("disk_rio or disk_wio arg error")
		return 6, nil
	}
	envs := rep.Configs.Envs
	envs["disk_rbw"] = fmt.Sprintf("%v", containerLimit.DiskRbw)
	envs["disk_wbw"] = fmt.Sprintf("%v", containerLimit.DiskWbw)
	envs["disk_rio"] = fmt.Sprintf("%v", containerLimit.DiskRio)
	envs["disk_wio"] = fmt.Sprintf("%v", containerLimit.DiskWio)

	// 更新配置的请求数据
	up := &containerPB.UpdateRequest{
		NodeId:      nodeInfo.ID,
		ContainerId: containerId,
		ResourceLimit: &containerPB.ResourceLimit{
			CpuLimit:        containerLimit.CPU,
			CpuPrio:         int64(containerLimit.CPUPrio),
			MemoryLimit:     float64(containerLimit.MemHard),
			MemorySoftLimit: float64(containerLimit.MemSoft),
			DiskPrio:        int64(containerLimit.DiskPrio),
		},
		RestartPolicy:  rep.Configs.RestartPolicy,
		Networks:       rep.Configs.Networks,
		SecurityConfig: rep.Configs.SecurityConfig,
		Mounts:         rep.Configs.Mounts,
		Envs:           envs,
		Cmd:            rep.Configs.Cmd,
	}

	logrus.Error(up)

	// upRsp, err := cli.Update(ctx_, up)
	// 更新配置, 容器运行中，更新配置会失败
	_, err = containerServer.Update(ctx_, up)
	if err != nil {
		logrus.Error(err)
		return 5, nil
	}

	return 0, nil
}

func (h *KsScmcHandler) SetContainerPrivileged(ctx context.Context, node_ip string, container_name string, isPrivileged bool) (int32, error) {
	return 0, nil
}

func (h *KsScmcHandler) SetContainerSysctls(ctx context.Context, node_ip string, container_name string, sysctl_name string, sysctl_value string) (int32, error) {
	return 0, nil
}

func (h *KsScmcHandler) SetContainerExtraHosts(ctx context.Context, node_ip string, container_name string, container_extrahosts []string) (int32, error) {
	nodeInfo, err := GetNodeId(node_ip)
	if err != nil {
		logrus.Error(err)
		return 1, nil
	}
	conn, err := internal.GetAgentConn(nodeInfo.Address)
	if err != nil {
		logrus.Error(err)
		return 2, nil
	}

	cli := containerPB.NewContainerClient(conn)

	ctx_, cancel := context.WithCancel(context.Background())
	defer cancel()
	containerId, err := GetContainerId(cli, container_name, ctx_)
	if err != nil {
		logrus.Error(err)
		return 3, nil
	}
	hostsRequest := &containerPB.SetHostsRequest{
		NodeId:              nodeInfo.ID,
		ContainerId:         containerId,
		ContainerExtrahosts: container_extrahosts,
	}
	_, err = cli.SetContainerExtraHosts(ctx_, hostsRequest)
	if err != nil {
		logrus.Error(err)
		return 10, nil
	}
	return 0, nil
}

func (h *KsScmcHandler) GetImagesList(ctx context.Context, node_ip string) (*ksscmc.GetImagesListResponse, error) {
	imageListResponse := &ksscmc.GetImagesListResponse{
		ImageList: []string{},
		RetCode:   1,
	}
	nodeInfo, err := GetNodeId(node_ip)
	if err != nil {
		logrus.Error(err)
		return imageListResponse, nil
	}
	conn, err := internal.GetAgentConn(nodeInfo.Address)
	if err != nil {
		logrus.Error(err)
		return imageListResponse, nil
	}
	imageListRequest := &imagePB.ListRequest{
		NodeId: nodeInfo.ID,
	}
	cli := imagePB.NewImageClient(conn)
	resp, err := cli.List(ctx, imageListRequest)
	if err != nil {
		logrus.Infof("get image list error: %v", err)
	}
	for _, item := range resp.Images {
		imageListResponse.ImageList = append(imageListResponse.ImageList, item.Name)
	}
	imageListResponse.RetCode = 0

	return imageListResponse, nil
}

func (h *KsScmcHandler) LoadImage(ctx context.Context, node_ip string, tarPath string) (int32, error) {
	nodeInfo, err := GetNodeId(node_ip)
	if err != nil {
		logrus.Error(err)
		return 1, nil
	}
	conn, err := internal.GetAgentConn(nodeInfo.Address)
	if err != nil {
		logrus.Error(err)
		return 2, nil
	}
	cli := containerPB.NewContainerClient(conn)
	loadRequest := &containerPB.LoadRequest{
		TarPath: tarPath,
	}
	stream, err := cli.LoadImage(ctx, loadRequest)
	if err != nil {
		logrus.Error(err)
		return 9, nil
	}
	start := time.Now()
	for {
		resp, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
				break
			}
			logrus.Error(err)
			return 9, nil
		}

		switch update := resp.Update.(type) {
		case *containerPB.LoadProgress_Percent:
			logrus.Infof("[进度] %.1f%% | 已用时间: %v\n",
				update.Percent,
				time.Since(start).Round(time.Second))
		case *containerPB.LoadProgress_Message:
			logrus.Infof("[日志] %s\n", update.Message)

		case *containerPB.LoadProgress_Result:
			if update.Result.Success {
				logrus.Infof("\n✅ 成功: %s\n", update.Result.Detail)
			} else {
				logrus.Infof("\n❌ 失败: %s\n", update.Result.Detail)
			}
		}
	}
	return 0, nil

}

func (h *KsScmcHandler) PauseContainer(ctx context.Context, node_ip string, container_name string) (int32, error) {
	nodeInfo, err := GetNodeId(node_ip)
	if err != nil {
		logrus.Error(err)
		return 1, nil
	}
	conn, err := internal.GetAgentConn(nodeInfo.Address)
	if err != nil {
		logrus.Error(err)
		return 2, nil
	}
	cli := containerPB.NewContainerClient(conn)

	ctx_, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	containerId, err := GetContainerId(cli, container_name, ctx_)
	if err != nil {
		logrus.Error(err)
		return 3, nil
	}
	pauseReply := &containerPB.PauseRequest{
		NodeId:      nodeInfo.ID,
		ContainerId: containerId,
	}
	if _, err = cli.Pause(ctx_, pauseReply); err != nil {
		logrus.Error(err)
		return 4, nil
	}
	return 0, nil
}

func (h *KsScmcHandler) UnpauseContainer(ctx context.Context, node_ip string, container_name string) (int32, error) {
	nodeInfo, err := GetNodeId(node_ip)
	if err != nil {
		logrus.Error(err)
		return 1, nil
	}
	conn, err := internal.GetAgentConn(nodeInfo.Address)
	if err != nil {
		logrus.Error(err)
		return 2, nil
	}
	cli := containerPB.NewContainerClient(conn)

	ctx_, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	containerId, err := GetContainerId(cli, container_name, ctx_)
	if err != nil {
		logrus.Error(err)
		return 3, nil
	}
	pauseReply := &containerPB.UnpauseRequest{
		NodeId:      nodeInfo.ID,
		ContainerId: containerId,
	}
	if _, err = cli.Unpause(ctx_, pauseReply); err != nil {
		logrus.Error(err)
		return 4, nil
	}
	return 0, nil
}
func (h *KsScmcHandler) LoginV3(ctx context.Context, user string, passwd string, node_ip string) (r string, err error) {
	return "", nil
}
func ThriftapiMain() {

	logrus.Infof("thriftapi serv:")
	logrus.Infof("CA: %s", common.Config.ApiTLS.CA)
	logrus.Infof("serverCert: %s", common.Config.ApiTLS.ServerCert)
	logrus.Infof("serverKey: %s", common.Config.ApiTLS.ServerKey)

	cfg, err := common.LoadApiTLSConfig()
	if err != nil {
		logrus.Errorf("loadTLSConfig failed! %v", err)
		log.Fatalf("loadTLSConfig failed! %v", err)
	}

	addr := common.Config.ApiController.Addr()
	logrus.Infof("addr: %s", addr)

	transport, err := thrift.NewTSSLServerSocket(addr, cfg)
	if err != nil {
		panic(err)
	}
	//fmt.Printf("%T\n", transport);

	ksscmchandler := &KsScmcHandler{}
	processor := ksscmc.NewKsScmcClientProcessor(ksscmchandler)

	/*multiplexedProcessor := thrift.NewTMultiplexedProcessor()

	        dshandler := &DockerStatsServer{}
		multiplexedProcessor.RegisterProcessor(
			"DockerStatsData", // 服务名（必须和生成代码中的一致）
			node.NewDockerStatsDataProcessor(dshandler),
		)

	        ksscmchandler := &KsScmcHandler{}
		multiplexedProcessor.RegisterProcessor(
			"KsScmcClient", // 服务名（必须和生成代码中的一致）
			ksscmc.NewKsScmcClientProcessor(ksscmchandler),
		)*/

	transportFactory := thrift.NewTBufferedTransportFactory(8192)
	//也可以选择Framed传输协议，但是必须确保服务端和客户端的传输协议一致
	//transportFactory := thrift.NewTBufferedTransportFactory(8192)
	//framedtransportFactory := thrift.NewTFramedTransportFactory(transportFactory)

	//可以选择任意一种编码协议，但是必须确保服务端和客户端的编码协议一致
	protocolFactory := thrift.NewTBinaryProtocolFactory(true, true) //布尔参数strictRead, strictWrite，读和写时是否加入版本校验。
	//protocolFactory := thrift.NewTCompactProtocolFactory()
	//protocolFactory := thrift.NewTJSONProtocolFactory()

	server := thrift.NewTSimpleServer4(
		// multiplexedProcessor,
		processor,
		transport,
		transportFactory,
		protocolFactory,
	)

	if err := server.Serve(); err != nil {
		panic(err)
	}
}
