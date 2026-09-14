// Package bindings 是壳层绑定：把引擎能力暴露给前端（设计文档 §3.3-3）。
// 本包禁止业务逻辑——只做引擎调用与类型透传。
package bindings

import (
	"context"
	"log"

	"github.com/xiaokentrl/phpbox-desktop/internal/engine/docker"
	"github.com/xiaokentrl/phpbox-desktop/internal/engine/faultmode"
)

// Docker 暴露容器管理能力；导出方法会被 wails3 生成前端绑定。
type Docker struct{}

// ContainerSummary 透传引擎类型（前端 TS 类型由绑定生成器产出）。
type ContainerSummary = docker.ContainerSummary

// ListContainers 返回全部容器（含已停止）。
func (d *Docker) ListContainers(ctx context.Context) ([]ContainerSummary, error) {
	c, err := docker.New()
	if err != nil {
		return nil, err
	}
	items, err := c.ListContainers(ctx)
	if err != nil {
		log.Printf("[绑定] Docker.ListContainers 失败: %v", err)
		return nil, err
	}
	log.Printf("[绑定] Docker.ListContainers → %d 个容器（前端绑定链路贯通）", len(items))
	return items, nil
}

// GetContainerLogs 读容器最后 tail 行日志（诊断视图；tail 0/越界回退 50）。
func (d *Docker) GetContainerLogs(ctx context.Context, name string, tail int) ([]string, error) {
	c, err := docker.New()
	if err != nil {
		return nil, err
	}
	lines, err := c.ContainerLogs(ctx, name, tail)
	if err != nil {
		log.Printf("[绑定] Docker.GetContainerLogs(%s) 失败: %v", name, err)
		return nil, err
	}
	log.Printf("[绑定] Docker.GetContainerLogs(%s) → %d 行", name, len(lines))
	return lines, nil
}

// FaultHit / FaultMode 透传引擎类型（§8.1 已知故障模式库，阶段 0 只读检测）。
type FaultHit = faultmode.Hit
type FaultMode = faultmode.Mode

// DetectFaults 扫描异常容器日志做已知故障模式匹配（只读；一键修复属 v1.1 引擎期，不提供）。
func (d *Docker) DetectFaults(ctx context.Context) ([]FaultHit, error) {
	c, err := docker.New()
	if err != nil {
		return nil, err
	}
	items, err := c.ListContainers(ctx)
	if err != nil {
		log.Printf("[绑定] Docker.DetectFaults 容器列表失败: %v", err)
		return nil, err
	}
	hits, err := faultmode.Detect(ctx, c, items)
	if err != nil {
		log.Printf("[绑定] Docker.DetectFaults 失败: %v", err)
		return nil, err
	}
	log.Printf("[绑定] Docker.DetectFaults → %d 处命中", len(hits))
	return hits, nil
}

// FaultModes 返回全部已知故障模式定义（诊断页参考表；CLI 兜底均为 bash 实测签名）。
func (d *Docker) FaultModes() []FaultMode {
	return faultmode.Modes()
}
