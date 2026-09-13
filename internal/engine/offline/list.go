// Package engine · offline 离线缓存扫描：只读遍历 phpbox 的 offline/ 目录树。
// 目录契约（AGENTS §1.2）：php/<版本>/{apk,pecl}/ 是构建闭包；
// mysql/pgsql/redis/<裸版本>/<svc>-<版本>.tar 与 nginx/<tag>/ 是已拉取镜像 tar。
package offline

import (
	"io/fs"
	"os"
	"path/filepath"
	"sort"
)

// Entry 一条离线缓存记录（一个服务版本目录）。
type Entry struct {
	Svc   string `json:"svc"`  // php / mysql / pgsql / redis / nginx
	Ver   string `json:"ver"`  // 裸版本号；nginx 用镜像 tag（如 alpine）
	Path  string `json:"path"` // 绝对路径
	Size  int64  `json:"size"` // 目录递归总字节数
	Files int    `json:"files"`
	Kind  string `json:"kind"` // image-tar（单 tar）/ closure（apk+pecl 闭包）
}

// List 扫描 offline/ 一级服务目录下的各版本子目录。
// 目录缺失返回空列表（安装前状态），不视为错误。
func List(root string) ([]Entry, error) {
	var out []Entry
	svcs, err := readDirNames(root)
	if err != nil {
		if os.IsNotExist(err) {
			return []Entry{}, nil // offline/ 尚未创建：空列表而非错误
		}
		return nil, err
	}
	for _, svc := range svcs {
		vers, err := readDirNames(filepath.Join(root, svc))
		if err != nil {
			continue // 单个服务目录不可读：跳过，不让整体失败
		}
		for _, ver := range vers {
			p := filepath.Join(root, svc, ver)
			e := Entry{Svc: svc, Ver: ver, Path: p, Kind: "image-tar"}
			if svc == "php" {
				e.Kind = "closure"
			}
			size, files, err := dirSize(p)
			if err != nil {
				continue
			}
			e.Size, e.Files = size, files
			out = append(out, e)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Svc != out[j].Svc {
			return out[i].Svc < out[j].Svc
		}
		return out[i].Ver < out[j].Ver
	})
	return out, nil
}

// readDirNames 列子目录名（仅目录，文件与隐藏项跳过）。
func readDirNames(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() {
			names = append(names, e.Name())
		}
	}
	return names, nil
}

// dirSize 递归累计大小与文件数。
func dirSize(dir string) (int64, int, error) {
	var total int64
	var files int
	err := filepath.WalkDir(dir, func(_ string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return nil // 单文件 stat 失败：不计入，不中断
		}
		total += info.Size()
		files++
		return nil
	})
	if err != nil {
		return 0, 0, err
	}
	return total, files, nil
}
