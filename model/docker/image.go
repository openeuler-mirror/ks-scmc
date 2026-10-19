package docker

import (
	"context"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	log "github.com/sirupsen/logrus"
)

func RemoveImage(ctx context.Context, repoTag string, force bool) error {
	cli, err := Client()
	if err != nil {
		return err
	}

	opt := types.ImageRemoveOptions{Force: force}
	if _, err := cli.ImageRemove(ctx, repoTag, opt); err != nil {
		log.Warnf("Remove image=%v err=%v", repoTag, err)
		return err
	}

	log.Infof("Remove image=%v OK", repoTag)
	return nil
}

func IsImageExist(imageRef string) (bool, error) {
	cli, err := Client()
	if err != nil {
		return false, err
	}

	if _, _, err = cli.ImageInspectWithRaw(context.Background(), imageRef); err != nil {
		if client.IsErrNotFound(err) {
			return false, nil
		}
		return false, err
	}

	return true, nil
}
