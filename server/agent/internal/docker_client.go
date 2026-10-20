package internal

import (
	"github.com/docker/docker/errdefs"
	log "github.com/sirupsen/logrus"

	"scmc/rpc"
)

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
