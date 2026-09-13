<script setup lang="ts">
// 应用壳 + 视图路由（阶段 0：内联视图；§22.1 晋升制——复用时抽组件）
import { computed, nextTick, onMounted, ref, watch } from 'vue'
import { t } from './i18n'
import { state, setRoute, setTheme, setAppLocale, initTheme, clearTask, taskRunning, toastBus,
  openInstall, openDanger, openExt, openSiteModal, closeModal, type Route, type ContainerRow } from './state'
import { dispatchTask, inWails, onWailsReady } from './api/task'
import { Events } from '@wailsio/runtime'
import { ListContainers } from '../bindings/github.com/xiaokentrl/phpbox-desktop/internal/bindings/docker'
import { ListBackups, DeleteBackup } from '../bindings/github.com/xiaokentrl/phpbox-desktop/internal/bindings/backup'
import { ReadPhpExtensions } from '../bindings/github.com/xiaokentrl/phpbox-desktop/internal/bindings/php'
import { ListOfflineCache, VerifyOfflineCache, PruneOfflineCache } from '../bindings/github.com/xiaokentrl/phpbox-desktop/internal/bindings/offline'
import { ListSites } from '../bindings/github.com/xiaokentrl/phpbox-desktop/internal/bindings/site'
import type { ContainerSummary } from '../bindings/github.com/xiaokentrl/phpbox-desktop/internal/engine/docker/models'

// ── 主题 ──
const THEMES = [
  { id: 'midnight', zh: '午夜蓝', en: 'Midnight' },
  { id: 'light', zh: '晨光白', en: 'Daylight' },
  { id: 'oled', zh: '极夜黑', en: 'OLED' },
  { id: 'forest', zh: '森林绿', en: 'Forest' },
  { id: 'ocean', zh: '深海蓝', en: 'Ocean' },
  { id: 'sakura', zh: '樱花粉', en: 'Sakura' },
]
const themePop = ref(false)
function pickTheme(id: string) { setTheme(id); themePop.value = false }
const themeName = computed(() => {
  const th = THEMES.find(x => x.id === state.theme)
  return th ? (state.locale === 'en-US' ? th.en : th.zh) : state.theme
})

// ── 导航 ──
const NAV = [
  { section: 'nav.business' },
  { id: 'sites', icon: '🔗' },
  { section: 'nav.services' },
  { id: 'php', icon: '🐘' }, { id: 'mysql', icon: '🐬' }, { id: 'pgsql', icon: '🐘' },
  { id: 'redis', icon: '⚡' }, { id: 'nginx', icon: '🌐' }, { id: 'go', icon: '🐹' },
  { section: 'nav.ops' },
  { id: 'backup', icon: '📦' }, { id: 'offline', icon: '🗄️' },
  { id: 'settings', icon: '⚙️' }, { id: 'overview', icon: '◈' },
]
function go(r: string) { setRoute(r as Route) }
function countFor(id: string): number | null {
  if (id === 'sites') return state.sites.length || null
  if (id === 'backup') return state.backups.length || null
  if (id === 'offline') return state.offlineCache.length || null
  const list = (state.installed as Record<string, string[]>)[id]
  return list ? list.length || null : null
}
const svcLabel = (id: string) => t('nav.' + id)

// ── 服务线视图（php/mysql/pgsql/redis/nginx）──
const SVC_META: Record<string, { icon: string; suggested: string[]; single?: boolean }> = {
  php:   { icon: '🐘', suggested: ['8.4', '8.3', '8.2', '8.1', '8.0', '7.4'] },
  mysql: { icon: '🐬', suggested: ['9.1', '8.4', '8.0', '5.7'] },
  pgsql: { icon: '🐘', suggested: ['17', '16', '15', '14'] },
  redis: { icon: '⚡', suggested: ['8', '7'] },
  nginx: { icon: '🌐', suggested: ['alpine', '1.25'], single: true },
}
// nginx 版本读 .env（NGINX_VERSION），单实例；其余线多版本
const PORT_MAP: Record<string, Record<string, string>> = {
  mysql: { '8.4': '3384', '8.0': '3380', '5.7': '3357', '9.1': '3391' },
  pgsql: { '17': '5417', '16': '5416', '15': '5415', '14': '5414' },
  redis: { '8': '6379', '7': '6377' },
  nginx: { alpine: '80', '1.25': '8025' },
}
function svcVersions(kind: string): string[] {
  return (state.installed as Record<string, string[]>)[kind] ?? []
}
function openInstallModal(kind: string) {
  const meta = SVC_META[kind]
  openInstall({ svc: kind, title: svcLabel(kind), suggested: meta.suggested, single: meta.single })
}

// ── 安装弹窗（本地 UI 状态）──
const installVer = ref('')
const installModal = computed(() => state.modal?.kind === 'install' ? state.modal : null)
watch(() => state.modal?.kind, () => { // 打开安装弹窗时重置输入（nginx 单实例预填）
  const m = installModal.value
  if (m) installVer.value = m.single ? (SVC_META[m.svc]?.suggested[0] ?? '') : ''
  if (state.modal?.kind === 'danger') { dcInput.value = ''; dcChecked.value = false; dcPurge.value = false }
})
function pickInstallVer(v: string) { installVer.value = v }
function installCmdPreview(): string {
  const m = installModal.value
  if (!m) return ''
  const v = installVer.value.trim()
  // nginx install 不带版本（版本由 .env 的 NGINX_VERSION 决定）；其余线 install <版本>
  return m.svc === 'nginx' ? 'phpbox nginx install' : v ? `phpbox ${m.svc} install ${v}` : `phpbox ${m.svc} install <版本>`
}
function confirmInstall() {
  const m = installModal.value
  if (!m) return
  const v = installVer.value.trim()
  if (m.svc !== 'nginx' && !v) { toastBus(t('mod.err.version'), 'err'); return }
  const args = m.svc === 'nginx' ? ['nginx', 'install'] : [m.svc, 'install', v]
  closeModal()
  dispatchTask(`${t('svc.install')}·${m.title} ${v}`, `phpbox ${args.join(' ')}`, args, {
    doneMsg: t('task.done'),
    fallback: [{ d: 400, lines: ['[INFO] 演示环境：安装流程模拟输出'] }],
    onDone: () => {
      if (v && !svcVersions(m.svc).includes(v)) ((state.installed as Record<string, string[]>)[m.svc] ??= []).push(v)
      loadContainers()
    },
  })
}

// ── 危险确认弹窗（卸载：勾选 + 输入匹配 + 可选 --purge）──
const dcInput = ref(''), dcChecked = ref(false), dcPurge = ref(false)
const dcModal = computed(() => state.modal?.kind === 'danger' ? state.modal : null)
const dcMatched = computed(() => dcModal.value ? dcInput.value.trim() === dcModal.value.expect : false)
const dcReady = computed(() => !!(dcMatched.value && dcChecked.value))
function dcConfirm() {
  const m = dcModal.value
  if (!m || !dcReady.value) return
  closeModal()
  m.onConfirm(dcPurge.value)
}

