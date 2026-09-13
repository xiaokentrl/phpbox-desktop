// Package docker · 日志读取：诊断视图的容器日志 tail（只读）。
// 阶段 0 契约：修复动作不存在（bash CLI 无 restart/chown/sock-clean 子命令，
// ui-spec §5.7 一键修复是 v1.1 + Go 引擎前提）——本层只提供事实，不提供修复。
package docker

import (
	"bytes"
	"context"
	"strconv"
	"strings"

	"github.com/moby/moby/api/pkg/stdcopy"
	"github.com/moby/moby/client"
)

// ContainerLogs 返回容器最后 tail 行日志（docker logs --tail 语义）。
// 输出按行拆分、去尾部空行；容器不存在/daemon 不可达原样返回错误（不吞、不转译）。
func (c *Client) ContainerLogs(ctx context.Context, name string, tail int) ([]string, error) {
	if tail <= 0 || tail > 500 {
		tail = 50
	}
	rc, err := c.api.ContainerLogs(ctx, name, client.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Tail:       strconv.Itoa(tail),
	})
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	// 非 tty 模式 docker 返回 stdcopy 多路复用流（8 字节头 + 载荷，实测每行带
	// \x02\x00… 头直出到 UI）；SDK 的 stdcopy.StdCopy 负责解复用，stdout/stderr
	// 汇入同一缓冲（诊断场景不区分来源，行序即时间序）。
	var buf bytes.Buffer
	if _, err := stdcopy.StdCopy(&buf, &buf, rc); err != nil {
		return nil, err
	}
	text := strings.TrimRight(buf.String(), "\n")
	if text == "" {
		return []string{}, nil
	}
	return strings.Split(text, "\n"), nil
}
