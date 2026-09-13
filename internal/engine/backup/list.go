// Package engine · backup 备份归档列表：只读扫描 phpbox 的 backups/ 目录。
// 归档本身是宿主机普通文件（cmd_backup 产物），无需 Docker。
package backup

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

// Entry 一份备份归档的描述。
type Entry struct {
	File string    `json:"file"` // 文件名（不含路径）
	Path string    `json:"path"` // 绝对路径
	Size int64     `json:"size"`
	At   time.Time `json:"at"`
}

// List 扫描备份目录，返回 *.tar.gz（按时间倒序，最新在前）。
func List(dir string) ([]Entry, error) {
	items, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return []Entry{}, nil // 目录尚无备份：空列表而非错误
		}
		return nil, err
	}
	var out []Entry
	for _, it := range items {
		if it.IsDir() || !strings.HasSuffix(it.Name(), ".tar.gz") {
			continue
		}
		info, err := it.Info()
		if err != nil {
			continue // 扫描期间被删除/不可读：跳过该条，不让整个列表失败
		}
		out = append(out, Entry{
			File: it.Name(),
			Path: filepath.Join(dir, it.Name()),
			Size: info.Size(),
			At:   info.ModTime(),
		})
	}
	// 时间倒序：新备份在前
	slices.SortFunc(out, func(a, b Entry) int { return b.At.Compare(a.At) })
	return out, nil
}
