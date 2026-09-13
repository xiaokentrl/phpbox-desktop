// 数据加载层：各 Wails 绑定 → state 单例。视图与壳只调用，不直接碰绑定。
import { inWails, onWailsReady } from './task'
import { state, toastBus, type OfflineRow, type SiteEntry, type GoProjectRow, type BackupRow, type EnvRow } from '../state'
import { cmpVerDesc } from '../utils'
import { ListContainers } from '../../bindings/github.com/xiaokentrl/phpbox-desktop/internal/bindings/docker'
import { ListBackups } from '../../bindings/github.com/xiaokentrl/phpbox-desktop/internal/bindings/backup'
import { ListOfflineCache } from '../../bindings/github.com/xiaokentrl/phpbox-desktop/internal/bindings/offline'
import { ListSites } from '../../bindings/github.com/xiaokentrl/phpbox-desktop/internal/bindings/site'
import { ListGoProjects } from '../../bindings/github.com/xiaokentrl/phpbox-desktop/internal/bindings/goprojects'
import { ReadEnv } from '../../bindings/github.com/xiaokentrl/phpbox-desktop/internal/bindings/env'

// 容器列表（总览/服务线/Go 状态共用的真实数据源）+ installed 派生
// installed 与 bash cmd_list 同源：phpbox-service/phpbox-version labels（缺 label 的容器不纳入）
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
