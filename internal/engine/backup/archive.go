// Package backup · 归档内容查看与导出（ui-spec §3.8）。
// 备份归档是普通文件（文件即接口）：查看 = gzip/tar 头流式解析（只读）；
// 导出 = 复制到用户选定目录（不改动 backups/ 本体，非引擎状态变更，无需 CLI 事务）。
package backup

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
)

// ArchiveEntry 归档内一个条目（路径为 tar 头原样，如 home/kentrl/phpbox/.env）。
type ArchiveEntry struct {
	Path  string `json:"path"`
	Size  int64  `json:"size"` // tar 头声明的字节（目录为 0）
	IsDir bool   `json:"isDir"`
}

// ArchiveInfo 归档内容摘要（流式读取，上限截断如实标注）。
type ArchiveInfo struct {
	Entries    []ArchiveEntry `json:"entries"`
	Truncated  bool           `json:"truncated"`   // 达到 maxEntries 上限，未读完（如实标注，不冒充全量）
	TotalBytes int64          `json:"totalBytes"`  // 已读条目的声明字节和（截断时是部分和）
}

// InspectArchive 流式读取 tar.gz 归档头（不解压文件内容），最多 maxEntries 条。
// maxEntries ≤ 0 时取 2000。
func InspectArchive(path string, maxEntries int) (ArchiveInfo, error) {
	if maxEntries <= 0 {
		maxEntries = 2000
	}
	f, err := os.Open(path)
	if err != nil {
		return ArchiveInfo{}, err
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return ArchiveInfo{}, fmt.Errorf("gzip 头不可读（归档损坏或非 tar.gz）: %w", err)
	}
	defer gz.Close()
	out := ArchiveInfo{Entries: make([]ArchiveEntry, 0, 256)}
	tr := tar.NewReader(gz)
	for {
		hd, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return out, fmt.Errorf("tar 头读取中断: %w", err) // 已读条目照常返回，错误一并上报
		}
		out.TotalBytes += hd.Size
		out.Entries = append(out.Entries, ArchiveEntry{
			Path:  hd.Name,
			Size:  hd.Size,
			IsDir: hd.Typeflag == tar.TypeDir,
		})
		if len(out.Entries) >= maxEntries {
			out.Truncated = true
			break
		}
	}
	return out, nil
}

// ExportArchive 把归档复制到 dst（用户经原生对话框选定的完整目标路径）。
// 复制到临时文件后 rename 原子落盘，避免半截文件被当成完整备份。
func ExportArchive(src, dst string) error {
	if src == "" || dst == "" {
		return fmt.Errorf("导出路径为空")
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	tmp := dst + ".part"
	out, err := os.OpenFile(tmp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		os.Remove(tmp)
		return err
	}
	if err := out.Close(); err != nil {
		os.Remove(tmp)
		return err
	}
	if err := os.Rename(tmp, dst); err != nil {
		os.Remove(tmp)
		return err
	}
	return nil
}
