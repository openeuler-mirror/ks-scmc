package tasks

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"scmc/model/db"
	"scmc/model/docker"
	"scmc/model/litedb"
	"scmc/model/warnlog"
	"scmc/rpc/pb/container"
	"scmc/rpc/pb/logging"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	log "github.com/sirupsen/logrus"
)

// BlkioHistory 存储容器上一次Blkio统计数据，用于计算速率（字节数+时间戳）
type BlkioHistory struct {
	LastReadBytes  uint64    // 上一次读字节总数
	LastWriteBytes uint64    // 上一次写字节总数
	LastTime       time.Time // 上一次统计时间
}

type NetworkLimit struct {
	RxLimit float64 // 接收带宽限制 (MB/s)
	TxLimit float64 // 发送带宽限制 (MB/s)
}

type NetworkHistory struct {
	LastRxBytes uint64    // 上一次接收字节总数
	LastTxBytes uint64    // 上一次发送字节总数
	LastTime    time.Time // 上一次统计时间
}

type ContainerNetLimitMap map[string]struct {
	RxLimit float64
	TxLimit float64
}

type ContainerResourceMonitor struct {
	mu            sync.Map // 节点监控锁
	cli           *client.Client
	lastAlertTime sync.Map // 告警时间锁
	blkioHistory  sync.Map // Blkio历史数据
	netHistory    sync.Map // 网络历史数据
}

func (c *ContainerResourceMonitor) Interval() time.Duration {
	return 10 * time.Second
}

func (c *ContainerResourceMonitor) RunAtInit() bool {
	return true
}

func (c *ContainerResourceMonitor) Run() {
	log.Debugf("ContainerResourceMonitor Run start")

	if c.cli == nil {
		cli, err := docker.Client()
		if err != nil {
			log.Errorf("Failed to create Docker client: %v", err)
			return
		}
		c.cli = cli
	}

	nodes, err := db.ListNodes()
	if err != nil {
		log.Errorf("Failed to list nodes: %v", err)
		return
	}
	for _, node := range nodes {
		go c.monitorNodeResources(node)
	}
}

func (c *ContainerResourceMonitor) monitorNodeResources(node *db.NodeInfo) {
	nodeAddr := node.Address
	log.Debugf("Monitor container resources on node: %s（%s）", node.Name, nodeAddr)

	if _, loaded := c.mu.LoadOrStore(nodeAddr, struct{}{}); loaded {
		log.Debugf("Node %s is already monitored", nodeAddr)
		return
	}
	defer c.mu.Delete(nodeAddr)

	containers, err := c.cli.ContainerList(context.Background(), types.ContainerListOptions{All: false})
	if err != nil {
		log.Errorf("Failed to list containers on node %s: %v", nodeAddr, err)
		return
	}

	runningContainerIDs := make(map[string]bool)
	for _, cnt := range containers {
		runningContainerIDs[cnt.ID] = true
	}

	for _, container := range containers {
		c.checkContainerResourceUsage(container, node)
	}

	c.blkioHistory.Range(func(key, value interface{}) bool {
		cntID := key.(string)
		if !runningContainerIDs[cntID] {
			c.blkioHistory.Delete(cntID)
			log.Debugf("Clean Blkio history for stopped container: %s", cntID[:12])
		}
		return true
	})

	c.netHistory.Range(func(key, value interface{}) bool {
		keyStr := key.(string)
		parts := strings.Split(keyStr, ":")
		if len(parts) != 2 {
			c.netHistory.Delete(keyStr)
			return true
		}

		cntID := parts[0]
		if !runningContainerIDs[cntID] {
			c.netHistory.Delete(cntID)
			log.Debugf("Clean Network history for stopped container %s, interface %s", cntID[:12], parts[1])
		}
		return true
	})
}

