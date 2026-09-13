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

	// daemon 长驻进程槽（daemon.go）：与任务队列独立
	daemonMu     sync.Mutex
	daemonID     string
	daemonCancel context.CancelFunc
}

// TaskEvent 任务事件（前端 EventsOn 消费）。
type TaskEvent struct {
	Line string `json:"line"`
	Cls  string `json:"cls"`
	// Phase 阶段日志推断（ui-spec §8：九态的降级实现——bash 引擎无原生状态事件，
	// 由日志行推断；推断失败留空，前端显示原始日志不臆造状态）
	Phase string `json:"phase,omitempty"`
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
	ev := TaskEvent{Line: line}
	switch {
	case strings.HasPrefix(line, "[OK]"):
		ev.Cls = "ok"
	case strings.HasPrefix(line, "[ERR]"):
		ev.Cls = "err"
	}
	ev.Phase = inferPhase(line)
	return ev
}

// phaseRules 模式→阶段映射表（ui-spec §8 契约："可配置映射表，bash 输出演进时
// 改表不改码"）。模式全部来自 bash 仓真实日志词汇的实测采集（grep 核对，非想象）：
//   - "初始化 X 配置" / "X 配置就绪"（install.sh:92/104）→ configured 前置
//   - "镜像构建成功" / "Docker 构建开始"（php/install.sh:93/102）→ preparing
//   - "X 安装完成"（install.sh:163 / php/install.sh:76）→ committed（healthy 由
//     容器状态派生，不经日志）
//   - "安装失败，自动清理"（install.sh:47）→ rolling_back
//   - "已卸载"（php/uninstall.sh:116）→ absent
// bash 冻结标记集实测仅 [INFO]/[OK]/[ERR]（log.sh）——无 [WARN]，不引入。
type phaseRule struct {
	re    *regexp.Regexp
	phase string
}

var phaseRules = []phaseRule{
	{regexp.MustCompile(`初始化 .* 配置|配置就绪`), "configured"},
	{regexp.MustCompile(`Docker 构建开始|镜像构建|离线构建|在线构建`), "preparing"},
	{regexp.MustCompile(`安装完成|端口已改为`), "committed"},
	{regexp.MustCompile(`安装失败.*清理|端口变更失败`), "rolling_back"},
	{regexp.MustCompile(`已卸载`), "absent"},
}

// inferPhase 阶段推断：首个命中返回；无命中返回空串（前端显示原始日志，不臆造）。
func inferPhase(line string) string {
	for _, r := range phaseRules {
		if r.re.MatchString(line) {
			return r.phase
		}
	}
	return ""
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
