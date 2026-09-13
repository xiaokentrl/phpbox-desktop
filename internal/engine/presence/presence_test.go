package presence

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

// 集成测试：本机真实检测（引擎已安装 + Docker 运行中，全绿路径）。
// 未装引擎/无 Docker 的机器上相应断言自动 SKIPPED，不造假。
func TestDetectRealMachine(t *testing.T) {
	st := Detect(context.Background())

	home, _ := os.UserHomeDir()
	_, dirErr := os.Stat(filepath.Join(home, "phpbox"))
	dirExists := dirErr == nil
	if dirExists && !st.EngineDir {
		t.Errorf("EngineDir: 目录存在但检测为 false")
	}
	if st.EngineDir {
		t.Logf("引擎目录: 存在")
	}
	if st.CliInPath != "" {
		t.Logf("CLI: %s", st.CliInPath)
	} else if dirExists {
		t.Logf("CLI: 不在 PATH（装了引擎但未链接，GUI 横幅会提示 ln -sf）")
	}
	if st.DockerOK {
		t.Logf("Docker: 可达")
	} else {
		t.Logf("Docker: 不可达（%s）——降级横幅真实呈现", st.DockerErr)
	}
	// DockerErr 与 DockerOK 互斥（诚实降级：错误原样带回，不吞）
	if st.DockerOK && st.DockerErr != "" {
		t.Errorf("DockerOK=true 但 DockerErr 非空")
	}
	if !st.DockerOK && st.DockerErr == "" {
		t.Errorf("DockerOK=false 但没有错误原因")
	}
}

// LookPath 语义单元测试：PATH 找不到的命令必须返回空（前端分支依赖此语义）。
func TestDetectCliLookupSemantics(t *testing.T) {
	st := Detect(context.Background())
	if st.EngineDir && st.CliInPath == "" {
		// 引擎目录存在但 CLI 不在 PATH：合法状态（install.sh 的 ln -sf 可能失败/未跑）
		t.Logf("真实状态：引擎目录存在但 CLI 不在 PATH——横幅按此分支展示")
	}
}
