package internal

import (
	"os"
	"path/filepath"
	"regexp"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	log "github.com/sirupsen/logrus"

	"scmc/common"
)

const (
	containerSocketPath = "/tmp/.X11-unix"
	containerAuthPath   = "/tmp/.xauth"
	authFile            = "Xauthority"
)
const containerNameChars = `[a-zA-Z0-9][a-zA-Z0-9_.-]`

var containerNamePattern = regexp.MustCompile(`^` + containerNameChars + `+$`)

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

	// AttachStdin, Tty: make sure container like OS(ubuntu, centos) will not exit after start
	config.AttachStdin = true
	config.Tty = true

	// set environment for display, shared X11 socket/auth file
	config.Env = append(config.Env, "DISPLAY=:0", "XAUTHORITY="+filepath.Join(containerAuthPath, authFile))
	hostConfig.Mounts = append(hostConfig.Mounts, mount.Mount{
		Type:   mount.TypeBind,
		Source: socketDir,
		Target: containerSocketPath,
	})
	hostConfig.Mounts = append(hostConfig.Mounts, mount.Mount{
		Type:   mount.TypeBind,
		Source: authDir,
		Target: containerAuthPath,
	})

	return nil
}
