// engine · offline 校验与清理：镜像 tar 做 gzip+tar 头可读性校验（对齐 bash
// _ensure_offline_image 的 tar 可读性契约）；PHP 闭包检查 apk/pecl 非空。
package offline

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// known 服务白名单：删除/校验入口只接受这些一级目录名。
var known = map[string]bool{"php": true, "mysql": true, "pgsql": true, "redis": true, "nginx": true}

// Result 校验结论。
type Result struct {
	OK     bool   `json:"ok"`
	Detail string `json:"detail"`
}

// sanitize 服务+版本合法性：两者都必须是裸名（无路径分隔符），服务必须在白名单内。
func sanitize(svc, ver string) error {
	if !known[svc] {
		return fmt.Errorf("未知离线服务: %q", svc)
	}
	if ver == "" || filepath.Base(ver) != ver || ver == "." || ver == ".." {
		return fmt.Errorf("非法版本目录名: %q", ver)
	}
	return nil
}

// Verify 轻量校验一条缓存：不逐字节读大 tar，只验 gzip 头 + 首 tar 条目可解析。
func Verify(root, svc, ver string) (Result, error) {
	if err := sanitize(svc, ver); err != nil {
		return Result{}, err
	}
	dir := filepath.Join(root, svc, ver)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return Result{OK: false, Detail: "目录不存在"}, nil
	}
	if svc == "php" {
		return verifyClosure(dir)
	}
	return verifyImageTar(dir, svc, ver)
}

// verifyClosure PHP 构建闭包：apk/ 与 pecl/ 必须存在且至少各有一个包。
func verifyClosure(dir string) (Result, error) {
	apk, apkErr := countFiles(filepath.Join(dir, "apk"))
	pecl, peclErr := countFiles(filepath.Join(dir, "pecl"))
	switch {
	case apkErr != nil || peclErr != nil:
		return Result{OK: false, Detail: "apk/pecl 目录缺失"}, nil
	case apk == 0 || pecl == 0:
		return Result{OK: false, Detail: fmt.Sprintf("闭包不完整（apk %d · pecl %d）", apk, pecl)}, nil
	}
	return Result{OK: true, Detail: fmt.Sprintf("apk %d 个 · pecl %d 个", apk, pecl)}, nil
}

// verifyImageTar 镜像 tar：定位 <svc>-<ver>.tar，验证 gzip + 首 tar 头。
func verifyImageTar(dir, svc, ver string) (Result, error) {
	name := fmt.Sprintf("%s-%s.tar", svc, ver)
	f, err := os.Open(filepath.Join(dir, name))
	if os.IsNotExist(err) {
		// 目录在但 tar 缺失：列出实际内容辅助诊断
		items, _ := os.ReadDir(dir)
		var actual []string
		for _, it := range items {
			actual = append(actual, it.Name())
		}
		return Result{OK: false, Detail: "缺失 " + name + "（现有: " + strings.Join(actual, ", ") + "）"}, nil
	}
	if err != nil {
		return Result{}, err
	}
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil {
		return Result{OK: false, Detail: name + " gzip 头损坏: " + err.Error()}, nil
	}
	defer gz.Close()
	if _, err := tar.NewReader(gz).Next(); err != nil {
		return Result{OK: false, Detail: name + " tar 头不可读: " + err.Error()}, nil
	}
	return Result{OK: true, Detail: name + " 可读（gzip/tar 头校验通过）"}, nil
}

// Remove 清理一条缓存：删除 offline/<svc>/<ver>/ 整目录。调用方（GUI）须先做危险确认。
func Remove(root, svc, ver string) error {
	if err := sanitize(svc, ver); err != nil {
		return err
	}
	dir := filepath.Join(root, svc, ver)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return fmt.Errorf("缓存条目不存在: offline/%s/%s", svc, ver)
	}
	return os.RemoveAll(dir)
}

// countFiles 目录内普通文件数。
func countFiles(dir string) (int, error) {
	items, err := os.ReadDir(dir)
	if err != nil {
		return 0, err
	}
	n := 0
	for _, it := range items {
		if !it.IsDir() {
			n++
		}
	}
	return n, nil
}
