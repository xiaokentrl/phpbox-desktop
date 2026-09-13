package env

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const sampleEnv = `# phpbox 用户覆盖层
PROJECT_NAME=phpbox

WWW_ROOT=/home/me/www
GO_PROXY=https://goproxy.cn,direct
`

func TestRead(t *testing.T) {
	p := filepath.Join(t.TempDir(), ".env")
	os.WriteFile(p, []byte(sampleEnv), 0o600)

	kvs, err := Read(p)
	if err != nil {
		t.Fatal(err)
	}
	// 3 键（注释与空行跳过）
	if len(kvs) != 3 {
		t.Fatalf("期望 3 个键，得到 %d: %+v", len(kvs), kvs)
	}
	byKey := map[string]string{}
	for _, kv := range kvs {
		byKey[kv.Key] = kv.Value
	}
	if byKey["GO_PROXY"] != "https://goproxy.cn,direct" {
		t.Errorf("GO_PROXY 解析错误: %q", byKey["GO_PROXY"])
	}
}

func TestReadQuotesAndMissing(t *testing.T) {
	// 成对引号剥除（对齐 bash _env_strip_quotes）
	p := filepath.Join(t.TempDir(), ".env")
	os.WriteFile(p, []byte("X=\"a b\"\nY='c d'\nZ=e=f\n"), 0o600)
	kvs, _ := Read(p)
	byKey := map[string]string{}
	for _, kv := range kvs {
		byKey[kv.Key] = kv.Value
	}
	if byKey["X"] != "a b" || byKey["Y"] != "c d" {
		t.Errorf("引号剥除错误: %+v", byKey)
	}
	if byKey["Z"] != "e=f" {
		t.Errorf("值含 = 应保留（首个 = 分割）: %+v", byKey)
	}

	// 缺失文件 = 空列表
	kvs, err := Read(filepath.Join(t.TempDir(), "nope"))
	if err != nil || len(kvs) != 0 {
		t.Fatalf("缺失文件应返回空列表: %v %+v", err, kvs)
	}
}

func TestPatch(t *testing.T) {
	p := filepath.Join(t.TempDir(), ".env")
	os.WriteFile(p, []byte(sampleEnv), 0o600)

	err := Patch(p, map[string]string{
		"WWW_ROOT":        "/new/www",    // 已存在：原行替换
		"PGSQL_DATA_ROOT": "/new/pgdata", // 不存在：追加
	})
	if err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(p)
	content := string(data)
	// 注释与空行保留、顺序稳定
	if !strings.HasPrefix(content, "# phpbox 用户覆盖层") {
		t.Errorf("注释丢失:\n%s", content)
	}
	if !strings.Contains(content, "\n\n") {
		t.Errorf("空行丢失:\n%q", content)
	}
	if strings.Count(content, "PROJECT_NAME=phpbox") != 1 {
		t.Errorf("未修改的行被改动:\n%s", content)
	}
	if !strings.Contains(content, "WWW_ROOT=/new/www") || !strings.Contains(content, "PGSQL_DATA_ROOT=/new/pgdata") {
		t.Errorf("替换/追加失败:\n%s", content)
	}

	// 追加的键能被 bash 语义读回
	kvs, _ := Read(p)
	byKey := map[string]string{}
	for _, kv := range kvs {
		byKey[kv.Key] = kv.Value
	}
	if byKey["PGSQL_DATA_ROOT"] != "/new/pgdata" || byKey["WWW_ROOT"] != "/new/www" {
		t.Errorf("写读不一致: %+v", byKey)
	}
}

func TestPatchRejectsInjection(t *testing.T) {
	p := filepath.Join(t.TempDir(), ".env")
	os.WriteFile(p, []byte(sampleEnv), 0o600)
	before, _ := os.ReadFile(p)

	for k, v := range map[string]string{
		"WWW_ROOT": "a\nPROJECT_NAME=evil",
		"K=EY":     "x",
		"GO_PROXY": `he"llo`,
		"NEW_KEY":  "with'quote",
	} {
		if err := Patch(p, map[string]string{k: v}); err == nil {
			t.Errorf("Patch(%q) 应拒绝", k)
		}
	}
	after, _ := os.ReadFile(p)
	if string(before) != string(after) {
		t.Error("拒绝路径不应触碰原文件")
	}
}
