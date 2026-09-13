// Package diskusage · 目录磁盘占用计算（总览页资源小部件 §3.11）。
// 真实递归 stat，不做估算；目录不存在返回 0（未初始化状态，不是错误）。
package diskusage

import (
	"io/fs"
	"path/filepath"
)

// Dir 返回目录递归总字节（符号链接不跟随）。目录缺失返回 0。
func Dir(path string) int64 {
	if path == "" {
		return 0
	}
	var total int64
	_ = filepath.WalkDir(path, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // 不可读的子目录：跳过但继续（占用统计是尽力而为的快照）
		}
		if d.Type().IsRegular() {
			if info, err := d.Info(); err == nil {
				total += info.Size()
			}
		}
		return nil
	})
	return total
}
