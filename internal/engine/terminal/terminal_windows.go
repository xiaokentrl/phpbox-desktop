//go:build windows

// Package terminal · Windows 形态：PTY 依赖 creack/pty 不支持 windows，
// 绑定层暴露的三方法在本平台返回明确错误（诚实降级：GUI 提示终端不可用，
// 引导用系统终端 + CLI 兜底命令），不假装有终端。
package terminal

import (
	"context"
	"errors"
	"os/exec"
)

// Session 与 unix 版同名，字段仅占位。
type Session struct{}

var errUnsupported = errors.New("嵌入式终端在 Windows 暂不可用（PTY 未接入）——请使用系统终端运行 phpbox 命令")

func Start(ctx context.Context, dir string) (*Session, error) {
	_ = exec.LookPath // 保持与 unix 文件同构的依赖面
	return nil, errUnsupported
}

func (s *Session) Read(p []byte) (int, error)  { return 0, errUnsupported }
func (s *Session) Write(p []byte) (int, error) { return 0, errUnsupported }
func (s *Session) Resize(cols, rows int) error  { return errUnsupported }
func (s *Session) WaitExit() error              { return errUnsupported }
func (s *Session) Close() error                  { return errUnsupported }
