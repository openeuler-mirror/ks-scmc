package model

import (
	"context"
	"net"
	"time"

	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"scmc/common"
	"scmc/rpc/pb/authz"
)

func authzClient(addr string) (authz.AuthzClient, error) {
	conn, err := grpc.Dial(
		addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithContextDialer(func(ctx context.Context, addr string) (net.Conn, error) {
			return net.DialTimeout("unix", addr, time.Second)
		}),
	)

	if err != nil {
		log.Warnf("connect authz socket err=%v", err)
		return nil, err
	}

	return authz.NewAuthzClient(conn), nil
}

func updateAuthzConfig(action authz.AUTHZ_ACTION, containerName, containerID string) error {
	cli, err := authzClient(common.Config.Agent.AuthzSock)
	if err != nil {
		return err
	}

	// 容器名 ID ID子串
	ids := []string{
		containerName,
		containerID[:10],
		containerID[:11],
		containerID[:12],
		containerID,
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*1)
	defer cancel()
	_, err = cli.UpdateConfig(ctx, &authz.UpdateConfigRequest{
		Action:       int64(action),
		ContainerIds: ids,
	})
	if err != nil {
		log.Warnf("rpc UpdateConfig action=%v ids=%v err=%v", action, ids, err)
		return err
	}

	return nil
}

func AddSensitiveContainers(containerName, containerID string) error {
	return updateAuthzConfig(authz.AUTHZ_ACTION_ADD_SENSITIVE_CONTAINER, containerName, containerID)
}

func DelSensitiveContainers(containerName, containerID string) error {
	return updateAuthzConfig(authz.AUTHZ_ACTION_DEL_SENSITIVE_CONTAINER, containerName, containerID)
}
