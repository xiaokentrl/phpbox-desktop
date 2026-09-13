// Package bindings · php 绑定：读取各 PHP 版本真实扩展状态。
// 修改经 Runner spawn `phpbox php extension add/remove`（镜像重建事务在 bash 侧）。
package bindings

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// Php 暴露 PHP 版本状态读取能力。
type Php struct{}

// ReadPhpExtensions 解析某 PHP 版本的真实扩展状态（config/php/<版本>/extensions.env）。
// 清洗语义对齐 bash _php_read_extensions：去注释/空行、去 KEY= 前缀、逗号拆分、去重、排序。
func (p *Php) ReadPhpExtensions(version string) ([]string, error) {
	if version == "" || filepath.Base(version) != version {
		return nil, fmt.Errorf("非法 PHP 版本号: %q", version)
	}
	path := filepath.Join(phpConfigDir(version), "extensions.env")
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, fmt.Errorf("版本未安装或无扩展状态: config/php/%s/extensions.env", version)
	}
	if err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	var out []string
	for _, ln := range strings.Split(string(data), "\n") {
		ln = strings.TrimSpace(ln)
		if ln == "" || strings.HasPrefix(ln, "#") {
			continue
		}
		if i := strings.Index(ln, "="); i >= 0 {
			ln = strings.TrimSpace(ln[i+1:])
		}
		for _, ext := range strings.Split(ln, ",") {
			ext = strings.TrimSpace(ext)
			if ext != "" && !seen[ext] {
				seen[ext] = true
				out = append(out, ext)
			}
		}
	}
	slices.Sort(out)
	log.Printf("[绑定] Php.ReadPhpExtensions(%s) → %d 个扩展", version, len(out))
	return out, nil
}
