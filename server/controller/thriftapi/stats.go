package thriftapi

import (
	"context"
	"time"

	"scmc/rpc/pb/container"
	"scmc/server/controller/internal"
)

func GetContainerStats(hostid string) (string, error) {
	conn, err := internal.GetAgentConn(hostid)
	if err != nil {
		return "", nil
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
