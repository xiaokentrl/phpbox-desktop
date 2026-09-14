// Package docker · 单容器资源占用（ui-spec §3.5 版本卡内存占用）。
// 读取通道：Docker API ContainerStatsOneShot（非流式），与总览资源小部件同类只读；
// 工作集口径与 docker stats 一致：cgroup usage − inactive_file（页缓存不计入）。
package docker

import (
	"context"
	"encoding/json"

	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/client"
)

// ContainerMem 容器内存快照（当前值，非流式）。
type ContainerMem struct {
	Name   string `json:"name"`   // 容器名
	MemUse int64  `json:"memUse"` // 工作集字节（usage − inactive_file）
}

// StatsMemory 读单容器内存工作集。容器不存在原样报错（前端如实呈现）。
// options Stream=false + 默认 IncludePreviousSample=false = 一次性快照（one-shot，daemon ≤28 兼容）。
func (c *Client) StatsMemory(ctx context.Context, name string) (ContainerMem, error) {
	resp, err := c.api.ContainerStats(ctx, name, client.ContainerStatsOptions{})
	if err != nil {
		return ContainerMem{}, err
	}
	defer resp.Body.Close()
	dec := json.NewDecoder(resp.Body)
	var st container.StatsResponse
	if err := dec.Decode(&st); err != nil {
		return ContainerMem{}, err
	}
	mem := workingSet(st.MemoryStats)
	return ContainerMem{Name: name, MemUse: mem}, nil
}

// workingSet 内存工作集：usage − inactive_file（cgroup v1 键 total_inactive_file，v2 键 inactive_file）。
// docker stats CLI 同款口径；无明细时退回 usage（不臆造）。
func workingSet(m container.MemoryStats) int64 {
	if m.Usage == 0 {
		return 0
	}
	if v, ok := m.Stats["inactive_file"]; ok && v < m.Usage {
		return int64(m.Usage - v)
	}
	if v, ok := m.Stats["total_inactive_file"]; ok && v < m.Usage {
		return int64(m.Usage - v)
	}
	return int64(m.Usage)
}
