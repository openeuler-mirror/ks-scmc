package server

import (
	"context"
	"log"
	"sync"

	"github.com/docker/docker/client"
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
			log.Printf("Ping: %v", err)
		} else {
			return cli, nil
		}
	}

	var err error
	cli, err = client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		log.Printf("try to connect to container daemon: %v", err)
		return nil, err
	}

	cli.NegotiateAPIVersion(context.Background())

	return cli, nil
}
