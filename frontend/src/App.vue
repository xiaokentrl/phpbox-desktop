<script setup lang="ts">
// 应用壳 + 视图路由（阶段 0：内联视图；§22.1 晋升制——复用时抽组件）
import { computed, nextTick, onMounted, ref, watch } from 'vue'
import { t } from './i18n'
import { state, setRoute, setTheme, setAppLocale, initTheme, initPwdPolicy, setPwdPolicy, clearTask, taskRunning, toastBus,
  openInstall, openDanger, openExt, openSiteModal, closeModal, pushNotif, notifUnread, markNotifsRead, clearNotifs,
  type Route, type ContainerRow } from './state'
import { dispatchTask, inWails, onWailsReady } from './api/task'
import { loadPresence, loadContainers, loadResourceUsage, loadGoImages } from './api/data'
import { Events } from '@wailsio/runtime'
import { ListContainers, GetContainerLogs } from '../bindings/github.com/xiaokentrl/phpbox-desktop/internal/bindings/docker'
import { ListBackups, DeleteBackup, InspectBackup, ExportBackup } from '../bindings/github.com/xiaokentrl/phpbox-desktop/internal/bindings/backup'
import { ReadPhpExtensions } from '../bindings/github.com/xiaokentrl/phpbox-desktop/internal/bindings/php'
import { ListOfflineCache, VerifyOfflineCache, PruneOfflineCache } from '../bindings/github.com/xiaokentrl/phpbox-desktop/internal/bindings/offline'
import { ListSites, ProbeSiteHealth } from '../bindings/github.com/xiaokentrl/phpbox-desktop/internal/bindings/site'
import { ListGoProjects } from '../bindings/github.com/xiaokentrl/phpbox-desktop/internal/bindings/goprojects'
import { StartDaemon, StopDaemon } from '../bindings/github.com/xiaokentrl/phpbox-desktop/internal/bindings/runner'
import { ReadEnv, PatchEnv } from '../bindings/github.com/xiaokentrl/phpbox-desktop/internal/bindings/env'
import { ExportDiagnosticBundle } from '../bindings/github.com/xiaokentrl/phpbox-desktop/internal/bindings/diag'
import { ListCreds, GetServicePassword } from '../bindings/github.com/xiaokentrl/phpbox-desktop/internal/bindings/creds'

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
  { id: 'diag', icon: '🩺' },
  { id: 'settings', icon: '⚙️' }, { id: 'overview', icon: '◈' },
]
function go(r: string) { setRoute(r as Route) }
function countFor(id: string): number | null {
  if (id === 'sites') return state.sites.length || null
  if (id === 'backup') return state.backups.length || null
  if (id === 'go') return state.goProjects.length || null
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
// 欢迎卡起步套件（§5.1：PHP+MySQL+Nginx 推荐）：命令与 bash cli.sh 签名逐条核对——
// php/mysql install 带版本参数；nginx install 无参数（固定 alpine）
const WELCOME_STEPS: { svc: string; ver: string }[] = [
  { svc: 'php', ver: '8.4' },
  { svc: 'mysql', ver: '8.0' },
  { svc: 'nginx', ver: 'alpine' },
]
// 端口不再硬编码：真实值经 Creds 绑定读 .env（MYSQL_80_PORT 等契约键）
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
      loadContainers() // installed 由容器 labels 重新派生（真实状态，无乐观拼接）
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
  const c = state.containers.find(x => x.Name.replace(/^\//, '') === want)
  return c?.State ?? ''
}
function copyCmd(cmd: string) {
  if (navigator.clipboard?.writeText) {
    navigator.clipboard.writeText(cmd).then(() => toastBus(t('toast.copied'), 'ok'), () => toastBus(cmd, 'info', 4000))
  } else toastBus(cmd, 'info', 4000) // 非 https/localhost 降级为展示
}

// ── 服务连接凭证（§3.3/3.4 版本卡连接区）：真实端口/DSN/密码（点击显示 8s 自动掩码）──
// creds 按 服务/版本 缓存；密码明文只在 revealPwd 的 8s 窗口内存在，超时即清除
const svcCreds = ref<Record<string, { port: string; user: string; hasPass: boolean; dsnMask: string }>>({})
const pwdShown = ref<Record<string, string>>({}) // key: svc/ver → 明文（8s 窗口）
const pwdTimers: Record<string, number> = {}
async function loadSvcCreds(svc: string) {
  if (!['mysql', 'pgsql', 'redis', 'nginx'].includes(svc) || !inWails()) return
  try {
    const list = (await ListCreds(svc, svcVersions(svc))) ?? []
    const map: typeof svcCreds.value = {}
    for (const c of list) map[`${svc}/${c.version}`] = { port: c.port, user: c.user, hasPass: !!c.hasPass, dsnMask: c.dsnMask }
    svcCreds.value = map
  } catch { svcCreds.value = {} } // 读取失败：连接区退回无端口形态，不显示假值
}
function credOf(svc: string, ver: string) { return svcCreds.value[`${svc}/${ver}`] }
async function revealPwd(svc: string, ver: string) {
  const key = `${svc}/${ver}`
  if (pwdShown.value[key]) { clearTimeout(pwdTimers[key]); delete pwdTimers[key]; delete pwdShown.value[key]; return } // 已显示 → 再点隐藏
  try {
    const p = await GetServicePassword(svc, ver)
    if (!p) return // 无密码（未装/无键）：不显示
    pwdShown.value = { ...pwdShown.value, [key]: p }
    pwdTimers[key] = window.setTimeout(() => { delete pwdShown.value[key] }, state.pwdPolicy.showSec * 1000) // 自动掩码（§3.10 时长可调）
  } catch { /* 读取失败保持掩码 */ }
}
function pwdLabel(svc: string, ver: string): string {
  const key = `${svc}/${ver}`
  return pwdShown.value[key] ?? '••••••••'
}
function dsnOf(svc: string, ver: string): string {
  const c = credOf(svc, ver)
  if (!c) return ''
  const p = pwdShown.value[`${svc}/${ver}`]
  // 复制明文受 §3.10 策略控制：allowCopy=false 时复制的命令永远带掩码（屏幕显示不受影响）
  const pass = p && state.pwdPolicy.allowCopy ? p : '***'
  if (svc === 'mysql') return `mysql -h127.0.0.1 -P${c.port} -u${c.user} -p${pass}`
  if (svc === 'pgsql') return `postgresql://${c.user}:${pass}@127.0.0.1:${c.port}/postgres`
  if (svc === 'redis') return `redis-cli -h 127.0.0.1 -p ${c.port} -a ${pass}`
  if (svc === 'nginx') return `http://localhost:${c.port}`
  return ''
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
          loadContainers() // installed 由容器 labels 重新派生（真实状态，无乐观删减）
        },
      })
    },
  })
}

// ── nginx 快捷操作（§3.6）：reload / 修改端口——真实 CLI（nginx/cli.sh: reload · port set <端口>）──
// 无「校验配置」按钮：nginx -t 只是 bash 内部 _nginx_validate，不是 CLI 子命令（诚实降级）
function nginxReload() {
  const args = ['nginx', 'reload']
  dispatchTask(t('ngx.reloadTask'), 'phpbox nginx reload', args, {
    doneMsg: t('ngx.reloadDone'),
    fallback: [{ d: 400, lines: ['[INFO] 演示环境：nginx reload 模拟输出', '[OK] nginx 配置已重载'] }],
  })
}
// 修改端口弹窗：本地输入态（modal 走 state 统一互斥）
const ngxPortModal = ref(false)
const ngxPortInput = ref('')
const NGX_PORT_RE = /^[1-9][0-9]{0,4}$/
const ngxPortValid = computed(() => NGX_PORT_RE.test(ngxPortInput.value) && +ngxPortInput.value <= 65535)
function ngxPortPreview(): string {
  const p = ngxPortInput.value.trim()
  return `phpbox nginx port set ${NGX_PORT_RE.test(p) && +p <= 65535 ? p : '<端口>'}`
}
function openNginxPortModal() {
  ngxPortInput.value = credOf('nginx', 'alpine')?.port ?? '' // 预填当前真实端口（.env NGINX_PORT）
  ngxPortModal.value = true
}
function confirmNginxPort() {
  if (!ngxPortValid.value || taskRunning()) return
  const p = ngxPortInput.value.trim()
  const args = ['nginx', 'port', 'set', p]
  ngxPortModal.value = false
  dispatchTask(t('ngx.portSetTask'), `phpbox nginx port set ${p}`, args, {
    doneMsg: t('ngx.portSetDone', { p }),
    fallback: [{ d: 400, lines: ['[INFO] 演示环境：端口变更模拟输出', `[OK] 端口已改为 ${p}`] }],
    onDone: () => { loadContainers(); loadSvcCreds('nginx') }, // 新端口经 .env 重读
  })
}

