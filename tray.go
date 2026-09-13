// 系统托盘（设计文档硬需求）：常驻状态、显示/隐藏主窗口、优雅退出。
// 图标复用 build/appicon.png；关闭按钮 X 拦截为隐藏到托盘（真正退出走托盘菜单）。
package main

import (
	_ "embed"
	"runtime"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

//go:embed build/appicon.png
var trayIcon []byte

func setupTray(app *application.App, win *application.WebviewWindow) {
	menu := application.NewMenu()

	// 显示/隐藏主窗口（默认项）
	menu.Add("显示 phpbox").OnClick(func(*application.Context) { win.Show() })
	menu.AddSeparator()

	// 快速打开常用视图（前端路由事件）
	open := func(route string) {
		app.Event.Emit("ui:navigate", map[string]string{"route": route})
		win.Show()
	}
	menu.Add("站点管理").OnClick(func(*application.Context) { open("sites") })
	menu.Add("服务总览").OnClick(func(*application.Context) { open("overview") })
	menu.Add("备份恢复").OnClick(func(*application.Context) { open("backup") })
	menu.AddSeparator()

	// 优雅退出：app.Quit 直接销毁事件循环，不经过窗口关闭链，不会被 X 拦截卡住
	menu.Add("退出").OnClick(func(*application.Context) { app.Quit() })

	tray := app.SystemTray.New()
	tray.SetIcon(trayIcon)
	tray.SetTooltip("phpbox Desktop · Docker LNMP 开发环境")
	tray.SetMenu(menu)

	// 单击托盘 = 显示主窗口
	tray.OnClick(func() { win.Show() })

	// 关闭按钮 X → 隐藏到托盘，不退出。
	// hook 按平台事件注册（三平台各自的关闭事件名），同步执行 + Cancel 短路默认销毁链；
	// app.Quit() 不走此链，退出不受影响。
	hideOnClose := func(event *application.WindowEvent) {
		win.Hide()
		event.Cancel()
	}
	switch runtime.GOOS {
	case "linux":
		win.RegisterHook(events.Linux.WindowDeleteEvent, hideOnClose)
	case "windows":
		win.RegisterHook(events.Windows.WindowClosing, hideOnClose)
	case "darwin":
		win.RegisterHook(events.Mac.WindowShouldClose, hideOnClose)
	}
}
