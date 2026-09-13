// Package bindings · php 绑定：读取各 PHP 版本真实扩展状态。
// 修改经 Runner spawn `phpbox php extension add/remove`（镜像重建事务在 bash 侧）。
package bindings

import (
	"fmt"
	"log"
	"path/filepath"

	phpengine "github.com/xiaokentrl/phpbox-desktop/internal/engine/php"
)

// Php 暴露 PHP 版本状态读取能力。
type Php struct{}

// ReadPhpExtensions 解析某 PHP 版本的真实扩展状态（config/php/<版本>/extensions.env）。
// 清洗语义在 engine/php（对齐 bash _php_read_extensions：去注释/空行、去 KEY= 前缀、
// 逗号拆分、去重、排序）。
func (p *Php) ReadPhpExtensions(version string) ([]string, error) {
	if version == "" || filepath.Base(version) != version {
		return nil, fmt.Errorf("非法 PHP 版本号: %q", version)
	}
	exts, err := phpengine.ReadExtensions(filepath.Join(phpConfigDir(version), "extensions.env"))
	if err != nil {
		return nil, fmt.Errorf("版本未安装或无扩展状态: config/php/%s/extensions.env", version)
	}
	log.Printf("[绑定] Php.ReadPhpExtensions(%s) → %d 个扩展", version, len(exts))
	return exts, nil
}
