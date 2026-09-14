package backup

import (
	"archive/tar"
	"compress/gzip"
	"os"
	"path/filepath"
	"testing"
)

// 写一个真实可读的最小 tar.gz（两条文件 + 一条目录），供 Inspect/Export 测试。
func writeMiniArchive(t *testing.T, path string) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	gz := gzip.NewWriter(f)
	tw := tar.NewWriter(gz)
	for _, e := range []struct {
		name string
		body string
		dir  bool
	}{
		{"home/x/.env", "A=1\n", false},
		{"home/x/config/", "", true},
		{"home/x/config/php.ini", "memory=1G", false},
	} {
		hdr := &tar.Header{Name: e.name, Mode: 0o644}
		if e.dir {
			hdr.Typeflag = tar.TypeDir
		} else {
			hdr.Size = int64(len(e.body))
		}
		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatal(err)
		}
		if e.body != "" {
			if _, err := tw.Write([]byte(e.body)); err != nil {
				t.Fatal(err)
			}
		}
	}
	tw.Close()
	gz.Close()
	f.Close()
}

func TestInspectArchive(t *testing.T) {
	dir := t.TempDir()
	arc := filepath.Join(dir, "backup-test.tar.gz")
	writeMiniArchive(t, arc)

	info, err := InspectArchive(arc, 0)
	if err != nil {
		t.Fatalf("InspectArchive: %v", err)
	}
	if len(info.Entries) != 3 {
		t.Fatalf("条目数 = %d, want 3（目录也计一条）", len(info.Entries))
	}
	if !info.Entries[1].IsDir {
		t.Error("第二条应是目录条目")
	}
	if info.Entries[0].Size != 4 {
		t.Errorf("首条 size = %d, want 4（A=1\\n）", info.Entries[0].Size)
	}
	if info.Truncated {
		t.Error("3 条不应触发截断")
	}
}

func TestInspectArchiveTruncated(t *testing.T) {
	dir := t.TempDir()
	arc := filepath.Join(dir, "backup-test.tar.gz")
	writeMiniArchive(t, arc)

	info, err := InspectArchive(arc, 2)
	if err != nil {
		t.Fatalf("InspectArchive: %v", err)
	}
	if len(info.Entries) != 2 || !info.Truncated {
		t.Fatalf("maxEntries=2 应读到 2 条且 Truncated=true，得到 %d 条 truncated=%v", len(info.Entries), info.Truncated)
	}
}

func TestInspectArchiveCorrupt(t *testing.T) {
	dir := t.TempDir()
	arc := filepath.Join(dir, "bad.tar.gz")
	if err := os.WriteFile(arc, []byte("这不是 gzip"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := InspectArchive(arc, 0); err == nil {
		t.Fatal("非 gzip 内容应报错（诚实失败，不返回假空列表）")
	}
}

func TestExportArchive(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "backup-test.tar.gz")
	writeMiniArchive(t, src)

	dst := filepath.Join(dir, "exported", "copy.tar.gz") // 目标目录不存在
	if err := ExportArchive(src, dst); err == nil {
		t.Fatal("目标目录不存在时应报错（不静默 mkdir——保存对话框已保证目录存在）")
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := ExportArchive(src, dst); err != nil {
		t.Fatalf("ExportArchive: %v", err)
	}
	a, err := os.ReadFile(src)
	if err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(dst)
	if err != nil {
		t.Fatal(err)
	}
	if len(a) != len(b) {
		t.Fatalf("导出副本 %d 字节 != 原件 %d 字节", len(b), len(a))
	}
	if _, err := os.Stat(dst + ".part"); !os.IsNotExist(err) {
		t.Error("临时 .part 文件应已 rename 消失")
	}
}
