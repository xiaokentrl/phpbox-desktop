// Package engine · go 自动发现：扫描 GO_PROJECTS_ROOT 直接子目录中含 go.mod 的项目。
// 判定与 bash _go_server 同构；运行状态由调用方（绑定层）用 docker 引擎注入，本包不依赖 Docker。
package goproject

import (
	"os"
	"path/filepath"
	"sort"
)

// Entry 一个 Go 项目。
type Entry struct {
	Name    string `json:"name"` // 项目名（目录名，即 CLI 参数形态）
	Dir     string `json:"dir"`  // 绝对路径
	Running bool   `json:"running"`
}

// List 扫描 projects 根目录。根目录缺失返回空列表（未初始化状态）。
// running 为"运行中的 go-<项目名> 容器名集合"，由绑定层从 Docker 引擎查询注入。
func List(root string, running map[string]bool) ([]Entry, error) {
	items, err := os.ReadDir(root)
	if err != nil {
		if os.IsNotExist(err) {
			return []Entry{}, nil
		}
		return nil, err
	}
	var out []Entry
	for _, it := range items {
		if !it.IsDir() {
			continue
		}
		if _, err := os.Stat(filepath.Join(root, it.Name(), "go.mod")); err != nil {
			continue // 无 go.mod：不是 Go 项目（bash 契约：直接子目录含 go.mod 才发现）
		}
		out = append(out, Entry{
			Name:    it.Name(),
			Dir:     filepath.Join(root, it.Name()),
			Running: running["go-"+it.Name()],
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}
