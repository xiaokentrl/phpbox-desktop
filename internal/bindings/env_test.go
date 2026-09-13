package bindings

import "testing"

// 集成测试：真实读取 ~/phpbox/.env。文件不存在（未安装 bash 引擎）时 SKIPPED。
// 注意：PatchEnv 只测白名单拒绝路径——真实写入会改动用户配置，不在测试中执行。
func TestReadEnvThroughBinding(t *testing.T) {
	kvs, err := (&Env{}).ReadEnv()
	if err != nil {
		t.Skipf(".env 不可读（%v）：SKIPPED", err)
	}
	t.Logf("绑定链路贯通：真实 .env → %d 个键（可编辑 %d）", len(kvs), countEditable(kvs))
	for _, kv := range kvs {
		if kv.Key == "" {
			t.Error("空键名（解析错误）")
		}
	}
	// 白名单键的标记必须与 envEditable 一致
	for _, kv := range kvs {
		if kv.Editable != envEditable[kv.Key] {
			t.Errorf("键 %s 可编辑标记不一致: %v", kv.Key, kv.Editable)
		}
	}
}

func TestPatchEnvRejectsSystemKeys(t *testing.T) {
	// 系统级键必须拒绝（改 PROJECT_NAME/前缀/分隔符会破坏 bash 侧容器命名与标签契约）
	for _, key := range []string{"PROJECT_NAME", "PHP_SERVICE_PREFIX", "LABEL_SEPARATOR", "CURRENT_UID", "IMAGE_PREFIX", "NEW_RANDOM_KEY"} {
		if err := (&Env{}).PatchEnv(map[string]string{key: "x"}); err == nil {
			t.Errorf("PatchEnv(%s) 应被白名单拒绝", key)
		}
	}
}