func (c *ContainerResourceMonitor) checkContainerResourceUsage(container types.Container, node *db.NodeInfo) {
	cntID := container.ID
	cntName := strings.TrimPrefix(container.Names[0], "/")
	log.Debugf("Check resource for container: %s（%s）", cntName, cntID[:12])

	stats, err := c.cli.ContainerStats(context.Background(), cntID, false)
	if err != nil {
		log.Warnf("Failed to get stats for container %s: %v", cntID[:12], err)
		return
	}
	defer stats.Body.Close()

	var v types.StatsJSON
	if err := json.NewDecoder(stats.Body).Decode(&v); err != nil {
		log.Warnf("Failed to decode stats for container %s: %v", cntID[:12], err)
		return
	}

	resourceLimit, err := c.getContainerResourceLimit(cntID)
	if err != nil {
		log.Warnf("Failed to get resource limit for container %s: %v", cntID[:12], err)
		return
	}

	cpuUsage := c.calculateCPUUsage(&v, resourceLimit)
	memoryUsage := c.calculateMemoryUsage(&v, resourceLimit)
	diskSpaceUsage := c.calculateDiskSpaceUsage(cntID, resourceLimit)

	readRate, writeRate := c.calculateDiskIORates(cntID, &v)
	readUsage := c.calculateDiskIOMonitorUsage(readRate, resourceLimit.DiskRbw)
	writeUsage := c.calculateDiskIOMonitorUsage(writeRate, resourceLimit.DiskWbw)

	c.checkAndAlert(cpuUsage, "cpu", "", container, node, resourceLimit, nil, "CPU使用率达到90%阈值")
	c.checkAndAlert(memoryUsage, "memory", "", container, node, resourceLimit, nil, "内存使用率达到90%阈值")
	c.checkAndAlert(diskSpaceUsage, "disk", "", container, node, resourceLimit, nil, "磁盘空间使用率达到90%阈值")
	if readRate > 0 {
		c.checkAndAlert(readUsage, "diskRead", "", container, node, resourceLimit, nil, "磁盘读速率达到90%阈值")
	}
	if writeRate > 0 {
		c.checkAndAlert(writeUsage, "diskWrite", "", container, node, resourceLimit, nil, "磁盘写速率达到90%阈值")
	}

	netLimitMap, err := c.getContainerResourceLimitNic(cntID)
	if err != nil {
		log.Warnf("Failed to get resource limit for container %s:%v", cntID[:12], err)
		return
	}

	if len(netLimitMap) == 0 {
		log.Debugf("Container %s has no network limit config, skip network rate calculation", cntID[:12])
		return
	}

	netRates := c.calculateNetworkRates(cntID, &v)
	for ifaceName, rates := range netRates {
		netLimit, hasLimit := netLimitMap[ifaceName]
		if !hasLimit {
			log.Debugf("Container %s interface %s: no network limit config, skip", cntID[:12], ifaceName)
			continue
		}

		rxUsage := c.calculateNetworkMonitorUsage(rates.RxRate, netLimit.RxLimit)
		txUsage := c.calculateNetworkMonitorUsage(rates.TxRate, netLimit.TxLimit)
		if rates.RxRate > 0 && netLimit.RxLimit > 0 {
			c.checkAndAlert(rxUsage, "netRx", ifaceName, container, node, resourceLimit, &netLimit, fmt.Sprintf("网卡[%s]接收速率达到90%%阈值", ifaceName))
		}
		if rates.TxRate > 0 && netLimit.TxLimit > 0 {
			c.checkAndAlert(txUsage, "netTx", ifaceName, container, node, resourceLimit, &netLimit, fmt.Sprintf("网卡[%s]发送速率达到90%%阈值", ifaceName))
		}
	}
}

func (c *ContainerResourceMonitor) calculateNetworkMonitorUsage(netRate float64, netLimit float64) float64 {
	if netLimit <= 0 || netRate <= 0 {
		return 0.0
	}
	usage := (netRate / float64(netLimit)) * 100.0
	log.Debugf("Container Network usage: %.2f%% (rate: %.2f MB/s, limit: %.2f MB/s)", usage, netRate, netLimit)

	return usage
}

