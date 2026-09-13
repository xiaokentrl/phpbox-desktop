// Package bindings · spawn runner：阶段 0 桌面壳通过 spawn bash phpbox CLI 驱动引擎。
// 输出经 Wails 事件流推送到前端任务抽屉；退出码经回调返回。
package bindings

import (
	"bufio"
	"context"
	"fmt"
	"os/exec"
	"sync"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// Runner spawn phpbox CLI 并流式转发输出。
type Runner struct {
	mu   sync.Mutex
	running bool
}

// TaskEvent 任务事件（前端 EventsOn 消费）。
type TaskEvent struct {
	Line string `json:"line"`
	Cls  string `json:"cls"`
}

func classify(line string) TaskEvent {
	switch {
	case len(line) >= 4 && line[:4] == "[OK]":
		return TaskEvent{Line: line, Cls: "ok"}
	case len(line) >= 5 && line[:5] == "[ERR]":
		return TaskEvent{Line: line, Cls: "err"}
	default:
		return TaskEvent{Line: line}
	}
}

// RunTask spawn bash phpbox CLI 并流式返回输出（事件名 task:log / task:done）。
func (r *Runner) RunTask(ctx context.Context, args []string) error {
	r.mu.Lock()
	if r.running {
		r.mu.Unlock()
		return fmt.Errorf("有任务正在运行")
	}
	r.running = true
	r.mu.Unlock()
	defer func() { r.mu.Lock(); r.running = false; r.mu.Unlock() }()

	app := application.Get()
	cmd := exec.CommandContext(ctx, "phpbox", args...)
	stdout, _ := cmd.StdoutPipe()
	stderr, _ := cmd.StderrPipe()
	if err := cmd.Start(); err != nil {
		app.Event.Emit("task:log", TaskEvent{Line: "[ERR] " + err.Error(), Cls: "err"})
		app.Event.Emit("task:done", TaskEvent{Line: "failed", Cls: "err"})
		return err
	}

	var wg sync.WaitGroup
	scan := func(rd *bufio.Reader) {
		defer wg.Done()
		for {
			line, err := rd.ReadString('\n')
			if line != "" {
				ev := classify(trimNL(line))
				app.Event.Emit("task:log", ev)
			}
			if err != nil {
				break
			}
		}
	}
	wg.Add(2)
	go scan(bufio.NewReader(stdout))
	go scan(bufio.NewReader(stderr))
	wg.Wait()

	if err := cmd.Wait(); err != nil {
		app.Event.Emit("task:done", TaskEvent{Line: "failed", Cls: "err"})
		return err
	}
	app.Event.Emit("task:done", TaskEvent{Line: "success", Cls: "ok"})
	return nil
}

func trimNL(s string) string {
	for len(s) > 0 && (s[len(s)-1] == '\n' || s[len(s)-1] == '\r') {
		s = s[:len(s)-1]
	}
	return s
}
