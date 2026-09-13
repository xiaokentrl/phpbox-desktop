package bindings

import (
	"context"
	"testing"
)

// TestListContainersThroughBinding 验证绑定层完整链路：
// 前端调用 → binding → engine → Docker daemon → 真实容器返回。
// 需要 Docker daemon 运行（本机开发前提），无 daemon 时跳过。
func TestListContainersThroughBinding(t *testing.T) {
	d := &Docker{}
	items, err := d.ListContainers(context.Background())
	if err != nil {
		t.Skipf("Docker daemon 不可达，跳过（%v）", err)
	}
	if len(items) == 0 {
		t.Fatal("绑定返回 0 个容器（daemon 可达但列表为空）")
	}
	running := 0
	for _, it := range items {
		if it.Name == "" || it.Image == "" || it.State == "" {
			t.Fatalf("容器字段缺失: %+v", it)
		}
		if it.State == "running" {
			running++
		}
	}
	t.Logf("绑定链路贯通：%d 个容器（%d 个运行中），首项 %s/%s",
		len(items), running, items[0].Name, items[0].Image)
}
