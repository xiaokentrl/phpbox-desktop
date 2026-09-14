# phpbox Desktop

**中文** | [English](#english)

跨平台桌面应用：多版本 Docker 开发环境管理器（PHP / MySQL / PostgreSQL / Redis / Nginx / Go），bash 版 [phpbox](https://github.com/xiaokentrl/phpbox) 的桌面化演进。

- 技术栈：Wails v3（Go 壳）+ Vue 3 + TypeScript + Vite
- 状态：阶段 0（全部模块已接真实数据，v0.1 收尾中）
- 文档：[docs/](docs/) — 设计蓝图 / 技术决策 ADR / UI 规格 / 前端开发规约 · 总纲见 [AGENTS.md](AGENTS.md)
- 平台：Linux（已实测）· macOS · Windows
- 许可：MIT

## 架构：GUI 是视图，CLI 与文件才是事实来源

桌面应用不重写 bash 引擎，也不维护平行状态：

```text
main.go / tray.go                 窗口、托盘、资产、服务注册
  → internal/bindings             8 服务 17 方法，只透传不写业务逻辑
    → internal/engine             只读解析：docker / site / health / backup / offline / goproject / env
      → bash phpbox CLI           事实引擎：一切变更事务、镜像构建、回滚在 bash 侧完成
```

- **读取**：Go 引擎直读真实文件（vhost / `.env` / `extensions.env` / `backups/` / `offline/`）与 Docker Engine API（本机容器状态、服务标签）。路径由 `bindings/paths.go` 统一派生——`BASE_DIR=~/phpbox` 固定布局，`OFFLINE_DIR` / `GO_PROJECTS_ROOT` / `NGINX_PORT` 等从 `.env` 读取，归一化规则逐行对齐 bash `lib/common/env.sh`。
- **变更**：Runner spawn `phpbox` CLI（`RunTask` 单任务队列 + daemon 长驻槽），任务输出流式进入任务抽屉，失败原样呈现。
- **安装状态**：以容器标签 `phpbox-service` / `phpbox-version` 为唯一事实来源，与 CLI `phpbox list` 同源。
- **每个 GUI 操作展示等价 CLI 命令**（可复制）——开发者可以从 GUI 学会 CLI，也可以随时脱离 GUI 直接用 CLI。

## 模块能力（全部真实数据，无模拟）

| 模块 | 能力 |
|---|---|
| 站点 | vhost 列表 / PHP 版本切换 / 删除 / hosts 解析显示 / HTTP 探活（三态 up / degraded / down，Go 侧 HEAD 127.0.0.1 + Host 头路由） |
| 服务线 | 五条服务线（PHP / MySQL / PostgreSQL / Redis / Nginx）容器状态 / 安装 / 卸载 / PHP 扩展管理（`extensions.env` 文件为事实来源） |
| 备份 | 列表 / 创建 / 恢复 / 删除（路径逃逸拒绝） |
| 离线缓存 | 扫描 / 校验（tar 与 gzip 头检查）/ 清理 |
| Go 项目 | 发现 / 测试 / 停止 / daemon 运行 |
| 设置 | `.env` 读写（系统级键白名单保护，行级 patch 保留注释与顺序） |
| 任务 | 任务抽屉流式输出 + daemon 日志卡 + 命令面板 ⌘K |

## 开发

前置：Go 1.27+、Node 20+、Docker；Linux 另需 `libgtk-3-dev` 与 `libayatana-appindicator3-dev`（托盘）。

```bash
# bash 引擎（事实来源）：先安装 CLI
~/phpbox/install.sh        # 或参考 phpbox 仓 README

# 桌面应用
wails3 task dev            # 开发模式（Go 侧改动需重启 dev，前端 HMR）
wails3 task build          # 生产构建 → bin/phpbox-desktop

# 引擎 POC 工具（阶段 0 遗留；go-task 未装时可 wails3 task 执行同名任务）
task engine:build         # 构建 phpboxd
task engine:list          # Engine API 容器列举
task purity                # 引擎层禁止 import 壳层检查

# 验证链（每个切片的必跑顺序）
cd frontend && npx vue-tsc --noEmit && npm run build
go vet ./... && go test ./internal/...
go build -o /tmp/phpbox-desktop .   # CGO 生产构建

# 绑定重生成（Go 方法签名变更后）
wails3 generate bindings -clean=true -ts -i
```

## 安装（Linux 用户级，无 root）

```bash
wails3 task build
./build/linux/install-user.sh
```

安装三个文件：`~/.local/bin/phpbox-desktop`、`~/.local/share/icons/hicolor/128x128/apps/phpbox-desktop.png`、`~/.local/share/applications/phpbox-desktop.desktop`（含 `StartupWMClass`，实测与窗口 WM_CLASS 精确匹配）。应用菜单搜索 phpbox-desktop 即可启动；卸载即删除上述三个文件。

## English

Cross-platform desktop app: multi-version Docker dev-environment manager (PHP / MySQL / PostgreSQL / Redis / Nginx / Go) — the desktop evolution of the bash [phpbox](https://github.com/xiaokentrl/phpbox) CLI.

- Stack: Wails v3 (Go shell) + Vue 3 + TypeScript + Vite
- Status: phase 0 (all modules on real data; v0.1 wrap-up)
- Docs: [docs/](docs/) — design blueprint / ADR / UI spec / frontend guidelines · project charter in [AGENTS.md](AGENTS.md)
- License: MIT

## Architecture: the GUI is a view; the CLI and files are the source of truth

The desktop app does not rewrite the bash engine and keeps no parallel state:

```text
main.go / tray.go                 window, tray, assets, service registration
  → internal/bindings             13 services / 31 methods, thin pass-through
    → internal/engine             read-only parsing: docker / site / health / backup / offline / goproject / goimages / env / creds / presence / php / diagbundle / diskusage / faultmode
      → bash phpbox CLI           the engine: all mutating transactions, image builds, rollback
```

- **Reads**: the Go engine reads real files (vhost / `.env` / `extensions.env` / `backups/` / `offline/`) and the Docker Engine API (local containers, service labels). Paths derive from `bindings/paths.go` — fixed `BASE_DIR=~/phpbox` layout; `OFFLINE_DIR` / `GO_PROJECTS_ROOT` / `NGINX_PORT` etc. come from `.env` with normalization rules mirroring bash `lib/common/env.sh` line by line.
- **Mutations**: the Runner spawns the `phpbox` CLI (`RunTask` single-task queue + a daemon slot); task output streams into the task drawer, failures shown verbatim.
- **Install state**: container labels `phpbox-service` / `phpbox-version` are the single source of truth, same as CLI `phpbox list`.
- **Every GUI action shows its equivalent CLI command** (copyable) — learn the CLI from the GUI, or leave the GUI for the CLI at any time.

## Modules (all real data, no mocks)

| Module | Capabilities |
|---|---|
| Sites | vhost list / PHP-version switch / removal / hosts resolution / HTTP health probe (up / degraded / down; HEAD via 127.0.0.1 + Host header on the Go side) / batch select with batch hosts + batch delete (sequential CLI spawn) / open in browser & file manager (Shell binding, port auto-appended) |
| Services | five service lines (PHP / MySQL / PostgreSQL / Redis / Nginx): container status / install / uninstall / PHP extensions (the `extensions.env` file is the source of truth) / real port & password reveal (timeout configurable) / DSN copy / nginx reload & port change |
| Backup | list / create / restore / delete (path-escape rejected) / archive contents viewer (gzip/tar header stream) / export to any directory (atomic copy) |
| Offline cache | scan / verify (tar & gzip header checks, per-entry last-verified time persisted in `offline/.verify-state`) / prune |
| Go projects | discover / test / stop / daemon run |
| Go images | golang:* listing with in-use state (bash-identical uninstall refusal) / install / uninstall --purge |
| Settings | `.env` read/write (system keys whitelisted; line-level patch keeps comments and order) / password display policy (local preference, not `.env`) |
| Diagnostics | presence banner (engine/CLI/env/Docker) / abnormal containers / log tail / known failure-pattern matching (§8.1 phase-0, read-only with real CLI ways out) / export diagnostic bundle (secrets masked) |
| Tasks | streaming task drawer + daemon log card + ⌘K command palette / resizable sidebar & drawer |

## Development

Prerequisites: Go 1.25+, Node 20+, Docker; on Linux the Wails v3 shell builds against GTK4: `libgtk-4-dev`, `libwebkitgtk-6.0-dev`, `libx11-dev` (and `libayatana-appindicator3-dev` for the tray). `go vet`/`go test` type-check the cgo'ed wails package too, so these headers are needed for the verification chain itself, not just the final build.

```bash
# bash engine (source of truth): install the CLI first
~/phpbox/install.sh        # or see the phpbox repo README

# desktop app
wails3 task dev            # dev mode (Go changes need a dev restart; frontend is HMR)
wails3 task build          # production build → bin/phpbox-desktop

# verification chain (run in this order for every slice)
cd frontend && npx vue-tsc --noEmit && npm run build
go vet ./... && go test ./internal/...
go build -o /tmp/phpbox-desktop .

# regenerate bindings (after Go method signature changes)
wails3 generate bindings -clean=true -ts -i
```

CI runs the same chain on every push/PR to `main` (`.github/workflows/ci.yml`): linux verify (vue-tsc → vite → vet → test → CGO build) plus a `GOOS=windows` cross-compile job asserting a PE32+ executable. Integration tests that need a live Docker daemon or the `phpbox` CLI auto-SKIP in CI — a green run proves the static chain, not full integration.

## Install (Linux, user-level, no root)

```bash
wails3 task build
./build/linux/install-user.sh
```

Installs three files: `~/.local/bin/phpbox-desktop`, `~/.local/share/icons/hicolor/128x128/apps/phpbox-desktop.png`, and `~/.local/share/applications/phpbox-desktop.desktop` (with `StartupWMClass`, verified to match the window's WM_CLASS exactly). Launch from the app menu; uninstall by removing those three files.
