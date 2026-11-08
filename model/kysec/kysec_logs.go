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
        warn_logs := make([]*db.WarnLog, 0)
        adt_log := r
        if adt_log == nil {
            continue
        }
        
        detail := ""
        ev_type := logging.EVENT_TYPE_WARN_FILE_LIST
        if adt_log.Type == 1 {
            ev_type = logging.EVENT_TYPE_WARN_PROCESS_LIST
            detail = detail + "进程白名单:"
        } else if adt_log.Type == 2 {
            ev_type = logging.EVENT_TYPE_WARN_FILE_LIST
            detail = detail + "文件保护:"
        } else {
            log.Warnf("FetchKernelLogs log type invalide: %d", adt_log.Type)
            continue
        }

        detail = detail + strings.Trim(string(adt_log.Sub[:]), "\x00")
        if adt_log.Op == 1 {
            detail = detail + " 执行"
        } else if adt_log.Op == 2 {
            detail = detail + " 修改"
        } else {
            log.Warnf("FetchKernelLogs invalide Op value: %d", adt_log.Op)
            continue
        }

        detail = detail + strings.TrimSpace(adt_log.Path) + " 操作结果:"
        if adt_log.Result == 0 {
            detail = detail + "放行"
        } else if adt_log.Result == 1 {
            detail = detail + "拒绝"
        } else {
            log.Warnf("FetchKernelLogs invalide Result value: %d", adt_log.Result)
            continue
        }

        detail = strings.TrimSpace(detail)
        detail = strings.Trim(detail, "\x00")

        container_id := ""
        container_name := ""
        ev_module := logging.EVENT_MODULE_NONE
        node_id, node_name, address, err := getNodeInfos()
        if err != nil {
            log.Warnf("FetchKernelLogs get node id failed!")
            continue
        }

        if strings.Trim(string(adt_log.Id[:]), "\x00") == "localhost" {
            ev_module = logging.EVENT_MODULE_NONE
        } else {
            ev_module = logging.EVENT_MODULE_CONTAINER
            container_id = strings.Trim(string(adt_log.Id[:]), "\x00")
            cfgs, err := litedb.GetContainerConfig(container_id)
            if err == nil {
                container_name = cfgs.ContainerName
            }
        }

        nodeinfo := fmt.Sprintf("%s(%s)", node_name, address)
        warn_logs = append(warn_logs, &db.WarnLog{
            EventType : int64(ev_type),
            EventModule : int64(ev_module),
            NodeId: int64(node_id),
            NodeInfo: nodeinfo,
            ContainerID: container_id,
            ContainerName: container_name,
            Detail: detail,
        })

        err = db.CreateWarnLog(warn_logs)
        if err != nil {
            log.Warnf("create kernel warn log failed!")
            continue
        }
    }
}
