package thriftapi

import (
	"context"
	"time"
	"strings"

	"scmc/rpc/pb/container"
	"scmc/server/controller/internal"
	log "github.com/sirupsen/logrus"
)


func GetContainerStats(hostid string) (string, error) {
	conn, err := internal.GetAgentConn(hostid)
	if err != nil {
		return "", err
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second * 5)
	defer cancel()

	cli := container.NewContainerClient(conn)
	reply, err := cli.GetContainerStats(ctx, &container.StatsRequest{ HostId : hostid,})
	if err != nil {
		return "", err
	}
	return reply.StatsData , nil
}

func (h *KsScmcHandler) GetAllContainers(ctx context.Context, node_name string) (r string, err error) {
	log.Infof("GetAllContainers node_name: %s", node_name)

	node_name = strings.TrimSpace(node_name)
	if node_name == "" {
		json_str := "{\"status\":-1,\"message\":\"Invalid parameter: parameter must not be empty!\",\"containers\":[]}"
		return json_str, nil
	}

	json_str, err := GetContainerStats(node_name)
	if err != nil {
		json_str = "{\"status\":-1,\"message\":\"" + err.Error() + "\",\"containers\":[]}"
	} else {
		if (len(json_str) == 0) {
			json_str = "{\"status\":0,\"message\":\"success\",\"containers\":[]}"
		}
	}

	return json_str, nil
}
