package docker

import (
        "context"
        "encoding/json"
        "fmt"
        "math"
        //"net/http"
        "strings"
	log "github.com/sirupsen/logrus"
        "github.com/docker/docker/api/types"
        //"github.com/docker/docker/client"
)

// ResponseJSON 定义响应的 JSON 结构
type ResponseJSON struct {
        Status    int            `json:"status"`    // 状态码
        Message   string         `json:"message"`   // 消息
        Containers []ContainerInfo `json:"containers"` // 容器信息列表
}

// ContainerInfo 定义容器的统计信息结构
type ContainerInfo struct {
        Name      string  `json:"name"`       // 容器名称
        Stat      int     `json:"stat"`       // 状态（0 表示正常）
        CPU       float64 `json:"cpu"`        // CPU 使用量
        CPUUsage  float64 `json:"cpu_usage"`  // CPU 使用率
        MemUsage  uint64  `json:"mem_usage"`  // 内存使用量 (MB)
        MemTotal  uint64  `json:"mem_total"`  // 内存总量 (MB)
        DiskUsage uint64  `json:"disk_usage"` // 磁盘使用量 (MB)
        DiskTotal uint64  `json:"disk_total"` // 磁盘总量 (MB)
        DiskRBW   uint64  `json:"disk_rbw"`   // 磁盘读取带宽 (bytes)
        DiskWBW   uint64  `json:"disk_wbw"`   // 磁盘写入带宽 (bytes)
        NetRBW    uint64  `json:"net_rbw"`    // 网络接收带宽 (bytes)
        NetWBW    uint64  `json:"net_wbw"`    // 网络发送带宽 (bytes)
        Image     string  `json:"image"`      // 镜像名称
        Version   string  `json:"version"`    // 镜像版本号
}

// 处理 /stats 请求
func HandleStatsRequest() (string, error) {
	// 获取 Docker 容器的统计信息
	containerInfos, err := getContainerStats()
	if err != nil {
		// 返回错误响应
		response := ResponseJSON{
			Status:  -1,
			Message: fmt.Sprintf("Error: %v", err),
			Containers: []ContainerInfo{},
		}

		//jsonData, err := json.MarshalIndent(response, "", "  ")
		jsonData, err := json.Marshal(response) // 输出一行，不进行美化处理
		if err != nil {
			log.Warnf("getContainerStats() error! marshaling response to JSON: %v", err)
			return "", nil
		}

		// 打印 JSON 数据
		log.Info(string(jsonData))
		return string(jsonData), nil
	}

	// 构建成功响应
	response := ResponseJSON {
		Status:    0,
		Message:   "success",
		Containers: containerInfos,
	}

	/* 没有容器数据的时候 */
	if len(containerInfos) == 0 {
		response = ResponseJSON {
			Status:    0,
			Message:   "success",
			Containers: []ContainerInfo{},
		}
	}

	log.Info("containers len: ", len(containerInfos))
	log.Info("containerInfos: ", containerInfos)

	// 将响应 JSON 序列化为字符串
	//jsonData, err := json.MarshalIndent(response, "", "  ")
	jsonData, err := json.Marshal(response)
	if err != nil {
		log.Warnf("getContainerStats() success! marshaling response to JSON: %v", err)
		return "", nil
	}

	// 打印 JSON 数据
	log.Info(string(jsonData))
	return string(jsonData), nil
}

