// Package diagbundle · 诊断包导出（ui-spec §10：日志+配置+版本信息，归宿 Go 引擎）。
// 消费方：诊断面板 §5.7 "未知故障 → 导出诊断包" 分支。
// 原则：全部真实数据（Docker API / 真实文件），敏感键脱敏（密码类只报存在不报值）。
package diagbundle

import (
	"context"
	"fmt"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/xiaokentrl/phpbox-desktop/internal/engine/docker"
	"github.com/xiaokentrl/phpbox-desktop/internal/engine/env"
	"github.com/xiaokentrl/phpbox-desktop/internal/engine/health"
	phpengine "github.com/xiaokentrl/phpbox-desktop/internal/engine/php"
	"github.com/xiaokentrl/phpbox-desktop/internal/engine/presence"
	"github.com/xiaokentrl/phpbox-desktop/internal/engine/site"
)

// Sources 诊断包采集的真实数据源（由绑定层注入路径，保持引擎可测试）。
type Sources struct {
	EnvFile       string // ~/phpbox/.env
	SitesDir      string // ~/phpbox/config/nginx/sites
	PhpConfigDir  func(version string) string // config/php/<ver>（extensions.env 所在）
	NginxPort     int                         // 健康探测端口（.env NGINX_PORT，默认 80）
	LogTail       int                         // 每容器日志尾行数（默认 30）
	IncludeSecrets bool                       // 恒为 false：密码键脱敏在 build 内强制执行
}

// secretKeys 密码类 .env 键（bash cli.sh：MYSQL_/PGSQL_<ver>_ROOT_PASSWORD、REDIS_<ver>_ROOT_PASSWORD）。
var secretKeySuffixes = []string{"PASSWORD", "PASSWORD_FILE", "SECRET", "TOKEN"}

// isSecretKey 键名以敏感后缀结尾即脱敏（宽松匹配宁多勿漏：诊断包可能外发）。
func isSecretKey(k string) bool {
	ku := strings.ToUpper(k)
	for _, s := range secretKeySuffixes {
		if strings.HasSuffix(ku, s) {
			return true
		}
	}
	return false
}

// Build 生成诊断包文本（各段失败不中断——诊断包的价值恰在"带病快照"，失败段原样记错误）。
func Build(ctx context.Context, src Sources) string {
	if src.LogTail <= 0 || src.LogTail > 200 {
		src.LogTail = 30
	}
	var b strings.Builder
	sec := func(name string) { fmt.Fprintf(&b, "\n===== %s =====\n", name) }

	sec("版本信息")
	fmt.Fprintf(&b, "生成时间: %s\n", time.Now().Format(time.RFC3339))
	fmt.Fprintf(&b, "phpbox Desktop (Go %s %s/%s)\n", runtime.Version(), runtime.GOOS, runtime.GOARCH)

	sec("引擎就绪度")
	st := presence.Detect(ctx)
	fmt.Fprintf(&b, "EngineDir(~/phpbox): %v\n", st.EngineDir)
	fmt.Fprintf(&b, "CLI in PATH: %s\n", orDash(st.CliInPath))
	fmt.Fprintf(&b, ".env exists: %v\n", st.HasEnv)
	fmt.Fprintf(&b, "Docker: %v\n", st.DockerOK)
	if st.DockerErr != "" {
		fmt.Fprintf(&b, "Docker error: %s\n", st.DockerErr)
	}

	sec("容器状态")
	var containers []docker.ContainerSummary
	if dc, err := docker.New(); err != nil {
		fmt.Fprintf(&b, "Docker API 不可达: %v\n", err)
	} else if items, err := dc.ListContainers(ctx); err != nil {
		fmt.Fprintf(&b, "容器列举失败: %v\n", err)
	} else {
		containers = items
		for _, c := range items {
			fmt.Fprintf(&b, "%-12s %-8s %-14s %s\n", strings.TrimPrefix(c.Name, "/"), c.State, c.Service, c.Version)
		}
	}

	sec("容器日志（每容器最后 30 行）")
	if len(containers) == 0 {
		b.WriteString("（无容器或 Docker 不可达）\n")
	} else if dc, err := docker.New(); err != nil {
		fmt.Fprintf(&b, "Docker API 不可达: %v\n", err)
	} else {
		for _, c := range containers {
			name := strings.TrimPrefix(c.Name, "/")
			fmt.Fprintf(&b, "\n--- %s ---\n", name)
			lines, err := dc.ContainerLogs(ctx, name, src.LogTail)
			if err != nil {
				fmt.Fprintf(&b, "日志读取失败: %v\n", err)
				continue
			}
			if len(lines) == 0 {
				b.WriteString("（无输出）\n")
			}
			for _, ln := range lines {
				b.WriteString(ln + "\n")
			}
		}
	}

	sec("站点（vhost 真实解析）")
	if sites, err := site.List(src.SitesDir); err != nil {
		fmt.Fprintf(&b, "站点读取失败: %v\n", err)
	} else if len(sites) == 0 {
		b.WriteString("（无站点）\n")
	} else {
		for _, s := range sites {
			fmt.Fprintf(&b, "%-24s php=%-8s hosts=%-6v root=%s\n", s.Domain, s.PHP, s.Hosts, s.Root)
		}
		b.WriteString("\n健康探测:\n")
		port := src.NginxPort
		if port <= 0 {
			port = 80
		}
		for _, s := range sites {
			r := health.Probe(s.Domain, port, 3*time.Second)
			fmt.Fprintf(&b, "%-24s %s code=%d err=%s\n", s.Domain, r.Status, r.Code, orDash(r.Err))
		}
	}

	sec("PHP 扩展（extensions.env 真实内容）")
	if vers, ok := phpVersionsFrom(containers); ok {
		for _, v := range vers {
			dir := src.PhpConfigDir(v)
			exts, err := phpengine.ReadExtensions(filepath.Join(dir, "extensions.env"))
			if err != nil {
				fmt.Fprintf(&b, "php %s: 读取失败 %v\n", v, err)
				continue
			}
			fmt.Fprintf(&b, "php %s: %s\n", v, strings.Join(exts, ","))
		}
	} else {
		b.WriteString("（无 phpbox 管理的 PHP 容器）\n")
	}

	sec("环境配置（.env，密码键脱敏）")
	if kvs, err := env.Read(src.EnvFile); err != nil {
		fmt.Fprintf(&b, ".env 读取失败: %v\n", err)
	} else if len(kvs) == 0 {
		b.WriteString("（.env 不存在或为空）\n")
	} else {
		for _, kv := range kvs {
			if isSecretKey(kv.Key) {
				fmt.Fprintf(&b, "%s=***（%d 字符，已脱敏）\n", kv.Key, len(kv.Value))
			} else {
				fmt.Fprintf(&b, "%s=%s\n", kv.Key, kv.Value)
			}
		}
	}

	sec("结束")
	b.WriteString("诊断包由 phpbox Desktop 生成：数据来自 Docker Engine API 与真实文件，未含任何 GUI 状态。\n")
	return b.String()
}

// phpVersionsFrom 从容器表提取 PHP 版本（labels 事实源）；无 php 容器返回 false。
func phpVersionsFrom(containers []docker.ContainerSummary) ([]string, bool) {
	seen := map[string]bool{}
	for _, c := range containers {
		if c.Service == "php" && c.Version != "" {
			seen[c.Version] = true
		}
	}
	if len(seen) == 0 {
		return nil, false
	}
	out := make([]string, 0, len(seen))
	for v := range seen {
		out = append(out, v)
	}
	sort.Strings(out)
	return out, true
}

func orDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}
