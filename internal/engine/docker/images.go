// Package docker · 镜像占用统计（总览页资源小部件 §3.11）。
package docker

import (
	"context"
	"sort"
	"strings"

	"github.com/moby/moby/client"
)

// ImageStat 一条镜像的占用摘要（phpbox 管理视角过滤可选）。
type ImageStat struct {
	ID    string   // 短 ID（12 位）
	Tags  []string // 仓库标签（顶层多个 repo:tag）
	Size  int64    // 字节（docker image ls SIZE）
	InUse bool     // 是否被任何容器使用（docker system df ACTIVE 语义）
}

// ImageUsage 镜像占用汇总：总量 + 明细（按大小降序）。
type ImageUsage struct {
	Total int64       // 全部镜像字节和
	Used  int64       // 正被容器使用的镜像字节和
	Items []ImageStat
}

// ListImageUsage 返回本机镜像占用。Docker 的镜像 size 是解压后大小
//（与 docker system df 的 Images SIZE 同源），含共享层会计。
func (c *Client) ListImageUsage(ctx context.Context) (ImageUsage, error) {
	res, err := c.api.ImageList(ctx, client.ImageListOptions{})
	if err != nil {
		return ImageUsage{}, err
	}
	// 容器使用集合：哪些镜像 ID 被引用
	containers, err := c.api.ContainerList(ctx, client.ContainerListOptions{All: true})
	if err != nil {
		return ImageUsage{}, err
	}
	inUse := map[string]bool{}
	for _, ct := range containers.Items {
		if ct.ImageID != "" {
			inUse[ct.ImageID] = true
		}
	}
	out := ImageUsage{}
	for _, img := range res.Items {
		tags := append([]string{}, img.RepoTags...)
		st := ImageStat{
			ID:    shortID(img.ID),
			Tags:  tags,
			Size:  img.Size,
			InUse: inUse[img.ID],
		}
		out.Total += img.Size
		if st.InUse {
			out.Used += img.Size
		}
		out.Items = append(out.Items, st)
	}
	// 大小降序
	sort.Slice(out.Items, func(i, j int) bool { return out.Items[i].Size > out.Items[j].Size })
	return out, nil
}

// shortID "sha256:abcd..." → "abcd..."（12 位显示形态，docker 惯例）。
func shortID(id string) string {
	if v, ok := strings.CutPrefix(id, "sha256:"); ok {
		if len(v) > 12 {
			return v[:12]
		}
		return v
	}
	return id
}
