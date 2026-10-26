package thriftapi

import (
	"context"
	"errors"
	"scmc/model/db"
	containerPB "scmc/rpc/pb/container"
)

//通过node name or ip查找node info
func GetNodeId(node_nameorip string) (*db.NodeInfo, error) {
	nodeInfo, err := db.QueryNodeByAddr(node_nameorip)
	if err != nil {
		nodeInfo, err = db.QueryNodeByName(node_nameorip)
		if err != nil {
			return nil, err
		}
	}
	return nodeInfo, nil
}
func GetContainerId(cli containerPB.ContainerClient, containerName string, ctx context.Context) (containerId string, err error) {

	r, err := cli.List(ctx, &containerPB.ListRequest{
		ListAll: true,
	})
	if err != nil {
		return "", err
	}

	for i := range r.Containers {
		if r.Containers[i].Info.Name == containerName {
			containerId = r.Containers[i].Info.Id
		}
	}
	if containerId == "" {
		return "", errors.New("containerId not found")
	}
	return containerId, nil
}
