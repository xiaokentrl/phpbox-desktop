package bindings

import (
	"slices"
	"testing"
)

func TestReadPhpExtensionsIntegration(t *testing.T) {
	// 真实环境：本机已装 8.4（extensions.env 含 15 个扩展）
	exts, err := (&Php{}).ReadPhpExtensions("8.4")
	if err != nil {
		t.Skipf("PHP 8.4 状态不可读（%v）：SKIPPED", err)
	}
	if !slices.Contains(exts, "gd") || !slices.Contains(exts, "opcache") {
		t.Errorf("基础扩展缺失: %v", exts)
	}
	// 契约：去重 + 排序（与 bash _php_read_extensions 输出一致）
	for i := 1; i < len(exts); i++ {
		if exts[i] == exts[i-1] {
			t.Errorf("重复扩展: %s", exts[i])
		}
		if exts[i] < exts[i-1] {
			t.Errorf("未排序: %v", exts)
			break
		}
	}
	t.Logf("8.4 → %d 个扩展", len(exts))
}

func TestReadPhpExtensionsRejectsBadVersion(t *testing.T) {
	// 版本号会拼进文件路径：路径形态必须拒绝（目录逃逸防护）
	for _, v := range []string{"", "../8.4", "8.4/x", "..", "a/b"} {
		if _, err := (&Php{}).ReadPhpExtensions(v); err == nil {
			t.Errorf("ReadPhpExtensions(%q) 应拒绝", v)
		}
	}
}
