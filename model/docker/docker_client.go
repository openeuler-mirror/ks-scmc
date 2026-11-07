package docker

import (
	"sync"

	"github.com/docker/docker/client"
)

var (
	cli     *client.Client
	cliErr  error
	cliOnce sync.Once
)

// Client creates a Docker API client based on the given Docker flags
func Client() (*client.Client, error) {
	httpHeaders := map[string]string{
		"Authz-User": "KS-SCMC-SERVICE", // 通过authz插件检查
	}

	cliOnce.Do(func() {
		cli, cliErr = client.NewClientWithOpts(
			client.FromEnv,
			client.WithAPIVersionNegotiation(),
			client.WithHTTPHeaders(httpHeaders),
		)
	})
	return cli, cliErr
}

func IsErrNotFound(err error) bool {
	return client.IsErrNotFound(err)
}
