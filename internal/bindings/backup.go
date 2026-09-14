// Package bindings · backup 绑定：扫描 phpbox backups/ 目录 + 归档删除/内容查看/导出。
// 壳层契约：列表/查看只读；创建/恢复经 Runner spawn `phpbox backup`/`restore`（事务在 bash 侧）；
// 删除/导出是普通文件操作（文件即接口），删除已过 GUI 三条件危险确认。
package bindings

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/wailsapp/wails/v3/pkg/application"

	"github.com/xiaokentrl/phpbox-desktop/internal/engine/backup"
)

// Backup 暴露备份归档管理能力。
type Backup struct{}

// BackupEntry 透传引擎类型。
type BackupEntry = backup.Entry

// ListBackups 列出全部备份归档（新在前）。
func (b *Backup) ListBackups() ([]BackupEntry, error) {
	entries, err := backup.List(backupsDir())
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
	path := filepath.Join(backupsDir(), name)
	if err := os.Remove(path); err != nil {
		log.Printf("[绑定] Backup.DeleteBackup(%s) 失败: %v", name, err)
		return err
	}
	log.Printf("[绑定] Backup.DeleteBackup → %s 已删除", name)
	return nil
}

// InspectBackup 查看归档内容（gzip/tar 头流式解析，不解压文件）。
// 与 DeleteBackup 同款文件名校验（防目录逃逸）。
func (b *Backup) InspectBackup(name string) (backup.ArchiveInfo, error) {
	if name == "" || filepath.Base(name) != name || name == "." || name == ".." {
		err := fmt.Errorf("非法备份文件名: %q", name)
		log.Printf("[绑定] Backup.InspectBackup 拒绝: %v", err)
		return backup.ArchiveInfo{}, err
	}
	info, err := backup.InspectArchive(filepath.Join(backupsDir(), name), 2000)
	if err != nil {
		log.Printf("[绑定] Backup.InspectBackup(%s) 失败: %v", name, err)
		return backup.ArchiveInfo{}, err
	}
	log.Printf("[绑定] Backup.InspectBackup(%s) → %d 条目（截断=%v）", name, len(info.Entries), info.Truncated)
	return info, nil
}

// ExportBackup 弹原生保存对话框把归档复制到任意目录（backups/ 本体不动）。
// 返回目标路径；取消返回空串（用户主动取消不是失败）。
func (b *Backup) ExportBackup(ctx context.Context, name string) (string, error) {
	if name == "" || filepath.Base(name) != name || name == "." || name == ".." {
		return "", fmt.Errorf("非法备份文件名: %q", name)
	}
	dst, err := application.Get().Dialog.SaveFile().
		SetFilename(name).
		AddFilter("备份归档 (tar.gz)", "*.tar.gz").
		PromptForSingleSelection()
	if err != nil {
		log.Printf("[绑定] Backup.ExportBackup 对话框失败: %v", err)
		return "", err
	}
	if dst == "" {
		return "", nil // 用户取消
	}
	if err := backup.ExportArchive(filepath.Join(backupsDir(), name), dst); err != nil {
		log.Printf("[绑定] Backup.ExportBackup 写入失败: %v", err)
		return "", err
	}
	log.Printf("[绑定] Backup.ExportBackup → %s", dst)
	return dst, nil
}
