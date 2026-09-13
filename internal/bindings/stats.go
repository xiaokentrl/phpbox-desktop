// Package bindings · stats 绑定：总览页资源小部件（ui-spec §3.11：offline/数据目录/镜像占用）。
// 全部只读真实测量：目录递归 stat + Docker 镜像 API；不做估算。
package bindings

import (
	"context"

	"github.com/xiaokentrl/phpbox-desktop/internal/engine/diskusage"
	"github.com/xiaokentrl/phpbox-desktop/internal/engine/docker"
)

// Stats 暴露资源占用统计能力。
type Stats struct{}

// DirUsage 一个目录的占用（路径 + 真实字节；目录不存在为 0）。
type DirUsage struct {
	Label string `json:"label"`
	Path  string `json:"path"`
	Bytes int64  `json:"bytes"`
}

// ResourceUsage 总览资源小部件数据：镜像（Docker API）+ 各数据目录（.env 契约路径）。
type ResourceUsage struct {
	Images   docker.ImageUsage `json:"images"`
	Dirs     []DirUsage        `json:"dirs"`
	DirsErrs []string          `json:"dirsErrs"` // 单目录测量失败不影响整体（快照尽力而为）
}

// GetResourceUsage 汇总镜像与数据目录占用。数据目录路径取 .env 契约键
//（env.sh:64/80/81 默认值对齐）：WWW_ROOT/MYSQL_DATA_ROOT/PGSQL_DATA_ROOT + 离线库。
func (s *Stats) GetResourceUsage(ctx context.Context) ResourceUsage {
	out := ResourceUsage{}
	if c, err := docker.New(); err == nil {
		if u, err := c.ListImageUsage(ctx); err == nil {
			out.Images = u
		}
	}
	// 目录清单：label 展示键（i18n 由前端负责，此处语义键）+ 真实路径
	home, _ := homeDirOf()
	dirs := []DirUsage{
		{Label: "offline", Path: offlineDir()},
		{Label: "www", Path: envPath("WWW_ROOT", joinPath(home, "www"))},
		{Label: "mysql", Path: envPath("MYSQL_DATA_ROOT", joinPath(home, "mysql-data"))},
		{Label: "pgsql", Path: envPath("PGSQL_DATA_ROOT", joinPath(home, "pgsql-data"))},
	}
	for _, d := range dirs {
		d.Bytes = diskusage.Dir(d.Path)
		out.Dirs = append(out.Dirs, d)
	}
	return out
}