// 计算容器的网卡速率并且返回
func (c *ContainerResourceMonitor) calculateNetworkRates(containerID string, v *types.StatsJSON) map[string]struct {
	RxRate float64
	TxRate float64
} {
	netRates := make(map[string]struct {
		RxRate float64
		TxRate float64
	})
	currentTime := time.Now()

	// 遍历容器所有网卡，逐个计算速率
	for ifaceName, netStats := range v.Networks {
		currentRx := netStats.RxBytes
		currentTx := netStats.TxBytes

		// 构建网卡维度的历史key: 容器ID:网卡名
		historyKey := fmt.Sprintf("%s:%s", containerID, ifaceName)
		historyVal, exists := c.netHistory.Load(historyKey)
		if !exists {
			c.netHistory.Store(historyKey, &NetworkHistory{
				LastRxBytes: currentRx,
				LastTxBytes: currentTx,
				LastTime:    currentTime,
			})
			log.Debugf("Init Network history for container %s interface %s", containerID[:12], ifaceName)
			netRates[ifaceName] = struct {
				RxRate float64
				TxRate float64
			}{0, 0}
			continue
		}

		// 读取历史数据计算速率
		history := historyVal.(*NetworkHistory)
		lastRx, lastTx := history.LastRxBytes, history.LastTxBytes
		lastTime := history.LastTime

		timeDelta := currentTime.Sub(lastTime).Seconds()
		if timeDelta <= 0.1 { // 时间差过小，跳过计算
			log.Debugf("Time delta too small(%.2fs) for container %s interface %s, skip network rate", timeDelta, containerID[:12], ifaceName)
			netRates[ifaceName] = struct {
				RxRate float64
				TxRate float64
			}{0, 0}
			continue
		}

		// 处理字节数重置(容器/网卡重启导致的负数差值)
		rxDelta := currentRx - lastRx
		if rxDelta < 0 {
			log.Warnf("Negative Rx delta for container %s interface %s (current: %d, last: %d)", containerID[:12], ifaceName, currentRx, lastRx)
			rxDelta = 0
		}
		txDelta := currentTx - lastTx
		if txDelta < 0 {
			log.Warnf("Negative Tx delta for container %s interface %s (current: %d, last: %d)", containerID[:12], ifaceName, currentTx, lastTx)
			txDelta = 0
		}
		// 转换为 MB/s
		rxRate := (float64(rxDelta) / timeDelta) / (1024 * 1024)
		txRate := (float64(txDelta) / timeDelta) / (1024 * 1024)

		// 更新当前网卡的历史数据
		c.netHistory.Store(historyKey, &NetworkHistory{
			LastRxBytes: currentRx,
			LastTxBytes: currentTx,
			LastTime:    currentTime,
		})

		// 存储当前网卡速率
		netRates[ifaceName] = struct {
			RxRate float64
			TxRate float64
		}{rxRate, txRate}

		log.Debugf("Container %s interface %s Network rate: rx=%.2f MB/s, tx=%.2f MB/s", containerID[:12], ifaceName, rxRate, txRate)
	}

	return netRates
}

// 获取网卡的限速信息
func (c *ContainerResourceMonitor) getContainerResourceLimitNic(containerID string) (ContainerNetLimitMap, error) {
	netLimitMap := make(ContainerNetLimitMap)

	netLimitList, err := litedb.ListNetworkLimit(containerID)
	if err != nil {
		log.Warnf("Get network limits for container %s err: %v", containerID[:12], err)
		return netLimitMap, nil
	}

	if len(netLimitList) == 0 {
		log.Debugf("Container %s has no network limit config", containerID[:12])
		return netLimitMap, nil
	}

	for _, netLimit := range netLimitList {
		if netLimit.ContainerInterfaceName == "" {
			log.Warnf("Container %s network limit has empty container interface name", containerID[:12])
			continue
		}

		netLimitMap[netLimit.ContainerInterfaceName] = struct {
			RxLimit float64
			TxLimit float64
		}{
			RxLimit: netLimit.NetRbw, // 下行(接收):MB/s
			TxLimit: netLimit.NetWbw, // 上行(发送):MB/s
		}

		log.Debugf("Container %s interface %s limit: rx=%.2f MB/s (%.2f MB), tx=%.2f MB/s (%.2f MB)",
			containerID[:12], netLimit.ContainerInterfaceName,
			netLimitMap[netLimit.ContainerInterfaceName].RxLimit, netLimit.NetRbw,
			netLimitMap[netLimit.ContainerInterfaceName].TxLimit, netLimit.NetWbw)
	}

	return netLimitMap, nil
}

