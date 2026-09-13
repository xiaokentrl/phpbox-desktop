// Package bindings · daemon 长驻进程槽：go run / go logs 等永不返回的命令通道。
// 与 RunTask（单任务队列，Wait 到退出）分开：StartDaemon 异步启动立即返回，
// 输出持续经 daemon:log 事件推送；StopDaemon 取消 context 结束进程。
// 单槽设计（与任务单队列同哲学）：同时至多一个 daemon，避免本地开发多进程打架。
package bindings

import (
	"bufio"
	"context"
	"fmt"
	"os/exec"
	"sync"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// DaemonEvent daemon 输出行事件（daemon:log）。
type DaemonEvent struct {
	ID   string `json:"id"`
	Line string `json:"line"`
	Cls  string `json:"cls"`
}

// DaemonState daemon 状态事件（daemon:state）：Running=false 且 Failed=true 表示异常退出。
type DaemonState struct {
	ID      string `json:"id"`
	Running bool   `json:"running"`
	Failed  bool   `json:"failed"`
}

// emitDaemon 与 emitEvent 同构的可注入边界（集成测试替身）。
var emitDaemon func(name string, data any) = emitDaemonViaWails

func emitDaemonViaWails(name string, data any) {
	if app := application.Get(); app != nil {
		app.Event.Emit(name, data)
	}
}

// StartDaemon 异步启动长驻命令并立即返回。id 为调用方给的进程标识（前端路由用）。
// 已有 daemon 运行时拒绝；进程退出（自然/取消）后槽位释放并广播 daemon:state。
func (r *Runner) StartDaemon(id string, args []string) error {
	r.daemonMu.Lock()
	if r.daemonID != "" {
		id := r.daemonID
		r.daemonMu.Unlock()
		return fmt.Errorf("长驻进程 %s 正在运行，请先停止", id)
	}
	ctx, cancel := context.WithCancel(context.Background())
	r.daemonID, r.daemonCancel = id, cancel
	r.daemonMu.Unlock()

	cmd := exec.CommandContext(ctx, "phpbox", args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		r.releaseDaemon()
		return err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		r.releaseDaemon()
		return err
	}
	if err := cmd.Start(); err != nil {
		r.releaseDaemon()
		emitDaemon("daemon:state", DaemonState{ID: id, Failed: true})
		return err
	}
	emitDaemon("daemon:state", DaemonState{ID: id, Running: true})

	go func() {
		var wg sync.WaitGroup
		scan := func(rd *bufio.Reader) {
			defer wg.Done()
			for {
				line, err := rd.ReadString('\n')
				if line != "" {
					clean := stripANSI(trimNL(line))
					if clean != "" {
						ev := classify(clean)
						emitDaemon("daemon:log", DaemonEvent{ID: id, Line: ev.Line, Cls: ev.Cls})
					}
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

		failed := ctx.Err() == nil && cmd.Wait() != nil
		// ctx 已取消（用户主动停止）不算失败
		r.releaseDaemon()
		emitDaemon("daemon:state", DaemonState{ID: id, Failed: failed})
	}()
	return nil
}

// StopDaemon 停止运行中的 daemon。id 不匹配（槽位空或他属）时报错而非误杀。
func (r *Runner) StopDaemon(id string) error {
	r.daemonMu.Lock()
	defer r.daemonMu.Unlock()
	if r.daemonID == "" {
		return fmt.Errorf("没有运行中的长驻进程")
	}
	if r.daemonID != id {
		return fmt.Errorf("长驻进程是 %s，不是 %s", r.daemonID, id)
	}
	r.daemonCancel()
	return nil
}

// DaemonRunning 返回当前 daemon 标识（空串=无）。
func (r *Runner) DaemonRunning() string {
	r.daemonMu.Lock()
	defer r.daemonMu.Unlock()
	return r.daemonID
}

// releaseDaemon 释放槽位（启动失败路径与退出路径共用）。
func (r *Runner) releaseDaemon() {
	r.daemonMu.Lock()
	r.daemonID, r.daemonCancel = "", nil
	r.daemonMu.Unlock()
}
