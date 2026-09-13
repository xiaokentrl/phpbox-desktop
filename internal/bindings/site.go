// Package bindings · site 绑定：读取真实站点 vhost 与 hosts 解析状态。
// 增删/切换/hosts 修改经 Runner spawn `phpbox site/hosts ...`（bash 侧事务 + sudo 边界）。
package bindings

import (
	"log"
	"os"
	"path/filepath"

	"github.com/xiaokentrl/phpbox-desktop/internal/engine/site"
)

// Site 暴露站点状态读取能力。
type Site struct{}

// SiteEntry 透传引擎类型。
type SiteEntry = site.Entry

// sitesDir phpbox 的站点 vhost 目录（config/nginx/sites/）。
func sitesDir() string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return "sites"
	}
	return filepath.Join(home, "phpbox", "config", "nginx", "sites")
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
