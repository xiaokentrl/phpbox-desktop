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
  → internal/bindings（8 服务 15 方法；只透传不写业务逻辑）
    → internal/engine（docker / backup / site / offline / goproject / env：只读解析与轻量校验）
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

**已接真实数据**：站点（vhost/hosts/切换/删除/**健康探活**——Go 侧 HEAD 127.0.0.1:NGINX_PORT + Host 头路由免 DNS，三态 up/degraded/down，实测 demo.test→403 degraded 为真实结果）、五服务线（安装/卸载/扩展，installed 由容器 phpbox-service/version labels 派生与 cmd_list 同源）、备份（列表/创建/恢复/删除）、离线缓存（扫描/校验/清理）、Go 项目（发现/测试/停止/daemon 运行）、设置（.env 白名单读写）、托盘（32×32 图标 + 关 X 隐藏）、任务抽屉 + daemon 通道。

**v0.1 待办**：Windows 构建验证、Dock/desktop file 安装（`build/linux/phpbox-desktop.desktop` 已生成未安装到 `~/.local/share/applications/`，窗口 WM_CLASS 匹配待验证）、README。已完成的 v0.1 项：站点健康探活（ec894e0）、命令面板 ⌘K（a50ffae）、绑定层路径统一读 .env。