// 获取 Docker 容器的统计信息
func getContainerStats() ([]ContainerInfo, error) {
        // 创建 Docker 客户端
        //cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	cli, err := Client()
        if err != nil {
                return nil, fmt.Errorf("failed to create Docker client: %v", err)
        }

        // 获取所有容器
        containers, err := cli.ContainerList(context.Background(), types.ContainerListOptions{All: true, Size: true})
        if err != nil {
                return nil, fmt.Errorf("failed to list containers: %v", err)
        }

        // 用于存储所有容器的统计信息
        var containerInfos []ContainerInfo

        // 遍历容器并获取统计信息
        for _, container := range containers {
                // 获取容器名称（去掉开头的 "/"）
                containerName := strings.TrimPrefix(container.Names[0], "/")
		log.Infof("containerName: %s", containerName)

                // 获取镜像名称和版本号
                imageName, imageVersion := parseImageName(container.Image)

                // 获取容器的统计信息流
                stats, err := cli.ContainerStats(context.Background(), container.ID, false)
                if err != nil {
                        log.Warnf("Error getting stats for container %s: %v\n", container.ID, err)
                        continue
                }
                defer stats.Body.Close()

                // 解析统计信息
                dec := json.NewDecoder(stats.Body)
                var v types.StatsJSON
                if err := dec.Decode(&v); err != nil {
                        log.Warnf("Error decoding stats for container %s: %v\n", container.ID, err)
                        continue
                }

                // 计算 CPU 使用率
                cpuUsage := calculateCPUPercent(&v)
		if math.IsNaN(cpuUsage) {
			cpuUsage = 0.00
		}
                cpuUsage = roundToTwoDecimalPlaces(cpuUsage) // 保留两位小数

                // 计算内存使用情况
                memoryUsage := v.MemoryStats.Usage / 1024 / 1024 // 转换为 MB
                memoryLimit := v.MemoryStats.Limit / 1024 / 1024 // 转换为 MB
                memoryPercent := calculateMemoryPercent(&v)
                memoryPercent = roundToTwoDecimalPlaces(memoryPercent) // 保留两位小数

                // 计算磁盘 I/O
                readBytes, writeBytes := calculateBlockIO(&v.BlkioStats)

                // 计算网络 I/O
                var rxBytes, txBytes uint64
                for _, netStats := range v.Networks {
                        rxBytes += netStats.RxBytes
                        txBytes += netStats.TxBytes
                }

                // 构建容器信息结构
                containerInfo := ContainerInfo{
                        Name:      containerName,
                        Stat:      0, // 假设状态正常
                        CPU:       float64(v.CPUStats.CPUUsage.TotalUsage) / 1e9, // 转换为 CPU 使用量
                        CPUUsage:  cpuUsage,
                        MemUsage:  memoryUsage,
                        MemTotal:  memoryLimit,
                        DiskUsage: 0, // 磁盘使用量需要额外获取
                        DiskTotal: 0, // 磁盘总量需要额外获取
                        DiskRBW:   readBytes,
                        DiskWBW:   writeBytes,
                        NetRBW:    rxBytes,
                        NetWBW:    txBytes,
                        Image:     imageName,
                        Version:   imageVersion,
                }

                // 将当前容器的统计信息添加到切片中
                containerInfos = append(containerInfos, containerInfo)
        }

        return containerInfos, nil
}

// 计算 CPU 使用率
func calculateCPUPercent(v *types.StatsJSON) float64 {
        cpuDelta := float64(v.CPUStats.CPUUsage.TotalUsage) - float64(v.PreCPUStats.CPUUsage.TotalUsage)
        systemDelta := float64(v.CPUStats.SystemUsage) - float64(v.PreCPUStats.SystemUsage)
	if (v.CPUStats.SystemUsage == v.PreCPUStats.SystemUsage) {
		return float64(0.00)
	}
        // cpuPercent1 := (cpuDelta / systemDelta) * float64(len(v.CPUStats.CPUUsage.PercpuUsage)) * 100.0
	cpuPercent := (float64(cpuDelta)/float64(systemDelta)) * float64(v.CPUStats.OnlineCPUs) * 100.0
        return cpuPercent
}

// 计算内存使用比率
func calculateMemoryPercent(v *types.StatsJSON) float64 {
        if v.MemoryStats.Limit == 0 {
                return 0.0 // 避免除零错误
        }
        return float64(v.MemoryStats.Usage) / float64(v.MemoryStats.Limit) * 100.0
}

// 计算磁盘 I/O
func calculateBlockIO(blkioStats *types.BlkioStats) (readBytes uint64, writeBytes uint64) {
        for _, stat := range blkioStats.IoServiceBytesRecursive {
                switch stat.Op {
                case "Read":
                        readBytes += stat.Value
                case "Write":
                        writeBytes += stat.Value
                }
        }
        return
}

// 解析镜像名称和版本号
func parseImageName(imageName string) (string, string) {
        // 镜像名称通常格式为 "image:tag" 或 "registry/image:tag"
        parts := strings.Split(imageName, ":")
        if len(parts) > 1 {
                return parts[0], parts[1] // 返回镜像名称和版本号
        }
        return parts[0], "latest" // 如果没有版本号，默认返回 "latest"
}

// 保留两位小数
func roundToTwoDecimalPlaces(value float64) float64 {
        return math.Round(value*100) / 100
}
