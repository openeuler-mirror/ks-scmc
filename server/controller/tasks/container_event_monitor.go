package tasks

import (
	"context"
	"fmt"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/client"
	"github.com/docker/docker/api/types/filters"

	"scmc/model/db"
	"scmc/model/warnlog"
	"scmc/rpc/pb/logging"
)

type ContainerEventMonitor struct{
	mu sync.Map			//存储每个节点的监控状态
	cli *client.Client
}

var lastDieTime time.Time

func (c *ContainerEventMonitor) Interval() time.Duration {
	return time.Minute
}

func (c *ContainerEventMonitor) RunAtInit() bool {
	return true
}


func (c *ContainerEventMonitor) monitorContainerEvents(cli *client.Client, node *db.NodeInfo) {
	


	if _,loaded := c.mu.LoadOrStore(node.Address,struct{}{});loaded{
		log.Debugf("Node %s (%s) is already being monitored.",node.Name,node.Address)
		return
	}
	
	log.Debugf("Monitoring container events on node: %s (%s)", node.Name, node.Address)
	
	// 创建事件过滤器
	filter := filters.NewArgs(
		filters.Arg("type", events.ContainerEventType), // 仅监听容器事件
		filters.Arg("event", "die"),                   // 捕捉容器停止事件
		// filters.Arg("event", "restart"),               // 捕捉容器重启事件
		filters.Arg("event", "start"),               // 捕捉容器重启事件
	)

	// 获取事件通道和错误通道
	eventChan, errChan := cli.Events(context.Background(), types.EventsOptions{
		Filters: filter,
	})

	// 处理事件流
	for {
		select {
		case event := <-eventChan:
			c.handleContainerEvent(event, node)
		case err := <-errChan:
			log.Warnf("Error receiving event: %v", err)
			c.mu.Delete((node.Address))
			return
		// case <-ctx.Done():
		// 	return
		}
	}
}


func (c *ContainerEventMonitor) handleContainerEvent(event events.Message, node *db.NodeInfo) {
	log.Debugf("ContainerEventMonitor handleContainerEvent event.Action:%s",event.Action)
	
	switch event.Action {
	case "die":
		// c.handleContainerDieEvent(event, node)
		exitCode,ok := event.Actor.Attributes["exitCode"]
		if !ok{
			log.Debugf("Exit code not found for container %.12s",event.Actor.ID)
			return
		}
		
		
		if exitCode == "0"{
			log.Debugf("Container %.12s died  normallyat %s code:%s ",event.Actor.ID,time.Unix(event.Time,0),exitCode)
		} else{
			log.Debugf("Container %.12s died  abnormallyat %s code:%s ",event.Actor.ID,time.Unix(event.Time,0),exitCode)
			lastDieTime = time.Unix(event.Time,0)
		}
	// case "restart":
	// 	c.handleContainerRestartEvent(event, node)
	case "start":
		
		if !lastDieTime.IsZero() && time.Since(lastDieTime) < 5*time.Second{
			log.Debugf("Container %.12s auto-restarted at %s ",event.ID,time.Unix(event.Time,0))
			
			c.handleContainerStartEvent(event, node)
		}
		lastDieTime = time.Time{}
	}
}

func (c *ContainerEventMonitor) handleContainerDieEvent(event events.Message, node *db.NodeInfo) {
	log.Debugf("Container %s died on node %s (%s)", event.Actor.ID, node.Name, node.Address)

	// 创建告警日志
	warnlog.Create([]*db.WarnLog{
		{
			NodeId:        node.ID,
			NodeInfo:      fmt.Sprintf("%s (%s)", node.Name, node.Address),
			EventType:     int64(logging.EVENT_TYPE_STOP_CONTAINER),
			EventModule:   int64(logging.EVENT_MODULE_CONTAINER),
			ContainerName: fmt.Sprintf("Container %.12s ", event.Actor.ID),
			Detail:        fmt.Sprintf("Container %s died unexpectedly", event.Actor.ID),
		},
	})
}

func (c *ContainerEventMonitor) handleContainerRestartEvent(event events.Message, node *db.NodeInfo) {
	log.Debugf("Container %s restarted on node %s (%s)", event.Actor.ID, node.Name, node.Address)

	// 创建告警日志
	warnlog.Create([]*db.WarnLog{
		{
			NodeId:        node.ID,
			NodeInfo:      fmt.Sprintf("%s (%s)", node.Name, node.Address),
			EventType:     int64(logging.EVENT_TYPE_RESTART_CONTAINER),
			EventModule:   int64(logging.EVENT_MODULE_CONTAINER),
			ContainerName: fmt.Sprintf("Container %.12s ", event.Actor.ID),
			Detail:        fmt.Sprintf("Container %s was restarted", event.Actor.ID),
		},
	})
}

func (c *ContainerEventMonitor) handleContainerStartEvent(event events.Message, node *db.NodeInfo) {
	log.Debugf("Container %.12s started on node %s (%s)", event.Actor.ID, node.Name, node.Address)
	
	
	containerJSON,err := c.cli.ContainerInspect(context.Background(),event.Actor.ID)
	
    if err != nil {
        log.Errorf("Failed to get restart policy for container %s: %v", event.Actor.ID, err)
        return
    }
	
	restartPolicy := containerJSON.HostConfig.RestartPolicy.Name
	maxRetryCount := containerJSON.HostConfig.RestartPolicy.MaximumRetryCount
	restartCount  := containerJSON.RestartCount
	
	var policyLog string
	switch restartPolicy {
	case "on-failure":
		policyLog = fmt.Sprintf("Container %.6s restarted due to failure", event.ID)
	case "always", "unless-stopped":
		policyLog = fmt.Sprintf("Container %.6s restarted automatically", event.ID)
	default:
		policyLog = fmt.Sprintf("Container %.6s started manually", event.ID)
	}
	log.Debugf(policyLog)

	var detail string
	if restartPolicy == "on-failure" {
		detail = fmt.Sprintf("容器 %.6s 异常退出重启（策略：%s，当前次数：%d，最大允许：%d）",
			event.ID, restartPolicy, restartCount, maxRetryCount)
	} else if restartPolicy == "always" || restartPolicy == "unless-stopped" {
		detail = fmt.Sprintf("容器 %.6s 异常退出重启（策略：%s，当前次数：%d）",
			event.ID, restartPolicy, restartCount)
	} else {
		detail = fmt.Sprintf("容器 %.6s 手动启动（策略：%s）", 
			event.ID, restartPolicy)
	}
	
	// 创建告警日志
	warnlog.Create([]*db.WarnLog{
		{
			NodeId:        node.ID,
			NodeInfo:      fmt.Sprintf("%s (%s)", node.Name, node.Address),
			EventType:     int64(logging.EVENT_TYPE_START_CONTAINER),
			EventModule:   int64(logging.EVENT_MODULE_CONTAINER),
			ContainerName: fmt.Sprintf("容器 %.12s", event.ID),
			Detail:        detail,
		},
	})
}

func (c *ContainerEventMonitor) Run() {
	log.Debugf("ContainerEventMonitor Run start ")

	// 获取 Docker 客户端
	if c.cli == nil {
		cli, err := client.NewClientWithOpts(client.WithVersion("1.41"))
		if err != nil {
			log.Errorf("Failed to create Docker client: %v", err)
			return
		}
		c.cli = cli
	}
	
	
	nodes, err := db.ListNodes()
	if err != nil {
		log.Errorf("Failed to list nodes: %v", err)
		return
	}

	for _, node := range nodes {
		go c.monitorContainerEvents(c.cli, node)
	}
}