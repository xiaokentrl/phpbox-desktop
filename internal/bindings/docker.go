// Package bindings 是壳层绑定：把引擎能力暴露给前端（设计文档 §3.3-3）。
// 本包禁止业务逻辑——只做引擎调用与类型透传。
package bindings

import (
	"context"

	"github.com/xiaokentrl/phpbox-desktop/internal/engine/docker"
)

// Docker 暴露容器管理能力；导出方法会被 wails3 生成前端绑定。
type Docker struct{}

// ContainerSummary 透传引擎类型（前端 TS 类型由绑定生成器产出）。
type ContainerSummary = docker.ContainerSummary

// ListContainers 返回全部容器（含已停止）。
func (d *Docker) ListContainers(ctx context.Context) ([]ContainerSummary, error) {
	c, err := docker.New()
	if err != nil {
		return nil, err
	}
	return c.ListContainers(ctx)
}
