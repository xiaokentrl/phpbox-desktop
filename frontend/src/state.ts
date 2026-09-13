// 应用状态（阶段 0：reactive 单例；store 数量增长后迁 Pinia——规约 §一 迁移成本注释）
import { reactive } from 'vue'
import { setLocale, type Locale } from './i18n'

export type Route = 'sites' | 'php' | 'mysql' | 'pgsql' | 'redis' | 'nginx' | 'go'
  | 'backup' | 'offline' | 'settings' | 'overview'

export interface SiteEntry { domain: string; php: string; root: string; hosts: boolean; health: 'up' | 'warn' | 'down' }
export interface ContainerRow { name: string; image: string; state: string }
export interface TaskLine { t: string; c?: '' | 'ok' | 'err' | 'meta' | 'dim' | 'cmd' }
export interface Task { label: string; cli: string; lines: TaskLine[]; phase: 'running' | 'success' | 'failed' }

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
  backups: [
    { file: 'backup-20260913-0439.tar.gz', size: '1.3 GB', at: '2026-09-13 04:39' },
  ],
  task: null as Task | null,
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

/* ─── 任务引擎（单队列）：桌面版走真实 spawn，浏览器降级模拟（§13）─── */
export interface Step { d: number; lines: (string | TaskLine)[] }

export function toastBus(msg: string, kind: 'ok' | 'err' | 'info', ttl = 3200) {
  window.dispatchEvent(new CustomEvent('phpbox:toast', { detail: { msg, kind, ttl } }))
}

export function runTask(label: string, cli: string, steps: Step[], opts: { doneMsg?: string; onDone?: () => void } = {}): boolean {
  if (state.task && state.task.phase === 'running') return false // 单队列（§2.3）
  const lines: TaskLine[] = [{ t: '$ ' + cli, c: 'cmd' }]
  state.task = { label, cli, lines, phase: 'running' }
  let i = 0
  const next = () => {
    const task = state.task
    if (!task || task.phase !== 'running') return
    if (i >= steps.length) {
      task.phase = 'success'
      toastBus(opts.doneMsg || '任务完成: ' + label, 'ok')
      if (opts.onDone) opts.onDone()
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
