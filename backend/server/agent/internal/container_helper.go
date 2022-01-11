package internal

import (
	"os"
	"path/filepath"
	"regexp"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	log "github.com/sirupsen/logrus"

	"ksc-mcube/common"
)

const (
	socketMountPath   = "/tmp/.X11-unix"
	authFileMountPath = "/tmp/.Xauthority"
)
const containerNameChars = `[a-zA-Z0-9][a-zA-Z0-9_.-]`

var containerNamePattern = regexp.MustCompile(`^` + containerNameChars + `+$`)

func containerGraphicSetup(containerName string, config *container.Config, hostConfig *container.HostConfig) error {
	socketDir := filepath.Join(common.GraphicConfigBaseDir, containerName, "socket")
	if err := os.MkdirAll(socketDir, 0755); err != nil {
		log.Warnf("mkdir %v: %v", socketDir, err)
		return err
	}

	authFile := filepath.Join(common.GraphicConfigBaseDir, containerName, "Xauthority")
	_, err := os.Stat(authFile)
	if os.IsNotExist(err) {
		file, err := os.Create(authFile)
		if err != nil {
			log.Warnf("create file %v: %v", authFile, err)
			return err
		}
		defer file.Close()
	}

	config.Env = append(config.Env, "DISPLAY=:0", "XAUTHORITY="+authFileMountPath)
	hostConfig.Mounts = append(hostConfig.Mounts, mount.Mount{
		Type:   mount.TypeBind,
		Source: socketDir,
		Target: socketMountPath,
	})
	hostConfig.Mounts = append(hostConfig.Mounts, mount.Mount{
		Type:   mount.TypeBind,
		Source: authFile,
		Target: authFileMountPath,
	})

	return nil
}
