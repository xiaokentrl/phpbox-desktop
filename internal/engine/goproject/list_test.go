package goproject

import (
	"os"
	"path/filepath"
	"testing"
)

func TestList(t *testing.T) {
	root := t.TempDir()
	// 两个合法项目（含 go.mod）
	for _, name := range []string{"beta-api", "alpha-web"} {
		dir := filepath.Join(root, name)
		os.MkdirAll(dir, 0o700)
		os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module x"), 0o600)
	}
	// 无 go.mod：排除
	os.MkdirAll(filepath.Join(root, "not-go"), 0o700)
	// 嵌套 go.mod（二级目录）：不算（仅直接子目录）
	os.MkdirAll(filepath.Join(root, "not-go", "nested"), 0o700)
	os.WriteFile(filepath.Join(root, "not-go", "nested", "go.mod"), []byte("m"), 0o600)
	// 普通文件：排除
	os.WriteFile(filepath.Join(root, "README.md"), []byte("x"), 0o600)

	entries, err := List(root, map[string]bool{"go-beta-api": true})
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Fatalf("期望 2 个项目，得到 %d: %+v", len(entries), entries)
	}
	// 排序契约：项目名字典序
	if entries[0].Name != "alpha-web" || entries[1].Name != "beta-api" {
		t.Errorf("排序失效: %+v", entries)
	}
	if !entries[1].Running || entries[0].Running {
		t.Errorf("运行状态注入错误: %+v", entries)
	}
}

func TestListMissingRoot(t *testing.T) {
	entries, err := List(filepath.Join(t.TempDir(), "nope"), nil)
	if err != nil {
		t.Fatalf("缺失根目录应返回空列表: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("期望空列表，得到 %d 项", len(entries))
	}
}
