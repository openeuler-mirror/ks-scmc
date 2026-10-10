package tasks

import (
	"net"
	"time"

	log "github.com/sirupsen/logrus"

	"scmc/common"
)

type Task interface {
	Interval() time.Duration
	RunAtInit() bool
	Run()
}

func AddTask(ts ...Task) {
	for _, t := range ts {
		go func(t Task) {
			if t.RunAtInit() {
				t.Run()
			}

			ticker := time.NewTicker(t.Interval())
			for range ticker.C {
				t.Run()
			}
		}(t)
	}
}

func isMaster() bool {
	ifaceName, vip := common.Config.Controller.VirtualIf, common.Config.Controller.VirtualIP
	if ifaceName == "" || vip == "" {
		return true
	}

	iface, err := net.InterfaceByName(ifaceName)
	if err != nil {
		log.Warnf("net.InterfaceByName iface=%s err=%v", ifaceName, err)
		return false
	}

	addrs, err := iface.Addrs()
	if err != nil {
		log.Warnf("get iface=%s address list, err=%v", ifaceName, err)
	}

	for _, i := range addrs {
		addr, _, err := net.ParseCIDR(i.String())
		if err != nil {
			log.Warnf("net.ParseCIDR(%s) err=%v", i.String(), err)
			return false
		}
		if addr.String() == vip {
			return true
		}
	}

	return false
}
