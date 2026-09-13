package bindings

import (
	"context"
	"os/exec"
	"strings"
	"sync"
	"testing"
)

// phpbox log.sh 无条件输出 ANSI 颜色码（非 tty 不关闭），spawn 链路必须先剥离再分类
func TestStripANSI(t *testing.T) {
	cases := []struct{ in, want string }{
		{"\x1b[0;32m[OK]   备份完成\x1b[0m", "[OK]   备份完成"},
		{"\x1b[1;36m[INFO] 检查 Docker Engine…\x1b[0m", "[INFO] 检查 Docker Engine…"},
		{"\x1b[0;31m[ERR]  失败\x1b[0m", "[ERR]  失败"},
		{"无颜色行", "无颜色行"},
		{"", ""},
	}
	for _, c := range cases {
		if got := stripANSI(c.in); got != c.want {
			t.Errorf("stripANSI(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestClassify(t *testing.T) {
	cases := []struct {
		in      string
		wantCls string
	}{
		{"[OK]   备份完成: /x.tar.gz", "ok"},
		{"[ERR]  归档包含非法路径", "err"},
		{"[INFO] 检查依赖…", ""},
		{"NAMES  phpbox service", ""},
	}
	for _, c := range cases {
		ev := classify(c.in)
		if ev.Line != c.in || ev.Cls != c.wantCls {
			t.Errorf("classify(%q) = {%q, %q}, want {%q, %q}", c.in, ev.Line, ev.Cls, c.in, c.wantCls)
		}
	}
	// 剥离后必须能命中前缀（spawn 真实输出的组合路径）
	if ev := classify(stripANSI("\x1b[0;31m[ERR]  端口被占用\x1b[0m")); ev.Cls != "err" {
		t.Errorf("stripANSI+classify 未命中 [ERR] 前缀: %+v", ev)
	}
}

// 阶段推断：断言行全部来自 bash 仓真实日志词汇（grep 采集，见 phaseRules 注释出处）
func TestInferPhase(t *testing.T) {
	cases := []struct {
		in        string
		wantPhase string
		wantCls   string
	}{
		{"[INFO] 初始化 mysql 8.0 配置...", "configured", ""},             // install.sh:92
		{"[OK]   mysql 8.0 配置就绪", "configured", "ok"},                // install.sh:104
		{"[INFO] Docker 构建开始：PHP 8.4，超时 900s", "preparing", ""},      // php/install.sh:93
		{"[INFO] 离线构建 PHP 8.4：apk 闭包 23 个包", "preparing", ""},        // php/install.sh:166
		{"[OK]   mysql 8.0 安装完成，端口 3306", "committed", "ok"},         // install.sh:163
		{"[OK]   mysql 8.0 端口已改为 3307", "committed", "ok"},           // install.sh:195
		{"[INFO] 安装失败，自动清理 mysql 8.0 的半安装状态...", "rolling_back", ""}, // install.sh:47
		{"[ERR]  端口变更失败，已回滚", "rolling_back", "err"},                 // install.sh:187
		{"[OK]   PHP 8.4 已卸载", "absent", "ok"},                       // php/uninstall:116
		{"[INFO] apk 下载器启动：镜像 x，容器 y", "", ""},                       // 无阶段词：不臆造
		{"[ERR]  未知错误", "", "err"},                                   // 错误但无阶段词
	}
	for _, c := range cases {
		ev := classify(c.in)
		if ev.Phase != c.wantPhase {
			t.Errorf("inferPhase(%q) = %q, want %q", c.in, ev.Phase, c.wantPhase)
		}
		if ev.Cls != c.wantCls {
			t.Errorf("classify(%q).Cls = %q, want %q", c.in, ev.Cls, c.wantCls)
		}
	}
}

func TestTrimNL(t *testing.T) {
	cases := []struct{ in, want string }{
		{"line\n", "line"},
		{"line\r\n", "line"},
		{"line", "line"},
		{"\n", ""},
	}
	for _, c := range cases {
		if got := trimNL(c.in); got != c.want {
			t.Errorf("trimNL(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

// 集成测试：注入替身 emitEvent，验证真实 spawn 链路（RunTask→pipe→分类→事件序列）。
// phpbox CLI 不在 PATH 时跳过（环境标记 SKIPPED，AGENTS §8）。
func TestRunTaskSpawnIntegration(t *testing.T) {
	if _, err := exec.LookPath("phpbox"); err != nil {
		t.Skip("phpbox CLI 不在 PATH：SKIPPED")
	}

	var mu sync.Mutex
	var logs []TaskEvent
	var done []TaskEvent
	orig := emitEvent
	emitEvent = func(name string, data TaskEvent) {
		mu.Lock()
		defer mu.Unlock()
		if name == "task:log" {
			logs = append(logs, data)
		} else if name == "task:done" {
			done = append(done, data)
		}
	}
	defer func() { emitEvent = orig }()

	r := &Runner{}
	err := r.RunTask(context.Background(), []string{"list"})
	if err != nil {
		t.Fatalf("RunTask(list) 失败: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	if len(done) == 0 || done[len(done)-1].Line != "success" {
		t.Fatalf("task:done 事件缺失或非 success: %+v", done)
	}
	if len(logs) == 0 {
		t.Fatal("task:log 事件为空：spawn 输出未转发")
	}
	var joined strings.Builder
	for _, l := range logs {
		joined.WriteString(l.Line)
		joined.WriteString("\n")
	}
	for _, forbidden := range []string{"\x1b", "[ERR]"} {
		if strings.Contains(joined.String(), forbidden) {
			t.Errorf("输出含未剥离的 ANSI 或错误行: %q", joined.String())
		}
	}
	t.Logf("收到 %d 行日志", len(logs))
}
