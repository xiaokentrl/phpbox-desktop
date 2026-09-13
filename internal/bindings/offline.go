// Package bindings · offline 绑定：离线缓存列表/校验/清理。
// 校验对齐 bash _ensure_offline_image 的 tar 可读性契约；清理前 GUI 须做三条件危险确认。
package bindings

import (
	"log"

	"github.com/xiaokentrl/phpbox-desktop/internal/engine/offline"
)

// Offline 暴露离线缓存管理能力。
type Offline struct{}

// OfflineEntry 透传引擎类型。
type OfflineEntry = offline.Entry

// OfflineResult 透传引擎类型。
type OfflineResult = offline.Result

// ListOfflineCache 列出全部离线缓存条目。
func (o *Offline) ListOfflineCache() ([]OfflineEntry, error) {
	entries, err := offline.List(offlineDir())
	if err != nil {
		log.Printf("[绑定] Offline.ListOfflineCache 失败: %v", err)
		return nil, err
	}
	log.Printf("[绑定] Offline.ListOfflineCache → %d 个条目", len(entries))
	return entries, nil
}

// VerifyOfflineCache 校验一条缓存（镜像 tar gzip+tar 头 / PHP apk+pecl 闭包）。
func (o *Offline) VerifyOfflineCache(svc, ver string) (OfflineResult, error) {
	res, err := offline.Verify(offlineDir(), svc, ver)
	if err != nil {
		log.Printf("[绑定] Offline.VerifyOfflineCache(%s,%s) 失败: %v", svc, ver, err)
		return OfflineResult{}, err
	}
	log.Printf("[绑定] Offline.VerifyOfflineCache(%s,%s) → ok=%v %s", svc, ver, res.OK, res.Detail)
	return res, nil
}

// PruneOfflineCache 清理一条缓存目录（GUI 已做三条件危险确认；引擎侧再做服务白名单+版本防逃逸）。
func (o *Offline) PruneOfflineCache(svc, ver string) error {
	if err := offline.Remove(offlineDir(), svc, ver); err != nil {
		log.Printf("[绑定] Offline.PruneOfflineCache(%s,%s) 失败: %v", svc, ver, err)
		return err
	}
	log.Printf("[绑定] Offline.PruneOfflineCache → offline/%s/%s 已清理", svc, ver)
	return nil
}
