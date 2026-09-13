// 数据加载层：各 Wails 绑定 → state 单例。视图与壳只调用，不直接碰绑定。
import { inWails } from './task'
import { state, toastBus, pushNotif, taskRunning, type OfflineRow, type SiteEntry, type GoProjectRow, type BackupRow, type EnvRow } from '../state'
import { cmpVerDesc } from '../utils'
import { ListContainers } from '../../bindings/github.com/xiaokentrl/phpbox-desktop/internal/bindings/docker'
import { ListBackups } from '../../bindings/github.com/xiaokentrl/phpbox-desktop/internal/bindings/backup'
import { ListOfflineCache } from '../../bindings/github.com/xiaokentrl/phpbox-desktop/internal/bindings/offline'
import { ListSites, ProbeSiteHealth } from '../../bindings/github.com/xiaokentrl/phpbox-desktop/internal/bindings/site'
import { ListGoProjects } from '../../bindings/github.com/xiaokentrl/phpbox-desktop/internal/bindings/goprojects'
import { ReadEnv } from '../../bindings/github.com/xiaokentrl/phpbox-desktop/internal/bindings/env'
import { Detect as DetectPresence } from '../../bindings/github.com/xiaokentrl/phpbox-desktop/internal/bindings/presence'

// 引擎就绪度（§5.1 首启检测）：三条件独立呈现，不合并布尔——开发者要分别知道缺什么。
// 浏览器降级保留 null（横幅不显示，不做假检测）。
export async function loadPresence() {
  if (!inWails()) return
  try {
    state.presence = await DetectPresence() ?? null
  } catch { state.presence = null }
}

// 容器列表（总览/服务线/Go 状态共用的真实数据源）+ installed 派生
// installed 与 bash cmd_list 同源：phpbox-service/phpbox-version labels（缺 label 的容器不纳入）
// 通知源（§2.2）：容器意外退出 = 上次刷新时 running、本次 exited 且非任务运行中
// （任务运行中的退出是事务的正常部分——如卸载重建，不误报）
const prevContainerStates = new Map<string, string>()
export async function loadContainers() {
  state.dockerErr = ''
  try {
    const rows = (await ListContainers()) ?? []
    state.containers = rows
    const map: Record<string, Set<string>> = {}
    for (const c of rows) {
      if (!c.Service) continue
      // nginx 单实例无 version label：回退真实镜像 tag（nginx:alpine → alpine），不假设默认值
      ;(map[c.Service] ??= new Set()).add(c.Version || c.Image.split(':')[1] || '?')
    }
    state.installed = {}
    for (const [svc, set] of Object.entries(map)) {
      state.installed[svc] = [...set].sort(cmpVerDesc)
    }
    // 意外退出检测：仅对 phpbox 管理容器（有 Service），且无任务在跑（任务中退出属事务）
    if (!taskRunning()) {
      for (const c of rows) {
        const name = c.Name.replace(/^\//, '')
        const prev = prevContainerStates.get(name)
        if (c.Service && prev === 'running' && c.State !== 'running') {
          pushNotif('container_exit', name, `${name} → ${c.State}`)
        }
      }
    }
    prevContainerStates.clear()
    for (const c of rows) prevContainerStates.set(c.Name.replace(/^\//, ''), c.State)
  } catch (e) { state.dockerErr = String(e) }
}

export async function loadBackups() {
  state.backupErr = ''
  if (!inWails()) return // 浏览器降级：保留空列表
  try {
    const rows = (await ListBackups()) ?? []
    state.backups = rows.map(r => ({ file: r.file, path: r.path, size: Number(r.size), at: String(r.at) })) as BackupRow[]
  } catch (e) { state.backupErr = String(e) }
}

export async function loadOffline() {
  state.offlineErr = ''
  if (!inWails()) return
  try {
    const rows = (await ListOfflineCache()) ?? []
    state.offlineCache = rows.map(r => ({
      svc: r.svc, ver: r.ver, path: r.path, size: Number(r.size), files: Number(r.files), kind: r.kind,
    })) as OfflineRow[]
    // 通知源（§2.2）：离线库阈值 > 2G 进通知中心（真实字节数求和，非估算）
    const total = state.offlineCache.reduce((s, r) => s + r.size, 0)
    if (total > 2 * 1024 ** 3) {
      pushNotif('offline_quota', 'offline', `${(total / 1024 ** 3).toFixed(1)}G`)
    }
  } catch (e) { state.offlineErr = String(e) }
}

export async function loadSites() {
  state.siteErr = ''
  if (!inWails()) return
  try {
    const rows = (await ListSites()) ?? []
    state.sites = rows.map(r => ({ domain: r.domain, php: r.php, root: r.root, hosts: !!r.hosts, health: '' })) as SiteEntry[]
    probeSites() // 健康探测异步补齐（失败不影响列表展示）
  } catch (e) { state.siteErr = String(e) }
}

// 站点健康：Go 侧 HEAD 探测（WebView fetch 跨源读不到状态码），结果逐站点回写
async function probeSites() {
  const domains = state.sites.map(s => s.domain)
  await Promise.all(domains.map(async d => {
    try {
      const res = await ProbeSiteHealth(d)
      const s = state.sites.find(x => x.domain === d)
      if (s && (res.status === 'up' || res.status === 'degraded' || res.status === 'down')) s.health = res.status
    } catch { /* 单站点探测失败保留 ''（未探测），不污染整体 */ }
  }))
}

export async function loadGoProjects() {
  if (!inWails()) return
  try {
    const rows = (await ListGoProjects()) ?? []
    state.goProjects = rows.map(r => ({ name: r.name, dir: r.dir, running: !!r.running })) as GoProjectRow[]
  } catch (e) { toastBus(String(e), 'err', 5000) }
}

export async function loadEnv() {
  state.envErr = ''
  if (!inWails()) return
  try {
    const rows = (await ReadEnv()) ?? []
    state.envRows = rows.map(r => ({ key: r.key, value: r.value, editable: !!r.editable })) as EnvRow[]
    const d: Record<string, string> = {}
    for (const r of rows) d[r.key] = r.value
    state.envDraft = d
  } catch (e) { state.envErr = String(e) }
}

// 启动入口在 App.vue onMounted（loadContainers 即时 + onWailsReady 其余域）。
// 不提供聚合函数：各域单独调用，避免二次包装造成加载时机分裂。
