// phpbox Desktop 主程序：Wails v3 壳层。
// 职责边界（设计文档 §3.3）：窗口/资产/服务注册在此；业务全部在 internal/engine。
package main

import (
	"embed"

	"github.com/wailsapp/wails/v3/pkg/application"

	"github.com/xiaokentrl/phpbox-desktop/internal/bindings"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	app := application.New(application.Options{
		Name:        "phpbox Desktop",
		Description: "Multi-version Docker dev-environment manager",
		Services: []application.Service{
			application.NewService(&bindings.Docker{}),
			application.NewService(&bindings.Runner{}),
			application.NewService(&bindings.Backup{}),
			application.NewService(&bindings.Php{}),
			application.NewService(&bindings.Offline{}),
			application.NewService(&bindings.Site{}),
			application.NewService(&bindings.GoProjects{}),
			application.NewService(&bindings.Env{}),
			application.NewService(&bindings.Presence{}),
			application.NewService(&bindings.Diag{}),
			application.NewService(&bindings.Stats{}),
			application.NewService(&bindings.Creds{}),
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
	})

	win := app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:  "phpbox Desktop",
		Width:  1280,
		Height: 800,
	})

	setupTray(app, win)

	if err := app.Run(); err != nil {
		panic(err)
	}
}