func (c *ContainerResourceMonitor) getContainerResourceLimit(containerID string) (*container.ResourceLimit, error) {
	inspect, err := c.cli.ContainerInspect(context.Background(), containerID)
	if err != nil {
		return nil, fmt.Errorf("inspect container failed: %v", err)
	}

	limit := &container.ResourceLimit{}

	if inspect.HostConfig.Resources.NanoCPUs > 0 {
		limit.CpuLimit = float64(inspect.HostConfig.Resources.NanoCPUs) / 1e9
		log.Debugf("Container %s CPU limit: %.2f cores", containerID[:12], limit.CpuLimit)
	}

	if inspect.HostConfig.Resources.Memory > 0 {
		limit.MemoryLimit = float64(inspect.HostConfig.Resources.Memory) / (1024 * 1024)
		log.Debugf("Container %s Memory limit: %.0f MB", containerID[:12], limit.MemoryLimit)
	}

	if inspect.HostConfig.StorageOpt != nil {
		if sizeStr, ok := inspect.HostConfig.StorageOpt["size"]; ok {
			var sizeMB float64
			if _, err := fmt.Sscanf(sizeStr, "%fM", &sizeMB); err == nil {
				limit.DiskLimit = sizeMB
				log.Debugf("Container %s Disk space limit: %.0f MB", containerID[:12], sizeMB)
			}
		}
	}

	cntLimit, err := litedb.GetContainerOutConfig(containerID)
	if err == nil && cntLimit != nil {
		limit.DiskRbw = cntLimit.Config.DiskRbw
		limit.DiskWbw = cntLimit.Config.DiskWbw
		log.Debugf("Container %s Disk IO limit: read=%.0f MB/s, write=%.0f MB/s",
			containerID[:12], float64(limit.DiskRbw), float64(limit.DiskWbw))
	}

	return limit, nil
}

func (c *ContainerResourceMonitor) calculateCPUUsage(v *types.StatsJSON, limit *container.ResourceLimit) float64 {
	if limit.CpuLimit <= 0 || v.CPUStats.OnlineCPUs == 0 {
		log.Debugf("Container has no valid CPU limit, skip CPU calculation")
		return 0.0
	}

	cpuDelta := float64(v.CPUStats.CPUUsage.TotalUsage) - float64(v.PreCPUStats.CPUUsage.TotalUsage)
	systemDelta := float64(v.CPUStats.SystemUsage) - float64(v.PreCPUStats.SystemUsage)
	if systemDelta == 0 {
		return 0.0
	}

	hostUsage := (cpuDelta / systemDelta) * float64(v.CPUStats.OnlineCPUs) * 100.0
	cntUsage := (hostUsage / 100.0) * (float64(v.CPUStats.OnlineCPUs) / limit.CpuLimit) * 100.0
	log.Debugf("Container CPU usage: %.2f%%（limit: %.2f cores）", cntUsage, limit.CpuLimit)
	return cntUsage
}

func (c *ContainerResourceMonitor) calculateMemoryUsage(v *types.StatsJSON, limit *container.ResourceLimit) float64 {
	if limit.MemoryLimit <= 0 || v.MemoryStats.Limit == 0 {
		log.Debugf("Container has no valid Memory limit, skip Memory calculation")
		return 0.0
	}

	used := v.MemoryStats.Usage
	if cache, ok := v.MemoryStats.Stats["cache"]; ok {
		used -= cache
	}
	limitBytes := limit.MemoryLimit * 1024 * 1024
	usage := (float64(used) / limitBytes) * 100.0
	log.Debugf("Container Memory usage: %.2f%%（used: %.0f MB, limit: %.0f MB）",
		usage, float64(used)/(1024*1024), limit.MemoryLimit)
	return usage
}

func (c *ContainerResourceMonitor) calculateDiskSpaceUsage(containerID string, limit *container.ResourceLimit) float64 {
	if limit.DiskLimit <= 0 {
		log.Debugf("Container has no valid Disk space limit, skip Disk calculation")
		return 0.0
	}

	inspect, err := c.cli.ContainerInspect(context.Background(), containerID)
	if err != nil || inspect.GraphDriver.Data["MergedDir"] == "" {
		log.Warnf("Failed to get container %s storage dir: %v", containerID[:12], err)
		return 0.0
	}
	mergedDir := inspect.GraphDriver.Data["MergedDir"]

	var dirSize int64
	_ = filepath.Walk(mergedDir, func(path string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			dirSize += info.Size()
		}
		return nil
	})

	usedMB := float64(dirSize) / (1024 * 1024)
	usage := (usedMB / limit.DiskLimit) * 100.0
	log.Debugf("Container Disk space usage: %.2f%%（used: %.2f MB, limit: %.0f MB）",
		usage, usedMB, limit.DiskLimit)
	return usage
}

