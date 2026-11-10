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
	"scmc/model/docker"
	containerPB "scmc/rpc/pb/container"
	imagePB "scmc/rpc/pb/image"
	"scmc/rpc/pb/logging"
	networkPB "scmc/rpc/pb/network"
	nodePB "scmc/rpc/pb/node"
	securityPB "scmc/rpc/pb/security"
	"scmc/server/controller/internal"
	"scmc/thrift/gen-go/ksscmc"

	"git.apache.org/thrift.git/lib/go/thrift"
	"github.com/sirupsen/logrus"
)

type KsScmcHandler struct {
}

func (h *KsScmcHandler) InstallContainer(nodeName string, image string, ver string, name string, desc string) (r string, err error) {
	code, err := h.installContainerInternal(nodeName, image, ver, name)
	return Response2String(code), nil
}

func (h *KsScmcHandler) installContainerInternal(nodeNameorip string, imageName string, imageVer string, containerName string) (int32, error) {
	//通过node_nameorip查找node的ID
	nodeInfo, err := GetNodeId(nodeNameorip)
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
		Name:  containerName,
		Image: imageName + ":" + imageVer,
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
	_, err = containerServer.Create(ctx_, createRequest)
	if err != nil {
		logrus.Error(err)
		return 4, nil
	}

	return 0, nil
}

func (h *KsScmcHandler) StartContainerV3(session_id string, cid string) (string, error) {
	nodeIP := db.SessionManagerGetNodeIP(session_id)
	var containerName *string = nil
	containerInfos, err := h.GetAllContainerList(*nodeIP)
	if err != nil {
		return ResponseFailed(err.Error()), nil
	}
	for _, containerInfo := range containerInfos.Containerlist {
		if containerInfo.ContainerID != cid {
			continue
		}
		containerName = &containerInfo.ContainerName
	}
	if containerName == nil {
		logrus.Error("Failed to get containerName from containerID")
		return ResponseFailed("Cannot found Container"), nil
	}
	code, _ := h.startContainerInternal(*nodeIP, *containerName)
	return Response2String(code), nil
}

func (h *KsScmcHandler) StartContainer(nodeName string, name string) (string, error) {
	code, _ := h.startContainerInternal(nodeName, name)
	return Response2String(code), nil
}

