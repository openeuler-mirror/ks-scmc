package docker

import (
	"archive/tar"
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"

	log "github.com/sirupsen/logrus"
)

func ContainerFileMD5(containerID, path string) (string, error) {
	cli, err := Client()
	if err != nil {
		return "", err
	}

	f, stat, err := cli.CopyFromContainer(context.Background(), containerID, path)
	if err != nil {
		log.Errorf("CopyFromContainer container=%v path=%v, err=%v", containerID, path, err)
		return "", err
	}
	defer f.Close()

	if stat.Mode.IsDir() {
		return "", fmt.Errorf("%v is directory", path)
	}

	r := tar.NewReader(f)
	h := md5.New()
	for {
		_, err := r.Next()
		if err != nil {
			if err == io.EOF {
				break
			}
			log.Errorf("ContainerFileMD5 read tar, err=%v", err)
			return "", err
		}

		_, err = io.Copy(h, r)
		if nil != err {
			log.Errorf("ContainerFileMD5 copy to md5 hasher, err=%v", err)
			return "", err
		}
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}
