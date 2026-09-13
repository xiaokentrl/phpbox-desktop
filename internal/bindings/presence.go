// Package bindings · presence 绑定：引擎就绪度检测（ui-spec §5.1 首启 Onboarding）。
// 只读透传；安装动作不做——引导文案指路 bash 仓 install.sh（阶段 0 契约）。
package bindings

import (
	"context"

	"github.com/xiaokentrl/phpbox-desktop/internal/engine/presence"
)

// Presence 暴露引擎存在性检测能力。
type Presence struct{}

// PresenceStatus 透传引擎类型。
type PresenceStatus = presence.Status

// Detect 返回三条件快照（引擎目录 / CLI PATH / Docker 可达）。
func (p *Presence) Detect(ctx context.Context) PresenceStatus {
	st := presence.Detect(ctx)
	return st
}