func (c *ContainerResourceMonitor) sumBlkioBytes(ioList []types.BlkioStatEntry, op string) uint64 {
	var total uint64
	for _, entry := range ioList {
		if strings.ToLower(entry.Op) == op {
			total += entry.Value
		}
	}
	return total
}

func (c *ContainerResourceMonitor) calculateDiskIORates(containerID string, v *types.StatsJSON) (float64, float64) {
	currentRead := c.sumBlkioBytes(v.BlkioStats.IoServiceBytesRecursive, "read")
	currentWrite := c.sumBlkioBytes(v.BlkioStats.IoServiceBytesRecursive, "write")
	currentTime := time.Now()

	historyVal, exists := c.blkioHistory.Load(containerID)
	if !exists {
		c.blkioHistory.Store(containerID, &BlkioHistory{
			LastReadBytes:  currentRead,
			LastWriteBytes: currentWrite,
			LastTime:       currentTime,
		})
		log.Debugf("Init Blkio history for container %s", containerID[:12])
		return 0.0, 0.0
	}
	history := historyVal.(*BlkioHistory)
	lastRead, lastWrite := history.LastReadBytes, history.LastWriteBytes
	lastTime := history.LastTime

	timeDelta := currentTime.Sub(lastTime).Seconds()
	if timeDelta <= 0.1 {
		log.Debugf("Time delta too small（%.2fs）for container %s, skip IO rate", timeDelta, containerID[:12])
		return 0.0, 0.0
	}

	readDelta := currentRead - lastRead
	if readDelta < 0 {
		log.Warnf("Negative read delta for container %s（current: %d, last: %d）", containerID[:12], currentRead, lastRead)
		readDelta = 0
	}
	writeDelta := currentWrite - lastWrite
	if writeDelta < 0 {
		log.Warnf("Negative write delta for container %s（current: %d, last: %d）", containerID[:12], currentWrite, lastWrite)
		writeDelta = 0
	}

	readRate := (float64(readDelta) / timeDelta) / (1024 * 1024)
	writeRate := (float64(writeDelta) / timeDelta) / (1024 * 1024)

	c.blkioHistory.Store(containerID, &BlkioHistory{
		LastReadBytes:  currentRead,
		LastWriteBytes: currentWrite,
		LastTime:       currentTime,
	})

	log.Debugf("Container %s IO rate: read=%.2f MB/s, write=%.2f MB/s", containerID[:12], readRate, writeRate)
	return readRate, writeRate
}

func (c *ContainerResourceMonitor) calculateDiskIOMonitorUsage(ioRate float64, ioLimit int64) float64 {
	if ioLimit <= 0 || ioRate <= 0 {
		return 0.0
	}
	usage := (ioRate / float64(ioLimit)) * 100.0
	log.Debugf("Container IO usage: %.2f%%（rate: %.2f MB/s, limit: %d MB/s）", usage, ioRate, ioLimit)
	return usage
}

