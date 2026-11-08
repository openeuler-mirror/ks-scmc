package common

import (
    "fmt"
    "net"
    "strings"
	log "github.com/sirupsen/logrus"
)

func extractIP(vip string) (string, error) {
	ip, _, err := net.ParseCIDR(vip)
	if err == nil {
		return ip.String(), nil
	}

	parsedIP := net.ParseIP(strings.TrimSpace(vip))
	if parsedIP == nil {
		return "", fmt.Errorf("Invalid IP format: %s", vip)
	}

	return parsedIP.String(), nil
}

func IsMaster() bool {
	ifaceName, vip := Config.Controller.VirtualIf, Config.Controller.VirtualIP
	if ifaceName == "" || vip == "" {
		return true
	}

	vip, err := extractIP(vip)
	if err != nil {
		log.Warnf("extractIP err=%v", err)
		return false
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
