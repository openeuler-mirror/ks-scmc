package tasks

import (
	"context"
	"strings"
	"syscall"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	log "github.com/sirupsen/logrus"

	"scmc/common"
	"scmc/model/docker"
	"scmc/model/litedb"
)

type AppRun struct{}

func (s *AppRun) Interval() time.Duration {
	return time.Second * 3
}

func (s *AppRun) RunAtInit() bool {
	return true
}

func (s *AppRun) Run() {
	if common.IsMaster() { // 主机模式
		ContaierAppRun()
	} else { // 非主机模式
		ContaierAppStop() // 停止备机 app 的运行
	}
}

// 如果是备份机器，那么 app 是不需要运行的
func ContaierAppStop() {
	// 获取当前机器上所有的 容器 ID
	cli, err := docker.Client()
	if err != nil {
		return
	}

	opts := types.ContainerListOptions{All: true, Size: true}
	containers, err := cli.ContainerList(context.Background(), opts)
	if err != nil {
		return
	}

	for _, c := range containers { // 如果开启了应用高可用，就需要检测应用是否已经拉起来了
		containerName := strings.TrimPrefix(c.Names[0], "/")
		data, err := litedb.GetContainerOutConfigByName(containerName)
		if err != nil {
			continue
		}
		if data.Config.AppRun {
			go stopAppEntry(c.ID)
		}
	}
}

// 如果是主模式并且开启了应用高可用，那么就要运行容器里面的 app
func ContaierAppRun() {
	// 获取当前机器上所有的 容器 ID
	cli, err := docker.Client()
	if err != nil {
		return
	}

	opts := types.ContainerListOptions{All: true, Size: true}
	containers, err := cli.ContainerList(context.Background(), opts)
	if err != nil {
		return
	}

	for _, c := range containers { // 如果开启了应用高可用，就需要检测应用是否已经拉起来了
		containerName := strings.TrimPrefix(c.Names[0], "/")
		data, err := litedb.GetContainerOutConfigByName(containerName)
		if err != nil {
			continue
		}

		if data.Config.AppRun {
			go runAppEntry(c.ID)
		}

	}
}

// 容器启动，启动容器内配置的相关应用
func runAppEntry(containerId string) {
	allApp, err := litedb.ListAppEntry(containerId)
	if err != nil || len(allApp) == 0 {
		return
	}

	cli, err := docker.Client()
	if err != nil {
		return
	}

	if !containerIsRunning(context.Background(), cli, containerId) {
		return
	}

	for _, app := range allApp {
		if app.IsGUI {
			continue
		}

		if appIsRunningWithCli(context.Background(), cli, app.IDResponse) {
			continue
		}

		idRes, err := cli.ContainerExecCreate(context.Background(), containerId, types.ExecConfig{
			Cmd: strings.Split(app.ExePath, " "),
		})

		if err != nil || idRes.ID == "" {
			continue
		}

		if err := cli.ContainerExecStart(context.Background(), idRes.ID, types.ExecStartCheck{
			Detach: true,
			Tty:    false,
		}); err != nil {
			log.Warnf("err = %v", err)
			continue
		}

		app.IDResponse = idRes.ID
		if err := litedb.UpdateAppEntry(app); err != nil {
			log.Warnf("err = %v", err)
			continue
		}
	}
}

// 备份机器停止 app 的运行
func stopAppEntry(containerId string) {
	allApp, err := litedb.ListAppEntry(containerId)
	if err != nil || len(allApp) == 0 {
		return
	}

	cli, err := docker.Client()
	if err != nil {
		return
	}

	if !containerIsRunning(context.Background(), cli, containerId) {
		return
	}

	for _, app := range allApp {
		if app.IsGUI {
			continue
		}

		execIns, err := cli.ContainerExecInspect(context.Background(), app.IDResponse)
		if err != nil {
			continue
		}

		if !execIns.Running {
			continue
		}

		syscall.Kill(execIns.Pid, syscall.SIGKILL)
	}
}

func appIsRunningWithCli(ctx context.Context, cli *client.Client, idResponse string) bool {
	if idResponse == "" {
		return false
	}

	execins, err := cli.ContainerExecInspect(ctx, idResponse)
	if err != nil {
		log.Infof("cli.ContainerExecInspect err:%s", err)
		return false
	}

	return execins.Running
}

func containerIsRunning(ctx context.Context, cli *client.Client, containerID string) bool {
	ins, err := cli.ContainerInspect(ctx, containerID)
	if err != nil {
		log.Infof("cli.ContainerInspect err:%s", err)
		return false
	}

	if ins.State == nil {
		return false
	}

	return ins.State.Running
}
