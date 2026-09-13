// Package bindings · go 绑定：自动发现的 Go 项目列表（文件系统 + 运行中容器合并）。
// run/stop/logs/test 等操作经 Runner spawn `phpbox go ...`（事务在 bash 侧）。
package bindings

import (
	"context"
	"log"
	"strings"

	"github.com/xiaokentrl/phpbox-desktop/internal/engine/docker"
	"github.com/xiaokentrl/phpbox-desktop/internal/engine/goproject"
)

// GoProjects 暴露 Go 项目发现能力。
type GoProjects struct{}

// GoProjectEntry 透传引擎类型。
type GoProjectEntry = goproject.Entry

// ListGoProjects 列出自动发现的 Go 项目及运行状态。
func (g *GoProjects) ListGoProjects(ctx context.Context) ([]GoProjectEntry, error) {
	running := map[string]bool{}
	c, err := docker.New()
	if err == nil {
		items, err := c.ListContainers(ctx)
		if err == nil {
			for _, it := range items {
				name := strings.TrimPrefix(it.Name, "/")
				if strings.HasPrefix(name, "go-") && it.State == "running" {
					running[name] = true
				}
			}
		}
		// Docker 不可用时仅返回文件系统发现结果，不整体失败（开发机可能没起 Docker）
	}
	entries, err := goproject.List(goProjectsRoot(), running)
	if err != nil {
		log.Printf("[绑定] GoProjects.ListGoProjects 失败: %v", err)
		return nil, err
	}
	log.Printf("[绑定] GoProjects.ListGoProjects → %d 个项目", len(entries))
	return entries, nil
}
