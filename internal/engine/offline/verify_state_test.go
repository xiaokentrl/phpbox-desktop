package offline

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// 验证状态读写往返 + 转义（detail 含 | 的边界）。
func TestVerifyStateRoundTrip(t *testing.T) {
	dir := t.TempDir()
	all := ReadVerifyState(dir)
	if len(all) != 0 {
		t.Fatalf("空目录应有零记录，得到 %d", len(all))
	}
	WriteVerifyState(dir, "mysql", "8.0", true, "mysql-8.0.tar 可读（gzip/tar 头校验通过）")
	WriteVerifyState(dir, "php", "8.4", false, "apk 闭包缺少|x 构建依赖")
	all = ReadVerifyState(dir)
	if len(all) != 2 {
		t.Fatalf("应有 2 条记录，得到 %d", len(all))
	}
	if !all["mysql/8.0"].OK {
		t.Error("mysql/8.0 应为 ok")
	}
	if all["php/8.4"].Detail != "apk 闭包缺少|x 构建依赖" {
		t.Errorf("detail 转义往返失败: %q", all["php/8.4"].Detail)
	}
	if time.Since(all["mysql/8.0"].At) > time.Minute {
		t.Error("时间戳异常")
	}
	RemoveVerifyState(dir, "php", "8.4")
	all = ReadVerifyState(dir)
	if len(all) != 1 {
		t.Fatalf("清理后应剩 1 条，得到 %d", len(all))
	}
	if _, ok := all["php/8.4"]; ok {
		t.Error("php/8.4 记录应已删除")
	}
}

// List 附着 LastVerified 的集成验证（临时目录构造缓存结构）。
func TestListCarriesVerifyState(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "mysql", "8.0"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "mysql", "8.0", "mysql-8.0.tar"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	WriteVerifyState(dir, "mysql", "8.0", true, "ok")
	entries, err := List(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].LastVerified == "" || !entries[0].LastVerifyOK {
		t.Fatalf("List 应附着验证状态: %+v", entries)
	}
}
