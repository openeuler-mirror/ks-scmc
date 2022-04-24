package internal

import (
	"context"
	"sync"

	"github.com/docker/docker/client"
	"github.com/docker/docker/errdefs"
	log "github.com/sirupsen/logrus"

	"scmc/rpc"
)

var (
	cli  *client.Client
	lock sync.Mutex
)

func dockerCli() (*client.Client, error) {
	lock.Lock()
	defer lock.Unlock()

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
		client.WithHTTPHeaders(map[string]string{"AuthZ-User": "KS-SCMC-SERVICE"}),
	)
	if err != nil {
		log.Warnf("try to connect to container daemon: %v", err)
		return nil, err
	}

	return cli, nil
}

func transDockerError(err error) error {
	log.Debugf("docker error %#v", err)
	if errdefs.IsInvalidParameter(err) {
		return rpc.ErrInvalidArgument
	} else if errdefs.IsNotFound(err) {
		return rpc.ErrNotFound
	} else if errdefs.IsConflict(err) {
		return rpc.ErrAlreadyExists
	}

	return rpc.ErrInternal
}
