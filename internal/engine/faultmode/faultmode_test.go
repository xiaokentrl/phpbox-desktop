package faultmode

import (
	"strings"
	"testing"

	"github.com/xiaokentrl/phpbox-desktop/internal/engine/docker"
)

// §8.1 阶段 0 检测条件：容器状态 + 日志关键词。只对非 running 的 phpbox 容器扫日志。
func TestDetectMatchesSockCrashLoop(t *testing.T) {
	// Detect 依赖 docker.Client 拉日志，无法单测；对 pure 部分构造最小验证：
	// 命中行收集逻辑内联在 Detect，此处验证模式表内容与关键词形态本身。
	m := modes[0]
	if m.ID != "sock_crash_loop" {
		t.Fatalf("首模式应为 sock_crash_loop，得到 %s", m.ID)
	}
	for _, kw := range m.Match {
		if kw != strings.ToLower(kw) {
			t.Errorf("关键词 %q 必须小写（匹配时日志统一 lower）", kw)
		}
	}
}

func TestModesComplete(t *testing.T) {
	ids := map[string]bool{}
	for _, m := range Modes() {
		if m.ID == "" || m.Title == "" || m.Cli == "" {
			t.Errorf("模式 %q 字段不完整（ID/Title/Cli 必填）", m.ID)
		}
		if len(m.Match) == 0 {
			t.Errorf("模式 %q 无检测关键词", m.ID)
		}
		ids[m.ID] = true
	}
	for _, want := range []string{"sock_crash_loop", "port_conflict", "image_missing", "dbdata_corrupt"} {
		if !ids[want] {
			t.Errorf("模式表缺 %s", want)
		}
	}
}

// 编译期断言：Hit 与容器摘要的字段形态被绑定层透传依赖。
var _ = docker.ContainerSummary{Name: "", Service: "", State: ""}
