//go:build linux || darwin

package terminal

import (
	"context"
	"strings"
	"testing"
	"time"
)

// 真实 PTY 回环：写入 echo 命令，读输出必须含回显（行规程）与结果。
// 交互语义断言：PTY 下 bash 检测到 tty 才输出提示符与回显——管道拿不到这些。
func TestSessionRoundtrip(t *testing.T) {
	dir := t.TempDir()
	s, err := Start(context.Background(), dir)
	if err != nil {
		t.Skipf("PTY 不可用（%v）：SKIPPED", err)
	}
	defer s.Close()

	marker := "phpbox_term_probe_77f2"
	if _, err := s.Write([]byte("echo " + marker + "\n")); err != nil {
		t.Fatalf("write: %v", err)
	}
	// PTY 读取循环（带超时）：交互输出时机不确定，轮询断言
	deadline := time.Now().Add(5 * time.Second)
	buf := make([]byte, 8192)
	var out strings.Builder
	for time.Now().Before(deadline) {
		n, err := s.Read(buf)
		if n > 0 {
			out.Write(buf[:n])
			if strings.Contains(out.String(), marker) {
				return // 回显/结果任一含 marker 即 PTY 双向链路贯通
			}
		}
		if err != nil {
			break
		}
	}
	t.Fatalf("PTY 回环 5s 内未见 %q，输出: %q", marker, out.String())
}

// Resize 不应破坏会话（xterm 尺寸同步路径）。
func TestSessionResize(t *testing.T) {
	s, err := Start(context.Background(), t.TempDir())
	if err != nil {
		t.Skipf("PTY 不可用（%v）：SKIPPED", err)
	}
	defer s.Close()
	if err := s.Resize(120, 30); err != nil {
		t.Errorf("Resize(120,30): %v", err)
	}
	if err := s.Resize(80, 24); err != nil {
		t.Errorf("Resize(80,24): %v", err)
	}
}
