# phpbox Desktop

**中文** | [English](#english)

跨平台桌面应用：多版本 Docker 开发环境管理器（PHP / MySQL / PostgreSQL / Redis / Nginx / Go），bash 版 [phpbox](https://github.com/xiaokentrl/phpbox) 的桌面化演进。

- 技术栈：Wails v3（Go 引擎 + Vue 3 + TypeScript + Naive UI）
- 状态：阶段 0 开发中（引擎 POC）
- 文档：[docs/](docs/) — 设计蓝图 / 技术决策 ADR / UI 规格 / 前端开发规约
- 平台：Linux · macOS · Windows（含系统托盘）
- 许可：MIT

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
