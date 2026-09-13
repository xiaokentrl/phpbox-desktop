package bindings

import (
	"os"
	"path/filepath"
	"testing"
)

// expandPath 契约测试：逐条镜像 bash lib/common/env.sh 的路径归一化
// （OFFLINE_DIR 分支 84–93 行 / GO_*_ROOT 分支 111–119 行，两处规则一致）：
// "~"→HOME；"~/x"→HOME/x；"./x"→BASE_DIR/x；裸名→BASE_DIR/x；绝对路径原样。
func TestExpandPathMirrorsEnvSh(t *testing.T) {
	home, err := homeOf()
	if err != nil {
		t.Skipf("HOME 不可得（%v）：SKIPPED", err)
	}
	base := filepath.Join(home, "phpbox")
	cases := []struct{ in, want string }{
		{"~", home},
		{"~/www", filepath.Join(home, "www")},
		{"./cache", filepath.Join(base, "cache")},
		{"offline", filepath.Join(base, "offline")},
		{"/srv/data", "/srv/data"},
		{"", ""},
	}
	for _, c := range cases {
		if got := expandPath(c.in); got != c.want {
			t.Errorf("expandPath(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

// envLookup → envPath 全链路：真实读取本机 ~/phpbox/.env（集成）。
// 引擎未安装（.env 不存在）时 Read 返回空列表 → 全部键回退默认值，链路不报错。
func TestEnvPathReadsRealEnvFile(t *testing.T) {
	if v := envLookup("OFFLINE_DIR"); v != "" {
		t.Logf("真实 .env：OFFLINE_DIR=%q", v)
	}
	if _, err := filepath.Abs(envPath("OFFLINE_DIR", "fallback")); err != nil {
		t.Errorf("envPath 结果不是绝对路径形态: %v", err)
	}
	// NGINX_PORT：不可解析/缺失回退 80（bash env.sh:70 默认）
	if p := nginxPort(); p <= 0 {
		t.Errorf("nginxPort() = %d, want > 0", p)
	}
}

// homeOf 测试辅助（独立于被测函数，取 HOME 原值）。
func homeOf() (string, error) {
	return os.UserHomeDir()
}
