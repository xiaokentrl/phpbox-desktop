// Package bindings · diag 绑定：诊断包导出（ui-spec §5.7 未知故障分支 → §10 Go 引擎归宿）。
// 内容契约：日志+配置+版本信息（引擎 diagbundle.Build）；密码键强制脱敏。
package bindings

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"

	"github.com/xiaokentrl/phpbox-desktop/internal/engine/diagbundle"
)

// Diag 暴露诊断包导出能力。
type Diag struct{}

// ExportDiagnosticBundle 弹原生保存对话框选目标路径，写入诊断包文本。
// 返回实际写入路径；取消返回空串（用户主动取消不是失败，无错误）。
func (d *Diag) ExportDiagnosticBundle(ctx context.Context) (string, error) {
	path, err := application.Get().Dialog.SaveFile().
		SetFilename("phpbox-diag-" + time.Now().Format("20060102-150405") + ".txt").
		AddFilter("诊断包 (txt)", "*.txt").
		PromptForSingleSelection()
	if err != nil {
		log.Printf("[绑定] Diag.ExportDiagnosticBundle 对话框失败: %v", err)
		return "", err
	}
	if path == "" {
		return "", nil // 用户取消
	}
	content := diagbundle.Build(ctx, diagbundle.Sources{
		EnvFile:      envFile(),
		SitesDir:     sitesDir(),
		PhpConfigDir: phpConfigDir,
		NginxPort:    nginxPort(),
		LogTail:      30,
	})
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		log.Printf("[绑定] Diag.ExportDiagnosticBundle 写入失败: %v", err)
		return "", err
	}
	log.Printf("[绑定] Diag.ExportDiagnosticBundle → %s（%d 字节）", path, len(content))
	return path, nil
}
