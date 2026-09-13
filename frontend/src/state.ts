// 应用状态（阶段 0：reactive 单例；store 数量增长后迁 Pinia——规约 §一 迁移成本注释）
import { reactive } from 'vue'
import type { ContainerSummary } from '../bindings/github.com/xiaokentrl/phpbox-desktop/internal/engine/docker/models'
import type { ResourceUsage as ResourceUsageModel } from '../bindings/github.com/xiaokentrl/phpbox-desktop/internal/bindings/models'
import { setLocale, t, type Locale } from './i18n'

export type Route = 'sites' | 'php' | 'mysql' | 'pgsql' | 'redis' | 'nginx' | 'go'
  | 'backup' | 'offline' | 'settings' | 'overview' | 'diag'

// 站点行（与 Go SiteEntry 对齐：php 为服务键如 php84）
// health：Go 侧 HEAD 探测结果（up/degraded/down；'' = 未探测）
export interface SiteEntry { domain: string; php: string; root: string; hosts: boolean; health: '' | 'up' | 'degraded' | 'down' }
// @deprecated 壳化重构过渡：改用绑定的 ContainerSummary
export interface ContainerRow { name: string; image: string; state: string }
export interface TaskLine { t: string; c?: '' | 'ok' | 'err' | 'meta' | 'dim' | 'cmd' }
// stage：Go 侧日志阶段推断（runner.go inferPhase：configured/preparing/committed/
// rolling_back/absent）；推断失败为 null——显示原始日志，不臆造状态（ui-spec §8）
export type TaskStage = 'configured' | 'preparing' | 'committed' | 'rolling_back' | 'absent'
export interface Task { label: string; cli: string; lines: TaskLine[]; phase: 'running' | 'success' | 'failed'; stage: TaskStage | null }

// 通知中心条目（§2.2）：kind 对应四个真实信号源；ts 排序用；dedup 时间内同 key 合并
export type NotifKind = 'task_done' | 'task_fail' | 'daemon_down' | 'offline_quota' | 'container_exit'
export interface Notif {
  id: number
  kind: NotifKind
  key: string          // 去重键（如同容器名+事件）
  msg: string
  at: number           // Date.now()
  read: boolean
}

// 危险确认弹窗载荷（§8.2 三条件：警告清单 + 勾选 + 输入匹配）
export interface DangerModal {
  kind: 'danger'
  title: string
  description?: string
  warnings: { text: string; keep?: boolean }[]
  checkboxLabel: string
  inputLabel: string
  expect: string
  placeholder?: string
  cliPreview: string
  confirmLabel: string
  purge?: { label: string } // 附加 --purge 勾选（卸载类）
  onConfirm: (purge: boolean) => void
}
// 安装弹窗载荷
export interface InstallModal {
  kind: 'install'
  svc: string
  title: string
  suggested: string[]
  single?: boolean
}
// 备份归档行（与 Go BackupEntry 对齐：file/path/size/at）
export interface BackupRow { file: string; path: string; size: number; at: string }
// 离线缓存行（与 Go OfflineEntry 对齐）
export interface OfflineRow { svc: string; ver: string; path: string; size: number; files: number; kind: string; lastVerified?: string; lastVerifyOk?: boolean }
// Go 项目行（与 Go GoProjectEntry 对齐）
export interface GoProjectRow { name: string; dir: string; running: boolean }
// .env 行（与 Go EnvKV 对齐；draft 为编辑副本）
export interface EnvRow { key: string; value: string; editable: boolean }
// 长驻进程（daemon）状态：go run / go logs 等永不返回命令的独立通道
export interface DaemonState { id: string; label: string; cli: string; lines: TaskLine[]; running: boolean; failed: boolean }
// PHP 扩展弹窗载荷（真实状态经 Php.ReadPhpExtensions 加载）
export interface ExtModal {
  kind: 'ext'
  version: string
}
// 新建站点弹窗载荷
export interface SiteModal {
  kind: 'site'
}