func (c *ContainerResourceMonitor) checkAndAlert(usage float64, resourceType string, ifaceName string, container types.Container, node *db.NodeInfo, limit *container.ResourceLimit, netLimit *struct{ RxLimit, TxLimit float64 }, alertMsg string) {
	hasValidLimit := false
	// var limitValue float64
	switch resourceType {
	case "cpu":
		hasValidLimit = limit.CpuLimit > 0
		// limitValue = limit.CpuLimit
	case "memory":
		hasValidLimit = limit.MemoryLimit > 0
		// limitValue = limit.MemoryLimit
	case "disk":
		hasValidLimit = limit.DiskLimit > 0
		// limitValue = limit.DiskLimit
	case "diskRead":
		hasValidLimit = limit.DiskRbw > 0
		// limitValue = float64(limit.DiskRbw)
	case "diskWrite":
		hasValidLimit = limit.DiskWbw > 0
		// limitValue = float64(limit.DiskWbw)
	case "netRx":
		hasValidLimit = (netLimit != nil && netLimit.RxLimit > 0)
		// if hasValidLimit {
		// 	limitValue = netLimit.RxLimit
		// }
	case "netTx":
		hasValidLimit = (netLimit != nil && netLimit.TxLimit > 0)
		// if hasValidLimit {
		// 	limitValue = netLimit.TxLimit
		// }
	}

	if !hasValidLimit || usage <= 0 {
		log.Debugf("Container %s %s: no valid limit or usage, skip alert", container.ID[:12], resourceType)
		return
	}

	thresholdPercent := 90.0
	if usage < thresholdPercent {
		log.Debugf("Container %s %s: usage %.2f%% < %.1f%% threshold, skip alert",
			container.ID[:12], resourceType, usage, thresholdPercent)
		return
	}

	alertKey := fmt.Sprintf("%s-%s-%s", container.ID, node.Address, resourceType)
	if lastTime, exists := c.lastAlertTime.Load(alertKey); exists {
		if time.Since(lastTime.(time.Time)) < 1*time.Minute { // 以前是 10 分钟上报一次，修改为 1 分钟上报一次
			log.Debugf("Alert suppressed for %s on container %s（within 1 min）", resourceType, container.ID[:12])
			return
		}
	}

	cntName := strings.TrimPrefix(container.Names[0], "/")
	// limitStr := c.getResourceLimitStr(resourceType, limit)
	// thresholdStr := c.getThresholdStr(resourceType, limitValue)
	// detail := fmt.Sprintf("容器【%s（ID：%s）】%s：使用率%.2f%% > 95%%阈值（限制：%s，95%%阈值：%s）",
	//		cntName, container.ID[:12], alertMsg, usage, limitStr, thresholdStr)

	detail := fmt.Sprintf("容器%s %s", cntName, alertMsg)

	warnlog.Create([]*db.WarnLog{
		{
			NodeId:        node.ID,
			NodeInfo:      fmt.Sprintf("%s（%s）", node.Name, node.Address),
			EventType:     int64(logging.EVENT_TYPE_TYPE_NONE),
			EventModule:   int64(logging.EVENT_MODULE_CONTAINER),
			ContainerName: cntName,
			Detail:        detail,
		},
	})

	c.lastAlertTime.Store(alertKey, time.Now())
	log.Infof("Resource alert triggered: %s", detail)
}

func (c *ContainerResourceMonitor) getResourceLimitStr(resourceType string, limit *container.ResourceLimit) string {
	switch resourceType {
	case "cpu":
		return fmt.Sprintf("%.2f 核心", limit.CpuLimit)
	case "memory":
		return fmt.Sprintf("%.0f MB", limit.MemoryLimit)
	case "disk":
		return fmt.Sprintf("%.0f MB", limit.DiskLimit)
	case "diskRead":
		return fmt.Sprintf("%d MB/s", limit.DiskRbw)
	case "diskWrite":
		return fmt.Sprintf("%d MB/s", limit.DiskWbw)
	default:
		return "无限制"
	}
}

func (c *ContainerResourceMonitor) getThresholdStr(resourceType string, limitValue float64) string {
	threshold := limitValue * 0.90
	switch resourceType {
	case "cpu":
		return fmt.Sprintf("%.2f 核心", threshold)
	case "memory":
		return fmt.Sprintf("%.2f MB", threshold)
	case "disk":
		return fmt.Sprintf("%.2f MB", threshold)
	case "diskRead", "diskWrite":
		return fmt.Sprintf("%.2f MB/s", threshold)
	default:
		return "未知"
	}
}

func (c *ContainerResourceMonitor) fromCPUShares(cpuShares int64) int64 {
	if cpuShares <= 1280 {
		return 1
	} else if cpuShares <= 1536 {
		return 2
	} else if cpuShares <= 2048 {
		return 3
	} else if cpuShares <= 2560 {
		return 4
	} else {
		return 5
	}
}

func (c *ContainerResourceMonitor) fromBlkioWeight(weight uint16) int64 {
	if weight <= 600 {
		return 1
	} else if weight <= 700 {
		return 2
	} else if weight <= 800 {
		return 3
	} else if weight <= 900 {
		return 4
	} else {
		return 5
	}
}
