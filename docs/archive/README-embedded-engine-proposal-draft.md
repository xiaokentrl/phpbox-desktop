# phpbox Desktop

**中文** | [English](#english)

跨平台桌面应用：多版本 Docker 开发环境管理器（PHP / MySQL / PostgreSQL / Redis / Nginx / Go），bash 版 [phpbox](https://github.com/xiaokentrl/phpbox) 的桌面化演进。

- 技术栈：Wails v3（Go 引擎 + Vue 3 + TypeScript + Naive UI）
- 状态：阶段 0 开发中（引擎 POC）
- 文档：[docs/](docs/) — 设计蓝图 / 技术决策 ADR / UI 规格 / 前端开发规约
- 平台：Linux · macOS · Windows（含系统托盘）
- 许可：MIT

## 一、工程总纲：旧项目 + 新项目不是单体重构

本项目的核心设计不是把旧 bash 版 `phpbox` 直接重写成 Go 代码，并不是“把所有业务逻辑硬塞进桌面项目里”。

正确的工程形态是：

- 新项目：桌面前端 + Go 壳层 + 打包入口
- 旧项目：真实业务引擎（bash/脚本）
- 连接方式：通过打包 + CLI 调用的方式把旧项目当作内置引擎
- 事实来源：`phpbox` 命令作为真实事实来源，桌面应用不维护一份独立、并行的 GUI-only 状态

因此，本项目的真实架构是：

- 一个“桌面前端”：负责展示、交互、命令预览和任务抽屉
- 一个“旧脚本引擎”：负责真正的 Docker / vhost / 配置 / 数据卷 / 备份 / 离线缓存 / Go 项目管理逻辑
- 两者通过打包和 CLI 连接：桌面层通过 Go 调用 `phpbox`，而不是直接重写业务逻辑

这才是最现实、最稳妥、最容易打包与后续维护的方案。

### 结论判定

如果要追求“最短时间、最少风险、最容易打包”，那么结论是：

- 不要把旧项目重写成 Go
- 不要在 Go 层复制旧逻辑形成第二套实现
- 把旧 `phpbox` 当作内置引擎打进桌面包
- 用 `phpbox` 命令作为真实事实来源
- 让桌面应用成为“控制层/聚合层”，而不是“业务事实引擎”

这也正是本仓库的工程原则：GUI 只读展示，CLI 驱动真实事务，让 bash 版 `phpbox` 继续承担正确性和回滚责任。

## 二、打包总纲（总纲）

### 2.1 设计目标

打包方案的目标是：

1. 保留旧项目 `phpbox` 的全部真实能力
2. 把它作为新桌面应用的运行时依赖打进去
3. 不把命令逻辑拆成两套实现
4. 保证桌面端启动后，任何写操作都走 `phpbox`，而不是 GUI 自己生成状态
5. 让 AI 能够基于目录结构和步骤自动完成打包、安装、验证

### 2.2 核心原则

- 旧项目是事实引擎，不是“被新项目重写的代码基底”
- 新项目是桌面壳层，与旧项目通过 CLI 接口连接
- 真实文件和目录必须保留原始结构：vhost、`.env`、`extensions.env`、`backups/`、`offline/`、`config/`、`compose/` 等都要保留
- 发布包必须包含完整旧引擎目录，且在运行时可由桌面程序复制到用户目录
- `phpbox` 必须稳定出现在 `PATH` 中，以便 Go 的 `exec.Command("phpbox", ...)` 正常工作

### 2.3 打包架构总览

```text
phpbox-desktop/
  app/                         # 桌面应用本体
  internal/                    # Go 绑定层 / 引擎适配层
  frontend/                    # Vue 3 / TS 前端
  build/                       # 打包脚本、安装脚本、平台配置
  resources/
    phpbox/                    # 旧 bash 引擎的内置资源包
      bin/                     # phpbox 命令和脚本入口
      lib/                     # 共用库、helpers、函数、模块代码
      config/                  # 默认配置目录
      compose/                 # docker-compose 目录
      docs/                    # 旧项目文档（若需要）
      backup/                  # 旧引擎原始备份目录（视实际项目而定）
      offline/                 # 旧引擎缓存目录（运行时可生成）
      logs/                    # 运行日志目录
      .env.example
      install.sh
      README.md
  assets/                      # 图标、桌面启动项等
  cmd/
    phpboxd/                   # 可选：Go 侧引擎 CLI / 未来 server mode
  main.go
  README.md
```

### 2.4 对接方式

打包时，必须把旧项目的真实目录结构复制到新项目的 `resources/phpbox/` 中；安装时再把这些内容同步到用户目录，例如：

```text
~/.local/share/phpbox-desktop/
  phpbox/
    bin/
    lib/
    config/
    compose/
    offline/
    backups/
    logs/
```

或者直接使用：

```text
~/phpbox/
```

如果旧项目原本就是约定到 `~/phpbox` 与 `~/www` 之类目录，就尽量保持一致，不要在新项目里制造另一套并行路径。这样可以保证：

- GUI 读文件的路径和 CLI 一致
- 用户可以直接在编辑器中手动改配置
- 任务日志、恢复、离线缓存、站点配置都继续对应旧项目本体

## 三、子纲：旧项目作为内置引擎的实施方案

### 3.1 子纲目标

将旧项目作为“内置引擎”接入桌面应用，目标是保留业务事实而不重写核心逻辑，重点控制两件事：

1. 复制完整旧项目目录到安装包中
2. 在桌面运行时正确安装/暴露 `phpbox` 到 `PATH`

### 3.2 目录与文件职责

```text
resources/phpbox/
  README.md                    # 说明这是旧 bash 引擎资源包
  install.sh                   # 安装脚本：复制到 ~/.phpbox 或 ~/phpbox
  .env.example                 # 默认环境变量模板
  bin/
    phpbox                     # 主命令入口
    phpboxd                    # 可选：Go 引擎 CLI
    *.sh                       # 旧项目脚本入口
  lib/
    common.sh                  # 通用函数
    env.sh                     # 环境变量定义
    docker.sh                  # Docker 相关函数
    nginx.sh
    site.sh
    backup.sh
    offline.sh
    go.sh
  config/
    nginx/
      sites/
      conf.d/
    php/
    mysql/
    postgres/
    redis/
  compose/
    docker-compose.yml
    services/
  logs/
  offline/
  backups/
  cache/
  tests/
```

### 3.3 运行时安装策略

运行时至少要支持两种形式：

#### 形式 A：内置复制到用户目录

```text
用户启动桌面应用
  → 检测 ~/phpbox 是否存在
  → 若不存在，从 resources/phpbox 复制到 ~/phpbox
  → 设置权限 chmod +x
  → 若需要，注入 PATH 或软链接 /usr/local/bin/phpbox
  → 验证 `phpbox --help` 成功
```

#### 形式 B：直接使用内置资源目录

```text
桌面应用启动
  → 检测 phpbox 是否存在于 PATH
  → 若不存在，使用内置资源目录中的 bin/phpbox 作为运行时命令
  → 配置环境变量，使其读取内置目录中的 config / lib / compose
```

优先使用“形式 A”，因为它最符合旧项目真实工作方式，用户也能直接使用 CLI。

## 四、具体实施流程（可供 AI 自动执行）

下面给出一份可直接流程化执行的方案，用于让 AI 或开发者按顺序落地：

### Step 1：确认旧项目真实界面和目录结构

执行内容：

1. 读取旧项目 `/home/kentrl/phpbox` 的根目录
2. 识别以下关键目录：
   - `bin/`
   - `lib/`
   - `config/`
   - `compose/`
   - `offline/`
   - `backups/`
   - `logs/`
   - `.env.example`
   - `install.sh`
3. 确认旧项目中 `phpbox` 的入口脚本定义方式
4. 确认是否依赖 `~/phpbox`、`~/www`、`docker compose` 等固定目录
5. 确认所有读写操作都依赖真实文件界面，不要新建 GUI-only 数据结构

输出要求：

- 一份目录清单
- 需要保留的关键文件列表
- 需要复制到新项目的资源目录结构

### Step 2：定义新项目中的资源包目录

执行内容：

1. 在新项目根目录创建：
   - `resources/phpbox/`
   - `resources/phpbox/bin/`
   - `resources/phpbox/lib/`
   - `resources/phpbox/config/`
   - `resources/phpbox/compose/`
   - `resources/phpbox/offline/`
   - `resources/phpbox/backups/`
   - `resources/phpbox/logs/`
2. 保持旧项目结构不变，尽量按原目录名复制
3. 不要做“重构化”改造，只做“资源打包整理”

输出要求：

- 资源目录完全和旧项目保持一致
- 配置、脚本、入口文件全部保留

### Step 3：将旧项目核心目录拷贝进新项目资源包

执行内容：

1. 复制旧项目下的：
   - `bin/`
   - `lib/`
   - `config/`
   - `compose/`
   - `install.sh`
   - `.env.example`
   - 相关脚本文件
2. 只在资源层做复制，不在源码层重写业务逻辑
3. 若旧项目包含测试目录、文档或脚本，可按“保留最低运行依赖”的原则取舍

输出要求：

- `resources/phpbox/` 中已包含所有运行时必须的文件
- 资源文件和旧项目目录结构一致

### Step 4：在新项目中编写安装器

执行内容：

1. 新建 Go 文件或脚本，例如：
   - `internal/bootstrap/install_phpbox.go`
   - `build/install-phpbox.sh`
2. 安装逻辑必须：
   - 检测 `~/phpbox` 是否存在
   - 若不存在，则复制 `resources/phpbox` 到 `~/phpbox`
   - 设置执行权限：`chmod +x` 对 `bin/*.sh` 和 `bin/phpbox`
   - 检查命令 `phpbox` 是否可执行
3. 若 `phpbox` 不在 `PATH`，则创建软链接：
   - `/usr/local/bin/phpbox`（Linux）
   - 或使用用户 PATH 配置

输出要求：

- 桌面应用首次启动时可以自动安装旧引擎
- `phpbox --help` 能返回成功结果

### Step 5：在 Go 侧提供统一执行入口

执行内容：

1. 新增统一命令执行器：
   - `Runner.RunTask()`
   - `exec.CommandContext(ctx, "phpbox", args...)`
2. 保持所有写操作都走这个入口
3. 不要在前端、Go 业务代码中另建状态存储
4. 所有任务输出都走事件流，前端展示日志

输出要求：

- 所有变更命令统一由 `phpbox` 驱动
- 任务日志流与 CLI 输出一致

### Step 6：验证真实业务链路

执行内容：

1. 确认 `phpbox list` 能返回数据
2. 确认 `phpbox site add` 等命令依旧能写入真实文件
3. 确认 `.env`、`config/`、`offline/`、`backups/` 等对象在真实文件系统中可见
4. 确认新项目前端读取的内容和 CLI 输出一致

输出要求：

- 真实链路可用
- CLI 是唯一事实来源

### Step 7：确保打包产物可复制到其他机器

执行内容：

1. 在 `build/` 中加入安装脚本：
   - `build/package.sh`
   - `build/install-runtime.sh`
2. 打包时将：
   - 新项目可执行文件
   - 旧项目资源包
   - 安装脚本
   - 平台桌面启动配置
   集成到安装包中
3. 用户安装后自动完成引擎初始化

输出要求：

- 交付物可以在新机器上开箱即用
- 不需要手工复制旧项目源码到用户目录

### Step 8：禁止的实现方式

AI 或开发者在落地时必须避免：

- 不要在 Go 业务中重写旧 `phpbox` 的核心逻辑
- 不要把 GUI 状态当成事实来源
- 不要复制一份“伪状态”配置，不要出现“UI 改了但文件没改”的情况
- 不要假设 Docker API 直接覆盖所有流程
- 不要让前端直接写 `.env` / vhost / backup / offline 文件而绕过 CLI

### Step 9：最终验收标准

交付结果必须满足：

- 桌面应用可以正常启动
- `phpbox` 命令从包内资源被正确安装到用户目录
- `phpbox` 真实功能继续正常工作
- 所有写操作经 CLI 执行
- 读取数据来自真实文件/ Docker / 旧项目状态
- 用户既可以从桌面界面操作，也可以直接用 CLI 继续管理

## 五、AI 自动执行模式（推荐写法）

为了让 AI 能够按流程自动完成，本方案应遵循以下指令格式：

```text
目标：把旧 phpbox bash 引擎整合为桌面应用内置运行时依赖。

任务分解：
1. 扫描旧项目目录结构，确认必须保留的 bin/lib/config/compose 文件。
2. 创建 resources/phpbox/ 目录，按原目录结构复制所有运行时依赖。
3. 编写安装器，确保首次启动时同步到 ~/phpbox。
4. 让 Go 层通过 exec.Command("phpbox", ...) 调用真实 CLI。
5. 保证所有变更仍通过 CLI 执行，不建立 GUI-only 状态。
6. 验证 `phpbox --help`、`phpbox list`、关键操作命令生效。
7. 打包产物必须包含资源包与安装脚本。
```

AI 完成工作时，必须遵守以下硬规则：

- 绝不重写旧项目的核心脚本逻辑
- 绝不制造并行状态
- 绝不绕过 CLI 做直接写入
- 绝不假设“桌面层可以替代 bash 事实引擎”

## 六、项目工程实践结论

最终结论非常明确：

这个项目不是“单体重构项目”，而是“桌面应用 + 内置旧脚本引擎”的组合工程。最现实、最少风险、最容易打包的做法，是：

- 新项目负责 shell + UI + Go 绑定
- 旧项目负责真实业务逻辑与真实事务
- 打包时把旧项目当作内置引擎资源包
- 运行时把 `phpbox` 安装到用户目录并挂到 PATH
- 所有业务写操作统一经 CLI 执行

这样既保持了旧项目的真实性，又保留了桌面应用的可视化与聚合能力，符合“事实来源-CLI-文件接口-桌面视图”这条工程主线。

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

## Packaging architecture and migration strategy

The desktop app is not a monolithic rewrite of the legacy project. Instead, it is a two-part system:

- a desktop frontend and shell; and
- a legacy bash engine that remains the real source of business behavior.

The packaging strategy is to bundle the legacy phpbox engine as an embedded runtime dependency, not to reimplement it in Go.

The runtime flow is:

1. Detect whether the legacy engine exists in the user environment.
2. If missing, copy the bundled engine from the desktop app resource folder into the expected runtime directory, usually `~/phpbox`.
3. Set executable permissions on the legacy shell scripts and command entrypoints.
4. Ensure the `phpbox` binary is available on PATH.
5. Let the Go sidebar and bindings call `phpbox` through `exec.Command("phpbox", ...)`.
6. Keep all writes and transactions inside the legacy CLI, so the GUI never creates a second, contradictory state model.

This is the lowest-risk migration path and the easiest to package for release.

## Packaging directory layout

```text
phpbox-desktop/
  app/
  internal/
  frontend/
  build/
  resources/
    phpbox/
      README.md
      install.sh
      .env.example
      bin/
      lib/
      config/
      compose/
      logs/
      offline/
      backups/
      cache/
      tests/
  cmd/
  main.go
  README.md
```

The important rule is that the embedded resource tree should mirror the original phpbox runtime layout as closely as possible. This preserves compatibility, avoids path bugs, and keeps the CLI usable outside the desktop UI.
