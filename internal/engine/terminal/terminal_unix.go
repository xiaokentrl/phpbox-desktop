//go:build linux || darwin

// Package terminal · 嵌入式终端引擎（ui-spec §9 v2 项，依赖仅本地 PTY 提前落地——
// 同 i18n 从 v1.1 提前例：不需要引擎 server 模式，只造本地交互通道）。
// 交互式终端必须真 PTY：phpbox 的 confirm_yes 读 tty、docker/进度条检测 tty——
// 管道会让交互全部失效。这里只做进程与 IO 通道，不解不修饰字节（不隐藏终端真相）。
package terminal

import (
	"context"
	"os"
	"os/exec"

	"github.com/creack/pty"
)

// Session 一次终端会话：PTY master fd + 命令进程。
type Session struct {
	ptmx *os.File
	cmd  *exec.Cmd
}
// Start 用 PTY 启动 shell（默认 bash，缺失回退 sh；$SHELL 不信——桌面应用场景固定交互 shell）。
// dir 为空时继承当前目录。返回 master 端，读写字节原样进出自 proc/stdin 与屏幕。
func Start(ctx context.Context, dir string) (*Session, error) {
	shell := "bash"
	if _, err := exec.LookPath(shell); err != nil {
		shell = "sh"
	}
	cmd := exec.CommandContext(ctx, shell)
	cmd.Env = append(os.Environ(), "TERM=xterm-256color")
	if dir != "" {
		cmd.Dir = dir
	}
	ptmx, err := pty.Start(cmd)
	if err != nil {
		return nil, err
	}
	return &Session{ptmx: ptmx, cmd: cmd}, nil
}

// Read 从 PTY 读原始字节（终端输出：命令回显/交互提示/ANSI 全原样）。
func (s *Session) Read(p []byte) (int, error) { return s.ptmx.Read(p) }

// Write 向 PTY 写字节（用户键入；前端 xterm onData 原样转交，回显由伪终端行规程做）。
func (s *Session) Write(p []byte) (int, error) { return s.ptmx.Write(p) }

// Resize 通知 PTY 窗口尺寸变化（xterm cols/rows）。
func (s *Session) Resize(cols, rows int) error {
	return pty.Setsize(s.ptmx, &pty.Winsize{Cols: uint16(cols), Rows: uint16(rows)})
}

// WaitExit 等待 shell 退出（关闭 master 后调用），返回退出错误。
func (s *Session) WaitExit() error { return s.cmd.Wait() }

// Close 关闭 master 端（shell 收到 SIGHUP 自行退出）。
func (s *Session) Close() error { return s.ptmx.Close() }