// ── 命令面板（⌘K）——原型 4.16 契约；条目只收录真实能力（GUI 不承诺不存在的能力）──
// 原型的 layout 四条与 cmd.scanGo（'go,server' 非真实签名）不收录：布局拖拽未实装、真实命令是 'phpbox go server'
const CMD_ITEMS: { labelKey: string; kbd: string; route?: Route; run?: () => void }[] = [
  { labelKey: 'nav.sites', kbd: '⌘1', route: 'sites' },
  { labelKey: 'nav.php', kbd: '⌘2', route: 'php' },
  { labelKey: 'nav.mysql', kbd: '⌘3', route: 'mysql' },
  { labelKey: 'nav.pgsql', kbd: '⌘4', route: 'pgsql' },
  { labelKey: 'nav.redis', kbd: '⌘5', route: 'redis' },
  { labelKey: 'nav.nginx', kbd: '⌘6', route: 'nginx' },
  { labelKey: 'nav.go', kbd: '⌘7', route: 'go' },
  { labelKey: 'nav.backup', kbd: '⌘8', route: 'backup' },
  { labelKey: 'nav.settings', kbd: '', route: 'settings' },
  { labelKey: 'nav.offline', kbd: '', route: 'offline' },
  { labelKey: 'nav.overview', kbd: '⌘9', route: 'overview' },
  { labelKey: 'cmd.newSite', kbd: '', run: () => openSiteModal() },
  { labelKey: 'cmd.backup', kbd: '', run: () => { setRoute('backup'); backupNow() } },
  { labelKey: 'cmd.doctor', kbd: '', run: () => { setRoute('overview'); runDiagnostics() } },
  { labelKey: 'cmd.theme', kbd: '', run: () => { themePop.value = true } },
]
const paletteQuery = ref('')
const paletteActive = ref(0)
const paletteInputEl = ref<HTMLInputElement | null>(null)
const paletteMatches = computed(() => {
  const q = paletteQuery.value.trim().toLowerCase()
  if (!q) return CMD_ITEMS
  return CMD_ITEMS.filter(it =>
    t(it.labelKey).toLowerCase().includes(q) || it.labelKey.toLowerCase().includes(q))
})
function openCmdPalette() {
  state.palette = true
  paletteQuery.value = ''; paletteActive.value = 0
  nextTick(() => paletteInputEl.value?.focus())
}
function closeCmdPalette() { state.palette = false }

// ── 通知中心（§2.2 四真实源）──
function toggleNotifs() {
  themePop.value = false // 弹层互斥
  state.notifOpen = !state.notifOpen
  if (state.notifOpen) markNotifsRead()
}
function notifIcon(kind: string): string {
  return kind === 'task_done' ? '✓' : kind === 'task_fail' ? '✕' : kind === 'daemon_down' ? '⛔'
    : kind === 'offline_quota' ? '🗄️' : '⏹'
}
function notifMsg(n: { kind: string; msg: string }): string {
  return t('notif.' + n.kind) + n.msg
}
function execCmd(it: { route?: Route; run?: () => void }) {
  closeCmdPalette()
  if (it.route) { setRoute(it.route); document.querySelector('.view')?.scrollTo({ top: 0 }) }
  else it.run?.()
}
function onPaletteKey(e: KeyboardEvent) {
  if (e.key === 'ArrowDown' || e.key === 'ArrowUp') {
    e.preventDefault()
    const n = paletteMatches.value.length
    if (!n) return
    paletteActive.value = (paletteActive.value + (e.key === 'ArrowDown' ? 1 : -1) + n) % n
  } else if (e.key === 'Enter') {
    e.preventDefault()
    const it = paletteMatches.value[paletteActive.value]
    if (it) execCmd(it)
  }
}

// ── 全局快捷键（原型 3350：Esc 面板 > 弹窗；⌘K 切换；⌘R 同步；⌘1-9 路由）──
const ROUTE_KEYS: Route[] = ['sites', 'php', 'mysql', 'pgsql', 'redis', 'nginx', 'go', 'backup', 'overview']
function onGlobalKey(e: KeyboardEvent) {
  if (e.key === 'Escape') {
    if (state.palette) { closeCmdPalette(); return }
    if (ngxPortModal.value) { ngxPortModal.value = false; return }
    if (goImgModal.value) { goImgModal.value = false; return }
    if (bkModal.value) { bkModal.value = false; return }
    if (state.modal) { closeModal(); return }
    return
  }
  const mod = e.ctrlKey || e.metaKey
  if (mod && e.key.toLowerCase() === 'k') {
    e.preventDefault()
    state.palette ? closeCmdPalette() : openCmdPalette()
    return
  }
  if (mod && e.key.toLowerCase() === 'r') {
    e.preventDefault()
    loadContainers(); toastBus(t('foot.sync'), 'ok', 1200)
    return
  }
  if (mod && /^[1-9]$/.test(e.key)) {
    const r = ROUTE_KEYS[parseInt(e.key) - 1]
    if (r) { e.preventDefault(); setRoute(r); document.querySelector('.view')?.scrollTo({ top: 0 }) }
  }
}
onMounted(() => document.addEventListener('keydown', onGlobalKey))
// 全局点击：关闭侧栏弹层（主题/通知）——原型该缺陷一并修复
onMounted(() => document.addEventListener('click', () => { themePop.value = false; state.notifOpen = false }))

// ── Toast ──
const toasts = ref<{ id: number; msg: string; kind: string }[]>([])
let toastSeq = 0
window.addEventListener('phpbox:toast', (e) => {
  const d = (e as CustomEvent).detail
  const id = ++toastSeq
  toasts.value.push({ id, msg: d.msg, kind: d.kind })
  setTimeout(() => { toasts.value = toasts.value.filter(x => x.id !== id) }, d.ttl || 3200)
})

// ── Docker 真实数据 + installed 派生（单一实现在 api/data.ts，此处仅引用）──
// 引擎就绪度横幅：任一条件缺失即降级展示（浏览器降级 presence=null 不显示，不做假检测）
const presence = computed(() => state.presence
  ? { ...state.presence, degraded: !state.presence.EngineDir || !state.presence.CliInPath || !state.presence.DockerOK }
  : null)
