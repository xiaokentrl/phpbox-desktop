package bindings

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// 注入隔离：状态文件指向临时目录，不碰真实 ~/phpbox/.task-state
func injectTaskState(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), ".task-state")
	orig := taskStateFile
	taskStateFile = func() string { return path }
	t.Cleanup(func() { taskStateFile = orig })
	return path
}

func TestTaskStateLifecycle(t *testing.T) {
	path := injectTaskState(t)

	saveTaskState("phpbox mysql install 8.0")
	st, err := readTaskState()
	if err != nil {
		t.Fatalf("readTaskState: %v", err)
	}
	if st.Cli != "phpbox mysql install 8.0" || st.Stage != "" || st.At == "" {
		t.Fatalf("save/read 回环字段缺失: %+v", st)
	}

	updateTaskStage("preparing")
	st, _ = readTaskState()
	if st.Stage != "preparing" || st.Cli != "phpbox mysql install 8.0" {
		t.Fatalf("阶段回写不应覆盖 cli/at: %+v", st)
	}

	clearTaskState()
	if _, err := readTaskState(); !os.IsNotExist(err) {
		t.Fatalf("clear 后文件应不存在: %v", err)
	}
	_ = path
}

// GetInterruptedTask 读取即清除（一次性通知）；无文件/损坏文件都按"无中断"返回。
func TestGetInterruptedTaskOneShot(t *testing.T) {
	injectTaskState(t)
	r := &Runner{}

	// 无记录
	if st, err := r.GetInterruptedTask(); err != nil || st.Cli != "" {
		t.Fatalf("无记录应返回零值: %+v %v", st, err)
	}

	// 有记录：返回后清除，二次调用为空
	writeTaskState(InterruptedTask{Cli: "phpbox php install 8.4", Stage: "preparing", At: "2026-09-14T08:00:00+08:00"})
	st, err := r.GetInterruptedTask()
	if err != nil || st.Cli != "phpbox php install 8.4" || st.Stage != "preparing" {
		t.Fatalf("中断记录读取失败: %+v %v", st, err)
	}
	if st2, _ := r.GetInterruptedTask(); st2.Cli != "" {
		t.Fatalf("读取即清除失败，二次仍返回: %+v", st2)
	}

	// 损坏 JSON：按无中断处理且清除残留
	if err := os.WriteFile(taskStateFile(), []byte("{broken"), 0o644); err != nil {
		t.Fatal(err)
	}
	if st, err := r.GetInterruptedTask(); err != nil || st.Cli != "" {
		t.Fatalf("损坏文件应按无中断返回: %+v %v", st, err)
	}
	if _, err := os.Stat(taskStateFile()); !os.IsNotExist(err) {
		t.Fatal("损坏文件应被清除")
	}
}

// RunTask 完成后状态文件必须清除（成功任务不留中断痕迹）——真实 spawn 集成。
// phpbox 不在 PATH 时 SKIPPED（环境惯例）。
func TestRunTaskClearsState(t *testing.T) {
	if _, err := exec.LookPath("phpbox"); err != nil {
		t.Skip("phpbox CLI 不在 PATH：SKIPPED")
	}
	path := injectTaskState(t)

	orig := emitEvent
	emitEvent = func(name string, data TaskEvent) {} // 测试环境无 Wails app，静默
	defer func() { emitEvent = orig }()

	r := &Runner{}
	if err := r.RunTask(context.Background(), []string{"list"}); err != nil {
		t.Fatalf("RunTask(list) 失败: %v", err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("任务成功完成后状态文件应已清除（残留会让下次启动误报中断）")
	}
}
