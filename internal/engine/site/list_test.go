package site

import (
	"os"
	"path/filepath"
	"testing"
)

// 构造一个与 add.sh 模板同构的 vhost
func writeVhost(t *testing.T, dir, domain, phpKey string) {
	t.Helper()
	conf := "server {\n" +
		"    listen 80;\n" +
		"    server_name " + domain + ";\n" +
		"    root /var/www/" + domain + ";\n" +
		"    location ~ \\.php$ {\n" +
		"        resolver 127.0.0.11 valid=10s ipv6=off;\n" +
		"        set $php_upstream " + phpKey + ":9000;\n" +
		"        fastcgi_pass $php_upstream;\n" +
		"    }\n" +
		"}\n"
	if err := os.WriteFile(filepath.Join(dir, domain+".conf"), []byte(conf), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestList(t *testing.T) {
	dir := t.TempDir()
	writeVhost(t, dir, "shop.test", "php84")
	writeVhost(t, dir, "legacy.test", "php74")
	// 备份快照：必须排除
	os.WriteFile(filepath.Join(dir, ".backup.shop.test.conf"), []byte("server{}"), 0o600)
	// 非 conf：排除
	os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("x"), 0o600)

	entries, err := List(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Fatalf("期望 2 个站点，得到 %d: %+v", len(entries), entries)
	}
	// 排序契约：域名字典序
	byDomain := map[string]Entry{}
	for _, e := range entries {
		byDomain[e.Domain] = e
	}
	if byDomain["shop.test"].PHP != "php84" {
		t.Errorf("shop.test upstream 解析错误: %+v", byDomain["shop.test"])
	}
	// root 契约：容器 /var/www/<域名> 映射回宿主 ~/www/<域名>（WWW_ROOT 默认值）
	if byDomain["legacy.test"].Root != "~/www/legacy.test" {
		t.Errorf("root 行解析错误: %+v", byDomain["legacy.test"])
	}
}

func TestListCustomRoot(t *testing.T) {
	// 用户手改 vhost root 到非 /var/www 前缀：原样保留，不臆造映射
	dir := t.TempDir()
	conf := "server { fastcgi_pass php80:9000; root /srv/custom; }"
	if err := os.WriteFile(filepath.Join(dir, "custom.test.conf"), []byte(conf), 0o600); err != nil {
		t.Fatal(err)
	}
	entries, err := List(dir)
	if err != nil || len(entries) != 1 {
		t.Fatalf("自定义 root 站点应能解析: %v %+v", err, entries)
	}
	if entries[0].Root != "/srv/custom" {
		t.Errorf("非 /var/www 前缀的 root 应原样返回: %+v", entries[0])
	}
}

func TestListLegacyVhost(t *testing.T) {
	// 旧版模板：直接 fastcgi_pass php80:9000，无 set 变量
	dir := t.TempDir()
	conf := "server { fastcgi_pass php80:9000; root /var/www/old.test; }"
	if err := os.WriteFile(filepath.Join(dir, "old.test.conf"), []byte(conf), 0o600); err != nil {
		t.Fatal(err)
	}
	entries, err := List(dir)
	if err != nil || len(entries) != 1 {
		t.Fatalf("旧版站点应能解析: %v %+v", err, entries)
	}
	if entries[0].PHP != "php80" {
		t.Errorf("旧版 fastcgi_pass 解析错误: %+v", entries[0])
	}
}

func TestListMissingDir(t *testing.T) {
	entries, err := List(filepath.Join(t.TempDir(), "nope"))
	if err != nil {
		t.Fatalf("缺失目录应返回空列表: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("期望空列表，得到 %d 项", len(entries))
	}
}
