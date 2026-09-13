package bindings

import "testing"

// phpbox log.sh 无条件输出 ANSI 颜色码（非 tty 不关闭），spawn 链路必须先剥离再分类
func TestStripANSI(t *testing.T) {
	cases := []struct{ in, want string }{
		{"\x1b[0;32m[OK]   备份完成\x1b[0m", "[OK]   备份完成"},
		{"\x1b[1;36m[INFO] 检查 Docker Engine…\x1b[0m", "[INFO] 检查 Docker Engine…"},
		{"\x1b[0;31m[ERR]  失败\x1b[0m", "[ERR]  失败"},
		{"无颜色行", "无颜色行"},
		{"", ""},
	}
	for _, c := range cases {
		if got := stripANSI(c.in); got != c.want {
			t.Errorf("stripANSI(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestClassify(t *testing.T) {
	cases := []struct {
		in      string
		wantCls string
	}{
		{"[OK]   备份完成: /x.tar.gz", "ok"},
		{"[ERR]  归档包含非法路径", "err"},
		{"[INFO] 检查依赖…", ""},
		{"NAMES  phpbox service", ""},
	}
	for _, c := range cases {
		ev := classify(c.in)
		if ev.Line != c.in || ev.Cls != c.wantCls {
			t.Errorf("classify(%q) = {%q, %q}, want {%q, %q}", c.in, ev.Line, ev.Cls, c.in, c.wantCls)
		}
	}
	// 剥离后必须能命中前缀（spawn 真实输出的组合路径）
	if ev := classify(stripANSI("\x1b[0;31m[ERR]  端口被占用\x1b[0m")); ev.Cls != "err" {
		t.Errorf("stripANSI+classify 未命中 [ERR] 前缀: %+v", ev)
	}
}

func TestTrimNL(t *testing.T) {
	cases := []struct{ in, want string }{
		{"line\n", "line"},
		{"line\r\n", "line"},
		{"line", "line"},
		{"\n", ""},
	}
	for _, c := range cases {
		if got := trimNL(c.in); got != c.want {
			t.Errorf("trimNL(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}
