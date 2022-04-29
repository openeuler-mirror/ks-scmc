// miscellaneous functions and workers
package internal

import (
	"time"

	"github.com/shirou/gopsutil/cpu"
	log "github.com/sirupsen/logrus"
)

// 第一次调用时将返回0(没有上一次调用数据)
// 后续调用计算与前一次调用结果差值, 实现CPU使用计算
// 返回CPU使用核心数
func cpuUsage() (float64, error) {
	cpus, err := cpu.Percent(0, true)
	if err != nil {
		return 0, err
	}

	sum := 0.0
	for _, v := range cpus {
		sum += v
	}
	return sum / 100.0, nil
}

// 定期获取主机CPU使用率
// 保证RPC请求主机使用率时返回较新的CPU使用率
func CPUUsageProbe() {
	for {
		_, err := cpuUsage()
		if err != nil {
			log.Infof("acquire cpu usage err=%v", err)
		}

		time.Sleep(time.Second * 10)
	}
}
