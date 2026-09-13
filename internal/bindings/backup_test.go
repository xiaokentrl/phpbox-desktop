package bindings

import (
	"path/filepath"
	"testing"
)

// 集成测试：真实扫描 ~/phpbox/backups/。目录不存在（未安装 bash 引擎）时 SKIPPED。
func TestListBackupsThroughBinding(t *testing.T) {
	dir := backupsDir()
	entries, err := (&Backup{}).ListBackups()
	if err != nil {
		t.Skipf("backups 目录不可读（%v）：SKIPPED", err)
	}
	t.Logf("绑定链路贯通：%s → %d 个归档", dir, len(entries))
	if len(entries) == 0 {
		return
	}
	// 与引擎契约：新在前 + 路径在目录内
	for i := 1; i < len(entries); i++ {
		if entries[i].At.After(entries[i-1].At) {
			t.Errorf("排序失效：第 %d 项比前一项新", i)
		}
		if filepath.Dir(entries[i].Path) != dir {
			t.Errorf("归档路径越界: %s", entries[i].Path)
		}
	}
}

func TestDeleteBackupRejectsPathEscape(t *testing.T) {
	// GUI 只应传裸文件名；含路径/.. 的输入必须被拒绝，不许触碰目录外文件
	for _, name := range []string{"", "..", "../../etc/passwd", "sub/dir/x.tar.gz", "./x.tar.gz"} {
		if err := (&Backup{}).DeleteBackup(name); err == nil {
			t.Errorf("DeleteBackup(%q) 应拒绝路径形态", name)
		}
	}
}
