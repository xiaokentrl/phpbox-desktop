// Package php · PHP 扩展状态读取（config/php/<版本>/extensions.env 真实文件）。
// 清洗语义对齐 bash _php_read_extensions：去注释/空行、去 KEY= 前缀、逗号拆分、去重、排序。
// 修改操作不经本包——重建事务在 bash 侧（phpbox php extension add/remove）。
package php

import (
	"os"
	"slices"
	"strings"
)

// ReadExtensions 解析某版本 extensions.env 的启用扩展清单。
// 文件不存在返回错误（版本未安装或无扩展状态，调用方决定呈现语义）。
func ReadExtensions(path string) ([]string, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, os.ErrNotExist
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
	return out, nil
}
