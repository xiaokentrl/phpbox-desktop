# phpbox Desktop 总纲

## 1. 项目定位

- **类型**：phpbox（bash 引擎）的跨平台桌面 GUI，Wails v3（Go 壳）+ Vue 3 前端。
- **目标用户：开发者**——给自己搭建本地 Docker 开发环境（LNMP 多版本、站点、Go 项目）用。这不是运维面板，不是生产工具，不是容器管理平台；所有产品决策以开发者工作流为第一优先。
- **与 bash 引擎的关系**：GUI 不替代 CLI，是 CLI 的可视化与聚合面；一切状态变更经 spawn `phpbox` CLI 事务完成（事务/回滚/校验都在 bash 侧），GUI 侧引擎只做只读解析。
- **与 bash 仓的分工**：`~/phpbox`（bash）已冻结为参考实现与事实引擎；本仓是活跃开发区。

## 2. 开发者视角原则（产品决策的最终依据）

- **透明对应 CLI**：每个 GUI 操作都展示等价 `phpbox ...` 命令（命令预览/可复制）。开发者应能从 GUI 学会 CLI，也能随时脱离 GUI 直接用 CLI——GUI 是视图，命令与文件才是事实来源。
- **文件即接口**：vhost、`.env`、`extensions.env`、`backups/`、`offline/` 都是普通文件/目录，开发者可以直接用编辑器改；GUI 读写它们，**禁止创造 GUI-only 的平行状态**。
- **不隐藏终端真相**：任务与长驻进程的输出流式可见（任务抽屉 / daemon 日志卡）；失败输出原样呈现，不吞不错误、不用成功文案掩盖失败。
- **信任但给全信息**：开发者懂 Docker、端口、路径。危险操作弹窗给完整影响清单（删什么、留什么、什么不受影响），由开发者自行决策，不做模糊恐吓，也不做过度防呆。
- **诚实降级**：无 Docker、未装 bash 引擎、浏览器预览时明确显示降级状态（Engine error / 空列表 / 模拟输出），绝不伪造数据。原型虚构过的能力要么实装要么移除（历史案例：offline verify 子命令、restore 相对路径、站点健康列、go run 直跑）。
- **GUI 的价值 = 状态聚合 + 批量便利**，不是唯一入口；任何功能在 CLI 不存在时，不在 GUI 假装它存在。

## 3. 架构事实（当前状态）

```text
main.go/tray.go（窗口、托盘、资产、服务注册）
  → internal/bindings（12 服务 25 方法；只透传不写业务逻辑）
    → internal/engine（docker / site / health / backup / offline / goproject / goimages / env / creds / presence / php / diagbundle / diskusage：只读解析与轻量校验）
      → bash phpbox CLI（事实引擎：变更事务、镜像构建、回滚全在这侧）

前端：Vue 3 + 轻量 i18n（中英双语强制）+ 6 主题 + reactive 单例 state
数据方向：读取 = Go 引擎直读文件与 Docker API；变更 = Runner spawn CLI（RunTask 单任务队列 + daemon 长驻槽）
```

- 路径经 `bindings/paths.go` 统一派生：`BASE_DIR` 硬编码 `~/phpbox`（install.sh 固定布局，.env 无此键）；`OFFLINE_DIR`/`GO_PROJECTS_ROOT`/`NGINX_PORT` 等从 `.env` 读取，路径键按 bash `env.sh` 归一化规则展开（`~` 展开HOME、`./` 相对 BASE_DIR、裸名拼 BASE_DIR、绝对路径原样）。
- spawn 的 CLI 签名、文件格式、容器命名规则**必须先在 bash 仓核实**再实现，禁止凭原型想象。

## 4. 工程执行流程

