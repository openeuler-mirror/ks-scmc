// miscellaneous functions and workers
package internal

import (
	"sync"
	"time"

	"github.com/shirou/gopsutil/cpu"
	log "github.com/sirupsen/logrus"
)

var globalCPUUsage cpuUsage

// 第一次调用时将返回0(没有上一次调用数据)
// 后续调用计算与前一次调用结果差值, 实现CPU使用计算
// 返回CPU使用核心数
func getCPUUsage(dur time.Duration) (float64, error) {
	cpus, err := cpu.Percent(dur, true)
	if err != nil {
		return 0, err
	}

	sum := 0.0
	for _, v := range cpus {
		sum += v
	}
	return sum / 100.0, nil
}

type cpuUsage struct {
	sync.RWMutex

	usedCores float64
	updatedAt time.Time
}

func (c *cpuUsage) update() error {
	v, err := getCPUUsage(0)
	if err != nil {
		return err
	}

	c.Lock()
	defer c.Unlock()

	c.usedCores = v
	c.updatedAt = time.Now()
	return nil
}

func (c *cpuUsage) get() (float64, error) {
	c.RLock()
	usedCores, updatedAt := c.usedCores, c.updatedAt
	c.RUnlock()

	if time.Since(updatedAt) <= time.Second {
		return usedCores, nil
	}

	log.Debugf("cpu usage data out-date, refresh")
	return getCPUUsage(time.Microsecond * 500)
}

// 定期获取主机CPU使用率
// 保证RPC请求主机使用率时返回较新的CPU使用率
func CPUUsageProbe() {
	for {
		if err := globalCPUUsage.update(); err != nil {
			log.Infof("update cpu usage err=%v", err)
		}

		time.Sleep(time.Second)
	}
}
