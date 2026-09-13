package diagbundle

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// 集成测试：本机真实采集（引擎+Docker 就绪环境全段有数据；缺什么记什么）。
// 核心断言：密码键脱敏（诊断包可能外发，脱敏失败是安全事故）。
func TestBuildRealAndSecretsMasked(t *testing.T) {
	home, _ := os.UserHomeDir()
	base := filepath.Join(home, "phpbox")
	out := Build(context.Background(), Sources{
		EnvFile:       filepath.Join(base, ".env"),
		SitesDir:      filepath.Join(base, "config", "nginx", "sites"),
		PhpConfigDir:  func(v string) string { return filepath.Join(base, "config", "php", v) },
		NginxPort:     80,
		LogTail:       5,
		IncludeSecrets: false,
	})
	// 段完整性：七段齐全（匹配段名前缀，缺段说明采集链断裂）
	for _, sec := range []string{"版本信息", "引擎就绪度", "容器状态", "容器日志", "站点（vhost", "PHP 扩展", "环境配置（.env"} {
		if !strings.Contains(out, "===== "+sec) {
			t.Errorf("诊断包缺段: %s", sec)
		}
	}
	// 脱敏断言：本机 .env 若含真实密码键（MYSQL_8_0_ROOT_PASSWORD 等），值绝不能出现
	envData, _ := os.ReadFile(filepath.Join(base, ".env"))
	for _, ln := range strings.Split(string(envData), "\n") {
		if i := strings.Index(ln, "="); i > 0 {
			k, v := ln[:i], strings.TrimSpace(ln[i+1:])
			if v != "" && isSecretKey(k) && len(v) > 3 {
				if strings.Contains(out, v) {
					t.Errorf("密码键 %s 的明文泄漏进诊断包", k)
				}
			}
		}
	}
	t.Logf("诊断包 %d 字节（本机真实采集）", len(out))
}

// isSecretKey 后缀匹配单元测试（宽松宁多勿漏）。
func TestIsSecretKey(t *testing.T) {
	for _, k := range []string{"MYSQL_8_0_ROOT_PASSWORD", "PGSQL_17_PASSWORD", "REDIS_8_ROOT_PASSWORD", "API_TOKEN", "MY_SECRET"} {
		if !isSecretKey(k) {
			t.Errorf("%q 应判定为敏感键", k)
		}
	}
	for _, k := range []string{"NGINX_PORT", "WWW_ROOT", "PHP_DEFAULT_EXTENSIONS", "CURRENT_UID"} {
		if isSecretKey(k) {
			t.Errorf("%q 不应判定为敏感键", k)
		}
	}
}