// 容器名规则（get_container_name）：php84 / mysql84 / redis8 / pg17 / nginx（前缀+去点版本）
function containerNameFor(kind: string, ver: string): string {
  if (kind === 'nginx') return 'nginx'
  if (kind === 'pgsql') return `pg${ver.replace(/\./g, '')}`
  return `${kind}${ver.replace(/\./g, '')}`
}
// 卡片运行状态：真实容器表（Name 是 /php84 形态）
function containerStateFor(kind: string, ver: string): string {
  const want = containerNameFor(kind, ver)
  const c = containers.value.find(x => x.Name.replace(/^\//, '') === want)
  return c?.State ?? ''
}
function copyCmd(cmd: string) {
  if (navigator.clipboard?.writeText) {
    navigator.clipboard.writeText(cmd).then(() => toastBus(t('toast.copied'), 'ok'), () => toastBus(cmd, 'info', 4000))
  } else toastBus(cmd, 'info', 4000) // 非 https/localhost 降级为展示
}
function openUninstallModal(kind: string, ver: string) {
  const isNginx = kind === 'nginx'
  openDanger({
    title: t('mod.uninstallTitle', { name: svcLabel(kind), ver }),
    description: t('mod.uninstallDesc'),
    warnings: isNginx ? [
      { text: t('mod.warn.uninstallCmd') },
      { text: t('mod.warn.configKeep'), keep: true },
    ] : [
      { text: t('mod.warn.container', { kind, ver }) },
      { text: t('mod.warn.config', { kind, ver }) },
      { text: t('mod.warn.dataKeep'), keep: true },
      { text: t('mod.warn.offlineKeep'), keep: true },
    ],
    checkboxLabel: isNginx ? t('mod.uninstallNginxCheck') : t('mod.uninstallCheck', { kind, ver }),
    inputLabel: t('mod.confirmInput'),
    expect: ver,
    placeholder: t('mod.confirmPlaceholder', { ver }),
    cliPreview: isNginx ? 'phpbox nginx uninstall' : `phpbox ${kind} uninstall ${ver}`,
    confirmLabel: t('mod.confirmUninstall'),
    purge: isNginx ? undefined : { label: t('mod.purge') },
    onConfirm: (purge) => {
      const args = isNginx ? ['nginx', 'uninstall'] : [kind, 'uninstall', ver, ...(purge ? ['--purge'] : [])]
      const label = `${t('mod.confirmUninstall')}·${svcLabel(kind)} ${ver}`
      dispatchTask(label, `phpbox ${args.join(' ')}`, args, {
        doneMsg: t('task.done'),
        fallback: [{ d: 400, lines: ['[INFO] 演示环境：卸载流程模拟输出'] }],
        onDone: () => {
          const list = (state.installed as Record<string, string[]>)[kind]
          if (list) { const i = list.indexOf(ver); if (i >= 0) list.splice(i, 1) }
          loadContainers()
        },
      })
    },
  })
}

// ── Toast ──
const toasts = ref<{ id: number; msg: string; kind: string }[]>([])
let toastSeq = 0
window.addEventListener('phpbox:toast', (e) => {
  const d = (e as CustomEvent).detail
  const id = ++toastSeq
  toasts.value.push({ id, msg: d.msg, kind: d.kind })
  setTimeout(() => { toasts.value = toasts.value.filter(x => x.id !== id) }, d.ttl || 3200)
})

// ── Docker 真实数据 ──
const containers = ref<ContainerSummary[]>([])
const dockerErr = ref('')
async function refreshContainers() { loadContainers() }
async function loadContainers() {
  dockerErr.value = ''
  try { containers.value = (await ListContainers()) ?? [] }
  catch (e) { dockerErr.value = String(e) }
}
onMounted(() => {
  initTheme(); loadContainers()
  onWailsReady(() => { loadBackups(); loadOffline(); loadSites() })
  initTrayNav() // 托盘菜单快速跳转（ui:navigate）
})
watch(() => state.route, (r) => {
  if (r === 'backup') loadBackups()
  if (r === 'offline') loadOffline()
  if (r === 'sites') loadSites()
})

// ── 任务抽屉（真实 spawn：桌面内经 Runner 绑定驱动 phpbox CLI）──
const drawerCollapsed = ref(false)
const drawerLogEl = ref<HTMLElement | null>(null)
const taskPhaseLabel = computed(() => state.task
  ? state.task.phase === 'running' ? t('task.running')
    : state.task.phase === 'success' ? t('task.success') : t('task.failed')
  : '')
watch(() => state.task?.lines.length, async () => { // 新日志行 → 滚到底部
  if (drawerCollapsed.value) return
  await nextTick()
  if (drawerLogEl.value) drawerLogEl.value.scrollTop = drawerLogEl.value.scrollHeight
})

// 环境诊断：真实链路 spawn `phpbox list`（只读、安全、输出稳定）
function runDiagnostics() {
  const ok = dispatchTask(t('task.diag.label'), 'phpbox list', ['list'], {
    doneMsg: t('task.diag.done'),
    fallback: [ // 浏览器降级演示（§13）
      { d: 400, lines: ['[INFO] 检查 Docker Engine…'] },
      { d: 500, lines: ['[OK]   Docker Engine 已连接', '[OK]   Compose 配置可解析'] },
      { d: 350, lines: ['[INFO] 已安装服务：php 8.4/8.2/8.0/7.4 · mysql 8.4/8.0/5.7 · nginx alpine'] },
    ],
    onDone: () => { loadContainers() },
  })
  if (!ok) toastBus(t('task.busy'), 'err')
}

// ── PHP 扩展管理弹窗（真实状态 + 目标集合 → 逐个 add/remove spawn）──
const EXT_LIB = ['apcu','memcached','mongodb','amqp','yaml','ssh2','swoole','event','grpc','protobuf','igbinary','msgpack','ds','uv','pthreads']
const EXT_PRESETS: Record<string, string[]> = {
  default: ['gd','redis','pdo_mysql','mysqli','pgsql','pdo_pgsql','zip','bcmath','intl','opcache','exif','soap','sockets','imagick'],
  minimal: ['opcache'],
  web:     ['gd','redis','pdo_mysql','mysqli','pgsql','pdo_pgsql','zip','bcmath','intl','opcache','exif','soap','sockets','imagick'],
  debug:   ['gd','redis','pdo_mysql','mysqli','pgsql','pdo_pgsql','zip','bcmath','intl','opcache','exif','soap','sockets','imagick','xdebug'],
}
const extModal = computed(() => state.modal?.kind === 'ext' ? state.modal : null)
const extInstalled = ref<string[]>([])   // extensions.env 真实状态
const extExtra = ref<string[]>([])       // 本次手动添加
const extSelected = ref<Set<string>>(new Set())
const extPreset = ref('')
const extInput = ref('')
const extErr = ref('')
const extLoading = ref(false)

watch(() => state.modal?.kind, async (k) => {
  if (k !== 'ext') return
  const m = extModal.value
  if (!m) return
  extInstalled.value = []; extExtra.value = []; extSelected.value = new Set()
  extPreset.value = ''; extInput.value = ''; extErr.value = ''; extLoading.value = true
  if (!inWails()) { // 浏览器降级：演示数据
    extInstalled.value = [...EXT_PRESETS.debug]
    extSelected.value = new Set(EXT_PRESETS.debug)
    extPreset.value = 'debug'
    extLoading.value = false
    return
  }
  try {
    const exts = (await ReadPhpExtensions(m.version)) ?? []
    extInstalled.value = exts
    extSelected.value = new Set(exts)
  } catch (e) { extErr.value = String(e) }
  extLoading.value = false
})

const extEnabledList = computed(() => [...extInstalled.value, ...extExtra.value])
const extSuggestList = computed(() =>
  EXT_LIB.filter(e => !extInstalled.value.includes(e) && !extExtra.value.includes(e)).slice(0, 12))
// 已知扩展集合：所有列表中出现过的（预设 ∪ 已装 ∪ 库）——目标集合里未知项视为"删除"外的保留项
const extKnown = computed(() => {
  const s = new Set<string>([...extInstalled.value, ...extExtra.value, ...EXT_LIB, ...Object.values(EXT_PRESETS).flat()])
  return s
})
const extDirty = computed(() => {
  const on = extSelected.value
  const orig = new Set(extInstalled.value)
  if (on.size !== orig.size) return true
  for (const e of on) if (!orig.has(e)) return true
  return false
})
const extAdds = computed(() => [...extSelected.value].filter(e => !extInstalled.value.includes(e)))
const extRemoves = computed(() => extInstalled.value.filter(e => !extSelected.value.has(e)))
function extToggle(e: string) {
  const s = new Set(extSelected.value)
  if (s.has(e)) s.delete(e); else s.add(e)
  extSelected.value = s
  extPreset.value = ''
}
function extPickPreset(p: string) {
  extPreset.value = p
  extSelected.value = new Set(EXT_PRESETS[p] ?? [])
  // 预设里未知的新扩展也进入 extra，保证 UI 可见
  for (const e of EXT_PRESETS[p] ?? []) {
    if (!extInstalled.value.includes(e) && !extExtra.value.includes(e)) extExtra.value.push(e)
  }
}
function extAddManual() {
  const name = extInput.value.trim()
  if (!name) return
  if (!/^[a-zA-Z0-9._-]+$/.test(name)) { toastBus(t('ext.badName'), 'err'); return }
  if (extEnabledList.value.includes(name)) { toastBus(t('ext.dupe', { name }), 'info', 1600); extInput.value = ''; return }
  extExtra.value.push(name)
  const s = new Set(extSelected.value); s.add(name); extSelected.value = s
  extInput.value = ''
  extPreset.value = ''
}
// 应用：逐个 add/remove 真实 spawn（每个重建镜像），串行执行，失败即停
function extApply() {
  const m = extModal.value
  if (!m || !extDirty.value) return
  const adds = extAdds.value, removes = extRemoves.value
  const ops: { args: string[]; cli: string; label: string }[] = [
    ...adds.map(e => ({ args: ['php', 'extension', 'add', m.version, e], cli: `phpbox php extension add ${m.version} ${e}`, label: `+${e}` })),
    ...removes.map(e => ({ args: ['php', 'extension', 'remove', m.version, e], cli: `phpbox php extension remove ${m.version} ${e}`, label: `-${e}` })),
  ]
  closeModal()
  runExtOps(ops, 0)
}
function runExtOps(ops: { args: string[]; cli: string; label: string }[], i: number) {
  if (i >= ops.length) { toastBus(t('task.done'), 'ok'); return }
  const op = ops[i]
  const ok = dispatchTask(`PHP 扩展 ${op.label}`, op.cli, op.args, {
    doneMsg: `${op.label} ✓`,
    fallback: [{ d: 400, lines: [`[INFO] 演示环境：${op.cli}`] }],
    onDone: () => runExtOps(ops, i + 1), // 上一个成功才执行下一个；失败链条自然中断
  })
  if (!ok) toastBus(t('task.busy'), 'err')
}


const backupErr = ref('')
async function loadBackups() {
  backupErr.value = ''
  if (!inWails()) return // 浏览器降级：保留空列表
  try {
    const rows = (await ListBackups()) ?? []
    state.backups = rows.map(r => ({ file: r.file, path: r.path, size: Number(r.size), at: String(r.at) }))
  } catch (e) { backupErr.value = String(e) }
}
function fmtSize(n: number): string {
  if (n >= 1024 ** 3) return (n / 1024 ** 3).toFixed(1) + ' GB'
  if (n >= 1024 ** 2) return (n / 1024 ** 2).toFixed(0) + ' MB'
  if (n >= 1024) return (n / 1024).toFixed(0) + ' KB'
  return n + ' B'
}
function fmtTime(iso: string): string {
  const d = new Date(iso)
  return isNaN(d.getTime()) ? iso : d.toLocaleString()
}
function backupNow() {
  const ok = dispatchTask(t('bk.task.now'), 'phpbox backup', ['backup'], {
    doneMsg: t('bk.done.now'),
    fallback: [{ d: 400, lines: ['[INFO] 演示环境：备份流程模拟输出'] }, { d: 600, lines: ['[OK]   备份完成: ~/phpbox/backups/'] }],
    onDone: () => { loadBackups() },
  })
  if (!ok) toastBus(t('task.busy'), 'err')
}
function openRestoreModal(b: { file: string; path: string; size: number }) {
  openDanger({
    title: t('bk.restoreTitle'),
    description: `${b.file}（${fmtSize(b.size)}）`,
    warnings: [
      { text: t('bk.warn.overwrite') },
      { text: t('bk.warn.stopSvc') },
      { text: t('bk.warn.noImages'), keep: true },
      { text: t('bk.warn.goReinstall'), keep: true },
      { text: t('bk.warn.reload'), keep: true },
    ],
    checkboxLabel: t('bk.restoreCheck'),
    inputLabel: t('bk.confirmFile'),
    expect: b.file,
    placeholder: t('bk.confirmFilePh'),
    // 绝对路径 + -y：spawn 非交互必须 -y；GUI 弹窗即确认界面（cmd_restore 的 tty 确认在桌面环境不可用）
    cliPreview: `phpbox restore ${b.path} -y`,
    confirmLabel: t('bk.confirmRestore'),
    onConfirm: () => {
      dispatchTask(t('bk.task.restore'), `phpbox restore ${b.path} -y`, ['restore', b.path, '-y'], {
        doneMsg: t('bk.done.restore'),
        fallback: [{ d: 400, lines: ['[INFO] 演示环境：恢复流程模拟输出'] }],
        onDone: () => { loadContainers(); loadBackups() },
      })
    },
  })
}
function openBackupDeleteModal(b: { file: string; size: number }) {
  openDanger({
    title: t('bk.deleteTitle'),
    description: `${b.file}（${fmtSize(b.size)}）`,
    warnings: [
      { text: t('bk.warn.deleteFile') },
      { text: t('bk.deleteDesc') },
      { text: t('bk.warn.envKeep'), keep: true },
    ],
    checkboxLabel: t('bk.deleteCheck'),
    inputLabel: t('bk.confirmFile'),
    expect: b.file,
    placeholder: t('bk.confirmFilePh'),
    cliPreview: `${t('bk.warn.deleteFile')}: ${b.file}`,
    confirmLabel: t('bk.confirmDelete'),
    onConfirm: async () => {
      if (!inWails()) { toastBus(t('bk.done.delete'), 'ok'); return }
      try {
        await DeleteBackup(b.file)
        toastBus(t('bk.done.delete'), 'ok')
        loadBackups()
      } catch (e) { toastBus(String(e), 'err', 5000) }
    },
  })
}

// ── 离线缓存（真实目录树：Offline 绑定扫描 ~/phpbox/offline/）──
const offlineErr = ref('')
const offlineVerifyState = ref<Record<string, 'busy' | 'ok' | 'bad' | string>>({}) // key: svc/ver
async function loadOffline() {
  offlineErr.value = ''
  if (!inWails()) return // 浏览器降级：保留空列表
  try {
    const rows = (await ListOfflineCache()) ?? []
    state.offlineCache = rows.map(r => ({
      svc: r.svc, ver: r.ver, path: r.path, size: Number(r.size), files: Number(r.files), kind: r.kind,
    }))
  } catch (e) { offlineErr.value = String(e) }
}
const offlineTotal = computed(() => state.offlineCache.reduce((s, r) => s + r.size, 0))
async function verifyOffline(svc: string, ver: string) {
  const key = `${svc}/${ver}`
  if (!inWails()) { toastBus(t('off.browserHint'), 'info'); return }
  offlineVerifyState.value[key] = 'busy'
  try {
    const res = await VerifyOfflineCache(svc, ver)
    offlineVerifyState.value[key] = res?.ok ? 'ok' : (res?.detail || 'bad')
    toastBus(`${key}: ${res?.detail ?? ''}`, res?.ok ? 'ok' : 'err', 5000)
  } catch (e) {
    offlineVerifyState.value[key] = 'bad'
    toastBus(String(e), 'err', 5000)
  }
}
async function verifyOfflineAll() {
  let ok = 0
  for (const r of state.offlineCache) {
    await verifyOffline(r.svc, r.ver)
    if (offlineVerifyState.value[`${r.svc}/${r.ver}`] === 'ok') ok++
  }
  toastBus(t('off.verifyAllDone', { ok, total: state.offlineCache.length }), ok === state.offlineCache.length ? 'ok' : 'info')
}
function openOfflinePruneModal(r: { svc: string; ver: string; size: number; files: number }) {
  openDanger({
    title: t('off.pruneTitle'),
    description: `${r.svc} ${r.ver} · ${fmtSize(r.size)} · ${r.files} ${t('off.files')}`,
    warnings: [
      { text: t('off.warn.del', { svc: r.svc, ver: r.ver, size: fmtSize(r.size) }) },
      { text: t('off.warn.redownload') },
      { text: t('off.warn.runtime'), keep: true },
    ],
    checkboxLabel: t('off.pruneCheck'),
    inputLabel: t('off.confirmInput'),
    expect: r.ver,
    placeholder: t('mod.confirmPlaceholder', { ver: r.ver }),
    cliPreview: `rm -rf ~/phpbox/offline/${r.svc}/${r.ver}/`,
    confirmLabel: t('off.confirmPrune'),
    onConfirm: async () => {
      if (!inWails()) { toastBus(t('off.browserHint'), 'info'); return }
      try {
        await PruneOfflineCache(r.svc, r.ver)
        toastBus(`${t('off.confirmPrune')} ✓ ${r.svc}/${r.ver}`, 'ok')
        loadOffline()
      } catch (e) { toastBus(String(e), 'err', 5000) }
    },
  })
}

// ── 站点（真实 vhost 状态：Site 绑定解析 config/nginx/sites/*.conf + /etc/hosts）──
const siteModal = computed(() => state.modal?.kind === 'site' ? state.modal : null)
const siteDomain = ref('')
const sitePhp = ref('')
const siteErr = ref('')
const DOMAIN_RE = /^[a-zA-Z0-9]([a-zA-Z0-9-]*[a-zA-Z0-9])?(\.[a-zA-Z0-9]([a-zA-Z0-9-]*[a-zA-Z0-9])?)+$/
const installedPhp = computed(() => (state.installed as Record<string, string[]>).php ?? [])
const phpVerToKey = (v: string) => 'php' + v.replace(/\./g, '')  // 8.4 → php84（bash 服务键）

async function loadSites() {
  if (!inWails()) return // 浏览器降级：保留空列表
  try {
    const rows = (await ListSites()) ?? []
    state.sites = rows.map(r => ({ domain: r.domain, php: r.php, root: r.root, hosts: !!r.hosts }))
  } catch (e) { toastBus(String(e), 'err', 5000) }
}
watch(() => state.modal?.kind, (k) => {
  if (k !== 'site') return
  siteDomain.value = ''; sitePhp.value = installedPhp.value[0] ?? ''; siteErr.value = ''
})
const siteCmdPreview = computed(() => {
  const d = siteDomain.value.trim(), p = sitePhp.value
  return d && p ? `phpbox site add ${d} --php ${p}` : 'phpbox site add <域名> --php <版本>'
})
function confirmSiteAdd() {
  const d = siteDomain.value.trim()
  if (!DOMAIN_RE.test(d)) { siteErr.value = t('site.err.domain'); return }
  if (state.sites.some(s => s.domain === d)) { siteErr.value = t('site.err.dupe'); return }
  if (!sitePhp.value) { siteErr.value = t('site.err.php'); return }
  if (!(state.installed as Record<string, string[]>).nginx?.length) { siteErr.value = t('site.err.noNginx'); return }
  closeModal()
  dispatchTask(t('site.addTask'), `phpbox site add ${d} --php ${sitePhp.value}`,
    ['site', 'add', d, '--php', sitePhp.value], {
    fallback: [{ d: 400, lines: [`[INFO] 演示环境：phpbox site add ${d} --php ${sitePhp.value}`] }],
    onDone: () => { loadSites(); toastBus(`http://${d}`, 'ok') },
  })
}
// 行内切换 PHP 版本：真实 spawn phpbox site switch（失败由抽屉显示，列表由重载还原）
function onSiteSwitch(e: Event, domain: string) {
  const sel = e.target as HTMLSelectElement
  const site = state.sites.find(s => s.domain === domain)
  const oldKey = site?.php ?? ''
  if (!site || oldKey === sel.value || !sel.value) return
  const ver = sel.value.replace(/^php/, '')
  dispatchTask(`${t('site.switchTask')}·${domain}`, `phpbox site switch ${domain} --php ${ver}`,
    ['site', 'switch', domain, '--php', ver], {
    fallback: [{ d: 400, lines: [`[INFO] 演示环境：phpbox site switch ${domain} --php ${ver}`] }],
    onDone: () => { loadSites() },
  })
}
function hostsToggle(domain: string, add: boolean) {
  const args = add ? ['hosts', 'add', domain] : ['hosts', 'remove', domain]
  dispatchTask(add ? t('site.hosts.taskAdd') : t('site.hosts.taskRemove'), `phpbox ${args.join(' ')}`, args, {
    fallback: [{ d: 400, lines: [`[INFO] 演示环境：phpbox hosts ${add ? 'add' : 'remove'} ${domain}`] }],
    onDone: () => { loadSites() },
  })
}
function openSiteRemoveModal(domain: string) {
  openDanger({
    title: t('site.removeTitle', { domain }),
    description: t('site.removeDesc'),
    warnings: [
      { text: t('site.warn.conf', { domain }) },
      { text: t('site.warn.source'), keep: true },
      { text: t('site.warn.hosts'), keep: true },
      { text: t('site.warn.rolling'), keep: true },
    ],
    checkboxLabel: t('site.removeCheck'),
    inputLabel: t('mod.confirmInput'),
    expect: domain,
    placeholder: t('mod.confirmPlaceholder', { ver: domain }),
    cliPreview: `phpbox site remove ${domain}`,
    confirmLabel: t('site.confirmRemove'),
    onConfirm: () => {
      dispatchTask(`${t('site.confirmRemove')}·${domain}`, `phpbox site remove ${domain}`,
        ['site', 'remove', domain], {
        fallback: [{ d: 400, lines: [`[INFO] 演示环境：phpbox site remove ${domain}`] }],
        onDone: () => { loadSites() },
      })
    },
  })
}

// ── 托盘导航（Go 侧 Emit ui:navigate {route}）──
let trayNavBound = false
function initTrayNav() {
  if (trayNavBound || !inWails()) return
  trayNavBound = true
  Events.On('ui:navigate', (ev) => {
    const r = (ev.data as any)?.route
    if (typeof r === 'string' && r) setRoute(r as Route)
  })
}

// ── 事件委托 ──
</script>

<template>
  <div class="app">
    <aside class="sidebar">
      <div class="brand">
        <div class="brand-logo">π</div>
        <div class="brand-meta">
          <div class="brand-name">phpbox</div>
          <div class="brand-status">
            <span class="dot" :class="dockerErr ? 'dot-err' : 'dot-ok'"></span>
            <span>{{ dockerErr ? 'Engine error' : t('brand.connected') }}</span>
          </div>
        </div>
      </div>
      <nav class="nav">
        <template v-for="item in NAV" :key="item.id || item.section">
          <div v-if="item.section" class="nav-section">{{ t(item.section) }}</div>
          <button v-else class="nav-item" :class="{ active: state.route === item.id }" @click="go(item.id!)">
            <span class="nav-icon">{{ item.icon }}</span>
            <span>{{ svcLabel(item.id!) }}</span>
            <span v-if="countFor(item.id!)" class="nav-badge">{{ countFor(item.id!) }}</span>
          </button>
        </template>
      </nav>
      <div class="sidebar-foot">
        <button class="btn-ghost" @click="refreshContainers"><span>↻ {{ t('btn.refresh') }}</span></button>
        <button class="btn-ghost" @click="setAppLocale(state.locale === 'zh-CN' ? 'en-US' : 'zh-CN')">
          <span>{{ state.locale === 'zh-CN' ? '🌐 English' : '🌐 中文' }}</span>
        </button>
        <button class="btn-ghost" @click.stop="themePop = !themePop">
          <span>🎨 {{ themeName }}</span>
        </button>
        <div v-if="themePop" class="theme-pop">
          <button v-for="th in THEMES" :key="th.id" class="theme-item" :class="{ active: state.theme === th.id }"
                  @click.stop="pickTheme(th.id)">
            <span>{{ state.locale === 'en-US' ? th.en : th.zh }}</span>
          </button>
        </div>
      </div>
    </aside>
    <main class="main">
      <div class="view">
        <div class="view-inner">

          <!-- ═══ 站点（默认首屏 · 真实 vhost 状态）═══ -->
          <template v-if="state.route === 'sites'">
            <header class="view-header">
              <div><h1>{{ t('sites.title') }}</h1><p class="view-sub">{{ t('sites.sub') }}</p></div>
              <div class="header-actions"><button class="btn btn-primary" @click="openSiteModal()">+ {{ t('sites.add') }}</button></div>
            </header>
            <div v-if="state.sites.length === 0" class="empty">
              <div class="empty-icon">🌍</div><h2>{{ t('sites.empty.title') }}</h2><p>{{ t('sites.empty.desc') }}</p>
              <button class="btn btn-primary" @click="openSiteModal()">{{ t('sites.add') }}</button>
            </div>
            <template v-else>
              <div class="summary">
                <div class="summary-item"><div class="summary-num">{{ state.sites.length }}</div><div class="summary-label">{{ t('sum.sites') }}</div></div>
                <div class="summary-item"><div class="summary-num" style="color:var(--ok)">{{ state.sites.filter(s => s.hosts).length }}</div><div class="summary-label">{{ t('sum.healthy') }}</div></div>
                <div class="summary-item"><div class="summary-num">{{ state.env.NGINX_PORT }}</div><div class="summary-label">{{ t('sum.port') }}</div></div>
              </div>
              <div class="table-wrap"><table>
                <thead><tr><th>{{ t('th.domain') }}</th><th>{{ t('th.php') }}</th><th>{{ t('th.root') }}</th><th>{{ t('th.hosts') }}</th><th></th></tr></thead>
                <tbody><tr v-for="s in state.sites" :key="s.domain">
                  <td><a class="site-domain" :href="'http://'+s.domain" target="_blank" rel="noopener"><span class="favicon">{{ s.domain[0].toUpperCase() }}</span>{{ s.domain }}</a></td>
                  <td><select class="php-select" :value="s.php" @change="onSiteSwitch($event, s.domain)">
                    <option v-for="v in installedPhp" :key="v" :value="phpVerToKey(v)" :selected="phpVerToKey(v)===s.php">{{ v }}</option>
                  </select></td>
                  <td><span class="mono dim">{{ s.root }}</span></td>
                  <td>
                    <button v-if="!s.hosts" class="btn btn-sm" @click="hostsToggle(s.domain, true)">{{ t('site.hosts.add') }}</button>
                    <span v-else class="chip chip-accent">{{ t('site.hosts.added') }}</span>
                  </td>
                  <td><div class="row-actions">
                    <button v-if="s.hosts" class="btn btn-sm" @click="hostsToggle(s.domain, false)">{{ t('site.hostsRemove') }}</button>
                    <button class="btn btn-sm btn-danger" @click="openSiteRemoveModal(s.domain)">{{ t('common.delete') }}</button>
                  </div></td>
                </tr></tbody></table></div>
            </template>
          </template>

          <!-- ═══ 总览（真实 Docker 数据）═══ -->
          <template v-else-if="state.route === 'overview'">
            <header class="view-header"><div><h1>{{ t('overview.title') }}</h1><p class="view-sub">{{ t('overview.sub') }}</p></div>
              <div class="header-actions">
                <button class="btn" :disabled="taskRunning()" @click="runDiagnostics">⚙ {{ t('overview.diag') }}</button>
                <button class="btn" @click="loadContainers">{{ t('btn.refresh') }}</button>
              </div></header>
            <p v-if="dockerErr" class="alert alert-danger">{{ dockerErr }}</p>
            <div v-else class="summary">
              <div class="summary-item"><div class="summary-num">{{ containers.length }}</div><div class="summary-label">Containers</div></div>
              <div class="summary-item"><div class="summary-num" style="color:var(--ok)">{{ containers.filter(c=>c.State==='running').length }}</div><div class="summary-label">Running</div></div>
            </div>
            <div class="table-wrap"><table>
              <thead><tr><th>Container</th><th>Image</th><th>State</th></tr></thead>
              <tbody><tr v-for="c in containers" :key="c.Name">
                <td class="mono">{{ c.Name.replace(/^\//,'') }}</td><td class="mono dim">{{ c.Image }}</td>
                <td><span class="status-pill" :class="c.State==='running'?'pill-ok':'pill-off'"><span class="pill-dot"></span>{{ c.State }}</span></td>
              </tr></tbody></table></div>
          </template>

          <!-- ═══ 服务线（版本卡片 + 安装/卸载真实链路）═══ -->
          <template v-else-if="['php','mysql','pgsql','redis','nginx'].includes(state.route)">
            <header class="view-header"><div><h1>{{ svcLabel(state.route) }}</h1><p class="view-sub">{{ t('svc.'+state.route+'.sub') }}</p></div>
              <div class="header-actions">
                <button class="btn btn-primary" :disabled="taskRunning()" @click="openInstallModal(state.route)">+ {{ t('svc.install') }} {{ svcLabel(state.route) }}</button>
              </div></header>
            <div v-if="svcVersions(state.route).length === 0" class="empty">
              <div class="empty-icon">{{ SVC_META[state.route]?.icon }}</div>
              <h2>{{ t('svc.empty.title', { name: svcLabel(state.route) }) }}</h2>
              <p>{{ t('svc.hint.'+state.route) }}</p>
              <button class="btn btn-primary" @click="openInstallModal(state.route)">{{ t('svc.install') }} {{ svcLabel(state.route) }}</button>
            </div>
            <div v-else class="grid grid-3">
              <article v-for="v in svcVersions(state.route)" :key="v" class="card version-card">
                <div class="version-card-head">
                  <span class="version-tag">{{ v }}</span>
                  <span class="status-pill" :class="containerStateFor(state.route, v) === 'running' ? 'pill-ok' : 'pill-off'">
                    <span class="pill-dot"></span>{{ containerStateFor(state.route, v) === 'running' ? t('svc.card.running') : t('health.down') }}
                  </span>
                </div>
                <div>
                  <div v-if="PORT_MAP[state.route]?.[v]" class="kv"><span class="k">{{ t('svc.card.port') }}</span><span class="v">{{ PORT_MAP[state.route][v] }}</span></div>
                  <div v-if="['mysql','pgsql'].includes(state.route)" class="kv"><span class="k">{{ t('svc.card.password') }}</span><span class="v">••••••••</span></div>
                  <div class="kv"><span class="k">{{ t('svc.card.config') }}</span><span class="v">{{ t('svc.card.configPath', { kind: state.route, ver: v }) }}</span></div>
                </div>
                <div class="version-card-foot">
                  <button v-if="state.route === 'php'" class="btn btn-sm btn-primary" @click="openExt({ version: v })">{{ t('btn.exts') }}</button>
                  <button v-else class="btn btn-sm" @click="copyCmd(`phpbox ${state.route} list`)">{{ t('svc.card.cmd') }}</button>
                  <button class="btn btn-sm btn-danger" @click="openUninstallModal(state.route, v)">{{ t('btn.uninstall') }}</button>
                </div>
              </article>
            </div>
          </template>

          <!-- ═══ Go / 备份 / 离线 / 设置（占位）═══ -->
          <template v-else-if="state.route === 'go'">
            <header class="view-header"><div><h1>Go</h1><p class="view-sub">go.mod auto-discovery</p></div></header>
            <div class="empty"><div class="empty-icon">🐹</div><h2>Go Projects</h2><p>Projects under ~/www with go.mod are auto-discovered.</p></div>
          </template>
          <template v-else-if="state.route === 'backup'">
            <header class="view-header"><div><h1>{{ t('bk.title') }}</h1><p class="view-sub">{{ t('bk.sub') }}</p></div>
              <div class="header-actions">
                <button class="btn btn-primary" :disabled="taskRunning()" @click="backupNow()">{{ t('bk.now') }}</button>
              </div></header>
            <div class="alert alert-warn" style="margin-bottom:16px"><strong>{{ t('bk.warn') }}</strong>{{ t('bk.warnBody') }}</div>
            <p v-if="backupErr" class="alert alert-danger">{{ t('bk.loadErr') }}: {{ backupErr }}</p>
            <div v-if="state.backups.length === 0 && !backupErr" class="empty">
              <div class="empty-icon">📦</div><h2>{{ t('bk.empty') }}</h2><p>{{ t('bk.emptyDesc') }}</p>
            </div>
            <div v-else-if="!backupErr" class="table-wrap"><table>
              <thead><tr><th style="width:34%">{{ t('th.archive') }}</th><th style="width:10%">{{ t('th.size') }}</th><th style="width:20%">{{ t('th.time') }}</th><th></th></tr></thead>
              <tbody><tr v-for="b in state.backups" :key="b.file">
                <td><span class="mono" style="font-size:12.5px">{{ b.file }}</span></td>
                <td><span class="mono dim">{{ fmtSize(b.size) }}</span></td>
                <td><span class="mono dim" style="font-size:12px">{{ fmtTime(b.at) }}</span></td>
                <td><div class="row-actions">
                  <button class="btn btn-sm" @click="copyCmd(b.path)">{{ t('common.copy') }}</button>
                  <button class="btn btn-sm" @click="openRestoreModal(b)">{{ t('bk.restore') }}</button>
                  <button class="btn btn-sm btn-danger" @click="openBackupDeleteModal(b)">{{ t('bk.delete') }}</button>
                </div></td>
              </tr></tbody></table></div>
          </template>
          <template v-else-if="state.route === 'offline'">
            <header class="view-header"><div><h1>{{ t('nav.offline') }}</h1><p class="view-sub">{{ t('off.sub') }}</p></div>
              <div class="header-actions">
                <button class="btn" :disabled="!state.offlineCache.length" @click="verifyOfflineAll()">{{ t('off.verifyAll') }}</button>
              </div></header>
            <p v-if="offlineErr" class="alert alert-danger">{{ t('off.loadErr') }}: {{ offlineErr }}</p>
            <div v-if="state.offlineCache.length === 0 && !offlineErr" class="empty">
              <div class="empty-icon">🗄️</div><h2>{{ t('off.empty') }}</h2><p>{{ t('off.emptyDesc') }}</p>
            </div>
            <template v-else-if="!offlineErr">
              <div class="summary">
                <div class="summary-item"><div class="summary-num">{{ fmtSize(offlineTotal) }}</div><div class="summary-label">{{ t('off.total') }}</div></div>
                <div class="summary-item"><div class="summary-num">{{ state.offlineCache.length }}</div><div class="summary-label">{{ t('off.entries') }}</div></div>
                <div class="summary-item"><div class="summary-num">{{ new Set(state.offlineCache.map(r => r.svc)).size }}</div><div class="summary-label">{{ t('off.services') }}</div></div>
              </div>
              <div class="table-wrap"><table>
                <thead><tr><th style="width:16%">{{ t('nav.services') }}</th><th style="width:12%">{{ t('th.php') }}</th><th style="width:12%">{{ t('th.size') }}</th><th style="width:10%">{{ t('off.files') }}</th><th style="width:26%">{{ t('th.state') }}</th><th></th></tr></thead>
                <tbody><tr v-for="r in state.offlineCache" :key="r.svc+'/'+r.ver">
                  <td><div class="svc-cell"><span class="svc-icon">{{ {php:'🐘',mysql:'🐬',pgsql:'🐘',redis:'⚡',nginx:'🌐'}[r.svc] || '📦' }}</span>{{ r.svc }}</div></td>
                  <td><span class="chip chip-accent">{{ r.ver }}</span></td>
                  <td><span class="mono dim">{{ fmtSize(r.size) }}</span></td>
                  <td><span class="mono dim">{{ r.files }}</span></td>
                  <td>
                    <span v-if="offlineVerifyState[r.svc+'/'+r.ver] === 'busy'" class="status-pill pill-warn"><span class="pill-dot"></span>{{ t('task.running') }}</span>
                    <span v-else-if="offlineVerifyState[r.svc+'/'+r.ver] === 'ok'" class="status-pill pill-ok"><span class="pill-dot"></span>{{ t('health.up') }}</span>
                    <span v-else-if="offlineVerifyState[r.svc+'/'+r.ver]" class="status-pill pill-err"><span class="pill-dot"></span>{{ offlineVerifyState[r.svc+'/'+r.ver] }}</span>
                    <span v-else class="chip">{{ r.kind === 'closure' ? 'apk+pecl' : 'image tar' }}</span>
                  </td>
                  <td><div class="row-actions">
                    <button class="btn btn-sm" @click="verifyOffline(r.svc, r.ver)">{{ t('off.verify') }}</button>
                    <button class="btn btn-sm btn-danger" @click="openOfflinePruneModal(r)">{{ t('off.prune') }}</button>
                  </div></td>
                </tr></tbody></table></div>
            </template>
          </template>
          <template v-else-if="state.route === 'settings'">
            <header class="view-header"><div><h1>{{ t('nav.settings') }}</h1><p class="view-sub">Theme · Language · Layout</p></div></header>
            <div class="card" style="margin-bottom:16px">
              <h3 style="font-size:14px;font-weight:600;margin-bottom:14px">{{ state.locale === 'en-US' ? 'Theme' : '主题' }}</h3>
              <div class="quick-picks">
                <button v-for="th in THEMES" :key="th.id" class="pick" :class="{ selected: state.theme === th.id }" @click="setTheme(th.id)">
                  {{ state.locale === 'en-US' ? th.en : th.zh }}
                </button>
              </div>
            </div>
            <div class="card" style="margin-bottom:16px">
              <h3 style="font-size:14px;font-weight:600;margin-bottom:14px">{{ state.locale === 'en-US' ? 'Language' : '语言' }}</h3>
              <div class="quick-picks">
                <button class="pick" :class="{ selected: state.locale === 'zh-CN' }" @click="setAppLocale('zh-CN')">简体中文</button>
                <button class="pick" :class="{ selected: state.locale === 'en-US' }" @click="setAppLocale('en-US')">English</button>
              </div>
            </div>
          </template>

        </div>
      </div>
      <!-- 任务抽屉：真实 phpbox CLI 输出流（task:log/task:done 事件驱动）-->
      <section class="drawer" :class="{ collapsed: drawerCollapsed }" v-if="state.task">
        <header class="drawer-head">
          <div class="drawer-left">
            <span class="drawer-dot" :class="state.task.phase"></span>
            <span class="drawer-label">{{ state.task.label }}</span>
            <span class="drawer-cmd">$ {{ state.task.cli }}</span>
          </div>
          <div class="drawer-right">
            <span class="drawer-status">{{ taskPhaseLabel }}</span>
            <button class="icon-btn" v-if="state.task.phase !== 'running'" :title="t('task.close')" @click="clearTask()">✕</button>
            <button class="icon-btn" @click="drawerCollapsed = !drawerCollapsed">{{ drawerCollapsed ? '▲' : '▼' }}</button>
          </div>
        </header>
        <div class="drawer-log" ref="drawerLogEl"><div v-for="(l, i) in state.task.lines" :key="i" class="log-line" :class="l.c">{{ l.t }}</div></div>
      </section>
    </main>
    <!-- 弹窗根：安装 / 危险确认（§8.2 三条件）-->
    <div class="modal-root" :class="{ open: !!state.modal }">
      <div v-if="installModal" class="modal">
        <div class="modal-head"><h3>{{ t('svc.install') }} {{ installModal.title }}</h3><p>{{ t('mod.installDesc') }}</p></div>
        <div class="modal-body">
          <div class="field">
            <label>{{ t('mod.version') }}</label>
            <input type="text" v-model="installVer" :placeholder="SVC_META[installModal.svc]?.suggested[0] || ''" :disabled="installModal.single" autocomplete="off">
            <div class="quick-picks">
              <button v-for="v in installModal.suggested" :key="v" class="pick" :class="{ selected: installVer === v }" @click="pickInstallVer(v)">{{ v }}</button>
            </div>
          </div>
          <div class="field"><label>{{ t('mod.willRun') }}</label>
            <div class="cmd-preview"><span class="prompt">$ </span>{{ installCmdPreview() }}</div>
          </div>
        </div>
        <div class="modal-foot">
          <button class="btn" @click="closeModal()">{{ t('mod.cancel') }}</button>
          <button class="btn btn-primary" @click="confirmInstall()">{{ t('mod.install') }}</button>
        </div>
      </div>
      <div v-else-if="dcModal" class="modal">
        <div class="danger-header">
          <div class="danger-icon">⚠</div>
          <div style="flex:1"><h3>{{ dcModal.title }}</h3><p v-if="dcModal.description">{{ dcModal.description }}</p></div>
        </div>
        <div class="modal-body">
          <ul class="danger-list">
            <li v-for="(w, i) in dcModal.warnings" :key="i" :class="{ keep: w.keep }">{{ w.text }}</li>
          </ul>
          <div class="danger-input-wrap">
            <label>{{ dcModal.inputLabel }} <span class="key">{{ dcModal.expect }}</span></label>
            <input type="text" v-model="dcInput" :placeholder="dcModal.placeholder" :class="{ match: dcMatched }" autocomplete="off" spellcheck="false">
          </div>
          <label class="danger-check" :class="{ checked: dcChecked }">
            <input type="checkbox" v-model="dcChecked">
            <span class="check-label">{{ dcModal.checkboxLabel }}</span>
          </label>
          <label v-if="dcModal.purge" class="danger-check" :class="{ checked: dcPurge }">
            <input type="checkbox" v-model="dcPurge">
            <span class="check-label">{{ dcModal.purge.label }}<strong style="color:var(--danger)">{{ t('mod.purgeIrreversible') }}</strong></span>
          </label>
          <div class="field"><label>{{ t('mod.willRun') }}</label>
            <div class="cmd-preview"><span class="prompt">$ </span>{{ dcModal.cliPreview }}</div>
          </div>
        </div>
        <div class="modal-foot">
          <button class="btn" @click="closeModal()">{{ t('mod.cancel') }}</button>
          <button class="btn btn-danger-solid" :disabled="!dcReady" @click="dcConfirm()">{{ dcModal.confirmLabel }}</button>
        </div>
      </div>
      <div v-else-if="extModal" class="modal modal-lg">
        <div class="modal-head">
          <h3>{{ t('ext.title', { ver: extModal.version }) }}</h3>
          <p>{{ t('ext.sub') }}</p>
        </div>
        <div class="modal-body">
          <p v-if="extErr" class="alert alert-danger">{{ t('ext.loadErr') }}: {{ extErr }}</p>
          <p v-else-if="extLoading" class="dim">{{ t('task.running') }}</p>
          <template v-else>
            <div class="field">
              <label>{{ t('ext.preset') }}</label>
              <div class="quick-picks">
                <button v-for="p in ['default','minimal','web','debug']" :key="p" class="pick"
                        :class="{ selected: extPreset === p }" @click="extPickPreset(p)">{{ t('ext.preset.'+p) }}</button>
              </div>
              <div class="hint">{{ t('ext.presetHint') }}</div>
            </div>
            <div class="field">
              <div style="display:flex;justify-content:space-between;align-items:center">
                <label>{{ t('ext.enabled') }}</label>
                <span class="mono" style="font-size:11.5px;color:var(--text-mute)">{{ t('ext.selected', { n: extSelected.size, total: extEnabledList.length }) }}</span>
              </div>
              <div class="ext-toggle-grid">
                <button v-if="extEnabledList.length === 0" disabled class="hint">{{ t('ext.none') }}</button>
                <button v-for="e in extEnabledList" :key="e" class="ext-toggle" :class="extSelected.has(e) ? 'on' : 'off'"
                        @click="extToggle(e)">{{ e }}</button>
              </div>
              <div class="hint">{{ t('ext.enabledHint') }}</div>
            </div>
            <div class="field">
              <div style="display:flex;justify-content:space-between;align-items:center">
                <label>{{ t('ext.suggest') }}</label>
                <span class="mono" style="font-size:11.5px;color:var(--text-mute)">{{ t('ext.selected', { n: extSuggestList.filter(e => extSelected.has(e)).length, total: extSuggestList.length }) }}</span>
              </div>
              <div class="ext-toggle-grid">
                <button v-if="extSuggestList.length === 0" disabled class="hint">{{ t('ext.none') }}</button>
                <button v-for="e in extSuggestList" :key="e" class="ext-toggle" :class="extSelected.has(e) ? 'on' : 'off'"
                        @click="extToggle(e)">{{ e }}</button>
              </div>
              <div class="hint">{{ t('ext.suggestHint') }}</div>
            </div>
            <div class="field">
              <label>{{ t('ext.add') }}</label>
              <div style="display:flex;gap:8px">
                <input type="text" v-model="extInput" :placeholder="t('ext.addPh')" autocomplete="off"
                       @keydown.enter.prevent="extAddManual()">
                <button class="btn btn-primary" @click="extAddManual()">+ {{ t('ext.addBtn') }}</button>
              </div>
              <div class="hint">{{ t('ext.addHint') }}</div>
            </div>
            <div class="alert alert-warn">
              <strong>{{ t('ext.flow') }}</strong>{{ t('ext.flowBody') }} {{ t('ext.warn.rebuild') }}
            </div>
          </template>
        </div>
        <div class="modal-foot">
          <button class="btn" @click="closeModal()">{{ t('mod.cancel') }}</button>
          <button class="btn btn-primary" :disabled="!extDirty || extLoading" @click="extApply()">
            {{ extDirty ? t('ext.applyCount', { n: extSelected.size }) : t('ext.applied') }}
          </button>
        </div>
      </div>
      <div v-else-if="siteModal" class="modal">
        <div class="modal-head"><h3>{{ t('site.modalTitle') }}</h3><p>{{ t('site.modalSub') }}</p></div>
        <div class="modal-body">
          <p v-if="installedPhp.length === 0" class="alert alert-warn">{{ t('site.noPhp') }}</p>
          <div class="field">
            <label>{{ t('site.domain') }}</label>
            <input type="text" v-model="siteDomain" :placeholder="t('site.domainPh')" autocomplete="off">
          </div>
          <div class="field">
            <label>{{ t('site.pickPhp') }}</label>
            <div class="quick-picks">
              <button v-for="v in installedPhp" :key="v" class="pick" :class="{ selected: sitePhp === v }"
                      @click="sitePhp = v">{{ v }}</button>
            </div>
          </div>
          <p v-if="siteErr" class="alert alert-danger">{{ siteErr }}</p>
          <div class="field"><label>{{ t('site.willRun') }}</label>
            <div class="cmd-preview"><span class="prompt">$ </span>{{ siteCmdPreview }}</div>
          </div>
        </div>
        <div class="modal-foot">
          <button class="btn" @click="closeModal()">{{ t('mod.cancel') }}</button>
          <button class="btn btn-primary" :disabled="installedPhp.length === 0" @click="confirmSiteAdd()">{{ t('sites.add') }}</button>
        </div>
      </div>
    </div>
    <div class="toast-root">
      <div v-for="x in toasts" :key="x.id" class="toast" :class="x.kind">{{ x.msg }}</div>
    </div>
  </div>
</template>