func (h *KsScmcHandler) startContainerInternal(nodeNameorip string, containerName string) (int32, error) {

	nodeInfo, err := GetNodeId(nodeNameorip)
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
	containerId, err := GetContainerId(cli, containerName, ctx_)
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

func (h *KsScmcHandler) StopContainer(nodeName string, name string) (string, error) {
	code, _ := h.stopContainerInternal(nodeName, name)
	return Response2String(code), nil
}

func (h *KsScmcHandler) StopContainerV3(session_id string, cid string) (string, error) {
	nodeIP := db.SessionManagerGetNodeIP(session_id)
	var containerName *string = nil
	containerInfos, err := h.GetAllContainerList(*nodeIP)
	if err != nil {
		return ResponseFailed(err.Error()), nil
	}
	for _, containerInfo := range containerInfos.Containerlist {
		if containerInfo.ContainerID != cid {
			continue
		}
		containerName = &containerInfo.ContainerName
	}
	if containerName == nil {
		logrus.Error("Failed to get containerName from containerID")
		return ResponseFailed("Cannot found Container"), nil
	}
	code, _ := h.stopContainerInternal(*nodeIP, *containerName)
	return Response2String(code), nil
}

func (h *KsScmcHandler) stopContainerInternal(nodeNameorip string, containerName string) (int32, error) {

	nodeInfo, err := GetNodeId(nodeNameorip)
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
	containerId, err := GetContainerId(cli, containerName, ctx_)
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

func (h *KsScmcHandler) UninstallContainer(nodeName string, name string) (string, error) {
	code, _ := h.uninstallContainerInternal(nodeName, name)
	return Response2String(code), nil
}

func (h *KsScmcHandler) uninstallContainerInternal(nodeNameorip string, containerName string) (int32, error) {

	nodeInfo, err := GetNodeId(nodeNameorip)
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

	containerId, err := GetContainerId(cli, containerName, ctx_)
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

func (h *KsScmcHandler) GetContainerStatus(nodeNameorip string, containerName string) (int32, error) {

	nodeInfo, err := GetNodeId(nodeNameorip)
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
		if r.Containers[i].Info.Name == containerName {
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

func (h *KsScmcHandler) GetContainerMetrics(nodeNameorip string, containerName string) (*ksscmc.GetContainerMetricsResponse, error) {

	containerMetricResponse := &ksscmc.ContainerMetric{}
	response := &ksscmc.GetContainerMetricsResponse{
		ContainerMetric: containerMetricResponse,
		RetCode:         1, // 成功码
	}

	nodeInfo, err := GetNodeId(nodeNameorip)
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
		if r.Containers[i].Info.Name == containerName {
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

func (h *KsScmcHandler) GetContainerImageVers(nodeNameorip string, name string) (r string, err error) {
	res, _ := h.GetImageVer4Container(nodeNameorip, name)
	retData := make(map[string]interface{})
	retData["status"] = res.RetCode
	retData["message"] = "failed"
	retData["image"] = ""
	retData["c_ver"] = ""
	retData["vers"] = ""
	if res.RetCode == 0 {
		retData["message"] = "success"
		retData["image"] = res.ContainerImage.ImageName
		retData["c_ver"] = res.ContainerImage.InuseVersion
		retData["vers"] = []string{
			res.ContainerImage.InuseVersion,
		}
	}
	jsonData, err := json.Marshal(retData)
	if err != nil {
		logrus.Error(err)
	}
	return string(jsonData), nil
	// return fmt.Sprintf("{\"image_name\":\"%s\",\"inuse_version\":\"%s\"}", res.ContainerImage.ImageName, res.ContainerImage.InuseVersion), nil
}

func (h *KsScmcHandler) GetImageVer4Container(nodeNameorip string, containerName string) (*ksscmc.GetImageVer4ContainerResponse, error) {
	response := &ksscmc.GetImageVer4ContainerResponse{
		ContainerImage: &ksscmc.ContainerImage{},
		RetCode:        1, // 成功码
	}

	nodeInfo, err := GetNodeId(nodeNameorip)
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
		if r.Containers[i].Info.Name == containerName {
			containerInfo = r.Containers[i].Info
		}
	}

	if containerInfo == nil {
		logrus.Errorf("Failed to find container: %s", containerName)
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

func (h *KsScmcHandler) GetContainer(nodeName string, containerName string) (int32, error) {

	nodeInfo, err := GetNodeId(nodeName)
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
		if r.Containers[i].Info.Name == containerName {
			containerInfo = r.Containers[i].Info
		}
	}

	if containerInfo == nil {
		logrus.Error(err)
		return 4, nil
	}

	return 0, nil
}

func (h *KsScmcHandler) UpdateContainer(nodeName string, containerName string, newVer string) (string, error) {
	code, _ := h.updateContainerInternal(nodeName, containerName, newVer)
	return Response2String(code), nil
}

func (h *KsScmcHandler) updateContainerInternal(nodeName string, containerName string, newVer string) (int32, error) {

	nodeInfo, err := GetNodeId(nodeName)
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
		if r.Containers[i].Info.Name == containerName {
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
	inspectReply.Configs.Image = imageArr[0] + ":" + newVer
	createRequest := &containerPB.CreateRequest{
		NodeId:  nodeInfo.ID,
		Configs: inspectReply.Configs,
	}

	containerServer := &internal.ContainerServer{}
	_, err = containerServer.Create(ctx_, createRequest)
	if err != nil {
		logrus.Error(err)
		return 8, nil
	}

	return 0, nil
}

func (h *KsScmcHandler) GetAlarmList(nodeName string) (string, error) {
	alarmList := AlarmResponse{
		Status:  1,
		Message: "failed",
		CAlarms: []CAlarm{},
		HAlarms: []HAlarm{},
	}
	alarmListJson, _ := json.Marshal(alarmList)
	nodeInfo, err := GetNodeId(nodeName)
	if err != nil {
		logrus.Error(err)
		return string(alarmListJson), nil
	}
	//容器告警 未读
	containerAlarm, err := db.GetUnreadWarnLog(50, 0, nodeInfo.ID, 2)
	if err != nil {
		return string(alarmListJson), nil
	}
	var readContainerAlarm, readNodeAlarm []*db.WarnLog
	if len(containerAlarm) < 50 {
		readSize := 50 - len(containerAlarm)
		readContainerAlarm, err = db.GetUnreadWarnLog(readSize, 1, nodeInfo.ID, 2)
		if err != nil {
			return string(alarmListJson), nil
		}
	}
	containerAlarm = append(containerAlarm, readContainerAlarm...)
	//节点告警
	nodeAlarm, err := db.GetUnreadWarnLog(50, 0, nodeInfo.ID, 1)
	if err != nil {
		return string(alarmListJson), nil
	}
	if len(nodeAlarm) < 50 {
		nodeReadSize := 50 - len(nodeAlarm)
		readNodeAlarm, err = db.GetUnreadWarnLog(nodeReadSize, 1, nodeInfo.ID, 1)
		if err != nil {
			return string(alarmListJson), nil
		}
	}
	nodeAlarm = append(nodeAlarm, readNodeAlarm...)
	var hAlarms []HAlarm
	var cAlarms []CAlarm
	for _, item := range containerAlarm {
		t := time.Unix(item.CreatedAt, 0)
		cAlarm := CAlarm{
			ID:      item.ID,
			Status:  boolToInt(item.HaveRead) ^ 1,
			Message: item.Detail,
			CTime:   t.Format("2006-01-02 15:04:05"),
			CName:   item.ContainerName,
		}
		cAlarms = append(cAlarms, cAlarm)
	}
	for _, h := range nodeAlarm {
		t := time.Unix(h.CreatedAt, 0)
		hAlarm := HAlarm{
			ID:      h.ID,
			Status:  boolToInt(h.HaveRead) ^ 1,
			Message: h.Detail,
			CTime:   t.Format("2006-01-02 15:04:05"),
		}
		hAlarms = append(hAlarms, hAlarm)
	}
	alarmList.Status = 0
	alarmList.Message = "success"
	alarmList.HAlarms = hAlarms
	alarmList.CAlarms = cAlarms
	alarmListJson, _ = json.Marshal(alarmList)
	return string(alarmListJson), nil
}

func (h *KsScmcHandler) GetAllNodeList() (*ksscmc.GetAllNodeListResponse, error) {

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

func (h *KsScmcHandler) GetAllContainers(nodeName string) (r string, err error) {
	resp := ksscmc.GetAllContainersResponse{
		Status:     0,
		Message:    "success",
		Containers: []*ksscmc.ContainerInfo{},
	}

	ctx_, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	nodeInfo, err := db.QueryNodeByName(nodeName)
	if err != nil {
		logrus.Error("Failed to get node by name: %s, because %s", nodeName, err.Error())
		resp.Status = 1
		resp.Message = "Failed to get node by name"
		return Struct2JsonString(resp), nil
	}
	conn, err := internal.GetAgentConn(nodeInfo.Address)
	cli := containerPB.NewContainerClient(conn)
	containerListReply, err := cli.List(ctx_, &containerPB.ListRequest{
		ListAll: true,
	})
	if err != nil {
		logrus.Error("Failed to list container, ", err)
		resp.Status = 1
		resp.Message = "Failed to list container"
		return Struct2JsonString(resp), nil
	}
	containerStatsReply, err := cli.GetContainerStats(ctx_, &containerPB.StatsRequest{})
	if err != nil {
		logrus.Error("Failed to get container stats, ", err)
		resp.Status = 1
		resp.Message = "Failed to get container stats"
		return Struct2JsonString(resp), nil
	}
	var containersStatsResponse docker.ResponseJSON
	if err := json.Unmarshal([]byte(containerStatsReply.StatsData), &containersStatsResponse); err != nil {
		logrus.Warnf("Failed to get container stats, because: ", err.Error())
	}
	containersStats := containersStatsResponse.Containers
	for _, containerInfo := range containerListReply.Containers {
		var containerStats *docker.ContainerInfo = nil
		for _, tmpContainerStats := range containersStats {
			if containerInfo.Info.Name != tmpContainerStats.Name {
				continue
			}
			containerStats = &tmpContainerStats
		}
		if containerStats == nil {
			logrus.Warnf("Failed to find container: %s stats", containerInfo.Info.Name)
			containerStats = &docker.ContainerInfo{}
		}
		tmp := ksscmc.ContainerInfo{
			Name:      containerInfo.Info.Name,
			UUID:      containerInfo.Info.Id,
			CPU:       containerStats.CPU,
			CPUUsage:  containerStats.CPUUsage,
			MemUsage:  int32(containerStats.MemUsage),
			MemTotal:  int32(containerStats.MemTotal),
			DiskUsage: containerInfo.Info.ResourceStat.DiskStat.Used,
			DiskTotal: int32(containerStats.DiskTotal),
			DiskRbw:   float64(containerStats.DiskRBW),
			DiskWbw:   float64(containerStats.DiskWBW),
			NetRbw:    float64(containerStats.NetRBW),
			NetWbw:    float64(containerStats.NetWBW),
			Image:     containerStats.Image,
			Version:   containerStats.Version,
			Ha:        "false",
			Level:     0,
		}
		if strings.EqualFold(containerInfo.Info.State, "running") {
			tmp.Stat = int32(ksscmc.ContainerStatus_ONLINE)
		} else {
			tmp.Stat = int32(ksscmc.ContainerStatus_OFFLINE)
		}
		containerInspectReply, err := cli.Inspect(ctx_, &containerPB.InspectRequest{
			NodeId:      containerInfo.NodeId,
			ContainerId: containerInfo.Info.Id,
		})

		for _, containerNetwork := range containerInspectReply.Configs.Networks {
			if err != nil {
				logrus.Warn("Failed to get container: %s network interface", containerInfo.Info.Name)
			}
			mask := net.CIDRMask(int(containerNetwork.IpPrefixLen), 32)
			if mask == nil {
				logrus.Error(fmt.Sprintf("Failed to conv IPPrefixLen: %d to string", int(containerNetwork.IpPrefixLen)))
			}
			tmp.Nets = append(tmp.Nets, &ksscmc.NetworkInterface{
				EthName: containerNetwork.Interface,
				IP:      containerNetwork.IpAddress,
				Mac:     containerNetwork.MacAddress,
				Gateway: containerNetwork.Gateway,
				Mask:    net.IP(mask).String(),
			})
		}
		resp.Containers = append(resp.Containers, &tmp)
	}
	return Struct2JsonString(resp), nil
}

func (h *KsScmcHandler) GetAllContainerList(nodeName string) (*ksscmc.GetAllContainerListResponse, error) {
	ctx_, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	response := &ksscmc.GetAllContainerListResponse{
		Containerlist: []*ksscmc.ContainerMetric{},
		RetCode:       1, // 成功码
	}

	// nodes, err := db.ListNodes()
	node, err := db.QueryNodeByName(nodeName)
	nodes := []*db.NodeInfo{node}
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

			r, err := containerPB.NewContainerClient(conn).List(ctx_, &containerPB.ListRequest{
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
			CPUUsage:        tmpcontainer.Info.ResourceStat.CpuStat.CoreUsed * 100,
			MemUsage:        int32(tmpcontainer.Info.ResourceStat.MemStat.Used),
			DiskUsage:       float64(tmpcontainer.Info.ResourceStat.DiskStat.Used / 1024),
		}
		if tmpcontainer.Info.State == "running" {
			item.ContainerStatus = ksscmc.ContainerStatus_ONLINE
		}
		if tmpcontainer.Info.ResourceStat.NetStat == nil {
			item.NetUpload = 0
			item.NetDownload = 0
		} else {
			item.NetUpload = float64(tmpcontainer.Info.ResourceStat.NetStat.Tx / 1024 / 1024)
			item.NetDownload = float64(tmpcontainer.Info.ResourceStat.NetStat.Rx / 1024 / 1024)
		}

		response.Containerlist = append(response.Containerlist, item)
	}

	logrus.Debugf("get container list: %v", response.Containerlist)
	response.RetCode = 0
	return response, nil
}

func (h *KsScmcHandler) GetAllContainersV2(nid string) (r string, err error) {
	nodeInfo, err := db.QueryNodeByUUID(nid)
	if err != nil {
		resp := &ksscmc.GetAgentNodesResponse{
			Status:  1,
			Message: "Failed to found container",
			Nodes:   []*ksscmc.AgentNode{},
		}
		return Struct2JsonString(resp), nil
	}
	return h.GetAllContainers(nodeInfo.Name)
}

func (h *KsScmcHandler) GetContainerNetwork(nodeNameorip string, containerName string) (r string, err error) {
	res, err := h.getContainerNetworkInternal(nodeNameorip, containerName)
	if res.RetCode != 0 {
		logrus.Error(err)
		return fmt.Sprintf("{\"status\": %d, \"message\": %s}", res.RetCode, "failed"), nil
	}
	retData := make(map[string]interface{})
	retData["status"] = res.RetCode
	retData["message"] = "success"
	retData["ip"] = res.ContainerNetInfo.IP
	retData["mask"] = res.ContainerNetInfo.Mask
	retData["gw"] = res.ContainerNetInfo.Gateway

	jsonData, err := json.Marshal(retData)
	if err != nil {
		logrus.Error(err)
		return Response2String(1), nil
	}

	return string(jsonData), nil
}

func (h *KsScmcHandler) getContainerNetworkInternal(nodeNameorip string, containerName string) (*ksscmc.GetContainerNetworkResponse, error) {

	response := &ksscmc.GetContainerNetworkResponse{
		ContainerNetInfo: &ksscmc.ContainerNetInfo{},
		RetCode:          1, // 成功码
	}

	nodeInfo, err := GetNodeId(nodeNameorip)
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
	containerId, err := GetContainerId(cli, containerName, ctx_)
	if err != nil {
		logrus.Error(err)
		return response, nil
	}

	containerServer := internal.ContainerServer{}
	inspectReq := containerPB.InspectRequest{
		NodeId:      nodeInfo.ID,
		ContainerId: containerId,
	}

	inspect, err := containerServer.Inspect(ctx_, &inspectReq)
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

func (h *KsScmcHandler) GetContainerNetRules(nodeName string, containerName string) (*ksscmc.GetContainerNetRulesResponse, error) {

	response := &ksscmc.GetContainerNetRulesResponse{
		ContainerNetRules: []*ksscmc.ContainerNetRule{},
		RetCode:           1, // 成功码
	}

	nodeInfo, err := GetNodeId(nodeName)
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
	containerId, err := GetContainerId(cli, containerName, ctx_)
	if err != nil {
		logrus.Error(err)
		return response, nil
	}

	containerServer := internal.ContainerServer{}
	inspectReq := containerPB.InspectRequest{
		NodeId:      nodeInfo.ID,
		ContainerId: containerId,
	}

	inspect, err := containerServer.Inspect(ctx_, &inspectReq)
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

func (h *KsScmcHandler) AddContainerIpPort(nodeNameorip string, containerName string, ipAddr string, port int32, net_proto string) (int32, error) {
	nodeInfo, err := GetNodeId(nodeNameorip)
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
	containerId, err := GetContainerId(cli, containerName, ctx_)
	if err != nil {
		logrus.Error(err)
		return 3, nil
	}

	containerServer := internal.ContainerServer{}
	inspectReq := containerPB.InspectRequest{
		NodeId:      nodeInfo.ID,
		ContainerId: containerId,
	}

	inspect, err := containerServer.Inspect(ctx_, &inspectReq)
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
	networkSettings[0].IpAddress = ipAddr
	updateReq.Networks = append(updateReq.Networks, networkSettings...)
	_, err = cli.Update(ctx_, &updateReq)
	if err != nil {
		logrus.Error(err)
		return 5, nil
	}
	return 0, nil
}

func (h *KsScmcHandler) AddContainerReqIpPort(nodeName string, containerName string, ipAddr string, port int32, netProto string) (int32, error) {

	nodeInfo, err := GetNodeId(nodeName)
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
	containerId, err := GetContainerId(cli, containerName, ctx_)
	if err != nil {
		logrus.Error(err)
		return 3, nil
	}
	containerServer := internal.ContainerServer{}
	inspectReq := containerPB.InspectRequest{
		NodeId:      nodeInfo.ID,
		ContainerId: containerId,
	}
	inspect, err := containerServer.Inspect(ctx_, &inspectReq)
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
		if rule.Addr == ipAddr && rule.Port == uint32(port) {
			isExist = true
			for _, prot := range rule.Protocols {
				if prot == netProto {
					isContainProtocol = true
				}
			}
			if !isContainProtocol {
				rule.Protocols = append(rule.Protocols, netProto)
			}
		}
	}
	if !isExist {
		securityCfg.NetworkRule.Rules = append(securityCfg.NetworkRule.Rules, &securityPB.NetworkRule{
			Protocols: []string{netProto},
			Addr:      ipAddr,
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
	_, err = containerServer.Update(ctx_, &updateReq)
	if err != nil {
		logrus.Error(err)
		return 5, nil
	}
	return 0, nil
}

func (h *KsScmcHandler) AddContainerEnv(nodeNameorip string, containerName string, envName string, envValue string) (r string, err error) {
	retCode, err := h.addContainerEnvInternal(nodeNameorip, containerName, envName, envValue)
	if err != nil {
		logrus.Error(err)
		return fmt.Sprintf("{\"status\": %d, \"message\": %s}", retCode, "failed"), nil
	}
	return fmt.Sprintf("{\"status\": %d, \"message\": %s}", retCode, "success"), nil
}

func (h *KsScmcHandler) addContainerEnvInternal(nodeNameorip string, containerName string, envName string, envValue string) (int32, error) {
	nodeInfo, err := GetNodeId(nodeNameorip)
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
	containerId, err := GetContainerId(cli, containerName, ctx_)
	if err != nil {
		logrus.Error(err)
		return 3, nil
	}
	envs := map[string]string{
		envName: envValue,
	}
	containerServer := internal.ContainerServer{}
	inspectReq := containerPB.InspectRequest{
		NodeId:      nodeInfo.ID,
		ContainerId: containerId,
	}
	inspect, err := containerServer.Inspect(ctx_, &inspectReq)
	if err != nil {
		logrus.Error(err)
		return 4, nil
	}
	oldEnvs := inspect.Configs.Envs
	if oldEnvs == nil {
		oldEnvs = map[string]string{}
	}
	for k, v := range envs {
		oldEnvs[k] = v
	}
	updateReq := containerPB.UpdateRequest{
		NodeId:         nodeInfo.ID,
		ContainerId:    containerId,
		ResourceLimit:  inspect.Configs.ResouceLimit,
		RestartPolicy:  inspect.Configs.RestartPolicy,
		Networks:       inspect.Configs.Networks,
		SecurityConfig: inspect.Configs.SecurityConfig,
		Mounts:         inspect.Configs.Mounts,
		Envs:           oldEnvs,
		Cmd:            inspect.Configs.Cmd,
	}
	_, err = cli.Update(ctx_, &updateReq)
	if err != nil {
		logrus.Error(err)
		return 5, nil
	}
	return 0, nil
}

func (h *KsScmcHandler) AddContainerLocalVolume(nodeNameorip string, containerName string, inDir string, outDir string, accessAuth ksscmc.AccessAuth) (int32, error) {

	nodeInfo, err := GetNodeId(nodeNameorip)
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
	containerId, err := GetContainerId(cli, containerName, ctx_)
	if err != nil {
		logrus.Error(err)
		return 3, nil
	}

	containerServer := internal.ContainerServer{}
	inspectReq := containerPB.InspectRequest{
		NodeId:      nodeInfo.ID,
		ContainerId: containerId,
	}

	inspect, err := containerServer.Inspect(ctx_, &inspectReq)
	if err != nil {
		logrus.Error(err)
		return 4, nil
	}

	logrus.Warnf("container inspect: %v", inspect)

	fileInfo, err := os.Stat(outDir)
	if os.IsNotExist(err) {
		err := os.MkdirAll(outDir, 0755)
		if err != nil {
			logrus.Warnf("create dir: %v faied", outDir)
			return 5, nil
		}
	} else if !fileInfo.IsDir() {
		logrus.Warnf("%v exist and is not dir", outDir)
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
		Source:   outDir,
		Target:   inDir,
		ReadOnly: accessAuth == ksscmc.AccessAuth_READ_ONLY,
	})

	updateReq.Mounts = append(updateReq.Mounts, mountSettings...)
	logrus.Warnf("updateReq: %v", updateReq)

	_, err = cli.Update(ctx_, &updateReq)
	if err != nil {
		logrus.Error(err)
		return 7, nil
	}

	return 0, nil
}

func (h *KsScmcHandler) SetContainerDisplayV2(nodeName string, containerName string, enabled bool, display string) (r string, err error) {
	return Response2String(1), nil
}

func (h *KsScmcHandler) SetContainerQuota(nodeName string, containerName string, containerLimit *ksscmc.ContainerLimit) (r string, err error) {
	res, err := h.setContainerQuotaInternal(nodeName, containerName, containerLimit)
	if err != nil {
		logrus.Error(err)
		return fmt.Sprintf("{\"status\": %d, \"message\": %s}", res, "failed"), nil
	}
	return fmt.Sprintf("{\"status\": %d, \"message\": %s}", res, "success"), nil
}

func (h *KsScmcHandler) setContainerQuotaInternal(nodeName string, containerName string, containerLimit *ksscmc.ContainerLimit) (int32, error) {

	nodeInfo, err := GetNodeId(nodeName)
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
	containerId, err := GetContainerId(cli, containerName, ctx_)
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
	//网络限速
	limit := &containerPB.ContainerLimit{
		NetRbw: containerLimit.NetRbw,
		NetWbw: containerLimit.NetWbw,
	}
	quotaRequest := &containerPB.SetQuotaRequest{
		ContainerId:    containerId,
		ContainerLimit: limit,
	}
	_, err = cli.SetContainerQuota(ctx_, quotaRequest)
	if err != nil {
		return 6, nil
	}
	// upRsp, err := cli.Update(ctx_, up)
	// 更新配置, 容器运行中，更新配置会失败
	_, err = containerServer.Update(ctx_, up)
	if err != nil {
		logrus.Error(err)
		return 5, nil
	}

	return 0, nil
}

func (h *KsScmcHandler) SetContainerPrivileged(nodeIp string, containerName string, isPrivileged bool) (int32, error) {
	return 0, nil
}

func (h *KsScmcHandler) SetContainerSysctls(nodeIp string, containerName string, sysctlName string, sysctlValue string) (int32, error) {
	return 0, nil
}

func (h *KsScmcHandler) SetContainerExtraHosts(nodeIp string, containerName string, containerExtrahosts []string) (int32, error) {
	nodeInfo, err := GetNodeId(nodeIp)
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
	containerId, err := GetContainerId(cli, containerName, ctx_)
	if err != nil {
		logrus.Error(err)
		return 3, nil
	}
	hostsRequest := &containerPB.SetHostsRequest{
		NodeId:              nodeInfo.ID,
		ContainerId:         containerId,
		ContainerExtrahosts: containerExtrahosts,
	}
	_, err = cli.SetContainerExtraHosts(ctx_, hostsRequest)
	if err != nil {
		logrus.Error(err)
		return 10, nil
	}
	return 0, nil
}

func (h *KsScmcHandler) GetImagesList(nodeIp string) (*ksscmc.GetImagesListResponse, error) {
	ctx_, _ := context.WithTimeout(context.Background(), time.Second*5)
	imageListResponse := &ksscmc.GetImagesListResponse{
		ImageList: []string{},
		RetCode:   1,
	}
	nodeInfo, err := GetNodeId(nodeIp)
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
	resp, err := cli.List(ctx_, imageListRequest)
	if err != nil {
		logrus.Infof("get image list error: %v", err)
	}
	for _, item := range resp.Images {
		imageListResponse.ImageList = append(imageListResponse.ImageList, item.Name)
	}
	imageListResponse.RetCode = 0

	return imageListResponse, nil
}

func (h *KsScmcHandler) LoadImage(nodeIp string, tarPath string) (int32, error) {
	ctx_, _ := context.WithTimeout(context.Background(), time.Second*5)
	nodeInfo, err := GetNodeId(nodeIp)
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
	stream, err := cli.LoadImage(ctx_, loadRequest)
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

func (h *KsScmcHandler) PauseContainer(nodeIp string, containerName string) (int32, error) {
	nodeInfo, err := GetNodeId(nodeIp)
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
	containerId, err := GetContainerId(cli, containerName, ctx_)
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

func (h *KsScmcHandler) UnpauseContainer(nodeIp string, containerName string) (int32, error) {
	nodeInfo, err := GetNodeId(nodeIp)
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
	containerId, err := GetContainerId(cli, containerName, ctx_)
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

func (h *KsScmcHandler) LoginV3(user string, passwd string, nodeIp string) (r string, err error) {
	respone := ksscmc.CommonRespone{}
	if nodeIp == "" {
		respone.Message = "nodeIp is invalid!"
		respone.Status = 1
		return Struct2JsonString(respone), nil
	}
	sessionID, err := db.SessionManagerCreateSession(nodeIp)
	if err != nil {
		logrus.Error("Failed to create session, because: ", err)
		respone.Message = "Failed to create session"
		respone.Status = 1
		return Struct2JsonString(respone), nil
	}
	respone.Message = *sessionID
	respone.Status = 0
	return Struct2JsonString(respone), nil
}

func (h *KsScmcHandler) GetVersionsV3(session_id string) (string, error) {
	resp := ksscmc.GetVersionsV3Response{}
	nodeIP := db.SessionManagerGetNodeIP(session_id)
	var targetNode *db.NodeInfo = nil
	nodes, err := db.ListNodes()
	if err != nil {
		logrus.Error("Failed to get node list")
		return ResponseFailed("Failed to get node list"), nil
	}
	for _, nodeInfo := range nodes {
		if nodeInfo.Address != *nodeIP {
			continue
		}
		targetNode = nodeInfo
	}
	if targetNode == nil {
		return ResponseFailed("Cannot found node by session"), nil
	}
	resp.Nodes = append(resp.Nodes, &ksscmc.NodeVersion{
		Name:     targetNode.Name,
		UUID:     targetNode.UUID,
		Versions: &ksscmc.Version{},
	})
	return Struct2JsonString(resp), nil
}

func (h *KsScmcHandler) GetAllContainersV3(session_id string, nid string) (r string, err error) {
	nodeIP := db.SessionManagerGetNodeIP(session_id)
	if nodeIP == nil {
		resp := &ksscmc.GetAgentNodesResponse{
			Status:  1,
			Message: "session id is invalid",
			Nodes:   []*ksscmc.AgentNode{},
		}
		return Struct2JsonString(resp), nil
	}
	nodeInfo, err := db.QueryNodeByAddr(*nodeIP)
	return h.GetAllContainers(nodeInfo.Name)
}

func (h *KsScmcHandler) CondGetAlarmListV3(session_id string, cond *ksscmc.GetAlarmCond) (r string, err error) {
	resp := ksscmc.CondGetAlarmListResponse{
		Status:  0,
		Message: "success",
		Total:   0, // todo
		Nums:    0,
		Pages:   0,
		Page:    0, //todo
		HAlarms: []*ksscmc.CondAlarms{},
		CAlarms: []*ksscmc.CondAlarms{},
	}
	nodeID := int64(0)
	nodeInfo, err := db.QueryNodeByName(cond.NodeName)
	readType := ksscmc.ReadType_ALL
	var entityType []int

	if err != nil {
		logrus.Error("Failed to get node by ip", err)
	} else {
		nodeID = nodeInfo.ID
	}
	// thrift 定义中 1：仅获取已读告警；2：仅获取未读告警
	// 但是内部定义 0:未读 1:已读
	if ksscmc.ReadType(cond.ReadType) == ksscmc.ReadType_READED {
		readType = 1
	}
	if ksscmc.ReadType(cond.ReadType) == ksscmc.ReadType_UNREAD {
		readType = 0
	}
	if cond.EntityType == 1 || cond.EntityType == 2 {
		entityType = append(entityType, int(cond.EntityType))
	} else {
		entityType = append(entityType, 1, 2)
	}
	// 没有对 AlarmType 做处理
	if cond.AlarmType != -1 {
		logrus.Warn("Unsupported special alarm type!")
	}
	pager, warnLogs, _ := db.GetWarnLog(int(readType),
		entityType,
		[]int{int(cond.AlarmType)},
		nodeID,
		cond.StartTime,
		cond.EndTime,
		int64(cond.Nums),
		int64(cond.Page),
		cond.Asc)

	for _, warnLog := range warnLogs {
		status := 0
		if warnLog.HaveRead {
			status = 0
		} else {
			status = 1
		}
		alarm := &ksscmc.CondAlarms{
			ID:        int32(warnLog.ID),
			Status:    int32(status),
			Message:   warnLog.Detail,
			AlarmType: int32(warnLog.EventType),
			CTime:     time.Unix(warnLog.CreatedAt, 0).Format("2025-07-25 18:00:00"),
			UTime:     time.Unix(warnLog.UpdatedAt, 0).Format("2025-07-25 18:00:00"),
		}
		if warnLog.EventModule == int64(logging.EVENT_MODULE_NODE) {
			resp.HAlarms = append(resp.HAlarms, alarm)
		}
		if warnLog.EventModule == int64(logging.EVENT_MODULE_CONTAINER) {
			resp.CAlarms = append(resp.CAlarms, alarm)
		}
	}
	resp.Nums = int32(len(warnLogs))
	resp.Page = int32(pager.PageSize)
	resp.Total = int32(pager.TotalRows)
	return Struct2JsonString(resp), nil
}

func (h *KsScmcHandler) GetAgentNodesV3(session_id string) (r string, err error) {
	nodeIP := db.SessionManagerGetNodeIP(session_id)
	resp := &ksscmc.GetAgentNodesResponse{}
	conn, err := internal.GetAgentConn(*nodeIP)
	if err != nil {
		logrus.Error("Failed to get agent connection, because ", err)
		resp.Status = 1
		resp.Message = err.Error()
		return Struct2JsonString(resp), nil
	}
	nodeInfos, err := db.ListNodes()
	if err != nil {
		logrus.Error("Failed to get node infos, because ", err)
		resp.Status = 1
		resp.Message = err.Error()
		return Struct2JsonString(resp), nil
	}
	var nodeInfo *db.NodeInfo
	for _, _nodeInfo := range nodeInfos {
		if _nodeInfo.Address != *nodeIP {
			continue
		}
		nodeInfo = _nodeInfo
	}
	nodeCli := nodePB.NewNodeClient(conn)
	nodeNetworkCli := networkPB.NewNetworkClient(conn)
	ctx_, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	nodeStatusReply, err := nodeCli.Status(ctx_, &nodePB.StatusRequest{})
	if err != nil {
		logrus.Error("Failed to get node status ", err)
		resp.Status = 1
		resp.Message = err.Error()
		return Struct2JsonString(resp), nil
	}
	nodeNetworkReply, err := nodeNetworkCli.List(ctx_, &networkPB.ListRequest{})
	if err != nil {
		logrus.Error("Failed to get node network interface ", err)
		resp.Status = 1
		resp.Message = err.Error()
		return Struct2JsonString(resp), nil
	}
	nodeNetworkInterfaces := append(nodeNetworkReply.RealIfs, nodeNetworkReply.VirtualIfs...)
	nets := []*ksscmc.NetworkInterfaceWithAgentNode{}
	for _, networkInterface := range nodeNetworkInterfaces {
		nets = append(nets, &ksscmc.NetworkInterfaceWithAgentNode{
			EthName: networkInterface.Name,
			IP:      networkInterface.IpAddress,
			Mac:     networkInterface.MacAddress,
			Gateway: networkInterface.Gateway,
			Netmask: networkInterface.IpMask,
		})
	}
	nodeStatu := nodeStatusReply.StatusList[0]
	var nodeState int32 = 1
	if nodeStatu.State == int64(nodePB.NodeState_Online) {
		nodeState = 0
	}
	resp.Status = 0
	resp.Message = "success"
	resp.Nodes = append(resp.Nodes, &ksscmc.AgentNode{
		Nets:                nets,
		Desc:                nodeInfo.Comment,
		Name:                nodeInfo.Name,
		UUID:                nodeInfo.UUID,
		IP:                  nodeInfo.Address,
		Stat:                nodeState,
		CPUUsage:            nodeStatu.CpuStat.UsedPercentage,
		Cpus:                nodeStatu.CpuStat.Total,
		MemUsage:            nodeStatu.MemStat.UsedPercentage,
		MemTotal:            float64(nodeStatu.MemStat.Total),
		DiskUsage:           nodeStatu.DiskStat.UsedPercentage / 1024,
		DiskTotal:           float64(nodeStatu.DiskStat.Total) / 1024,
		DrbdUsage:           0.0, // todo
		DrbdTotal:           0.0, // todo
		NetRbw:              0.0,
		NetWbw:              0.0,
		DiskRbw:             0.0,
		DiskWbw:             0.0,
		DrbdRbw:             0.0,
		DrbdWbw:             0.0,
		ContainerNums:       int32(nodeStatu.ContainerStat.Total),
		OnlineContainerNums: int32(nodeStatu.ContainerStat.Running),
		MaxContainers:       9999,
		AlarmNums:           0, // todo,目前所有获取日志的接口都需要分页，获取日志数量总数待实现。
	})
	return Struct2JsonString(resp), nil
}

func (h *KsScmcHandler) SetAlarmReadV3(session_id string, node_name string, id int32) (r string, err error) {
	if err := db.SetWarnLogRead([]int64{int64(id)}); err != nil {
		return ResponseFailed(err.Error()), nil
	}
	return ResponseSucceed(), nil
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
