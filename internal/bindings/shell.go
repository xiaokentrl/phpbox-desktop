// Package bindings · shell 绑定：宿主桌面集成（打开浏览器/文件管理器）。
// 只透传 Wails BrowserManager，不碰任何 phpbox 状态——与引擎读写无关的纯桌面便利。
package bindings

import (
	"context"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// Shell 暴露宿主桌面集成能力。
type Shell struct{}

// validSiteURL 站点地址白名单：仅 http(s) 完整前缀（防 file:// 等任意协议经 webview 打开）。
func validSiteURL(url string) bool {
	return len(url) >= 8 && (url[:7] == "http://" || url[:8] == "https://")
}

// OpenSiteBrowser 用系统默认浏览器打开站点 URL。
// url 必须是 http(s) 完整地址（前端拼接端口）；WebView 内 <a target=_blank> 不可靠，统一走此绑定。
func (s *Shell) OpenSiteBrowser(ctx context.Context, url string) error {
	if !validSiteURL(url) {
		return errInvalidURL
	}
	return application.Get().Browser.OpenURL(url)
}

// OpenInFileManager 用系统文件管理器打开目录（站点根目录等）。
// 目录不存在时原样返回错误，由前端如实提示。
func (s *Shell) OpenInFileManager(ctx context.Context, dir string) error {
	if dir == "" {
		return errInvalidURL
	}
	return application.Get().Browser.OpenFile(dir)
}

var errInvalidURL = &invalidURLError{}

type invalidURLError struct{}

func (e *invalidURLError) Error() string { return "非法地址参数" }
