package kysec

import (
	// "time"
	// "strconv"
	"fmt"
	"net"
	"strings"

	log "github.com/sirupsen/logrus"
	"scmc/model/db"
	"scmc/model/litedb"
	"scmc/rpc/pb/logging"
)

/*
 * 获取本机所有IP
 */
func getAllIPs() ([]string, error) {
	var ips []string

	ifaces, err := net.Interfaces()
	if err != nil {
		log.Warnf("getAllIPs failed: %v", err)
		return nil, err
	}

	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 {
			continue // 接口未激活
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}

			if ip == nil || ip.IsLoopback() {
				continue
			}

			ip = ip.To4()
			if ip == nil {
				continue // 不是IPv4地址
			}

			ips = append(ips, ip.String())
		}
	}

	if len(ips) == 0 {
		return nil, fmt.Errorf("getAllIPs failed, no valid ip")
	}
	return ips, nil
}

/*
 * 获取本机 nodeid
 */

func getNodeInfos() (int64, string, string, error) {
	ips, err := getAllIPs()
	if err != nil {
		return 0, "", "", err
	}
	nodes, err := db.ListNodes()
	if err != nil {
		return 0, "", "", err
	}

	var nodeid int64 = 0
	var nodename string = ""
	var address string = ""
	for _, n := range nodes {
		for _, ip := range ips {
			if n.Address == ip {
				nodeid = n.ID
				nodename = n.Name
				address = n.Address
			}
		}
	}

	return nodeid, nodename, address, nil
}

var auditLogChan = make(chan *AuditLog, 10)

func FetchKernelLogs() {
	for r := range auditLogChan {
		warnLogs := make([]*db.WarnLog, 0)
		adtLog := r
		if adtLog == nil {
			continue
		}

		detail := ""
		evType := logging.EVENT_TYPE_WARN_FILE_LIST
		if adtLog.Type == 1 {
			evType = logging.EVENT_TYPE_WARN_PROCESS_LIST
			detail = detail + "进程白名单:"
		} else if adtLog.Type == 2 {
			evType = logging.EVENT_TYPE_WARN_FILE_LIST
			detail = detail + "文件保护:"
		} else {
			log.Warnf("FetchKernelLogs log type invalide: %d", adtLog.Type)
			continue
		}

		detail = detail + strings.Trim(string(adtLog.Sub[:]), "\x00")
		if adtLog.Op == 1 {
			detail = detail + " 执行"
		} else if adtLog.Op == 2 {
			detail = detail + " 修改"
		} else {
			log.Warnf("FetchKernelLogs invalide Op value: %d", adtLog.Op)
			continue
		}

		detail = detail + strings.TrimSpace(adtLog.Path) + " 操作结果:"
		if adtLog.Result == 0 {
			detail = detail + "放行"
		} else if adtLog.Result == 1 {
			detail = detail + "拒绝"
		} else {
			log.Warnf("FetchKernelLogs invalide Result value: %d", adtLog.Result)
			continue
		}

		detail = strings.TrimSpace(detail)
		detail = strings.Trim(detail, "\x00")

		containerId := ""
		containerName := ""
		evModule := logging.EVENT_MODULE_NONE
		node_id, node_name, address, err := getNodeInfos()
		if err != nil {
			log.Warnf("FetchKernelLogs get node id failed!")
			continue
		}

		if strings.Trim(string(adtLog.Id[:]), "\x00") == "localhost" {
			evModule = logging.EVENT_MODULE_NONE
		} else {
			evModule = logging.EVENT_MODULE_CONTAINER
			containerId = strings.Trim(string(adtLog.Id[:]), "\x00")
			cfgs, err := litedb.GetContainerConfig(containerId)
			if err == nil {
				containerName = cfgs.ContainerName
			}
		}

		nodeinfo := fmt.Sprintf("%s(%s)", node_name, address)
		warnLogs = append(warnLogs, &db.WarnLog{
			EventType:     int64(evType),
			EventModule:   int64(evModule),
			NodeId:        int64(node_id),
			NodeInfo:      nodeinfo,
			ContainerID:   containerId,
			ContainerName: containerName,
			Detail:        detail,
		})

		err = db.CreateWarnLog(warnLogs)
		if err != nil {
			log.Warnf("create kernel warn log failed!")
			continue
		}
	}
}
