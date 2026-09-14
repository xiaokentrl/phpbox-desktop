package bindings

import (
	"context"
	"strings"
	"testing"

	engineEnv "github.com/xiaokentrl/phpbox-desktop/internal/engine/env"
)

// TestListGoImagesThroughBinding：绑定链路 → Docker 镜像表筛 golang:*。
// 需要 Docker daemon，无 daemon 时 SKIPPED；无 golang 镜像 = 空列表也是合法结果（如实）。
func TestListGoImagesThroughBinding(t *testing.T) {
	g := &GoProjects{}
	list, err := g.ListGoImages(context.Background())
	if err != nil {
		t.Skipf("Docker daemon 不可达，跳过（%v）", err)
	}
	defaultTag := ""
	if kvs, err := engineEnv.Read(envFile()); err == nil {
		for _, kv := range kvs {
			if kv.Key == "GO_DEFAULT_VERSION" {
				defaultTag = kv.Value
			}
		}
	}
	defaults := 0
	for _, img := range list {
		if !strings.HasPrefix(img.Image, "golang:") {
			t.Errorf("非 golang 镜像混入: %+v", img)
		}
		if img.Tag == "" || img.Size < 0 {
			t.Errorf("镜像字段缺失: %+v", img)
		}
		if img.InUse && img.UsedBy == "" {
			t.Errorf("InUse=true 但 UsedBy 为空（应给出引用容器名）: %+v", img)
		}
		if img.Default {
			defaults++
		}
	}
	if defaults > 1 {
		t.Errorf("默认版本标记应至多一个（GO_DEFAULT_VERSION=%q），得到 %d 个", defaultTag, defaults)
	}
	t.Logf("绑定链路贯通：%d 个 golang 镜像（默认 %q）", len(list), defaultTag)
}
