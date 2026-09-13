// Package engine · env phpbox .env 文件读写。
// 解析语义对齐 bash _env_read_file：KEY=VALUE（首个 = 分割）、跳过注释/空行、剥成对引号；
// 写入采用行级 patch + 原子替换（临时文件 → rename），注释与键顺序保留。
package env

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// KV 一个 .env 条目。
type KV struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// Read 解析 .env 全部键值（按文件顺序，含注释键外的全部条目）。
// 文件不存在返回空列表（首次安装前状态）。
func Read(path string) ([]KV, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return []KV{}, nil
	}
	if err != nil {
		return nil, err
	}
	var out []KV
	for _, ln := range strings.Split(string(data), "\n") {
		ln = strings.TrimRight(ln, "\r")
		trimmed := strings.TrimSpace(ln)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		i := strings.Index(ln, "=")
		if i <= 0 {
			continue // 无 = 或键为空：跳过（bash 同样跳过）
		}
		key := strings.TrimSpace(ln[:i])
		val := strings.TrimSpace(ln[i+1:])
		out = append(out, KV{Key: key, Value: stripQuotes(val)})
	}
	return out, nil
}

// stripQuotes 剥掉成对包裹的单/双引号（对齐 bash _env_strip_quotes）。
func stripQuotes(s string) string {
	if len(s) >= 2 {
		first, last := s[0], s[len(s)-1]
		if (first == '"' && last == '"') || (first == '\'' && last == '\'') {
			return s[1 : len(s)-1]
		}
	}
	return s
}

// Patch 行级更新：已存在的键原行替换（保留注释与顺序），不存在的键追加到尾部。
// changes 中禁止换行与引号字符（防止破坏 KEY=VALUE 行格式 / shell 语义）。
// 原子性：写临时文件后 rename，失败不触碰原文件。
func Patch(path string, changes map[string]string) error {
	for k, v := range changes {
		if strings.Contains(k, "=") || strings.ContainsAny(v, "\"'\n") {
			return fmt.Errorf("非法键值（含引号/换行/=）: %s", k)
		}
	}
	data, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	lines := strings.Split(string(data), "\n")
	pending := make(map[string]string, len(changes))
	for k, v := range changes {
		pending[k] = v
	}
	out := make([]string, 0, len(lines)+len(changes))
	for _, ln := range lines {
		trimmed := strings.TrimSpace(ln)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			out = append(out, ln)
			continue
		}
		i := strings.Index(ln, "=")
		if i <= 0 {
			out = append(out, ln)
			continue
		}
		key := strings.TrimSpace(ln[:i])
		if v, ok := pending[key]; ok {
			out = append(out, key+"="+v)
			delete(pending, key)
			continue
		}
		out = append(out, ln)
	}
	// 未命中的键追加到尾部
	for k, v := range pending {
		out = append(out, k+"="+v)
	}
	content := strings.Join(out, "\n")
	if len(lines) > 0 && lines[len(lines)-1] == "" && len(out) > 0 && out[len(out)-1] != "" {
		content += "\n"
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".env.patch-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	if _, err := tmp.WriteString(content); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return err
	}
	if err := os.Rename(tmpName, path); err != nil {
		os.Remove(tmpName)
		return err
	}
	return nil
}
