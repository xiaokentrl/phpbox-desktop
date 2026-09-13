# phpbox Desktop

**中文** | [English](#english)

跨平台桌面应用：多版本 Docker 开发环境管理器（PHP / MySQL / PostgreSQL / Redis / Nginx / Go），bash 版 [phpbox](https://github.com/xiaokentrl/phpbox) 的桌面化演进。

- 技术栈：Wails v3（Go 引擎 + Vue 3 + TypeScript + Naive UI）
- 状态：阶段 0 开发中（引擎 POC）
- 文档：[docs/](docs/) — 设计蓝图 / 技术决策 ADR / UI 规格 / 前端开发规约
- 平台：Linux · macOS · Windows（含系统托盘）
- 许可：MIT

## 技术方案与项目框架

本项目采用 Wails v3 作为桌面应用框架，核心架构是：

- 桌面壳层：Wails v3（Go + Webview）
- 后端引擎：Go，负责系统交互、Docker API、文件与环境读写
- 前端：Vue 3 + TypeScript + Vite，负责界面渲染与交互
- UI 组件：Naive UI（根据实现中已有约定）
- 真实业务事实来源：bash 版 `phpbox` CLI，而非 GUI 自己维护一份并行状态

整体设计遵循“GUI 只读展示 + CLI 驱动事务变更”的原则：

- 读取状态：Go 引擎直接读取配置文件、Docker 状态与环境数据
- 变更操作：通过 `phpbox` CLI spawn 执行真实事务，保证与 bash 版本一致
- 数据边界：vhost、`.env`、`extensions.env`、`backups/`、`offline/` 等视为真实文件接口，不创建 GUI-only 状态

这使得应用既保留桌面体验，又能与现有 bash 工具链保持一致，避免前后端状态脱节。

## 开发

```bash
# 前置：Go 1.27+（Linux 桌面托盘另需 libgtk-3 与 libayatana-appindicator）
task engine:build   # 构建引擎 CLI（phpboxd）
task engine:list    # 列出容器（Engine API POC）
task purity         # 引擎零依赖检查（engine 禁止 import 壳层）
```

## English

Cross-platform desktop app: multi-version Docker dev-environment manager (PHP / MySQL / PostgreSQL / Redis / Nginx / Go) — the desktop evolution of the bash [phpbox](https://github.com/xiaokentrl/phpbox) CLI.

- Stack: Wails v3 (Go engine + Vue 3 + TypeScript + Naive UI)
- Status: phase 0 (engine POC)
- License: MIT

## Technical architecture and project framework

This project uses Wails v3 as its desktop application framework. The architecture is:

- Desktop shell: Wails v3 (Go + Webview)
- Backend engine: Go, responsible for system integration, Docker API access, file and environment handling
- Frontend: Vue 3 + TypeScript + Vite for UI rendering and interactions
- UI components: Naive UI (as used by the current implementation)
- Source of truth for business operations: the bash `phpbox` CLI, not a GUI-only parallel state model

The design follows the principle of “GUI for viewing + CLI for real mutations”:

- Read operations: the Go engine reads configuration files, Docker status, and environment data directly
- Mutation operations: run the actual transaction flow through `phpbox` CLI, keeping behavior aligned with the bash version
- File boundary: vhost files, `.env`, `extensions.env`, `backups/`, and `offline/` are treated as real filesystem interfaces instead of hidden GUI state

This keeps the desktop app user-friendly while preserving compatibility with the existing bash toolchain and avoiding state drift between GUI and CLI.