async function refreshContainers() { loadContainers() }
// 欢迎卡（§5.1 末节点）：引擎就绪 + Docker 可达 + 已装服务为零 → 推荐起步套件。
// installed 为零的判定真实（容器 labels 派生，与 cmd_list 同源）；任一服务装好即消失。
const showWelcome = computed(() => {
  const p = state.presence
  return !!p && p.EngineDir && !!p.CliInPath && p.DockerOK && !state.dockerErr
    && Object.keys(state.installed).length === 0 && state.containers.length === 0
})
onMounted(() => {
  initTheme(); initPwdPolicy(); loadContainers()
  onWailsReady(() => { loadPresence(); loadBackups(); loadOffline(); loadSites(); loadGoProjects() })
  initTrayNav() // 托盘菜单快速跳转（ui:navigate）
  initDaemonEvents() // 长驻进程通道（go run / go logs）
})
watch(() => state.route, (r) => {
  if (r === 'backup') loadBackups()
  if (r === 'offline') loadOffline()
  if (r === 'sites') loadSites()
  if (r === 'go') { loadGoProjects(); loadGoImages() }
  if (r === 'diag') { loadContainers(); loadPresence() } // 诊断页进入即刷新信号（存在性 + 容器状态）
  if (r === 'overview') loadResourceUsage() // 资源小部件（目录递归 stat 是实时快照，不缓存）
  if (r === 'settings') loadEnv()
  if (['mysql', 'pgsql', 'redis', 'nginx'].includes(r)) { loadSvcCreds(r) } // 凭证线进入即读真实端口/密码契约
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

// ── 诊断视图（§5.7 阶段 0 降级形态：只读信号聚合，无一键修复——bash CLI 无
//    restart/chown/sock-clean 子命令，修复属 v1.1 Go 引擎前提；CLI 兜底命令展示）──
// 异常容器：非 running 状态（exited/restarting/paused…）即排查候选；仅 phpbox 管理的
// 容器（有 Service label）纳入，第三方容器（如 dpanel）不掺入
const diagAbnormal = computed(() => state.containers.filter(c => c.Service && c.State !== 'running'))

// ── 总览聚合（§3.11 增强）：按 phpbox 服务线分组（labels 事实源）+ 第三方容器独立区 ──
const SVC_ICONS: Record<string, string> = { php: '🐘', mysql: '🐬', pgsql: '🐘', redis: '⚡', nginx: '🌐', go: '🐹' }
const SVC_ORDER = ['php', 'mysql', 'pgsql', 'redis', 'nginx', 'go']
// 分组行：Service label 归线（同一服务多版本合并成组行，rowspan 聚合展示）
const overviewGroups = computed(() => {
  const map = new Map<string, typeof state.containers>()
  for (const c of state.containers) {
    if (!c.Service) continue
    const arr = map.get(c.Service) ?? []
    arr.push(c)
    map.set(c.Service, arr)
  }
  const order = SVC_ORDER.filter(s => map.has(s))
  for (const s of [...map.keys()]) if (!order.includes(s)) order.push(s) // 未知服务线（labels 演进）排后但如实显示
  return order.map(service => ({ service, icon: SVC_ICONS[service] ?? '📦', items: map.get(service)! }))
})
// 无 phpbox labels 的容器（如 dpanel）：不归组不隐藏，独立呈现让开发者自己判断
const otherContainers = computed(() => state.containers.filter(c => !c.Service))
const abnormalContainers = computed(() => state.containers.filter(c => c.Service && c.State !== 'running'))
// 日志卡状态：当前选中容器 + tail 行（50 行默认；展示原始错误，不吞不修饰）
const diagSel = ref('')
const diagLogs = ref<string[]>([])
const diagLogErr = ref('')
const diagLogBusy = ref(false)
async function openDiagLogs(name: string) {
  diagSel.value = name
  diagLogs.value = []
  diagLogErr.value = ''
  if (!inWails()) return // 浏览器降级：不伪造日志
  diagLogBusy.value = true
  try {
    diagLogs.value = (await GetContainerLogs(name, 50)) ?? []
  } catch (e) {
    diagLogErr.value = String(e) // 容器不存在/daemon 不可达原样呈现
  } finally {
    diagLogBusy.value = false
  }
}
// 导出诊断包（§5.7 未知故障分支）：真实采集（presence/容器/日志/站点/扩展/.env 脱敏）
// → 原生保存对话框；取消静默（用户主动取消不是失败），成功 toast 带路径
const diagExportBusy = ref(false)
async function exportDiagBundle() {
  if (!inWails()) { toastBus(t('diag.export.browserOnly'), 'err', 4000); return }
  diagExportBusy.value = true
  try {
    const path = await ExportDiagnosticBundle()
    if (path) toastBus(`${t('diag.export.done')}: ${path}`, 'ok', 5000)
  } catch (e) {
    toastBus(String(e), 'err', 5000)
  } finally {
    diagExportBusy.value = false
  }
}

// ── PHP 扩展管理弹窗（真实状态 + 目标集合 → 逐个 add/remove spawn）──
const EXT_LIB = ['apcu','memcached','mongodb','amqp','yaml','ssh2','swoole','event','grpc','protobuf','igbinary','msgpack','ds','uv','pthreads']
const extModal = computed(() => state.modal?.kind === 'ext' ? state.modal : null)
const extInstalled = ref<string[]>([])   // extensions.env 真实状态
const extExtra = ref<string[]>([])       // 本次手动添加
const extSelected = ref<Set<string>>(new Set())
const extInput = ref('')
const extErr = ref('')
const extLoading = ref(false)

watch(() => state.modal?.kind, async (k) => {
  if (k !== 'ext') return
  const m = extModal.value
  if (!m) return
  extInstalled.value = []; extExtra.value = []; extSelected.value = new Set()
  extInput.value = ''; extErr.value = ''; extLoading.value = true
  if (!inWails()) { // 浏览器降级：演示数据
    const demo = ['gd','redis','pdo_mysql','mysqli','pgsql','pdo_pgsql','zip','bcmath','intl','opcache','exif','soap','sockets','imagick','xdebug']
    extInstalled.value = demo
    extSelected.value = new Set(demo)
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
}
function extAddManual() {
  const name = extInput.value.trim()
  if (!name) return
  if (!/^[a-zA-Z0-9._-]+$/.test(name)) { toastBus(t('ext.badName'), 'err'); return }
  if (extEnabledList.value.includes(name)) { toastBus(t('ext.dupe', { name }), 'info', 1600); extInput.value = ''; return }
  extExtra.value.push(name)
  const s = new Set(extSelected.value); s.add(name); extSelected.value = s
  extInput.value = ''
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

// ── 备份内容查看 / 导出（§3.8）：归档是普通文件——gzip/tar 头解析只读；导出=复制到任意目录 ──
const bkBusy = ref<Record<string, boolean>>({})
const bkModal = ref(false)
const bkEntries = ref<{ path: string; size: number; isDir: boolean }[]>([])
const bkTruncated = ref(false)
const bkModalFile = ref('')
const bkModalErr = ref('')
async function inspectBackup(file: string) {
  bkModalFile.value = file
  bkEntries.value = []
  bkTruncated.value = false
  bkModalErr.value = ''
  bkModal.value = true
  if (!inWails()) { bkModalErr.value = t('bk.browserOnly'); return }
  bkBusy.value[file] = true
  try {
    const info = await InspectBackup(file)
    bkEntries.value = (info?.entries ?? []).map(e => ({ path: e.path, size: Number(e.size), isDir: !!e.isDir }))
    bkTruncated.value = !!info?.truncated
  } catch (e) {
    bkModalErr.value = String(e) // 损坏归档/gzip 头不可读：原样呈现
  } finally { bkBusy.value[file] = false }
}
async function exportBackup(b: { file: string }) {
  if (!inWails()) { toastBus(t('bk.browserOnly'), 'err', 4000); return }
  bkBusy.value[b.file] = true
  try {
    const dst = await ExportBackup(b.file) // 原生保存对话框（取消返回空串，静默）
    if (dst) toastBus(`${t('bk.exportDone')}: ${dst}`, 'ok', 6000)
  } catch (e) {
    toastBus(String(e), 'err', 6000)
  } finally { bkBusy.value[b.file] = false }
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
// 环形图分片（§3.9 总览：按服务分色）：真实 size 求和分组；SVG 纯手绘不引依赖
const SVC_COLORS: Record<string, string> = {
  php: '#8b5cf6', mysql: '#5b9cff', pgsql: '#74acff', redis: '#ff5c5c', nginx: '#3dd68c', go: '#f5a623',
}
const offlineSlices = computed(() => {
  const bySvc = new Map<string, number>()
  for (const r of state.offlineCache) bySvc.set(r.svc, (bySvc.get(r.svc) ?? 0) + r.size)
  const total = offlineTotal.value
  if (total <= 0) return []
  let acc = 0 // 累计弧度（SVG arc 从 -90° 起顺时针）
  return [...bySvc.entries()].sort((a, b) => b[1] - a[1]).map(([svc, size]) => {
    const start = acc / total
    acc += size
    const end = acc / total
    const a0 = start * Math.PI * 2 - Math.PI / 2, a1 = end * Math.PI * 2 - Math.PI / 2
    const R = 52, CX = 60, CY = 60
    const large = end - start > 0.5 ? 1 : 0
    const x0 = CX + R * Math.cos(a0), y0 = CY + R * Math.sin(a0)
    const x1 = CX + R * Math.cos(a1), y1 = CY + R * Math.sin(a1)
    return {
      svc, size, pct: size / total,
      d: `M ${x0.toFixed(2)} ${y0.toFixed(2)} A ${R} ${R} 0 ${large} 1 ${x1.toFixed(2)} ${y1.toFixed(2)}`,
      color: SVC_COLORS[svc] || '#8b95ab',
    }
  })
})
async function verifyOffline(svc: string, ver: string) {
  const key = `${svc}/${ver}`
  if (!inWails()) { toastBus(t('off.browserHint'), 'info'); return }
  offlineVerifyState.value[key] = 'busy'
  try {
    const res = await VerifyOfflineCache(svc, ver)
    offlineVerifyState.value[key] = res?.ok ? 'ok' : (res?.detail || 'bad')
    toastBus(`${key}: ${res?.detail ?? ''}`, res?.ok ? 'ok' : 'err', 5000)
    await loadOffline() // 刷新"最后验证"列（结论已落盘 .verify-state）
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
// 域名校验对齐 bash _valid_domain：每段以字母数字开头结尾，点分段可有零段（允许 localhost 等单段域名）
const DOMAIN_RE = /^[a-zA-Z0-9]([a-zA-Z0-9-]*[a-zA-Z0-9])?(\.[a-zA-Z0-9]([a-zA-Z0-9-]*[a-zA-Z0-9])?)*$/
const installedPhp = computed(() => (state.installed as Record<string, string[]>).php ?? [])
const phpVerToKey = (v: string) => 'php' + v.replace(/\./g, '')  // 8.4 → php84（bash 服务键）
// 服务键 → 展示名（php84 → 8.4；未知键原样展示）
const phpKeyLabel = (k: string) => installedPhp.value.find(v => phpVerToKey(v) === k) ?? k

// 健康徽章（§3.1 三态）：'' 未探测灰 / up 绿 / degraded 黄 / down 灰（未启动）
const HEALTH_PILL: Record<string, string> = { '': 'pill-off', up: 'pill-ok', degraded: 'pill-warn', down: 'pill-off' }
const HEALTH_TXT: Record<string, string> = { '': 'site.health.none', up: 'health.up', degraded: 'health.warn', down: 'health.down' }

async function loadSites() {
  if (!inWails()) return // 浏览器降级：保留空列表
  try {
    const rows = (await ListSites()) ?? []
    state.sites = rows.map(r => ({ domain: r.domain, php: r.php, root: r.root, hosts: !!r.hosts, health: '' as const }))
    // 健康探测异步补齐：Go 侧 HEAD（WebView fetch 读不到状态码），单站失败不影响列表
    Promise.all(state.sites.map(async s => {
      try {
        const res = await ProbeSiteHealth(s.domain)
        if (res.status === 'up' || res.status === 'degraded' || res.status === 'down') s.health = res.status
      } catch { /* 保留 ''（未探测） */ }
    }))
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
  // 服务键 php84 → 版本号 8.4：不能简单去前缀（php84 去掉 php 是 84，丢了点号，
  // bash 侧 label 匹配 phpbox-version=8.4 必失败）——从已安装版本表反查唯一前缀匹配
  const ver = installedPhp.value.find(v => phpVerToKey(v) === sel.value)
  if (!ver) return
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

// ── Go 项目（真实发现：goproject 绑定扫描 ~/www + go-<项目> 容器状态）──
const GO_ROOT_LABEL = '~/www'
async function loadGoProjects() {
  if (!inWails()) return // 浏览器降级：保留空列表
  try {
    const rows = (await ListGoProjects()) ?? []
    state.goProjects = rows.map(r => ({ name: r.name, dir: r.dir, running: !!r.running }))
  } catch (e) { toastBus(String(e), 'err', 5000) }
}
function goTask(action: 'test' | 'stop', name: string) {
  // go run/logs 是长驻进程（compose exec / logs -f），不走单任务抽屉（会占死队列），v0.1 专用通道
  const label = action === 'test' ? t('gp.task.test') : t('gp.task.stop')
  dispatchTask(`${label}·${name}`, `phpbox go ${action} ${name}`, ['go', action, name], {
    fallback: [{ d: 400, lines: [`[INFO] 演示环境：phpbox go ${action} ${name}`] }],
    onDone: () => { loadGoProjects() },
  })
}

// ── Go 镜像管理（§3.7）：装卸经 CLI spawn（bash 契约 install [版本] / uninstall <版本> [--purge]）──
// 版本参数形态：alpine / latest / 纯数字点分（validate_version ^[0-9]+(\.[0-9]+){0,2}$；latest 归一化 alpine）
const GO_VER_RE = /^[0-9]+(\.[0-9]+){0,2}$/
const goImgVer = ref('')
const goImgModal = ref(false)
const goImgValid = computed(() => goImgVer.value === 'alpine' || goImgVer.value === 'latest' || GO_VER_RE.test(goImgVer.value))
function goImgPreview(): string {
  const v = goImgVer.value.trim()
  const ok = v === 'alpine' || v === 'latest' || GO_VER_RE.test(v)
  return `phpbox go install ${ok ? v : '<版本>'}`
}
function openGoImgInstall() {
  goImgVer.value = 'alpine' // bash 默认（_go_install requested:-alpine）
  goImgModal.value = true
}
function confirmGoImgInstall() {
  if (!goImgValid.value || taskRunning()) return
  const v = goImgVer.value.trim()
  const args = ['go', 'install', v]
  goImgModal.value = false
  dispatchTask(t('gimg.installTask'), `phpbox go install ${v}`, args, {
    doneMsg: t('gimg.installDone', { v }),
    fallback: [{ d: 400, lines: ['[INFO] 演示环境：Go 镜像安装模拟输出'] }],
    onDone: () => { loadGoImages(); loadEnv() }, // GO_DEFAULT_VERSION 会写入 .env，一并重读
  })
}
function openGoImgUninstall(img: { tag: string; image: string; inUse: boolean; usedBy: string; default: boolean }) {
  if (img.inUse) { // bash 同源拒绝：ancestor 被容器用时 uninstall 会失败——GUI 直接给出原因
    toastBus(t('gimg.inUseErr', { names: img.usedBy }), 'err', 6000)
    return
  }
  openDanger({
    title: t('gimg.unTitle', { image: img.image }),
    description: t('gimg.unDesc'),
    warnings: [
      { text: t('gimg.unWarn.image', { image: img.image }) },
      ...(img.default ? [{ text: t('gimg.unWarn.default', { v: img.tag }) }] : []),
      { text: t('gimg.unWarn.cacheKeep'), keep: true },
    ],
    checkboxLabel: t('gimg.unCheck', { image: img.image }),
    inputLabel: t('mod.confirmInput'),
    expect: img.tag,
    placeholder: img.tag,
    cliPreview: `phpbox go uninstall ${img.tag}`,
    confirmLabel: t('mod.confirmUninstall'),
    purge: { label: t('gimg.purge') },
    onConfirm: (purge) => {
      const args = ['go', 'uninstall', img.tag, ...(purge ? ['--purge'] : [])]
      dispatchTask(t('gimg.unTask'), `phpbox ${args.join(' ')}`, args, {
        doneMsg: t('gimg.unDone', { image: img.image }),
        fallback: [{ d: 400, lines: ['[INFO] 演示环境：Go 镜像卸载模拟输出'] }],
        onDone: () => { loadGoImages(); loadEnv() },
      })
    },
  })
}

// ── 长驻进程通道（daemon）：go run / go logs 的持续输出流 ──
let daemonBound = false
function initDaemonEvents() {
  if (daemonBound || !inWails()) return
  daemonBound = true
  Events.On('daemon:log', (ev) => {
    const d = ev.data as any
    if (!state.daemon || d?.id !== state.daemon.id) return
    state.daemon.lines.push({ t: String(d.line ?? ''), c: d.cls || 'dim' })
    if (state.daemon.lines.length > 400) state.daemon.lines.splice(0, state.daemon.lines.length - 400) // 环形截断
  })
  Events.On('daemon:state', (ev) => {
    const d = ev.data as any
    if (!state.daemon || d?.id !== state.daemon.id) return
    state.daemon.running = !!d.running
    state.daemon.failed = !!d.failed
    if (!d.running) { // 退出：任务模型刷新项目状态
      toastBus(`${state.daemon.label}: ${d.failed ? t('dm.failed') : t('dm.exited')}`, d.failed ? 'err' : 'info')
      if (d.failed) pushNotif('daemon_down', state.daemon.id, state.daemon.label) // §2.2 通知源：daemon 断连
    }
  })
}
async function startGoRun(name: string) {
  const id = `go:${name}`
  if (state.daemon?.running) { toastBus(t('dm.busy'), 'err'); return }
  state.daemon = { id, label: `${t('gp.run')}·${name}`, cli: `phpbox go run ${name}`, lines: [{ t: '$ phpbox go run ' + name, c: 'cmd' }], running: true, failed: false }
  try {
    await StartDaemon(id, ['go', 'run', name])
  } catch (e) {
    state.daemon = null
    toastBus(String(e), 'err', 5000)
  }
}
async function stopDaemonRun() {
  const d = state.daemon
  if (!d) return
  try {
    await StopDaemon(d.id)
    toastBus(t('dm.stopped') + ' · ' + t('dm.stopHint'), 'info', 4200)
  } catch (e) { toastBus(String(e), 'err', 5000) }
}
const daemonLogEl = ref<HTMLElement | null>(null)
watch(() => state.daemon?.lines.length, async () => { // 新日志行 → 滚到底部
  if (!state.daemon) return
  await nextTick()
  if (daemonLogEl.value) daemonLogEl.value.scrollTop = daemonLogEl.value.scrollHeight
})

// ── 设置页 · 环境配置（真实 .env 读写：白名单键可编辑，系统键只读）──
interface EnvRow { key: string; value: string; editable: boolean }
const envRows = ref<EnvRow[]>([])
const envDraft = ref<Record<string, string>>({})
const envErr = ref('')
async function loadEnv() {
  envErr.value = ''
  if (!inWails()) return // 浏览器降级：保留 state.env 演示值
  try {
    const rows = (await ReadEnv()) ?? []
    envRows.value = rows.map(r => ({ key: r.key, value: r.value, editable: !!r.editable }))
    const d: Record<string, string> = {}
    for (const r of rows) d[r.key] = r.value
    envDraft.value = d
  } catch (e) { envErr.value = String(e) }
}
const envDirty = computed(() => envRows.value.some(r => (envDraft.value[r.key] ?? '') !== r.value))
async function saveEnv() {
  if (!envDirty.value) { toastBus(t('env.noChange'), 'info'); return }
  const changes: Record<string, string> = {}
  for (const r of envRows.value) {
    const now = envDraft.value[r.key] ?? ''
    if (now !== r.value) changes[r.key] = now
  }
  try {
    await PatchEnv(changes)
    toastBus(t('env.saved'), 'ok')
    toastBus(t('env.rebuildHint'), 'info', 4200)
    loadEnv()
  } catch (e) { toastBus(String(e), 'err', 5000) }
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
            <span class="dot" :class="state.dockerErr ? 'dot-err' : 'dot-ok'"></span>
            <span>{{ state.dockerErr ? 'Engine error' : t('brand.connected') }}</span>
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
        <button class="btn-ghost" @click="openCmdPalette"><span>⌘K {{ t('foot.palette') }}</span></button>
        <button class="btn-ghost notif-btn" @click.stop="toggleNotifs">
          <span>🔔</span>
          <span v-if="notifUnread()" class="notif-badge">{{ notifUnread() > 9 ? '9+' : notifUnread() }}</span>
        </button>
        <button class="btn-ghost" @click="refreshContainers"><span>↻ {{ t('btn.refresh') }}</span></button>
        <button class="btn-ghost" @click="setAppLocale(state.locale === 'zh-CN' ? 'en-US' : 'zh-CN')">
          <span>{{ state.locale === 'zh-CN' ? '🌐 English' : '🌐 中文' }}</span>
        </button>
        <button class="btn-ghost" @click.stop="themePop = !themePop; state.notifOpen = false">
          <span>🎨 {{ themeName }}</span>
        </button>
        <div v-if="themePop" class="theme-pop">
          <button v-for="th in THEMES" :key="th.id" class="theme-item" :class="{ active: state.theme === th.id }"
                  @click.stop="pickTheme(th.id)">
            <span>{{ state.locale === 'en-US' ? th.en : th.zh }}</span>
          </button>
        </div>
        <!-- 通知中心（§2.2：四真实源，无模拟推送） -->
        <div v-if="state.notifOpen" class="notif-pop">
          <div class="notif-head">
            <span>{{ t('notif.title') }}</span>
            <button v-if="state.notifications.length" class="btn btn-sm" @click.stop="clearNotifs()">{{ t('notif.clear') }}</button>
          </div>
          <div v-if="state.notifications.length === 0" class="notif-empty">{{ t('notif.empty') }}</div>
          <div v-else class="notif-list">
            <div v-for="n in state.notifications" :key="n.id" class="notif-item" :class="{ unread: !n.read }">
              <span class="notif-icon">{{ notifIcon(n.kind) }}</span>
              <div class="notif-body">
                <div class="notif-msg">{{ notifMsg(n) }}</div>
                <div class="notif-time">{{ new Date(n.at).toLocaleTimeString() }}</div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </aside>
    <main class="main">
      <!-- ═══ 引擎就绪度横幅（§5.1 首启检测 · 诚实降级：三条件独立，不做假检测）═══ -->
      <div v-if="presence?.degraded" class="view-inner presence-banner">
        <div v-if="!presence.EngineDir" class="alert alert-warn">
          <strong>{{ t('presence.engine.title') }}</strong>
          <p>{{ t('presence.engine.desc') }}</p>
          <code class="cmd-line">~/phpbox/install.sh</code>
        </div>
        <div v-else-if="!presence.CliInPath" class="alert alert-warn">
          <strong>{{ t('presence.cli.title') }}</strong>
          <p>{{ t('presence.cli.desc') }}</p>
          <code class="cmd-line">sudo ln -sf "$HOME/phpbox/bin/phpbox" /usr/local/bin/phpbox</code>
        </div>
        <div v-if="presence.EngineDir && !presence.DockerOK" class="alert alert-danger">
          <strong>{{ t('presence.docker.title') }}</strong>
          <p>{{ presence.DockerErr || t('presence.docker.desc') }}</p>
        </div>
      </div>
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
                <div class="summary-item"><div class="summary-num" style="color:var(--ok)">{{ state.sites.filter(s => s.health === 'up').length }}</div><div class="summary-label">{{ t('sum.healthy') }}</div></div>
                <div class="summary-item"><div class="summary-num">{{ state.env.NGINX_PORT }}</div><div class="summary-label">{{ t('sum.port') }}</div></div>
              </div>
              <div class="table-wrap"><table>
                <thead><tr><th>{{ t('th.domain') }}</th><th>{{ t('th.php') }}</th><th>{{ t('th.root') }}</th><th>{{ t('th.health') }}</th><th>{{ t('th.hosts') }}</th><th></th></tr></thead>
                <tbody><tr v-for="s in state.sites" :key="s.domain">
                  <td><a class="site-domain" :href="'http://'+s.domain" target="_blank" rel="noopener"><span class="favicon">{{ s.domain[0].toUpperCase() }}</span>{{ s.domain }}</a></td>
                  <td><select class="php-select" :value="s.php" @change="onSiteSwitch($event, s.domain)">
                    <option v-for="v in installedPhp" :key="v" :value="phpVerToKey(v)" :selected="phpVerToKey(v)===s.php">{{ v }}</option>
                    <!-- 站点绑定的 PHP 已卸载：保留选项并标记（诚实呈现，切换它会被 CLI 拒绝） -->
                    <option v-if="!installedPhp.some(v => phpVerToKey(v) === s.php)" :value="s.php" selected>
                      {{ t('site.phpUninstalled', { ver: phpKeyLabel(s.php) }) }}
                    </option>
                  </select></td>
                  <td><span class="mono dim">{{ s.root }}</span></td>
                  <td><span class="status-pill" :class="HEALTH_PILL[s.health]"><span class="pill-dot"></span>{{ t(HEALTH_TXT[s.health]) }}</span></td>
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
            <p v-if="state.dockerErr" class="alert alert-danger">{{ state.dockerErr }}</p>

            <!-- 欢迎卡（§5.1 末节点）：三条件就绪 + 零服务 → 推荐起步套件（逐个真实安装，无假一键全家桶） -->
            <div v-if="showWelcome" class="card welcome-card">
              <h2>{{ t('welcome.title') }}</h2>
              <p class="dim">{{ t('welcome.desc') }}</p>
              <h3 style="margin:18px 0 4px">{{ t('welcome.bundle') }}</h3>
              <p class="dim" style="font-size:12.5px">{{ t('welcome.bundleDesc') }}</p>
              <div class="grid grid-3" style="margin-top:12px">
                <article v-for="(s, i) in WELCOME_STEPS" :key="s.svc" class="welcome-step">
                  <div class="welcome-step-head">
                    <span>{{ t('welcome.step', { n: i + 1 }) }}</span>
                    <span>{{ SVC_META[s.svc].icon }}</span>
                  </div>
                  <div class="version-tag">{{ s.ver }}</div>
                  <code class="mono dim welcome-cmd">{{ s.svc === 'nginx' ? 'phpbox nginx install' : `phpbox ${s.svc} install ${s.ver}` }}</code>
                  <button class="btn btn-primary btn-sm" :disabled="taskRunning()" @click="openInstallModal(s.svc)">
                    {{ t('svc.install') }} {{ svcLabel(s.svc) }}
                  </button>
                </article>
              </div>
            </div>

            <div v-else class="summary">
              <div class="summary-item"><div class="summary-num">{{ state.containers.length }}</div><div class="summary-label">Containers</div></div>
              <div class="summary-item"><div class="summary-num" style="color:var(--ok)">{{ state.containers.filter(c=>c.State==='running').length }}</div><div class="summary-label">Running</div></div>
            </div>

            <!-- 资源占用小部件（§3.11：镜像 Docker API + 数据目录递归 stat，真实测量） -->
            <div v-if="state.resourceUsage" class="card resource-card">
              <div class="res-block">
                <h4>{{ t('res.images') }}</h4>
                <div class="res-big">{{ fmtSize(state.resourceUsage.images.Total) }}</div>
                <div class="res-sub">{{ t('res.imagesUsed', { used: fmtSize(state.resourceUsage.images.Used), n: (state.resourceUsage.images.Items ?? []).length }) }}</div>
              </div>
              <div class="res-divider"></div>
              <div class="res-block res-block-wide">
                <h4>{{ t('res.dirs') }}</h4>
                <ul class="res-dirs">
                  <li v-for="d in state.resourceUsage.dirs ?? []" :key="d.label">
                    <span class="res-dir-label">{{ t('res.dir.' + d.label) }}</span>
                    <span class="mono dim res-dir-path">{{ d.path }}</span>
                    <span class="mono res-dir-size" :class="{ dim: d.bytes === 0 }">{{ fmtSize(d.bytes) }}</span>
                  </li>
                </ul>
              </div>
            </div>

            <!-- 异常聚合区（§3.11）：phpbox 管理容器非 running → 一眼看到 + 直达诊断页 -->
            <div v-if="abnormalContainers.length > 0" class="alert alert-warn" style="margin-bottom:16px">
              <strong>{{ t('overview.abnormalTitle', { n: abnormalContainers.length }) }}</strong>
              <span class="mono" style="font-size:12px">{{ abnormalContainers.map(c => c.Name.replace(/^\//,'') + ' (' + c.State + ')').join('  ') }}</span>
              <button class="btn btn-sm" style="margin-left:8px" @click="go('diag')">{{ t('overview.abnormalGo') }}</button>
            </div>

            <div v-if="!showWelcome" class="table-wrap"><table>
              <thead><tr><th style="width:18%">{{ t('overview.col.service') }}</th><th>Container</th><th>Image</th><th>State</th></tr></thead>
              <tbody>
                <template v-for="g in overviewGroups" :key="g.service">
                  <tr v-for="(c, i) in g.items" :key="c.Name">
                    <td v-if="i === 0" :rowspan="g.items.length">
                      <span class="svc-cell"><span class="svc-icon">{{ g.icon }}</span>{{ g.service }}</span>
                    </td>
                    <td class="mono">{{ c.Name.replace(/^\//,'') }}</td><td class="mono dim">{{ c.Image }}</td>
                    <td><span class="status-pill" :class="c.State==='running'?'pill-ok':'pill-off'"><span class="pill-dot"></span>{{ c.State }}</span></td>
                  </tr>
                </template>
                <tr v-for="c in otherContainers" :key="c.Name">
                  <td><span class="svc-cell"><span class="svc-icon">🧩</span>{{ t('overview.otherSvc') }}</span></td>
                  <td class="mono">{{ c.Name.replace(/^\//,'') }}</td><td class="mono dim">{{ c.Image }}</td>
                  <td><span class="status-pill" :class="c.State==='running'?'pill-ok':'pill-off'"><span class="pill-dot"></span>{{ c.State }}</span></td>
                </tr>
              </tbody></table></div>
          </template>

          <!-- ═══ 诊断（§5.7 阶段 0：只读信号聚合 + 日志 tail + CLI 兜底；无一键修复）═══ -->
          <template v-else-if="state.route === 'diag'">
            <header class="view-header"><div><h1>{{ t('diag.title') }}</h1><p class="view-sub">{{ t('diag.sub') }}</p></div>
              <div class="header-actions">
                <button class="btn" @click="loadContainers(); loadPresence()">{{ t('btn.refresh') }}</button>
                <button class="btn" :disabled="taskRunning()" @click="runDiagnostics">⚙ {{ t('overview.diag') }}</button>
                <button class="btn btn-primary" :disabled="diagExportBusy" @click="exportDiagBundle">{{ t('diag.export') }}</button>
              </div></header>

            <div class="grid grid-3">
              <article class="card">
                <h3>{{ t('diag.sig.engine') }}</h3>
                <template v-if="state.presence">
                  <p v-if="state.presence.EngineDir" class="diag-ok">{{ t('diag.sig.engineOk') }}</p>
                  <p v-else class="diag-bad">{{ t('presence.engine.title') }}</p>
                  <p v-if="state.presence.CliInPath" class="diag-ok">{{ t('diag.sig.cliOk') }} <code class="mono">{{ state.presence.CliInPath }}</code></p>
                  <p v-else class="diag-bad">{{ t('presence.cli.title') }}</p>
                </template>
                <p v-else class="dim">{{ t('diag.sig.browser') }}</p>
              </article>
              <article class="card">
                <h3>{{ t('diag.sig.docker') }}</h3>
                <p v-if="state.dockerErr" class="diag-bad">{{ state.dockerErr }}</p>
                <template v-else-if="state.presence">
                  <p v-if="state.presence.DockerOK" class="diag-ok">{{ t('diag.sig.dockerOk') }}</p>
                  <p v-else class="diag-bad">{{ state.presence.DockerErr || t('presence.docker.title') }}</p>
                </template>
                <p v-else class="dim">{{ t('diag.sig.browser') }}</p>
              </article>
              <article class="card">
                <h3>{{ t('diag.sig.containers') }}</h3>
                <p>{{ state.containers.length }} total · <span style="color:var(--ok)">{{ state.containers.filter(c=>c.State==='running').length }} running</span></p>
                <p v-if="diagAbnormal.length === 0" class="diag-ok">{{ t('diag.sig.allRun') }}</p>
                <p v-else class="diag-bad">{{ t('diag.sig.abn', { n: diagAbnormal.length }) }}</p>
              </article>
            </div>

            <div v-if="diagAbnormal.length" class="card" style="margin-top:14px">
              <h3>{{ t('diag.abn.title') }}</h3>
              <p class="dim">{{ t('diag.abn.desc') }}</p>
              <div class="table-wrap"><table>
                <thead><tr><th>Container</th><th>Service</th><th>State</th><th></th></tr></thead>
                <tbody><tr v-for="c in diagAbnormal" :key="c.Name">
                  <td class="mono">{{ c.Name.replace(/^\//,'') }}</td>
                  <td>{{ c.Service }}<span v-if="c.Version" class="dim"> / {{ c.Version }}</span></td>
                  <td><span class="status-pill pill-off"><span class="pill-dot"></span>{{ c.State }}</span></td>
                  <td>
                    <button class="btn btn-sm" @click="openDiagLogs(c.Name.replace(/^\//,''))">{{ t('diag.logs.view') }}</button>
                    <button class="btn btn-sm btn-ghost" @click="copyCmd(`docker logs ${c.Name.replace(/^\//,'')} --tail 50`)">docker logs</button>
                  </td>
                </tr></tbody></table></div>
            </div>

            <div class="card" style="margin-top:14px">
              <h3>{{ t('diag.logs.title') }}</h3>
              <p class="dim">{{ t('diag.logs.desc') }}</p>
              <div class="diag-logbar">
                <button v-for="c in state.containers.filter(x=>x.Service)" :key="c.Name"
                        class="btn btn-sm" :class="{ 'btn-primary': diagSel === c.Name.replace(/^\//,'') }"
                        @click="openDiagLogs(c.Name.replace(/^\//,''))">
                  {{ c.Name.replace(/^\//,'') }}
                </button>
              </div>
              <template v-if="diagSel">
                <p v-if="diagLogBusy" class="dim">{{ t('diag.logs.loading') }}</p>
                <p v-else-if="diagLogErr" class="alert alert-danger">{{ diagLogErr }}</p>
                <pre v-else-if="diagLogs.length" class="diag-logview"><code v-for="(ln,i) in diagLogs" :key="i">{{ ln }}
</code></pre>
                <p v-else class="dim">{{ t('diag.logs.empty') }}</p>
              </template>
              <p v-else class="dim">{{ t('diag.logs.pick') }}</p>
            </div>

            <div class="card" style="margin-top:14px">
              <h3>{{ t('diag.cli.title') }}</h3>
              <p class="dim">{{ t('diag.cli.desc') }}</p>
              <div class="cmd-preview" v-for="cmd in ['phpbox list', 'phpbox php list', 'phpbox site list', 'docker ps -a']" :key="cmd">
                <span class="prompt">$ </span>{{ cmd }}
              </div>
            </div>
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
                  <div v-if="credOf(state.route, v)?.port" class="kv"><span class="k">{{ t('svc.card.port') }}</span><span class="v mono">{{ credOf(state.route, v)!.port }}</span></div>
                  <div v-if="['mysql','pgsql','redis'].includes(state.route)" class="kv">
                    <span class="k">{{ t('svc.card.password') }}</span>
                    <span class="v mono pwd-reveal" @click="revealPwd(state.route, v)" :title="t('svc.card.pwdHint')">{{ credOf(state.route, v)?.hasPass === false ? t('svc.card.noPwd') : pwdLabel(state.route, v) }}</span>
                  </div>
                  <div v-if="credOf(state.route, v)?.port" class="kv">
                    <span class="k">DSN</span>
                    <span class="v mono dsn-reveal" @click="copyCmd(dsnOf(state.route, v))" :title="t('svc.card.dsnCopy')">{{ credOf(state.route, v)!.dsnMask }}</span>
                  </div>
                  <div class="kv"><span class="k">{{ t('svc.card.config') }}</span><span class="v">{{ t('svc.card.configPath', { kind: state.route, ver: v }) }}</span></div>
                </div>
                <div class="version-card-foot">
                  <button v-if="state.route === 'php'" class="btn btn-sm btn-primary" @click="openExt({ version: v })">{{ t('btn.exts') }}</button>
                  <button v-if="state.route === 'nginx'" class="btn btn-sm" :disabled="taskRunning()" @click="nginxReload">{{ t('ngx.reload') }}</button>
                  <button v-if="state.route === 'nginx'" class="btn btn-sm" :disabled="taskRunning()" @click="openNginxPortModal()">{{ t('ngx.portSet') }}</button>
                  <button v-if="state.route !== 'php' && state.route !== 'nginx'" class="btn btn-sm" @click="copyCmd(`phpbox ${state.route} list`)">{{ t('svc.card.cmd') }}</button>
                  <button class="btn btn-sm btn-danger" @click="openUninstallModal(state.route, v)">{{ t('btn.uninstall') }}</button>
                </div>
              </article>
            </div>
          </template>

          <!-- ═══ Go / 备份 / 离线 / 设置（占位）═══ -->
          <template v-else-if="state.route === 'go'">
            <header class="view-header">
              <div><h1>{{ t('gp.title') }}</h1><p class="view-sub">{{ t('gp.sub', { root: GO_ROOT_LABEL }) }}</p></div>
              <div class="header-actions"><button class="btn" @click="loadGoProjects()">{{ t('btn.refresh') }}</button></div>
            </header>
            <!-- 长驻进程卡片：go run 持续输出流（daemon 通道）-->
            <div v-if="state.daemon" class="card" style="margin-bottom:16px">
              <div style="display:flex;justify-content:space-between;align-items:center;gap:10px;padding:12px 16px 8px">
                <div style="display:flex;align-items:center;gap:10px;min-width:0">
                  <span class="drawer-dot" :class="state.daemon.running ? 'running' : state.daemon.failed ? 'failed' : 'success'"></span>
                  <strong style="font-size:13.5px">{{ state.daemon.label }}</strong>
                  <span class="drawer-cmd">$ {{ state.daemon.cli }}</span>
                </div>
                <div style="display:flex;align-items:center;gap:8px">
                  <span class="drawer-status">{{ state.daemon.running ? t('dm.running') : state.daemon.failed ? t('dm.failed') : t('dm.exited') }}</span>
                  <button v-if="state.daemon.running" class="btn btn-sm btn-danger" @click="stopDaemonRun()">{{ t('dm.stopRun') }}</button>
                  <button v-else class="btn btn-sm" @click="state.daemon = null">✕</button>
                </div>
              </div>
              <div ref="daemonLogEl" style="max-height:220px;overflow-y:auto;padding:10px 16px;background:var(--bg);border-top:1px solid var(--border-2)">
                <div v-for="(l, i) in state.daemon.lines" :key="i" class="log-line" :class="l.c">{{ l.t }}</div>
              </div>
            </div>
            <div v-if="state.goProjects.length === 0" class="empty">
              <div class="empty-icon">🐹</div><h2>{{ t('gp.empty.title') }}</h2>
              <p>{{ t('gp.empty.desc', { root: GO_ROOT_LABEL }) }}</p>
            </div>
            <div v-else class="table-wrap"><table>
              <thead><tr><th style="width:24%">{{ t('gp.col.project') }}</th><th style="width:34%">{{ t('gp.col.dir') }}</th><th style="width:16%">{{ t('gp.col.status') }}</th><th></th></tr></thead>
              <tbody><tr v-for="p in state.goProjects" :key="p.name">
                <td><div class="svc-cell"><span class="svc-icon">🐹</span>{{ p.name }}</div></td>
                <td><span class="mono dim" style="font-size:12px">{{ p.dir }}</span></td>
                <td><span class="status-pill" :class="p.running ? 'pill-ok' : 'pill-off'"><span class="pill-dot"></span>{{ p.running ? t('gp.running') : t('gp.stopped') }}</span></td>
                <td><div class="row-actions">
                  <button class="btn btn-sm btn-primary" :disabled="state.daemon?.running && state.daemon.id !== `go:${p.name}`"
                          :title="t('dm.busy')" @click="startGoRun(p.name)">{{ t('gp.run') }}</button>
                  <button class="btn btn-sm" :disabled="taskRunning()" @click="goTask('test', p.name)">{{ t('gp.test') }}</button>
                  <button v-if="p.running" class="btn btn-sm" :disabled="taskRunning()" @click="goTask('stop', p.name)">{{ t('gp.stop') }}</button>
                </div></td>
              </tr></tbody></table></div>

            <!-- Go 镜像卡（§3.7）：真实 golang:* 镜像表 + 被引用状态；装卸经 CLI 任务通道 -->
            <div class="card" style="margin-top:16px;padding:16px">
              <div style="display:flex;justify-content:space-between;align-items:center;gap:10px;margin-bottom:12px">
                <div><h3 style="font-size:14px;font-weight:600">{{ t('gimg.title') }}</h3>
                  <p class="view-sub" style="margin:2px 0 0">{{ t('gimg.sub') }}</p></div>
                <div class="row-actions">
                  <button class="btn btn-sm" @click="loadGoImages()">{{ t('btn.refresh') }}</button>
                  <button class="btn btn-sm btn-primary" :disabled="taskRunning()" @click="openGoImgInstall()">{{ t('gimg.install') }}</button>
                </div>
              </div>
              <p v-if="state.goImagesErr" class="alert alert-danger">{{ t('gimg.loadErr') }}: {{ state.goImagesErr }}</p>
              <p v-else-if="state.goImages.length === 0" class="dim" style="padding:8px 0">{{ t('gimg.empty') }}</p>
              <div v-else class="table-wrap"><table>
                <thead><tr><th style="width:20%">{{ t('gimg.col.tag') }}</th><th style="width:26%">{{ t('gimg.col.image') }}</th><th style="width:12%">{{ t('th.size') }}</th><th style="width:26%">{{ t('th.state') }}</th><th></th></tr></thead>
                <tbody><tr v-for="img in state.goImages" :key="img.image">
                  <td><div class="svc-cell"><span class="svc-icon">🐹</span>{{ img.tag }}
                    <span v-if="img.default" class="chip chip-accent" style="margin-left:6px">{{ t('gimg.defaultChip') }}</span></div></td>
                  <td><span class="mono dim" style="font-size:12px">{{ img.image }}</span></td>
                  <td><span class="mono dim">{{ fmtSize(img.size) }}</span></td>
                  <td>
                    <span v-if="img.inUse" class="status-pill pill-warn" :title="img.usedBy"><span class="pill-dot"></span>{{ t('gimg.inUse') }}</span>
                    <span v-else class="status-pill pill-off"><span class="pill-dot"></span>{{ t('gimg.idle') }}</span>
                  </td>
                  <td><div class="row-actions">
                    <button class="btn btn-sm btn-danger" :disabled="taskRunning()" @click="openGoImgUninstall(img)">{{ t('btn.uninstall') }}</button>
                  </div></td>
                </tr></tbody></table></div>
            </div>
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
                  <button class="btn btn-sm" :disabled="bkBusy[b.file]" @click="inspectBackup(b.file)">{{ t('bk.contents') }}</button>
                  <button class="btn btn-sm" :disabled="bkBusy[b.file]" @click="exportBackup(b)">{{ t('bk.export') }}</button>
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
              <!-- 总占用环形图（§3.9：按服务分色，真实字节数派生） -->
              <div class="card donut-card">
                <div class="donut-wrap">
                  <svg viewBox="0 0 120 120" class="donut" role="img" :aria-label="t('off.total')">
                    <circle cx="60" cy="60" r="52" fill="none" stroke="var(--surface-3)" stroke-width="16"/>
                    <path v-for="s in offlineSlices" :key="s.svc" :d="s.d" fill="none" :stroke="s.color"
                          stroke-width="16" stroke-linecap="butt"/>
                  </svg>
                  <div class="donut-center">
                    <div class="donut-num">{{ fmtSize(offlineTotal) }}</div>
                    <div class="donut-label">{{ t('off.total') }}</div>
                  </div>
                </div>
                <ul class="donut-legend">
                  <li v-for="s in offlineSlices" :key="s.svc">
                    <span class="legend-dot" :style="{ background: s.color }"></span>
                    <span>{{ s.svc }}</span>
                    <span class="mono dim">{{ fmtSize(s.size) }}</span>
                    <span class="dim">{{ (s.pct * 100).toFixed(0) }}%</span>
                  </li>
                </ul>
              </div>
              <div class="summary">
                <div class="summary-item"><div class="summary-num">{{ fmtSize(offlineTotal) }}</div><div class="summary-label">{{ t('off.total') }}</div></div>
                <div class="summary-item"><div class="summary-num">{{ state.offlineCache.length }}</div><div class="summary-label">{{ t('off.entries') }}</div></div>
                <div class="summary-item"><div class="summary-num">{{ new Set(state.offlineCache.map(r => r.svc)).size }}</div><div class="summary-label">{{ t('off.services') }}</div></div>
              </div>
              <div class="table-wrap"><table>
                <thead><tr><th style="width:14%">{{ t('nav.services') }}</th><th style="width:10%">{{ t('th.php') }}</th><th style="width:11%">{{ t('th.size') }}</th><th style="width:8%">{{ t('off.files') }}</th><th style="width:23%">{{ t('th.state') }}</th><th style="width:14%">{{ t('off.lastVerify') }}</th><th></th></tr></thead>
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
                  <td>
                    <span v-if="r.lastVerified" class="mono" :style="{ color: r.lastVerifyOk ? 'var(--ok)' : 'var(--danger)', fontSize: '11.5px' }">{{ r.lastVerified }}</span>
                    <span v-else class="dim" style="font-size:11.5px">—</span>
                  </td>
                  <td><div class="row-actions">
                    <button class="btn btn-sm" @click="verifyOffline(r.svc, r.ver)">{{ t('off.verify') }}</button>
                    <button class="btn btn-sm btn-danger" @click="openOfflinePruneModal(r)">{{ t('off.prune') }}</button>
                  </div></td>
                </tr></tbody></table></div>
            </template>
          </template>
          <template v-else-if="state.route === 'settings'">
            <header class="view-header"><div><h1>{{ t('nav.settings') }}</h1><p class="view-sub">Theme · Language · Security · Layout</p></div></header>
            <div class="card" style="margin-bottom:16px">
              <h3 style="font-size:14px;font-weight:600;margin-bottom:14px">{{ state.locale === 'en-US' ? 'Theme' : '主题' }}</h3>
              <div class="quick-picks">
                <button v-for="th in THEMES" :key="th.id" class="pick" :class="{ selected: state.theme === th.id }" @click="setTheme(th.id)">
                  {{ state.locale === 'en-US' ? th.en : th.zh }}
                </button>
              </div>
            </div>
            <!-- 密码显示策略（§3.10 安全组）：GUI 本地偏好（localStorage）——bash .env 无此契约键，不写 .env 造平行状态 -->
            <div class="card" style="margin-bottom:16px">
              <h3 style="font-size:14px;font-weight:600;margin-bottom:14px">{{ t('pwd.title') }}</h3>
              <div class="pwd-policy-row">
                <div>
                  <div>{{ t('pwd.showSec') }}</div>
                  <p class="view-sub" style="margin-top:2px">{{ t('pwd.showSecHint') }}</p>
                </div>
                <div class="quick-picks" style="justify-content:flex-end">
                  <button v-for="n in [3, 8, 30, 60]" :key="n" class="pick" :class="{ selected: state.pwdPolicy.showSec === n }"
                          @click="setPwdPolicy({ showSec: n })">{{ n }}s</button>
                </div>
              </div>
              <div class="pwd-policy-row">
                <div>
                  <div>{{ t('pwd.allowCopy') }}</div>
                  <p class="view-sub" style="margin-top:2px">{{ t('pwd.allowCopyHint') }}</p>
                </div>
                <div class="quick-picks" style="justify-content:flex-end">
                  <button class="pick" :class="{ selected: state.pwdPolicy.allowCopy }" @click="setPwdPolicy({ allowCopy: true })">{{ t('pwd.on') }}</button>
                  <button class="pick" :class="{ selected: !state.pwdPolicy.allowCopy }" @click="setPwdPolicy({ allowCopy: false })">{{ t('pwd.off') }}</button>
                </div>
              </div>
            </div>
            <div class="card" style="margin-bottom:16px">
              <h3 style="font-size:14px;font-weight:600;margin-bottom:14px">{{ state.locale === 'en-US' ? 'Language' : '语言' }}</h3>
              <div class="quick-picks">
                <button class="pick" :class="{ selected: state.locale === 'zh-CN' }" @click="setAppLocale('zh-CN')">简体中文</button>
                <button class="pick" :class="{ selected: state.locale === 'en-US' }" @click="setAppLocale('en-US')">English</button>
              </div>
            </div>
            <div class="card" style="margin-bottom:16px">
              <div style="display:flex;justify-content:space-between;align-items:center;margin-bottom:10px">
                <div>
                  <h3 style="font-size:14px;font-weight:600">{{ t('env.title') }}</h3>
                  <p class="view-sub" style="margin-top:3px">{{ t('env.sub') }}</p>
                </div>
                <button class="btn btn-primary" :disabled="!envDirty" @click="saveEnv()">{{ t('env.save') }}</button>
              </div>
              <p v-if="envErr" class="alert alert-danger">{{ t('env.loadErr') }}: {{ envErr }}</p>
              <div v-else-if="envRows.length" style="display:flex;flex-direction:column;gap:8px">
                <div v-for="r in envRows" :key="r.key" style="display:flex;align-items:center;gap:10px">
                  <span class="mono" style="font-size:12px;color:var(--text-mute);width:210px;flex-shrink:0">{{ r.key }}</span>
                  <input v-if="r.editable" type="text" v-model="envDraft[r.key]"
                         :style="(envDraft[r.key] ?? '') !== r.value ? 'border-color:var(--accent)' : ''">
                  <span v-else class="mono dim" style="font-size:12px">{{ r.value }}</span>
                  <span class="chip" :class="r.editable ? 'chip-accent' : ''">{{ r.editable ? t('env.editable') : t('env.readonly') }}</span>
                </div>
              </div>
              <p v-else class="dim">{{ t('env.loadErr') }}</p>
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
            <span v-if="state.task.phase === 'running' && state.task.stage" class="stage-pill">{{ t('stage.' + state.task.stage) }}</span>
            <span class="drawer-status">{{ taskPhaseLabel }}</span>
            <button class="icon-btn" v-if="state.task.phase !== 'running'" :title="t('task.close')" @click="clearTask()">✕</button>
            <button class="icon-btn" @click="drawerCollapsed = !drawerCollapsed">{{ drawerCollapsed ? '▲' : '▼' }}</button>
          </div>
        </header>
        <div class="drawer-log" ref="drawerLogEl"><div v-for="(l, i) in state.task.lines" :key="i" class="log-line" :class="l.c">{{ l.t }}</div></div>
      </section>
    </main>
    <!-- 弹窗根：安装 / 危险确认（§8.2 三条件）/ nginx 端口（独立本地态）-->
    <div class="modal-root" :class="{ open: !!state.modal || ngxPortModal || goImgModal || bkModal }">
      <div v-if="bkModal" class="modal modal-lg" @click.stop>
        <div class="modal-head">
          <h3>{{ t('bk.contentsTitle', { file: bkModalFile }) }}</h3>
          <p>{{ bkTruncated ? t('bk.truncated') : t('bk.entriesCount', { n: bkEntries.length }) }}</p>
        </div>
        <div class="modal-body">
          <p v-if="bkModalErr" class="alert alert-danger">{{ bkModalErr }}</p>
          <p v-else-if="bkEntries.length === 0" class="dim">{{ t('task.running') }}</p>
          <div v-else class="bk-entries">
            <div v-for="(e, i) in bkEntries" :key="i" class="bk-entry" :class="{ dir: e.isDir }">
              <span class="mono" style="font-size:11.5px">{{ e.isDir ? '📁' : '📄' }} {{ e.path }}</span>
              <span v-if="!e.isDir" class="mono dim" style="font-size:11px">{{ fmtSize(e.size) }}</span>
            </div>
          </div>
        </div>
        <div class="modal-foot">
          <button class="btn btn-sm" @click="copyCmd(`tar -tzf ${bkModalFile} | head -100`)">tar -tzf</button>
          <button class="btn btn-primary" @click="bkModal = false">{{ t('common.close') }}</button>
        </div>
      </div>
      <div v-else-if="ngxPortModal" class="modal" @click.stop>
        <div class="modal-head">
          <h3>{{ t('ngx.portSet') }}</h3>
          <p>{{ t('ngx.portDesc') }}</p>
        </div>
        <div class="modal-body">
          <div class="field">
            <label>{{ t('ngx.port') }}</label>
            <input type="text" v-model="ngxPortInput" inputmode="numeric" :placeholder="credOf('nginx', 'alpine')?.port || '80'" autocomplete="off" spellcheck="false">
            <div v-if="ngxPortInput && !ngxPortValid" class="hint alert alert-danger" style="margin-top:8px">{{ t('ngx.portInvalid') }}</div>
          </div>
          <div class="field"><label>{{ t('mod.willRun') }}</label>
            <div class="cmd-preview"><span class="prompt">$ </span>{{ ngxPortPreview() }}</div>
          </div>
        </div>
        <div class="modal-foot">
          <button class="btn" @click="ngxPortModal = false">{{ t('mod.cancel') }}</button>
          <button class="btn btn-primary" :disabled="!ngxPortValid || taskRunning()" @click="confirmNginxPort()">{{ t('ngx.portSetGo') }}</button>
        </div>
      </div>
      <div v-else-if="goImgModal" class="modal" @click.stop>
        <div class="modal-head">
          <h3>{{ t('gimg.installTitle') }}</h3>
          <p>{{ t('gimg.installDesc') }}</p>
        </div>
        <div class="modal-body">
          <div class="field">
            <label>{{ t('mod.version') }}</label>
            <input type="text" v-model="goImgVer" placeholder="alpine" autocomplete="off" spellcheck="false">
            <div class="quick-picks">
              <button v-for="v in ['alpine', 'latest', '1.25', '1.24', '1.23']" :key="v" class="pick" :class="{ selected: goImgVer === v }" @click="goImgVer = v">{{ v }}</button>
            </div>
            <div v-if="goImgVer && !goImgValid" class="hint alert alert-danger" style="margin-top:8px">{{ t('gimg.verInvalid') }}</div>
          </div>
          <div class="alert alert-warn">{{ t('gimg.installWarn') }}</div>
          <div class="field"><label>{{ t('mod.willRun') }}</label>
            <div class="cmd-preview"><span class="prompt">$ </span>{{ goImgPreview() }}</div>
          </div>
        </div>
        <div class="modal-foot">
          <button class="btn" @click="goImgModal = false">{{ t('mod.cancel') }}</button>
          <button class="btn btn-primary" :disabled="!goImgValid || taskRunning()" @click="confirmGoImgInstall()">{{ t('mod.install') }}</button>
        </div>
      </div>
      <div v-else-if="installModal" class="modal">
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
    <div class="cmd-palette" :class="{ open: state.palette }" @mousedown.self="closeCmdPalette">
      <div class="cmd-palette-box" @keydown="onPaletteKey">
        <input ref="paletteInputEl" v-model="paletteQuery" :placeholder="t('cmd.placeholder')"
               spellcheck="false" autocomplete="off" @input="paletteActive = 0">
        <div class="cmd-palette-list">
          <div v-for="(it, i) in paletteMatches" :key="it.labelKey"
               class="cmd-palette-item" :class="{ active: i === paletteActive }"
               @mouseenter="paletteActive = i" @click="execCmd(it)">
            <span>{{ t(it.labelKey) }}</span>
            <span v-if="it.kbd" class="kbd">{{ it.kbd }}</span>
          </div>
          <div v-if="paletteMatches.length === 0" class="cmd-palette-empty">{{ t('cmd.empty') }}</div>
        </div>
      </div>
    </div>
  </div>
</template>
