package docker

import (
	"context"
	"io/ioutil"

	"github.com/docker/cli/cli/command"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	log "github.com/sirupsen/logrus"

	"scmc/common"
)

func prepareRegistry() (cli *client.Client, encodedAuth string, err error) {
	cli, err = Client()
	if err != nil {
		log.Errorf("try to connect to container daemon: %v", err)
		return nil, "", err
	}

	var auth = types.AuthConfig{
		Username:      common.Config.Registry.Username,
		Password:      common.Config.Registry.Password,
		ServerAddress: common.Config.Registry.Addr,
	}

	encodedAuth, err = command.EncodeAuthToBase64(auth)
	if err != nil {
		return nil, "", err
	}

	return cli, encodedAuth, nil
}

func PullImage(imageRef string) error {
	cli, encodedAuth, err := prepareRegistry()
	if err != nil {
		log.Errorf("pullImage %v", err)
		return err
	}

	reader, err := cli.ImagePull(context.Background(), imageRef, types.ImagePullOptions{
		All:          false,
		RegistryAuth: encodedAuth,
	})
	if err != nil {
		log.Errorf("pullImage %v", err)
		return err
	}

	defer reader.Close()
	if _, err := ioutil.ReadAll(reader); err != nil {
		log.Errorf("readall %v", err)
		return err
	}
	log.Debugf("pull %v done", imageRef)

	return nil
}

func PushImage(repoTag string) error {
	cli, encodedAuth, err := prepareRegistry()
	if err != nil {
		log.Errorf("pushImage %v", err)
		return err
	}

	reader, err := cli.ImagePush(context.Background(), repoTag, types.ImagePushOptions{
		All:          false,
		RegistryAuth: encodedAuth,
	})
	if err != nil {
		log.Errorf("pushImage %v", err)
		return err
	}

	defer reader.Close()
	if _, err := ioutil.ReadAll(reader); err != nil {
		log.Errorf("readall %v", err)
		return err
	}
	log.Debugf("push %v done", repoTag)

	return nil
}

func RemoveImage(force bool, repoTags ...string) error {
	cli, err := Client()
	if err != nil {
		log.Errorf("try to connect to container daemon: %v", err)
		return err
	}

	for _, repoTag := range repoTags {
		if _, err := cli.ImageRemove(context.Background(), repoTag, types.ImageRemoveOptions{}); err != nil {
			log.Warnf("remove image=%v err=%v", repoTag, err)
		}
	}
	return nil
}

func IsImageExist(imageRef string) (bool, error) {
	cli, err := Client()
	if err != nil {
		log.Errorf("try to connect to container daemon: %v", err)
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
