// Package engine · health 站点健康 HEAD 探测。
// 必须在 Go 侧探测：WebView 内 fetch 跨源读不到状态码（Wails Issue #1642 官方建议）。
// 探测目标 127.0.0.1:<NGINX_PORT>，Host 头携带域名——nginx 按 server_name 路由，
// 不依赖 /etc/hosts 解析（域名未加 hosts 也能探活）。
package health

import (
	"fmt"
	"net/http"
	"time"
)

// Status 站点健康三态（UI 规格 §3.1：🟢 正常 / 🟡 降级 / ⚪ 未启动）。
type Status string

const (
	Up       Status = "up"       // 2xx/3xx：nginx 服务中
	Degraded Status = "degraded" // 4xx/5xx：有响应但异常（404 无索引 / 502 PHP 容器异常）
	Down     Status = "down"     // 连接失败：nginx 未启动或端口未监听
)

// Result 探测结果（结构化返回，前端映射徽章）。
type Result struct {
	Status Status `json:"status"`
	Code   int    `json:"code"`          // HTTP 状态码（Down 时 0）
	Err    string `json:"err,omitempty"` // 连接失败原因（Down 时非空）
}

// Probe 对 127.0.0.1:port 发 HEAD 请求，Host 头设为 domain（vhost 路由）。
// timeout 是硬上限（长操作截止时间契约），超时归为 Down。
func Probe(domain string, port int, timeout time.Duration) Result {
	req, err := http.NewRequest(http.MethodHead, fmt.Sprintf("http://127.0.0.1:%d/", port), nil)
	if err != nil {
		return Result{Status: Down, Err: err.Error()}
	}
	req.Host = domain
	client := &http.Client{Timeout: timeout}
	resp, err := client.Do(req)
	if err != nil {
		return Result{Status: Down, Err: err.Error()}
	}
	defer resp.Body.Close()
	if resp.StatusCode < 400 {
		return Result{Status: Up, Code: resp.StatusCode}
	}
	return Result{Status: Degraded, Code: resp.StatusCode}
}
