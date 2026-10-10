package docker

import (
	"context"
	"sync"

	"github.com/docker/docker/client"
	log "github.com/sirupsen/logrus"
)

var (
	cli      *client.Client
	cliGuard sync.Mutex
)

func Client() (*client.Client, error) {
	cliGuard.Lock()
	defer cliGuard.Unlock()

	if cli != nil {
		_, err := cli.Ping(context.Background())
		if err != nil {
			log.Warnf("ping container daemon: %v", err)
		} else {
			return cli, nil
		}
	}

	var err error
	cli, err = client.NewClientWithOpts(
		client.FromEnv,
		client.WithAPIVersionNegotiation(),
		client.WithHTTPHeaders(map[string]string{"Authz-User": "KS-SCMC-SERVICE"}),
	)
	if err != nil {
		log.Warnf("try to connect to container daemon: %v", err)
		return nil, err
	}

	return cli, nil
}

func IsErrNotFound(err error) bool {
	return client.IsErrNotFound(err)
}
