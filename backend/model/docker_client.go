package model

import (
	"context"
	"sync"

	"github.com/docker/docker/client"
	log "github.com/sirupsen/logrus"
)

var (
	dockerCli      *client.Client
	dockerCliGuard sync.Mutex
)

func DockerClient() (*client.Client, error) {
	dockerCliGuard.Lock()
	defer dockerCliGuard.Unlock()

	if dockerCli != nil {
		_, err := dockerCli.Ping(context.Background())
		if err != nil {
			log.Warnf("ping container daemon: %v", err)
		} else {
			return dockerCli, nil
		}
	}

	var err error
	dockerCli, err = client.NewClientWithOpts(
		client.FromEnv,
		client.WithAPIVersionNegotiation(),
		client.WithHTTPHeaders(map[string]string{"Authz-User": "KS-SCMC-SERVICE"}),
	)
	if err != nil {
		log.Warnf("try to connect to container daemon: %v", err)
		return nil, err
	}

	return dockerCli, nil
}