export const state = reactive({
  route: 'sites' as Route,
  theme: (localStorage.getItem('phpbox-theme') || 'midnight'),
  env: {
    PROJECT_NAME: 'phpbox', WWW_ROOT: '~/www',
    MYSQL_DATA_ROOT: '~/mysql-data', PGSQL_DATA_ROOT: '~/pgsql-data',
    NGINX_PORT: '80', NGINX_VERSION: 'alpine',
    GO_PROJECTS_ROOT: '~/www', GO_PROXY: 'https://goproxy.cn,direct',
  } as Record<string, string>,
  // 真实数据：经 Site 绑定解析 config/nginx/sites/*.conf 加载
  sites: [] as SiteEntry[],
  containers: [] as ContainerSummary[], // 真实 Docker API（总览/服务线/Go 共用；installed 的派生源）
  // 已安装服务线：从真实容器 phpbox-service/phpbox-version labels 派生（与 bash cmd_list 同源），
  // 由 loadContainers 刷新；不在容器里的版本不出现（诚实降级：Docker 不可达时各服务线显示空态）
  installed: {} as Record<string, string[]>,
  // 真实数据：经 Backup 绑定扫描 ~/phpbox/backups/ 加载
  backups: [] as BackupRow[],
  // 真实数据：经 Offline 绑定扫描 ~/phpbox/offline/ 加载
  offlineCache: [] as OfflineRow[],
  // 真实数据：经 GoProjects 绑定发现 ~/www 下的 go.mod 项目
  goProjects: [] as GoProjectRow[],
  // 长驻进程通道（Runner.StartDaemon/StopDaemon）：单槽，与任务队列独立
  daemon: null as DaemonState | null,
  // 各数据域的加载错误（视图切换不丢失）
  dockerErr: '', backupErr: '', offlineErr: '', siteErr: '', envErr: '',
  // 引擎就绪度（presence.Detect）：null = 未探测（浏览器降级）；字段见绑定 PresenceStatus
  presence: null as null | { EngineDir: boolean; CliInPath: string; HasEnv: boolean; DockerOK: boolean; DockerErr: string },
  // 资源占用（Stats.GetResourceUsage，§3.11）：null = 未加载（浏览器降级不显示，不伪造）
  // 直接采用生成绑定类型（engine 类型无 json tag，字段名/可空性与生成器对齐）
  resourceUsage: null as null | ResourceUsageModel,
  // .env 编辑区（settings 模块）
  envRows: [] as EnvRow[],
  envDraft: {} as Record<string, string>,
  // 离线缓存逐条校验状态（key: svc/ver）
  offlineVerifyState: {} as Record<string, string>,
  task: null as Task | null,
  // 通知中心（§2.2）：真实信号源——任务完成/失败、daemon 断连、离线库阈值、
  // 容器意外退出。只记录真实发生的事件，不做定时器轮询类的推断
  notifications: [] as Notif[],
  notifOpen: false,
  modal: null as InstallModal | DangerModal | ExtModal | SiteModal | null,
  palette: false, // 命令面板（⌘K）
  locale: (localStorage.getItem('phpbox-locale') || 'zh-CN') as Locale,
})

export function setRoute(r: Route) { state.route = r }

export function setTheme(id: string) {
  state.theme = id
  document.documentElement.dataset.theme = id
  try { localStorage.setItem('phpbox-theme', id) } catch { /* 忽略 */ }
}
export function initTheme() {
  const saved = localStorage.getItem('phpbox-theme')
  const id = saved === 'midnight' || saved === 'light' || saved === 'oled' || saved === 'forest'
    || saved === 'ocean' || saved === 'sakura' ? saved : 'midnight'
  setTheme(id)
}
export function setAppLocale(l: Locale) {
  state.locale = l
  setLocale(l)
}
export function initLocale() {
  document.documentElement.lang = state.locale
}

/* ─── 弹窗 store：安装 / 危险确认（卸载等破坏性操作 §8.2 三条件）─── */
export function openInstall(m: Omit<InstallModal, 'kind'>): void {
  state.modal = { kind: 'install', ...m }
}
export function openDanger(m: Omit<DangerModal, 'kind'>): void {
  state.modal = { kind: 'danger', ...m }
}
export function openExt(m: Omit<ExtModal, 'kind'>): void {
  state.modal = { kind: 'ext', ...m }
}
export function openSiteModal(): void {
  state.modal = { kind: 'site' }
}
export function closeModal(): void { state.modal = null }

