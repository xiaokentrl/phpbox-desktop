// 数据加载层：各 Wails 绑定 → state 单例。视图与壳只调用，不直接碰绑定。
import { inWails, onWailsReady } from './task'
import { state, toastBus, type OfflineRow, type SiteEntry, type GoProjectRow, type BackupRow, type EnvRow } from '../state'
import { ListContainers } from '../../bindings/github.com/xiaokentrl/phpbox-desktop/internal/bindings/docker'
import { ListBackups } from '../../bindings/github.com/xiaokentrl/phpbox-desktop/internal/bindings/backup'
import { ListOfflineCache } from '../../bindings/github.com/xiaokentrl/phpbox-desktop/internal/bindings/offline'
import { ListSites } from '../../bindings/github.com/xiaokentrl/phpbox-desktop/internal/bindings/site'
import { ListGoProjects } from '../../bindings/github.com/xiaokentrl/phpbox-desktop/internal/bindings/goprojects'
import { ReadEnv } from '../../bindings/github.com/xiaokentrl/phpbox-desktop/internal/bindings/env'

// 容器列表（总览/服务线/Go 状态共用的真实数据源）
export async function loadContainers() {
  state.dockerErr = ''
  try { state.containers = (await ListContainers()) ?? [] }
  catch (e) { state.dockerErr = String(e) }
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
  } catch (e) { state.offlineErr = String(e) }
}

export async function loadSites() {
  state.siteErr = ''
  if (!inWails()) return
  try {
    const rows = (await ListSites()) ?? []
    state.sites = rows.map(r => ({ domain: r.domain, php: r.php, root: r.root, hosts: !!r.hosts })) as SiteEntry[]
  } catch (e) { state.siteErr = String(e) }
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

// 启动期统一入口：等 Wails Core 注入后拉全部真实数据
export function loadAllOnReady() {
  onWailsReady(() => { loadBackups(); loadOffline(); loadSites(); loadGoProjects() })
}
