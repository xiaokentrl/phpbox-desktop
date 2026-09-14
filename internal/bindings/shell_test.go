package bindings

import (
	"context"
	"strings"
	"testing"
)

// TestShellURLValidation：validSiteURL 是 Shell 绑定唯一的前置防线——
// 仅 http(s) 完整前缀放行（防 file:// 等任意协议经 webview 打开）。
// 纯函数直测（application.Get() 在测试进程为 nil，放行路径只验校验器本身）。
func TestShellURLValidation(t *testing.T) {
	cases := []struct {
		url  string
		want bool
	}{
		{"http://demo.test", true},
		{"http://demo.test:8080", true},
		{"https://example.com", true},
		{"", false},
		{"file:///etc/passwd", false},
		{"javascript:alert(1)", false},
		{"ftp://x", false},
		{"http:/missing-slash", false},
		{"http:/", false},
	}
	for _, c := range cases {
		if got := validSiteURL(c.url); got != c.want {
			t.Errorf("validSiteURL(%q) = %v, want %v", c.url, got, c.want)
		}
	}
	// 绑定方法对非法地址的拒绝文案（不触达桌面 API）
	s := &Shell{}
	if err := s.OpenSiteBrowser(context.Background(), "file:///etc/passwd"); err == nil || !strings.Contains(err.Error(), "非法") {
		t.Errorf("OpenSiteBrowser 非法地址应拒绝并说明原因，得到: %v", err)
	}
	// OpenInFileManager 空路径拒绝
	if err := s.OpenInFileManager(context.Background(), ""); err == nil {
		t.Error("OpenInFileManager(空串) 应拒绝")
	}
}

// TestGetContainerMemoryThroughBinding：绑定链路 → Docker API one-shot stats。
// 需要至少一个运行中容器，无 daemon/无运行容器时 SKIPPED。
func TestGetContainerMemoryThroughBinding(t *testing.T) {
	d := &Docker{}
	items, err := d.ListContainers(context.Background())
	if err != nil {
		t.Skipf("Docker daemon 不可达，跳过（%v）", err)
	}
	var target string
	for _, it := range items {
		if it.State == "running" {
			target = strings.TrimPrefix(it.Name, "/")
			break
		}
	}
	if target == "" {
		t.Skip("无运行中容器：SKIPPED")
	}
	m, err := d.GetContainerMemory(context.Background(), target)
	if err != nil {
		t.Fatalf("GetContainerMemory(%s) 失败: %v", target, err)
	}
	if m.Name != target || m.MemUse < 0 {
		t.Fatalf("内存快照字段异常: %+v", m)
	}
	t.Logf("绑定链路贯通：%s 工作集 %d 字节", target, m.MemUse)
}
