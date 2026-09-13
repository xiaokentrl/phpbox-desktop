package docker

import (
	"context"

	"github.com/moby/moby/client"
)

// ContainerSummary 是容器列表的引擎输出（phpbox 管理视角，非 docker 原生结构）。
type ContainerSummary struct {
	Name    string // 容器名（去前导斜杠）
	Image   string
	State   string // running / exited / paused ...
	Service string // phpbox-service label（如 php/mysql；非 phpbox 容器为空）
	Version string // phpbox-version label（如 8.4；php 线含点，与 compose 分片名一致）
}

// ListContainers 返回全部容器（含已停止），按创建时间倒序由 daemon 决定。
func (c *Client) ListContainers(ctx context.Context) ([]ContainerSummary, error) {
	list, err := c.api.ContainerList(ctx, client.ContainerListOptions{All: true})
	if err != nil {
		return nil, err
	}
	out := make([]ContainerSummary, 0, len(list.Items))
	for _, ct := range list.Items {
		name := ""
		if len(ct.Names) > 0 {
			name = ct.Names[0]
		}
		out = append(out, ContainerSummary{
			Name:    name,
			Image:   ct.Image,
			State:   string(ct.State),
			Service: ct.Labels["phpbox-service"],
			Version: ct.Labels["phpbox-version"],
		})
	}
	return out, nil
}

// 编译期断言：客户端实现官方 APIClient（接口漂移即刻暴露）。
var _ client.APIClient = (*client.Client)(nil)
