// Package bindings · 嵌入式终端绑定（ui-spec §9 v2 项提前落地，依赖仅本地 PTY）。
// 单槽（与 daemon 同哲学）；输入经 TerminalWrite 原样写 PTY，输出字节流经
// terminal:data 事件推送（不解不修饰——不隐藏终端真相），resize 由前端驱动。
// Windows 平台 StartTerminal 返回明确错误（engine stub），前端诚实降级。
package bindings

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/wailsapp/wails/v3/pkg/application"

	"github.com/xiaokentrl/phpbox-desktop/internal/engine/terminal"
)

// Terminal 暴露嵌入式终端能力。
type Terminal struct{}

// termMu 保护单槽；termSession 为 nil 表示无会话。
var (
	termMu      sync.Mutex
	termSession *terminal.Session
	termCtx     context.CancelFunc
)

// TerminalStart 启动终端会话（$SHELL 固定为 bash/sh + TERM=xterm-256color）。
// 已有会话时返回错误（前端应先 StopTerminal——同 daemon 单槽语义）。
func (t *Terminal) TerminalStart(dir string) error {
	termMu.Lock()
	defer termMu.Unlock()
	if termSession != nil {
		return fmt.Errorf("终端会话已在运行")
	}
	ctx, cancel := context.WithCancel(context.Background())
	s, err := terminal.Start(ctx, dir)
	if err != nil {
		cancel()
		log.Printf("[绑定] Terminal.TerminalStart 失败: %v", err)
		return err
	}
	termSession, termCtx = s, cancel
	go pumpTerminal(s)
	log.Printf("[绑定] Terminal.TerminalStart → 会话已启动（dir=%q）", dir)
	return nil
}

// pumpTerminal 输出泵：PTY → terminal:data 事件（原字节转字符串；会话关闭即停止）。
func pumpTerminal(s *terminal.Session) {
	buf := make([]byte, 8192)
	for {
		n, err := s.Read(buf)
		if n > 0 {
			if app := application.Get(); app != nil {
				app.Event.Emit("terminal:data", string(buf[:n]))
			}
		}
		if err != nil {
			break
		}
	}
	// shell 退出/读端关闭：清槽 + 广播终结（前端显示会话结束，不静默吞）
	termMu.Lock()
	termSession, termCtx = nil, nil
	termMu.Unlock()
	if app := application.Get(); app != nil {
		app.Event.Emit("terminal:data", "\x1b[?25h\r\n\u001b[38;5;240m[会话结束]\u001b[0m\r\n")
	}
}

// TerminalWrite 用户键入原样写入 PTY（回显由行规程做，绑定不加工）。
func (t *Terminal) TerminalWrite(data string) error {
	termMu.Lock()
	s := termSession
	termMu.Unlock()
	if s == nil {
		return fmt.Errorf("没有运行中的终端会话")
	}
	_, err := s.Write([]byte(data))
	return err
}

// TerminalResize xterm cols/rows 变化同步 PTY 窗口尺寸。
func (t *Terminal) TerminalResize(cols, rows int) error {
	termMu.Lock()
	s := termSession
	termMu.Unlock()
	if s == nil {
		return fmt.Errorf("没有运行中的终端会话")
	}
	return s.Resize(cols, rows)
}

// TerminalStop 关闭会话（master 关闭 → shell 收 SIGHUP 退出，输出泵自然结束清槽）。
func (t *Terminal) TerminalStop() error {
	termMu.Lock()
	s, cancel := termSession, termCtx
	termSession, termCtx = nil, nil
	termMu.Unlock()
	if s == nil {
		return fmt.Errorf("没有运行中的终端会话")
	}
	if cancel != nil {
		cancel()
	}
	err := s.Close()
	log.Printf("[绑定] Terminal.TerminalStop → 会话已关闭")
	return err
}

// TerminalRunning 是否有活动会话（前端切路由时恢复 UI 状态用）。
func (t *Terminal) TerminalRunning() bool {
	termMu.Lock()
	defer termMu.Unlock()
	return termSession != nil
}
