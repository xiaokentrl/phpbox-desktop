// Package bindings · spawn runner：阶段 0 桌面壳通过 spawn bash phpbox CLI 驱动引擎。
// 输出经 Wails 事件流推送到前端任务抽屉；退出码经回调返回。
package bindings

import (
	"bufio"
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
	"sync"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// Runner spawn phpbox CLI 并流式转发输出。
type Runner struct {
	mu      sync.Mutex
	running bool
}

// TaskEvent 任务事件（前端 EventsOn 消费）。
type TaskEvent struct {
	Line string `json:"line"`
	Cls  string `json:"cls"`
}

// emitViaWails 默认事件发射：经 Wails 应用事件总线；无运行中的应用时静默丢弃。
// emitEvent 为可替换边界：集成测试注入替身后可脱离 GUI 验证 spawn 链路（§6.2）。
var emitEvent func(name string, data TaskEvent) = emitViaWails

func emitViaWails(name string, data TaskEvent) {
	if app := application.Get(); app != nil {
		app.Event.Emit(name, data)
	}
}

func classify(line string) TaskEvent {
	switch {
	case strings.HasPrefix(line, "[OK]"):
		return TaskEvent{Line: line, Cls: "ok"}
	case strings.HasPrefix(line, "[ERR]"):
		return TaskEvent{Line: line, Cls: "err"}
	default:
		return TaskEvent{Line: line}
	}
}

// ansiRe 匹配 ANSI 转义序列（phpbox log() 无条件输出颜色码，非 tty 管道也不关闭）
var ansiRe = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)

func stripANSI(s string) string {
	return ansiRe.ReplaceAllString(s, "")
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

	cmd := exec.CommandContext(ctx, "phpbox", args...)
	stdout, _ := cmd.StdoutPipe()
	stderr, _ := cmd.StderrPipe()
	if err := cmd.Start(); err != nil {
		emitEvent("task:log", TaskEvent{Line: "[ERR] " + err.Error(), Cls: "err"})
		emitEvent("task:done", TaskEvent{Line: "failed", Cls: "err"})
		return err
	}

	var wg sync.WaitGroup
	scan := func(rd *bufio.Reader) {
		defer wg.Done()
		for {
			line, err := rd.ReadString('\n')
			if line != "" {
				clean := stripANSI(trimNL(line))
				if clean != "" {
					emitEvent("task:log", classify(clean))
				}
				if err != nil {
					break
				}
				continue
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
		emitEvent("task:done", TaskEvent{Line: "failed", Cls: "err"})
		return err
	}
	emitEvent("task:done", TaskEvent{Line: "success", Cls: "ok"})
	return nil
}

func trimNL(s string) string {
	for len(s) > 0 && (s[len(s)-1] == '\n' || s[len(s)-1] == '\r') {
		s = s[:len(s)-1]
	}
	return s
}