1. 每切片先核实 bash 侧真实契约（CLI 签名 / 文件格式 / 命名规则）。
2. 实现顺序：引擎 → 绑定（+ main.go 注册 + `wails3 generate bindings -clean=true -ts -i` 重生成）→ 前端。
3. 验证顺序：`vue-tsc` → `vite build` → `go vet` / `go test ./internal/...` → CGO production build → dev 窗口运行时日志。
4. 测试哲学：真实环境集成测试优先（环境不可用时明确 `SKIPPED`）；断言不得依赖演示数据；注入替身走 `emitEvent`/`emitDaemon` 可替换边界。
5. 每个切片一个提交并推送（用户既定规则）；提交信息记录根因与证据。
6. 原型（`docs/ui-preview/`）是设计参考不是事实；与 bash 契约冲突时以 bash 为准，并记录差异。
7. wails3 dev 开发流注意事项：
   - **Go 侧任何变更需要重启 dev**（前端 HMR 不会重编 Go）。
   - `pkill -f` 模式必须用字符类转义（如 `[w]ails3`）防止匹配到执行 shell 自身导致命令自杀。

## 5. 实施优先级顺序（开发顺序总则）

按“最小真实闭环 → 模块扩展”的顺序推进，优先级如下：

1. **Site**：vhost / hosts / 站点切换 / 删除
2. **Docker / service**：容器状态 / 安装 / 卸载 / 扩展
3. **Backup**：列表 / 创建 / 恢复 / 删除
4. **Offline**：扫描 / 校验 / 清理
5. **Go Project**：发现 / 测试 / 停止 / daemon 运行
6. **Settings**：`.env` 读写 / 白名单管理
7. **Tasks**：任务抽屉 / daemon 日志 / 流式状态反馈

原则：
- 每个模块先核实 bash CLI 与文件接口，再实现 engine → binding → frontend。
- 先做最贴近真实业务与已存在事实的模块，减少返工和状态偏差。
- 不在前端制造 GUI-only 状态；功能必须能回溯到真实 CLI / 文件 / Docker 状态。

## 6. 阶段边界

