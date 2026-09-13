// Package engine · site 站点状态读取：解析 nginx sites/ 下的 vhost conf。
// 只读；增删切换经 Runner spawn `phpbox site add/switch/remove`（事务在 bash 侧）。
package site

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// Entry 一个站点的描述。
type Entry struct {
	Domain string `json:"domain"`
	PHP    string `json:"php"`   // vhost upstream 指向的 PHP 服务键（如 php84）
	Root   string `json:"root"`  // 站点目录（/var/www/<域名> 容器路径映射 WWW_ROOT/<域名>）
	Hosts  bool   `json:"hosts"` // 域名是否已在 /etc/hosts 解析（127.0.0.1）
}

var (
	// set $php_upstream php84:9000; —— add.sh 模板的固定形态
	upstreamRe = regexp.MustCompile(`set\s+\$php_upstream\s+([a-z0-9.]+):9000`)
	// 旧版模板直接写 fastcgi_pass php84:9000
	legacyRe = regexp.MustCompile(`fastcgi_pass\s+([a-z0-9.]+):9000`)
	// root /var/www/<域名>;
	rootRe = regexp.MustCompile(`root\s+(\S+);`)
)

// List 解析 sites 目录全部 *.conf。目录不存在返回空列表（未安装状态）。
func List(sitesDir string) ([]Entry, error) {
	items, err := os.ReadDir(sitesDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []Entry{}, nil
		}
		return nil, err
	}
	var out []Entry
	for _, it := range items {
		if it.IsDir() || !strings.HasSuffix(it.Name(), ".conf") {
			continue
		}
		if strings.HasPrefix(it.Name(), ".") {
			continue // .backup.* 恢复快照：非活跃站点
		}
		data, err := os.ReadFile(filepath.Join(sitesDir, it.Name()))
		if err != nil {
			continue // 单个 conf 不可读：跳过，不让整体失败
		}
		conf := string(data)
		domain := strings.TrimSuffix(it.Name(), ".conf")
		e := Entry{Domain: domain, Root: "~/www/" + domain}
		if m := upstreamRe.FindStringSubmatch(conf); m != nil {
			e.PHP = m[1]
		} else if m := legacyRe.FindStringSubmatch(conf); m != nil {
			e.PHP = m[1]
		}
		if m := rootRe.FindStringSubmatch(conf); m != nil {
			e.Root = m[1]
		}
		out = append(out, e)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Domain < out[j].Domain })
	return out, nil
}

// HostsResolved 检查域名是否在 /etc/hosts 中指向 127.0.0.1（与 bash _hosts_add 同判定）。
func HostsResolved(domain string) bool {
	data, err := os.ReadFile("/etc/hosts")
	if err != nil {
		return false
	}
	for _, ln := range strings.Split(string(data), "\n") {
		ln = strings.TrimSpace(ln)
		if ln == "" || strings.HasPrefix(ln, "#") {
			continue
		}
		fields := strings.Fields(ln)
		if len(fields) >= 2 && (fields[0] == "127.0.0.1" || fields[0] == "::1") {
			for _, f := range fields[1:] {
				if f == domain {
					return true
				}
			}
		}
	}
	return false
}
