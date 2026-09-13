package bindings

import (
	"os/exec"
	"strings"
	"sync"
	"testing"
	"time"
)

// collectDaemon 收集 daemon 事件（注入替身），返回停止收集的函数。
func collectDaemon() (func() (logs []DaemonEvent, states []DaemonState), func()) {
	var mu sync.Mutex
	var logs []DaemonEvent
	var states []DaemonState
	orig := emitDaemon
	emitDaemon = func(name string, data any) {
		mu.Lock()
		defer mu.Unlock()
		switch d := data.(type) {
		case DaemonEvent:
			if name == "daemon:log" {
				logs = append(logs, d)
			}
		case DaemonState:
			if name == "daemon:state" {
				states = append(states, d)
			}
		}
	}
	get := func() ([]DaemonEvent, []DaemonState) {
		mu.Lock()
		defer mu.Unlock()
		return logs, states
	}
	restore := func() { emitDaemon = orig }
	return get, restore
}

// 集成测试：daemon 全生命周期——异步启动立即返回、state 事件 running→exited、
// 槽位释放后可再次启动。phpbox 不在 PATH 时 SKIPPED。
func TestDaemonLifecycle(t *testing.T) {
	if _, err := exec.LookPath("phpbox"); err != nil {
		t.Skip("phpbox CLI 不在 PATH：SKIPPED")
	}
	get, restore := collectDaemon()
	defer restore()

	r := &Runner{}
	if err := r.StartDaemon("test:list", []string{"list"}); err != nil {
		t.Fatalf("StartDaemon 失败: %v", err)
	}
	if r.DaemonRunning() != "test:list" {
		t.Errorf("启动后槽位应为 test:list，得到 %q", r.DaemonRunning())
	}
	// 单槽互斥：运行中再启动必须被拒
	if err := r.StartDaemon("other", []string{"list"}); err == nil {
		t.Error("单槽互斥失效：第二个 daemon 未被拒绝")
	}

	// phpbox list 秒退：等槽位释放（上限 10s）
	deadline := time.Now().Add(10 * time.Second)
	for r.DaemonRunning() != "" && time.Now().Before(deadline) {
		time.Sleep(100 * time.Millisecond)
	}
	if r.DaemonRunning() != "" {
		t.Fatal("daemon 退出后槽位未释放")
	}

	logs, states := get()
	if len(states) == 0 {
		t.Fatal("未收到任何 daemon:state 事件")
	}
	if states[0].ID != "test:list" || !states[0].Running {
		t.Errorf("首个 state 应为 running: %+v", states[0])
	}
	last := states[len(states)-1]
	if last.Running {
		t.Errorf("末个 state 应为退出态: %+v", last)
	}
	if last.Failed {
		t.Errorf("phpbox list 正常退出不应标记失败: %+v", last)
	}
	joined := strings.Join(linesOf(logs), "\n")
	if strings.Contains(joined, "\x1b") || strings.Contains(joined, "[ERR]") {
		t.Errorf("daemon 输出未剥离 ANSI 或含错误: %q", joined)
	}
}

func TestDaemonStopMismatch(t *testing.T) {
	r := &Runner{}
	if err := r.StopDaemon("nope"); err == nil {
		t.Error("空槽位 StopDaemon 应报错")
	}
}

func linesOf(logs []DaemonEvent) []string {
	out := make([]string, len(logs))
	for i, l := range logs {
		out[i] = l.Line
	}
	return out
}