**已接真实数据**：站点（vhost/hosts/切换/删除/**健康探活**——Go 侧 HEAD 127.0.0.1:NGINX_PORT + Host 头路由免 DNS，三态 up/degraded/down，实测 demo.test→403 degraded 为真实结果）、五服务线（安装/卸载/扩展，installed 由容器 phpbox-service/version labels 派生与 cmd_list 同源；**真实连接凭证**（§3.3/3.4）——端口/用户/DSN 掩码经 creds 引擎读 .env 契约键（`<SVC>_<去点版本>_PORT`/`_ROOT_PASSWORD`，install.sh:8/38），密码点击显示自动掩码（时长可配 §3.10）、明文复制可禁；**nginx 快捷操作**（§3.6）——reload/修改端口真实 CLI（nginx/cli.sh），端口读 NGINX_PORT 单实例键，`nginx -t` 仅 bash 内部函数无 CLI 子命令故不设按钮）、**Go 镜像管理**（§3.7）——golang:* 真实镜像表 + InUse（bash uninstall 同款拒绝语义）+ .env GO_DEFAULT_VERSION 默认标记，install/uninstall --purge 经 CLI spawn（goimages 引擎 normalizeTag 即 _go_resolve_version 逆映射）、备份（列表/创建/恢复/删除/**内容查看**（§3.8）——gzip/tar 头流式解析 2000 条截断如实标注、**导出到任意目录**——原生保存对话框 + 原子复制）、离线缓存（扫描/校验——逐 tar 头校验 + 损坏如实报错，结果持久 offline/.verify-state 文件即接口/清理）、Go 项目（发现/测试/停止/daemon 运行）、设置（.env 白名单读写；**密码显示策略**（§3.10 安全组）——GUI 本地偏好 localStorage，不写 .env 造平行状态（bash 无此契约键））、**引擎就绪度检测**（presence 三条件独立：~/phpbox 目录 / phpbox CLI in PATH / Docker 可达 1s ping，缺失时顶部诚实降级横幅指路 bash 仓 install.sh，GUI 不内嵌不代装——§5.1 阶段 0 契约）、**总览聚合**（§3.11 增强——容器表按 Service label 分组 rowspan 聚合 + 无 labels 第三方容器独立区 + 非 running phpbox 实例异常聚合区跳诊断）、托盘（32×32 图标 + 关 X 隐藏）、任务抽屉 + daemon 通道 + **任务阶段日志推断**（ui-spec §8 阶段 0 降级：九态事件由日志行推断——runner.go phaseRules 可配置映射表，模式全部 grep 采集自 bash 真实日志词汇（初始化配置/配置就绪→configured、Docker 构建开始/离线构建→preparing、安装完成→committed、安装失败自动清理/端口变更失败→rolling_back、已卸载→absent）；bash 冻结标记实测仅 [INFO]/[OK]/[ERR]，无 [WARN] 不引入；推断失败留空显示原始日志不臆造；任务抽屉运行中显示阶段徽章）、**诊断视图**（§5.7 阶段 0 降级形态：presence 信号卡 + 异常容器列表 + 容器日志 tail 50 行——engine/docker stdcopy 解复用 + CLI 兜底命令展示；**无一键修复**——bash CLI 无 restart/chown/sock-clean 子命令，修复属 v1.1 Go 引擎前提，不假装）、**诊断包导出**（§10 Go 引擎归宿项落地：七段真实采集——版本/就绪度/容器状态/每容器日志 tail 30/站点+健康/PHP 扩展/.env 密码键强制脱敏；原生保存对话框选路径，engine/php 扩展解析同时从绑定层抽取归位）、**通知中心**（§2.2 四真实信号源：任务完成/失败、daemon 断连、离线库 >2G 阈值、容器意外退出——60s 去重 50 条上限，仅记录真实事件不做轮询推断）。

**v0.1 待办**：无代码项——余下为运行时验证类：Windows 真机运行验证（静态验证已完成：`GOOS=windows CGO_ENABLED=0` 交叉编译 ✓ 出 19.9MB exe、全仓 vet+测试二进制编译 ✓；Windows 上 wails 走 WebView2 纯系统调用无需 CGO。真机未验项：窗口/托盘运行行为、`phpbox` CLI spawn 在 Windows 的可用性——引擎本身是 bash 项目，Windows 侧需 WSL，属运行时事实待真机核实）。已完成的 v0.1 项：站点健康探活（ec894e0）、命令面板 ⌘K（a50ffae）、绑定层路径统一读 .env（b62f0c7）、Dock/desktop file 安装（用户级 `build/linux/install-user.sh`：`~/.local/bin` + hicolor 图标 + `~/.local/share/applications`；desktop file 含 `StartupWMClass=phpbox-desktop`，生成任务行级追加该键，实测 WM_CLASS=二进制 basename 精确匹配）、README（对齐现状重写：模块能力表/验证链/用户级安装，服务/方法计数随绑定重生成输出同步——当前 12 服务 25 方法；04:12 出现于工作区的"内置引擎打包"方向草稿已存 docs/archive/ 待用户裁决，未合入——该方向此前已按 ui-spec §5.1 否决）。

**ui-spec §10 待实现清单收口状态**（2026-09-14）：离线缓存 list/verify/prune ✓（engine/offline 早已落地）；导出诊断包 ✓（281ab8d，engine/diagbundle）；任务阶段推断 ✓（f3c0185，§8 降级形态——九态原生推送留 Go 引擎期）；站点健康 Go 侧探测 ✓（ec894e0）。**任务锁状态查询 + 陈旧锁清理**：bash 引擎实测无任何锁机制（flock/lockfile/并发保护全无，grep 全仓核实）——该能力纯属性 Go 引擎期新增，无 bash 契约可对照，**留待 Go 引擎阶段**，GUI 不提前假装。
