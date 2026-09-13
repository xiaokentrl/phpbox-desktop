// Package bindings · backup 绑定：扫描 phpbox backups/ 目录 + 归档删除。
// 壳层契约：列表只读；创建/恢复经 Runner spawn `phpbox backup`/`restore`（事务在 bash 侧）。
package bindings

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/xiaokentrl/phpbox-desktop/internal/engine/backup"
)

// Backup 暴露备份归档管理能力。
type Backup struct{}

// BackupEntry 透传引擎类型。
type BackupEntry = backup.Entry

// backupDir phpbox 的备份目录（env.sh 默认：BASE_DIR/backups；BASE_DIR=$HOME/phpbox）。
// 暂不读用户 .env 覆盖——阶段 0 固定默认，v0.1 引入 .env 解析后统一。
func backupDir() string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return "backups"
	}
	return filepath.Join(home, "phpbox", "backups")
}

// ListBackups 列出全部备份归档（新在前）。
func (b *Backup) ListBackups() ([]BackupEntry, error) {
	entries, err := backup.List(backupDir())
	if err != nil {
		log.Printf("[绑定] Backup.ListBackups 失败: %v", err)
		return nil, err
	}
	log.Printf("[绑定] Backup.ListBackups → %d 个归档", len(entries))
	return entries, nil
}

// DeleteBackup 删除一份归档文件（GUI 侧已做三条件危险确认；此处拒绝含路径的文件名，防目录逃逸）。
func (b *Backup) DeleteBackup(name string) error {
	if name == "" || filepath.Base(name) != name || name == "." || name == ".." {
		err := fmt.Errorf("非法备份文件名: %q", name)
		log.Printf("[绑定] Backup.DeleteBackup 拒绝: %v", err)
		return err
	}
	path := filepath.Join(backupDir(), name)
	if err := os.Remove(path); err != nil {
		log.Printf("[绑定] Backup.DeleteBackup(%s) 失败: %v", name, err)
		return err
	}
	log.Printf("[绑定] Backup.DeleteBackup → %s 已删除", name)
	return nil
}
