package backup

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func writeArchive(t *testing.T, dir, name string, size int, age time.Duration) {
	t.Helper()
	path := filepath.Join(dir, name)
	data := make([]byte, size)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	past := time.Now().Add(-age)
	if err := os.Chtimes(path, past, past); err != nil {
		t.Fatal(err)
	}
}

func TestList(t *testing.T) {
	dir := t.TempDir()
	writeArchive(t, dir, "backup-new.tar.gz", 10, 1*time.Hour)
	writeArchive(t, dir, "backup-old.tar.gz", 20, 48*time.Hour)
	writeArchive(t, dir, "not-archive.txt", 5, 0)    // 非 tar.gz：过滤
	writeArchive(t, dir, "partial.tar.gz.tmp", 5, 0) // 后缀不符：过滤
	if err := os.Mkdir(filepath.Join(dir, "sub.tar.gz"), 0o700); err != nil {
		t.Fatal(err) // 目录同名后缀：过滤
	}

	entries, err := List(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Fatalf("期望 2 个归档，得到 %d: %+v", len(entries), entries)
	}
	if entries[0].File != "backup-new.tar.gz" {
		t.Errorf("倒序失效：首项应为 backup-new.tar.gz，得到 %s", entries[0].File)
	}
	if entries[0].Size != 10 || entries[1].Size != 20 {
		t.Errorf("大小不匹配: %+v", entries)
	}
}

func TestListMissingDir(t *testing.T) {
	// 目录不存在 = 尚无备份：空列表而非错误（安装前状态）
	entries, err := List(filepath.Join(t.TempDir(), "nope"))
	if err != nil {
		t.Fatalf("缺失目录不应报错: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("期望空列表，得到 %d 项", len(entries))
	}
}

func TestListEmptyDir(t *testing.T) {
	entries, err := List(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Errorf("期望空列表，得到 %d 项", len(entries))
	}
}
