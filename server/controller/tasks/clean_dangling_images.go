// 清理悬挂镜像
package tasks

import (
	"context"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"scmc/model/docker"
)

type CleanImages struct{}

func (s *CleanImages) Interval() time.Duration {
	return 5 * time.Minute
}

func (s *CleanImages) RunAtInit() bool {
	return true
}

func (s *CleanImages) Run() {
	cleanDanglingImages()
}

func cleanDanglingImages() {
	cli, err := docker.Client()
	if err != nil {
		log.Warnf("Get docker client err: %v", err)
		return
	}

	// 创建过滤器：只保留悬空镜像（dangling=true）
	filterArgs := filters.NewArgs()
	filterArgs.Add("dangling", "true") // 关键过滤条件

	// 配置镜像列表选项，应用过滤器
	listOpts := types.ImageListOptions{
		Filters: filterArgs,
	}

	// 获取过滤后的镜像列表（仅 <none>:<none> 镜像）
	images, err := cli.ImageList(context.Background(), listOpts)
	if err != nil {
		log.Warnf("ImageList err: %v", err)
		return
	}

	if len(images) == 0 {
		// log.Debugf("没有可清理的悬挂镜像(<none>:<none>)")
		return
	}

	// 清理悬挂镜像...
	removedCount := 0
	for _, img := range images {
		resp, err := cli.ImageRemove(context.Background(), img.ID, types.ImageRemoveOptions{Force: true})
		if err != nil {
			log.Errorf("清理 %s 失败: %v", img.ID[:12], err)
			continue
		}

		for _, r := range resp {
			if r.Deleted != "" {
				log.Debugf(" 已删除镜像层: %s\n", r.Deleted[:12])
			}
			if r.Untagged != "" {
				log.Warnf(" 已取消标签: %s\n", r.Untagged)
			}
		}
		removedCount++
	}

	log.Debugf("\n清理完成：成功删除 %d 个悬挂镜像，失败 %d 个\n", removedCount, len(images)-removedCount)
}
