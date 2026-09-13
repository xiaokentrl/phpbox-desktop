// Package engine · presence phpbox 引擎存在性检测（ui-spec §5.1 首启 Onboarding 的三条件）。
// 只读检测，不做任何安装动作——安装引导走 UI 指路 bash 仓 install.sh（阶段 0 契约），
// GUI 不复制、不内嵌引擎（否决过 resources/phpbox 打包方向，见 docs/archive）。
package presence

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/moby/moby/client"
)

// Status 引擎就绪度快照（三条件独立呈现，不合并成单一布尔——开发者要分别知道缺什么）。
type Status struct {
	EngineDir bool   // ~/phpbox 目录存在（install.sh 的固定布局）
	CliInPath string // exec.LookPath("phpbox") 结果：可执行文件绝对路径；空 = 不在 PATH
	HasEnv    bool   // ~/phpbox/.env 存在（引擎已初始化过的标志）
	DockerOK  bool   // Docker Engine 可达（1s ping）
	DockerErr string // Docker 不可达原因（可呈现给开发者）
}

// Detect 返回三条件快照。目录检测用 os.Stat（目录/文件都算"存在"），
// CLI 检测用 LookPath（PATH 里任何位置都算——用户可能装在 /usr/local/bin 或自定目录）。
func Detect(ctx context.Context) Status {
	home, _ := os.UserHomeDir()
	st := Status{}
	if home != "" {
		dir := filepath.Join(home, "phpbox")
		if _, err := os.Stat(dir); err == nil {
			st.EngineDir = true
		}
		if _, err := os.Stat(filepath.Join(dir, ".env")); err == nil {
			st.HasEnv = true
		}
	}
	if p, err := exec.LookPath("phpbox"); err == nil {
		st.CliInPath = p
	}
	st.DockerOK, st.DockerErr = dockerPing(ctx)
	return st
}

// dockerPing 1s 截止的 Engine API 探活；失败把错误原样带回（不吞、不转译）。
func dockerPing(ctx context.Context) (bool, string) {
	pingCtx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()
	api, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return false, err.Error()
	}
	defer api.Close()
	if _, err := api.Ping(pingCtx, client.PingOptions{}); err != nil {
		return false, err.Error()
	}
	return true, ""
}
