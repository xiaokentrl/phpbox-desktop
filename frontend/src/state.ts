// 应用状态（阶段 0：reactive 单例；store 数量增长后迁 Pinia——规约 §一 迁移成本注释）
import { reactive } from 'vue'
import { setLocale, t, type Locale } from './i18n'

export type Route = 'sites' | 'php' | 'mysql' | 'pgsql' | 'redis' | 'nginx' | 'go'
  | 'backup' | 'offline' | 'settings' | 'overview'

export interface SiteEntry { domain: string; php: string; root: string; hosts: boolean; health: 'up' | 'warn' | 'down' }
export interface ContainerRow { name: string; image: string; state: string }
export interface TaskLine { t: string; c?: '' | 'ok' | 'err' | 'meta' | 'dim' | 'cmd' }
export interface Task { label: string; cli: string; lines: TaskLine[]; phase: 'running' | 'success' | 'failed' }

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

export const state = reactive({
  route: 'sites' as Route,
  theme: (localStorage.getItem('phpbox-theme') || 'midnight'),
  env: {
    PROJECT_NAME: 'phpbox', WWW_ROOT: '~/www',
    MYSQL_DATA_ROOT: '~/mysql-data', PGSQL_DATA_ROOT: '~/pgsql-data',
    NGINX_PORT: '80', NGINX_VERSION: 'alpine',
    GO_PROJECTS_ROOT: '~/www', GO_PROXY: 'https://goproxy.cn,direct',
  } as Record<string, string>,
  // 演示数据：桌面版经 bash spawn（phpbox site add 等）读写真实状态
  sites: [
    { domain: 'shop.test', php: '8.4', root: '~/www/shop', hosts: true, health: 'up' },
    { domain: 'legacy.test', php: '7.4', root: '~/www/legacy', hosts: true, health: 'down' },
    { domain: 'api.test', php: '8.4', root: '~/www/api', hosts: false, health: 'warn' },
  ] as SiteEntry[],
  containers: [] as ContainerRow[],
  installed: { php:['8.4','8.2','8.0','7.4'], mysql:['8.4','8.0','5.7'], pgsql:['17'], redis:['8'], nginx:['alpine'] } as Record<string, string[]>,
  // 真实数据：经 Backup 绑定扫描 ~/phpbox/backups/ 加载
  backups: [] as BackupRow[],
  task: null as Task | null,
  modal: null as InstallModal | DangerModal | null,
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
export function closeModal(): void { state.modal = null }

/* ─── 任务引擎（单队列）：桌面版走真实 spawn，浏览器降级模拟（§13）─── */
export interface Step { d: number; lines: (string | TaskLine)[] }

export function toastBus(msg: string, kind: 'ok' | 'err' | 'info', ttl = 3200) {
  window.dispatchEvent(new CustomEvent('phpbox:toast', { detail: { msg, kind, ttl } }))
}

// 任务原语：真实链路（src/api/task.ts 事件订阅）与模拟链路（runTask）共用
export function startTask(label: string, cli: string): boolean {
  if (state.task && state.task.phase === 'running') return false // 单队列（§2.3）
  state.task = { label, cli, lines: [{ t: '$ ' + cli, c: 'cmd' }], phase: 'running' }
  return true
}
export function pushTaskLine(line: string, cls?: TaskLine['c']): void {
  const task = state.task
  if (!task || task.phase !== 'running') return
  task.lines.push(cls === undefined ? classify(line) : { t: line, c: cls })
}
export function finishTask(phase: 'success' | 'failed', opts: { doneMsg?: string; onDone?: () => void } = {}): void {
  const task = state.task
  if (!task || task.phase !== 'running') return
  task.phase = phase
  if (phase === 'success') {
    toastBus(opts.doneMsg || t('task.done') + ': ' + task.label, 'ok')
    if (opts.onDone) opts.onDone()
  } else {
    toastBus(t('task.failed') + ': ' + task.label, 'err')
  }
}
export function clearTask(): void { state.task = null }
export function taskRunning(): boolean {
  return !!state.task && state.task.phase === 'running'
}

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
