// Package docker 封装 Docker Engine API 客户端（设计文档 §5.4）。
// 连接差异（socket/named pipe）由官方 SDK 内部封装；API 版本协商必须开启。
package docker

import (

	"github.com/moby/moby/client"
)

// Client 是引擎对 Docker daemon 的唯一入口（阶段 0 POC：仅容器列举）。
type Client struct {
	api *client.Client
}

// New 建立 Engine API 连接：FromEnv 读取标准环境变量，
// WithAPIVersionNegotiation 自动协商客户端与 daemon 的最高共同 API 版本（官方推荐）。
func New() (*Client, error) {
	api, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, err
	}
	return &Client{api: api}, nil
}
