package internal

import (
	"github.com/docker/docker/api/types"

	pb "ksc-mcube/rpc/pb/container"
)

func calculateCPUPercentUnix(previousCPU, previousSystem uint64, v *types.StatsJSON) float64 {
	var (
		cpuPercent = 0.0
		// calculate the change for the cpu usage of the container in between readings
		cpuDelta = float64(v.CPUStats.CPUUsage.TotalUsage) - float64(previousCPU)
		// calculate the change for the entire system between readings
		systemDelta = float64(v.CPUStats.SystemUsage) - float64(previousSystem)
		onlineCPUs  = float64(v.CPUStats.OnlineCPUs)
	)

	if onlineCPUs == 0.0 {
		onlineCPUs = float64(len(v.CPUStats.CPUUsage.PercpuUsage))
	}
	if systemDelta > 0.0 && cpuDelta > 0.0 {
		cpuPercent = (cpuDelta / systemDelta) * onlineCPUs * 100.0
	}
	return cpuPercent
}

func calculateBlockIO(blkio types.BlkioStats) (uint64, uint64) {
	var blkRead, blkWrite uint64
	for _, bioEntry := range blkio.IoServiceBytesRecursive {
		if len(bioEntry.Op) == 0 {
			continue
		}
		switch bioEntry.Op[0] {
		case 'r', 'R':
			blkRead = blkRead + bioEntry.Value
		case 'w', 'W':
			blkWrite = blkWrite + bioEntry.Value
		}
	}
	return blkRead, blkWrite
}

func calculateMemUsageUnixNoCache(mem types.MemoryStats) uint64 {
	// cgroup v1
	if v, isCgroup1 := mem.Stats["total_inactive_file"]; isCgroup1 && v < mem.Usage {
		return mem.Usage - v
	}
	// cgroup v2
	if v := mem.Stats["inactive_file"]; v < mem.Usage {
		return mem.Usage - v
	}
	return mem.Usage
}

func calculateMemPercentUnixNoCache(limit, usedNoCache float64) float64 {
	// MemoryStats.Limit will never be 0 unless the container is not running and we haven't
	// got any data from cgroup
	if limit != 0 {
		return usedNoCache / limit * 100.0
	}
	return 0
}

func calculateNetwork(network map[string]types.NetworkStats) (uint64, uint64) {
	var rx, tx uint64

	for _, v := range network {
		rx += v.RxBytes
		tx += v.TxBytes
	}
	return rx, tx
}

func calculateContainerStat(v *types.StatsJSON) *pb.ContainerStatus {
	previousCPU := v.PreCPUStats.CPUUsage.TotalUsage
	previousSystem := v.PreCPUStats.SystemUsage

	var r = pb.ContainerStatus{
		CpuStat: &pb.CpuStat{
			Percentage: calculateCPUPercentUnix(previousCPU, previousSystem, v),
		},
		MemStat: &pb.MemoryStat{
			Used:  calculateMemUsageUnixNoCache(v.MemoryStats),
			Limit: v.MemoryStats.Limit,
		},
		BlockStat: &pb.BlockStat{},
		NetStat:   &pb.NetworkStats{},
	}

	r.MemStat.Percentage = calculateMemPercentUnixNoCache(float64(r.MemStat.Limit), float64(r.MemStat.Used))
	r.BlockStat.Read, r.BlockStat.Write = calculateBlockIO(v.BlkioStats)
	r.NetStat.Rx, r.NetStat.Tx = calculateNetwork(v.Networks)

	return &r
}
