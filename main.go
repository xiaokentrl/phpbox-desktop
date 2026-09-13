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
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
	})

	app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:  "phpbox Desktop",
		Width:  1280,
		Height: 800,
	})

	if err := app.Run(); err != nil {
		panic(err)
	}
}
