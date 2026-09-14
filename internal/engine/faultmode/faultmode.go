// Package faultmode · 已知故障模式库（ui-spec §8.1）阶段 0 降级形态：
// 只读检测（容器状态 + 日志关键词匹配）+ 真实 CLI 兜底命令。
// 不做一键修复——bash CLI 无修复类子命令（sock 清理/属主治愈均不存在），
// 修复属 v1.1 Go 引擎期（AGENTS 阶段边界）；GUI 只给事实与 CLI 出路，不假装能修。
// 模式表来自 ui-spec §8.1（phpbox 真实踩坑经验产品化）。
package faultmode

import (
	"context"
	"strings"

	"github.com/xiaokentrl/phpbox-desktop/internal/engine/docker"
)

// Mode 一个已知故障模式的检测定义。
type Mode struct {
	ID     string   `json:"id"`     // 稳定标识（前端徽章/跳转用）
	Title  string   `json:"title"`  // 模式名（中英由前端 i18n 按标题键处理——传固定 zh 键）
	Match  []string `json:"match"`  // 日志关键词（任一命中即匹配；全部小写，匹配时统一 lower）
	Scopes []string `json:"scopes"` // 受影响服务线（提示用，如 mysql/pgsql/nginx/php）
	Cli    string   `json:"cli"`    // 真实 CLI 兜底命令（grep 核实存在的子命令组合）
}

// Hit 一次检测结果：模式 + 命中的容器与命中行（证据，不是猜测）。
type Hit struct {
	Mode      Mode     `json:"mode"`
	Container string   `json:"container"` // 容器名
	Lines     []string `json:"lines"`      // 命中的日志行（截断到最多 3 行）
}

// modes §8.1 模式表（CLI 兜底全部对 bash cli.sh 实测签名核实）。
// 阶段 0 检测面：容器重启循环（State= restarting）+ 日志关键词。
var modes = []Mode{
	{
		ID:     "sock_crash_loop",
		Title:  "fault.sock",
		Match:  []string{"operation not permitted", ".sock"},
		Scopes: []string{"mysql", "pgsql"},
		// bash 无 sock 清理子命令：兜底 = 卸载重装（用户决策后自行执行）
		Cli: "phpbox mysql uninstall <版本> [--purge] && phpbox mysql install <版本>",
	},
	{
		ID:     "port_conflict",
		Title:  "fault.port",
		Match:  []string{"bind failed", "address already in use", "port is already allocated"},
		Scopes: []string{"mysql", "pgsql", "redis", "nginx", "php"},
		// port set 是真 CLI（mysql/pgsql/redis: port <版本> <端口>；nginx: port <端口>）——GUI 提供迁移出路
		Cli: "phpbox mysql port <版本> <新端口>  # 或 nginx: phpbox nginx port <新端口>",
	},
	{
		ID:     "image_missing",
		Title:  "fault.image",
		Match:  []string{"no such image", "image not found", "manifest unknown"},
		Scopes: []string{"php", "mysql", "pgsql", "redis", "nginx"},
		// 镜像缺失：重装对应服务（离线优先由 bash install 自身决定）
		Cli: "phpbox <服务> install <版本>  # 离线命中则零下载，否则联网拉取",
	},
	{
		ID:     "dbdata_corrupt",
		Title:  "fault.dbdata",
		Match:  []string{"ibdata1", "different filesystem", "innoDB: initialization"},
		Scopes: []string{"mysql", "pgsql"},
		Cli:    "phpbox backup  # 数据目录异常时先备份；恢复用 phpbox restore <备份文件> -y",
	},
}

// Modes 返回全部已知模式定义（诊断页参考表数据源）。
func Modes() []Mode { return modes }

// Detect 扫描异常容器（非 running 的 phpbox 容器）日志 tail 100 行做模式匹配。
// 只报命中（证据：容器名 + 命中行），未命中返回空——不做概率性"疑似"提示。
func Detect(ctx context.Context, c *docker.Client, containers []docker.ContainerSummary) ([]Hit, error) {
	var hits []Hit
	for _, ct := range containers {
		// 只排查 phpbox 管理容器（有 Service label）且非 running：重启循环/退出的才有诊断价值
		if ct.Service == "" || ct.State == "running" {
			continue
		}
		logs, err := c.ContainerLogs(ctx, strings.TrimPrefix(ct.Name, "/"), 100)
		if err != nil {
			continue // 日志读不到（清理中的容器等）：跳过，不是故障证据
		}
		for _, m := range modes {
			var lines []string
			for _, l := range logs {
				low := strings.ToLower(l)
				for _, kw := range m.Match {
					if strings.Contains(low, kw) {
						lines = append(lines, l)
						break
					}
				}
				if len(lines) >= 3 {
					break
				}
			}
			if len(lines) > 0 {
				hits = append(hits, Hit{Mode: m, Container: strings.TrimPrefix(ct.Name, "/"), Lines: lines})
			}
		}
	}
	return hits, nil
}
