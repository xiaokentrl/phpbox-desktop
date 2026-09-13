// Package bindings · site 绑定：读取真实站点 vhost 与 hosts 解析状态。
// 增删/切换/hosts 修改经 Runner spawn `phpbox site/hosts ...`（bash 侧事务 + sudo 边界）。
package bindings

import (
	"log"
	"os"
	"path/filepath"
	"strconv"
	"time"

	engineEnv "github.com/xiaokentrl/phpbox-desktop/internal/engine/env"
	"github.com/xiaokentrl/phpbox-desktop/internal/engine/health"
	"github.com/xiaokentrl/phpbox-desktop/internal/engine/site"
)

// Site 暴露站点状态读取能力。
type Site struct{}

// SiteEntry 透传引擎类型。
type SiteEntry = site.Entry

// SiteHealth 透传引擎类型（status: up/degraded/down + code + err）。
type SiteHealth = health.Result

// probeTimeout 单站点探测硬上限（§12.2 长操作截止时间契约）。
const probeTimeout = 3 * time.Second

// sitesDir phpbox 的站点 vhost 目录（config/nginx/sites/）。
func sitesDir() string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return "sites"
	}
	return filepath.Join(home, "phpbox", "config", "nginx", "sites")
}

// nginxPort 读 .env 的 NGINX_PORT；缺失/不可解析回退 80（bash 默认）。
// envFile() 复用 env.go 的同包定义（~/phpbox/.env）。
func nginxPort() int {
	kvs, err := engineEnv.Read(envFile())
	if err != nil {
		return 80
	}
	for _, kv := range kvs {
		if kv.Key == "NGINX_PORT" {
			if p, err := strconv.Atoi(kv.Value); err == nil && p > 0 {
				return p
			}
			break
		}
	}
	return 80
}

// ListSites 列出全部站点（vhost 真实状态 + hosts 解析）。
func (s *Site) ListSites() ([]SiteEntry, error) {
	entries, err := site.List(sitesDir())
	if err != nil {
		log.Printf("[绑定] Site.ListSites 失败: %v", err)
		return nil, err
	}
	for i := range entries {
		entries[i].Hosts = site.HostsResolved(entries[i].Domain)
	}
	log.Printf("[绑定] Site.ListSites → %d 个站点", len(entries))
	return entries, nil
}

// ProbeSiteHealth 对域名发 HEAD 探测（Go 侧：WebView fetch 跨源读不到状态码）。
// 目标 127.0.0.1:NGINX_PORT，Host 头路由——不依赖 /etc/hosts。
func (s *Site) ProbeSiteHealth(domain string) SiteHealth {
	res := health.Probe(domain, nginxPort(), probeTimeout)
	log.Printf("[绑定] Site.ProbeSiteHealth(%s) → %s", domain, res.Status)
	return res
}
