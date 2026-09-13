package offline

import (
	"archive/tar"
	"compress/gzip"
	"os"
	"path/filepath"
	"testing"
)

// 写一个最小合法 gzip+tar（含一个文件条目），模拟镜像离线包
func writeFakeImageTar(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	gz := gzip.NewWriter(f)
	defer gz.Close()
	tw := tar.NewWriter(gz)
	defer tw.Close()
	if err := tw.WriteHeader(&tar.Header{Name: "manifest.json", Size: 2}); err != nil {
		t.Fatal(err)
	}
	if _, err := tw.Write([]byte("{}")); err != nil {
		t.Fatal(err)
	}
}

func mkClosure(t *testing.T, dir string, apkN, peclN int) {
	t.Helper()
	for _, sub := range []string{"apk", "pecl"} {
		if err := os.MkdirAll(filepath.Join(dir, sub), 0o700); err != nil {
			t.Fatal(err)
		}
	}
	for i := 0; i < apkN; i++ {
		os.WriteFile(filepath.Join(dir, "apk", fakeName(i)+".apk"), []byte("x"), 0o600)
	}
	for i := 0; i < peclN; i++ {
		os.WriteFile(filepath.Join(dir, "pecl", fakeName(i)+".tgz"), []byte("x"), 0o600)
	}
}

func fakeName(i int) string { return string(rune('a' + i)) }

func TestList(t *testing.T) {
	root := t.TempDir()
	mkClosure(t, filepath.Join(root, "php", "8.4"), 2, 1)
	writeFakeImageTar(t, filepath.Join(root, "mysql", "8.0", "mysql-8.0.tar"))
	os.WriteFile(filepath.Join(root, "mysql", "8.0", "stray.txt"), []byte("x"), 0o600) // 混入文件：不成为条目

	entries, err := List(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Fatalf("期望 2 条（php/8.4 + mysql/8.0），得到 %d: %+v", len(entries), entries)
	}
	// 排序契约：服务名字典序（mysql < php）
	if entries[0].Svc != "mysql" || entries[0].Ver != "8.0" {
		t.Errorf("首项应为 mysql/8.0: %+v", entries[0])
	}
	if entries[0].Kind != "image-tar" || entries[0].Files != 2 {
		t.Errorf("mysql 条目形态错误（tar + 混入文件 = 2 文件，目录递归计数契约）: %+v", entries[0])
	}
	php := entries[1]
	if php.Kind != "closure" || php.Files != 3 {
		t.Errorf("php 闭包应为 3 文件: %+v", php)
	}
}

func TestListMissingRoot(t *testing.T) {
	entries, err := List(filepath.Join(t.TempDir(), "nope"))
	if err != nil {
		t.Fatalf("缺失根目录应返回空列表: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("期望空列表，得到 %d 项", len(entries))
	}
}

func TestVerifyClosure(t *testing.T) {
	root := t.TempDir()
	mkClosure(t, filepath.Join(root, "php", "8.4"), 2, 1)
	res, err := Verify(root, "php", "8.4")
	if err != nil || !res.OK {
		t.Fatalf("完整闭包应通过: %+v err=%v", res, err)
	}
	// pecl 为空 → 不完整
	mkClosure(t, filepath.Join(root, "php", "7.4"), 1, 0)
	res, err = Verify(root, "php", "7.4")
	if err != nil || res.OK {
		t.Fatalf("pecl 为空应失败: %+v err=%v", res, err)
	}
}

func TestVerifyImageTar(t *testing.T) {
	root := t.TempDir()
	writeFakeImageTar(t, filepath.Join(root, "redis", "8", "redis-8.tar"))
	if res, err := Verify(root, "redis", "8"); err != nil || !res.OK {
		t.Fatalf("合法 tar 应通过: %+v err=%v", res, err)
	}
	// tar 缺失 → 明确失败并报告现有内容
	res, err := Verify(root, "redis", "7")
	if err != nil || res.OK {
		t.Fatalf("缺失 tar 应失败: %+v err=%v", res, err)
	}
	// 损坏 gzip → 头校验失败
	os.MkdirAll(filepath.Join(root, "nginx", "alpine"), 0o700)
	os.WriteFile(filepath.Join(root, "nginx", "alpine", "nginx-alpine.tar"), []byte("not gzip"), 0o600)
	if res, err := Verify(root, "nginx", "alpine"); err != nil || res.OK {
		t.Fatalf("损坏 tar 应失败: %+v err=%v", res, err)
	}
}

func TestRemoveAndEscape(t *testing.T) {
	root := t.TempDir()
	mkClosure(t, filepath.Join(root, "php", "8.4"), 1, 1)

	// 路径逃逸与服务白名单：一律拒绝，不得触碰目录外
	for _, bad := range [][2]string{{"php", "../8.4"}, {"php", "a/b"}, {"../../etc", "8.4"}, {"unknown", "8.4"}, {"php", ""}} {
		if err := Remove(root, bad[0], bad[1]); err == nil {
			t.Errorf("Remove(%q,%q) 应拒绝", bad[0], bad[1])
		}
	}
	if err := Remove(root, "php", "8.4"); err != nil {
		t.Fatalf("合法清理失败: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "php", "8.4")); !os.IsNotExist(err) {
		t.Error("清理后目录应不存在")
	}
	if _, err := os.Stat(root); err != nil {
		t.Errorf("根目录不应被误删: %v", err)
	}
}