/* ─── 任务引擎（单队列）：桌面版走真实 spawn，浏览器降级模拟（§13）─── */
export interface Step { d: number; lines: (string | TaskLine)[] }

export function toastBus(msg: string, kind: 'ok' | 'err' | 'info', ttl = 3200) {
  window.dispatchEvent(new CustomEvent('phpbox:toast', { detail: { msg, kind, ttl } }))
}

// 任务原语：真实链路（src/api/task.ts 事件订阅）与模拟链路（runTask）共用
export function startTask(label: string, cli: string): boolean {
  if (state.task && state.task.phase === 'running') return false // 单队列（§2.3）
  state.task = { label, cli, lines: [{ t: '$ ' + cli, c: 'cmd' }], phase: 'running', stage: null }
  return true
}
export function pushTaskLine(line: string, cls?: TaskLine['c']): void {
  const task = state.task
  if (!task || task.phase !== 'running') return
  task.lines.push(cls === undefined ? classify(line) : { t: line, c: cls })
}
// 记录任务最新阶段（Go 侧 task:log 事件携带 phase；浏览器模拟任务不产生）
export function pushTaskStage(stage: TaskStage): void {
  const task = state.task
  if (!task || task.phase !== 'running') return
  task.stage = stage
}
export function finishTask(phase: 'success' | 'failed', opts: { doneMsg?: string; onDone?: () => void } = {}): void {
  const task = state.task
  if (!task || task.phase !== 'running') return
  task.phase = phase
  if (phase === 'success') {
    toastBus(opts.doneMsg || t('task.done') + ': ' + task.label, 'ok')
    pushNotif('task_done', task.label, task.label + ' ✓')
    if (opts.onDone) opts.onDone()
  } else {
    toastBus(t('task.failed') + ': ' + task.label, 'err')
    pushNotif('task_fail', task.label, task.label)
  }
}
export function clearTask(): void { state.task = null }
export function taskRunning(): boolean {
  return !!state.task && state.task.phase === 'running'
}

// ── 通知中心（§2.2 四真实源）──
let notifSeq = 0
const notifSeen = new Map<string, number>() // key → 上次推送时间（去重窗口 60s）
const NOTIF_DEDUP_MS = 60_000

export function pushNotif(kind: NotifKind, key: string, msg: string): void {
  const now = Date.now()
  const last = notifSeen.get(kind + '|' + key) ?? 0
  if (now - last < NOTIF_DEDUP_MS) return // 窗口内同源事件合并（如容器反复退出只报一次）
  notifSeen.set(kind + '|' + key, now)
  state.notifications.unshift({ id: ++notifSeq, kind, key, msg, at: now, read: false })
  if (state.notifications.length > 50) state.notifications.length = 50 // 上限防泄漏
}
export function notifUnread(): number { return state.notifications.filter(n => !n.read).length }
export function markNotifsRead(): void { for (const n of state.notifications) n.read = true }
export function clearNotifs(): void { state.notifications = [] }

export function runTask(label: string, cli: string, steps: Step[], opts: { doneMsg?: string; onDone?: () => void } = {}): boolean {
  if (!startTask(label, cli)) return false
  let i = 0
  const next = () => {
    const task = state.task
    if (!task || task.phase !== 'running') return
    if (i >= steps.length) {
      finishTask('success', opts)
      return
    }
    const st = steps[i++]
    for (const raw of st.lines) task.lines.push(typeof raw === 'string' ? classify(raw) : raw)
    setTimeout(next, st.d || 500)
  }
  next()
  return true
}

function classify(line: string): TaskLine {
  if (line.startsWith('[OK]')) return { t: line, c: 'ok' }
  if (line.startsWith('[ERR]')) return { t: line, c: 'err' }
  if (line.startsWith('[INFO]')) return { t: line, c: '' }
  return { t: line, c: 'dim' }
}
