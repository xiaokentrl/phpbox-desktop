package bindings

import (
	"context"
	"testing"
)

// 集成测试：真实发现 ~/www 下的 go.mod 项目 + Docker 运行状态合并。
func TestListGoProjectsThroughBinding(t *testing.T) {
	entries, err := (&GoProjects{}).ListGoProjects(context.Background())
	if err != nil {
		t.Skipf("Go 项目发现失败（%v）：SKIPPED", err)
	}
	t.Logf("绑定链路贯通：%s → %d 个项目（运行中 %d）",
		goProjectsRoot(), len(entries), countRunning(entries))
	for _, e := range entries {
		if e.Name == "" || e.Dir == "" {
			t.Errorf("项目字段不完整: %+v", e)
		}
	}
}

func countRunning(entries []GoProjectEntry) int {
	n := 0
	for _, e := range entries {
		if e.Running {
			n++
		}
	}
	return n
}
