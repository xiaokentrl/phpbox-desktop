//go:build linux || darwin

package bindings

import (
	"strings"
	"sync"
	"testing"
	"time"
)

// 终端绑定链路集成：TerminalStart → PTY → pump → emitTerminalData 替身收流，
// TerminalWrite 键入 → 回显；TerminalStop 关槽。复用注入惯例（emitEvent/emitDaemon 同款）。
// 集成前提：runner_test.go 的 injectTaskState 若在同测试二进制运行会互不影响
// （全局 termSession 槽是本测试私有状态，测试串行语义由 -p 包级保证）。
func TestTerminalBindingRoundtrip(t *testing.T) {
	var mu sync.Mutex
	var out strings.Builder
	orig := emitTerminalData
	emitTerminalData = func(data string) {
		mu.Lock()
		defer mu.Unlock()
		out.WriteString(data)
	}
	defer func() { emitTerminalData = orig }()

	tm := &Terminal{}
	if err := tm.TerminalStart(t.TempDir()); err != nil {
		t.Skipf("PTY 不可用（%v）：SKIPPED", err)
	}

	// 键入命令：经 PTY 行规程回显 + bash 执行结果，双向都必须可见
	marker := "term_binding_probe_5d3c"
	if err := tm.TerminalWrite("echo " + marker + "\n"); err != nil {
		t.Fatal(err)
	}
	ok := false
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) && !ok {
		mu.Lock()
		ok = strings.Contains(out.String(), marker)
		mu.Unlock()
		if !ok {
			time.Sleep(50 * time.Millisecond)
		}
	}
	if !ok {
		mu.Lock()
		got := out.String()
		mu.Unlock()
		t.Fatalf("PTY 链路 5s 内未见 marker 回显/结果，输出: %q", got)
	}

	// Resize 不报错（xterm 尺寸同步路径）
	if err := tm.TerminalResize(100, 30); err != nil {
		t.Errorf("TerminalResize: %v", err)
	}

	if err := tm.TerminalStop(); err != nil {
		t.Fatalf("TerminalStop: %v", err)
	}
	// 槽位释放：二次 Stop 应明确报"没有会话"
	if err := tm.TerminalStop(); err == nil || !strings.Contains(err.Error(), "没有") {
		t.Errorf("双 Stop 应报无会话，得到: %v", err)
	}

	// 会话终结标记必须广播（前端依赖它显示会话结束）。
	// 注意不能持锁等：emit 替身也要拿这把锁，持锁 sleep = 泵饿死（首轮失败根因）
	deadline = time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		mu.Lock()
		done := strings.Contains(out.String(), "会话结束")
		mu.Unlock()
		if done {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Error("未见 [会话结束] 终结标记（Stop 后 3s 内）")
}

// 单槽语义：Start 后未 Stop 再 Start 必须拒绝（daemon 同款防打架）。
func TestTerminalSingleSlot(t *testing.T) {
	tm := &Terminal{}
	if err := tm.TerminalStart(""); err != nil {
		t.Skipf("PTY 不可用（%v）：SKIPPED", err)
	}
	defer tm.TerminalStop() //nolint:errcheck
	if err := tm.TerminalStart(""); err == nil || !strings.Contains(err.Error(), "已在运行") {
		t.Errorf("单槽下二次 Start 应拒绝，得到: %v", err)
	}
	if !tm.TerminalRunning() {
		t.Error("TerminalRunning 应为 true")
	}
}
