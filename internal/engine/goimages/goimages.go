// Package goimages · Go 镜像管理（ui-spec §3.7）：golang:* 镜像的发现/占用/被引用状态。
// 契约来源 bash lib/go/common/install.sh + server.sh：
//   - 版本形态：alpine/latest 归一化为 golang:alpine；<v> 归一化为 golang:<v>-alpine
//   - uninstall 前置校验：镜像被任何容器（ancestor）使用时 bash 会拒绝并提示先 go stop
//   - --purge 语义：额外删 GO_CACHE_ROOT/<GO_CACHE_VERSION>（latest 对应 alpine）
// GUI 不做平行状态：列表 = Docker 真实镜像，被引用 = 真实容器引用；变更全部经 CLI spawn。
package goimages

import (
	"context"
	"strings"

	"github.com/xiaokentrl/phpbox-desktop/internal/engine/docker"
)

// GoImage 一条 Go 镜像状态（全部由 Docker 真实状态派生）。
type GoImage struct {
	Tag     string `json:"tag"`     // CLI 参数形态：alpine / 1.24 / 1.24.3（用户装卸用它）
	Image   string `json:"image"`   // 完整镜像名：golang:alpine / golang:1.24-alpine
	Size    int64  `json:"size"`    // 字节
	InUse   bool   `json:"inUse"`   // 被容器使用（docker ancestor 语义）——bash uninstall 会拒绝
	UsedBy  string `json:"usedBy"`  // 引用它的容器名（可能多个，逗号连接；未用为空）
	Default bool   `json:"default"` // .env GO_DEFAULT_VERSION 指向的版本（当前默认）
}

// normalizeTag docker 镜像 tag → CLI 版本参数形态（bash _go_resolve_version 的逆映射）：
// golang:alpine → alpine；golang:1.24-alpine → 1.24。
func normalizeTag(ref string) (tag, image string) {
	if _, t, ok := strings.Cut(ref, ":"); ok {
		image = "golang:" + t
		if t == "alpine" {
			return "alpine", image
		}
		if v, ok := strings.CutSuffix(t, "-alpine"); ok {
			return v, image
		}
		return t, image // 非 -alpine 变体（如 golang:1.24-bookworm）：原样展示，CLI 装它会被归一化，如实报告
	}
	return ref, ref
}

// List 从 Docker 真实镜像表筛出 golang 镜像并标注被引用状态。
// defaultTag 是 .env GO_DEFAULT_VERSION（如 "alpine"）；Docker 不可达时返回错误（调用方降级显示）。
func List(ctx context.Context, defaultTag string, c *docker.Client) ([]GoImage, error) {
	usage, err := c.ListImageUsage(ctx)
	if err != nil {
		return nil, err
	}
	// 容器名 → 其镜像引用（go 容器直接引用 golang:tag，无中间镜像，字符串匹配即够）
	cts, err := c.ListContainers(ctx)
	if err != nil {
		return nil, err
	}
	var out []GoImage
	for _, img := range usage.Items {
		for _, ref := range img.Tags {
			repo, _, _ := strings.Cut(ref, ":")
			if repo != "golang" {
				continue
			}
			tag, image := normalizeTag(ref)
			gi := GoImage{
				Tag:     tag,
				Image:   image,
				Size:    img.Size,
				InUse:   img.InUse,
				Default: tag == defaultTag,
			}
			if img.InUse {
				var users []string
				for _, ct := range cts {
					if ct.Image == ref || ct.Image == image {
						users = append(users, strings.TrimPrefix(ct.Name, "/"))
					}
				}
				gi.UsedBy = strings.Join(users, ", ")
			}
			out = append(out, gi)
			break // 同一镜像 ID 的其余 RepoTag（<none> 等）不重复出
		}
	}
	return out, nil
}
