package internal

import (
	"os"
	"path/filepath"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	log "github.com/sirupsen/logrus"

	"scmc/common"
)

const (
	containerSocketPath  = "/tmp/.X11-unix"
	containerAuthPath    = "/tmp/.xauth"
	authFile             = "Xauthority"
	containerNamePattern = `^[a-zA-Z0-9][a-zA-Z0-9_.-]+$`
)

func containerGraphicSetup(containerName string, config *container.Config, hostConfig *container.HostConfig) error {
	hostPath := filepath.Join(common.Config.Agent.ContainerExtraDataBasedir, containerName)
	socketDir := filepath.Join(hostPath, "socket")
	if err := os.MkdirAll(socketDir, 0755); err != nil {
		log.Warnf("mkdir %v: %v", socketDir, err)
		return err
	}

	authDir := filepath.Join(hostPath, "xauth")
	if err := os.MkdirAll(authDir, 0755); err != nil {
		log.Warnf("mkdir %v: %v", authDir, err)
		return err
	}

	// set environment for display, shared X11 socket/auth file
	config.Env = append(config.Env, "DISPLAY=:0", "XAUTHORITY="+filepath.Join(containerAuthPath, authFile))

	var mnts []mount.Mount
	for _, m := range hostConfig.Mounts {
		if m.Target == containerSocketPath || m.Target == containerAuthPath {
			continue // remove duplicate mnt point
		}
		mnts = append(mnts, m)
	}

	hostConfig.Mounts = append(mnts,
		mount.Mount{
			Type:   mount.TypeBind,
			Source: socketDir,
			Target: containerSocketPath,
		},
		mount.Mount{
			Type:   mount.TypeBind,
			Source: authDir,
			Target: containerAuthPath,
		},
	)

	return nil
}

func removeContainerGraphicSetup(containerName string) error {
	hostPath := filepath.Join(common.Config.Agent.ContainerExtraDataBasedir, containerName)
	return os.RemoveAll(hostPath)
}
