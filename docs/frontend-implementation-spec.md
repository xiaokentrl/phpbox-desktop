# phpbox Desktop 前端落地实施方案
给执行 AI 的说明：本方案是唯一真源。任何与本方案冲突的"更优雅写法"都不许采用。原型 HTML 是最终 UI/交互基准，逐像素、逐毫秒、逐文案对齐。

一、总纲：五条不可违背的约束
UI 100% 还原：所有 CSS 类名、DOM 层级、间距、圆角、动画曲线、颜色变量，逐一对照原型 HTML 的 <style> 段落。不许"顺手优化"（例如把 .btn 改成 Button 的 BEM、把 0.13s 改成 0.15s）。

交互 100% 还原：

行内编辑：click 进入编辑，Enter 提交，Escape 取消，blur 100ms 后取消

PHP 下拉：切换后 900ms 后 toast + 重渲染（不是立即）

任务日志：每行 90–290ms 随机延迟，cmd 行固定 120ms

语言切换按钮：zh ↔ en 双向，文案 English / 中文

数据 100% 还原：state 里的初始值（4 个站点、3 个备份、6 条离线缓存、PHP 8.4/8.3/8.0 等）照抄，不得删减"看起来没用的"字段。

文案 100% 还原：MESSAGES['zh-CN'] 和 MESSAGES['en-US'] 每条 key 一字不差。新增 key 必须先加进两份字典。

不许引入新依赖：只用 vue@3 / pinia / vue-router@4 / vue-i18n（若不想装 i18n 库，可用原型里的 t() 实现） / wails。不许装 tailwind、unocss、element-plus、antdv、naive。

二、落地顺序（7 个 Phase，每 Phase 可独立验收）
Phase	范围	验收标准
P0	工程骨架 + 样式 tokens + i18n 字典	pnpm dev 起得来，空白页背景色 = --bg
P1	api/client + api/events + stores/app + stores/layout	控制台 client.probe() 返回 'wails' | 'http'
P2	layouts/ + components/ui/ + TaskDrawer + ToastRoot + CommandPalette	侧栏可拖宽、抽屉可拖高、主题切换、⌘K 可弹
P3	stores/env + stores/services + modules/services/（registry 驱动）	mysql 一条线：空态 → 安装 → 端口/密码行内编辑 → 启停 → 卸载
P4	stores/sites + modules/sites/ 全套	站点 CRUD + PHP 下拉切换 + 端口行内 + 伪静态弹窗 + vhost 弹窗
P5	stores/task + useTaskRunner + 全部 meta dispatch	每个 runTask 调用点都能正确改 store，抽屉日志逐行出现
P6	modules/php（ExtensionsButton）+ go/backup/offline/overview/settings	与原型逐屏对照无差异
P7	托盘模拟 + 快捷键 + 边界态（Esc、空态、脏状态）	全部键盘/鼠标操作与原型一致
每完成一个 Phase 必须与原型 HTML 并排跑起来做像素对照。不许连续做完两个 Phase 再回来验证。

三、目录结构详解（最终版）
text
frontend/
├── package.json
├── vite.config.ts
├── tsconfig.json
├── index.html
├── wailsjs/                       # Wails 生成，不改
└── src/
    ├── main.ts
    ├── App.vue
    ├── env.d.ts
    ├── api/                        ─── 通信层（10 文件）
    │   ├── types.ts
    │   ├── client.ts
    │   ├── wails.ts
    │   ├── http.ts
    │   ├── events.ts
    │   ├── sites.ts
    │   ├── services.ts
    │   ├── service-config.ts
    │   ├── extensions.ts
    │   ├── go.ts
    │   ├── backup.ts
    │   ├── offline.ts
    │   └── settings.ts
    ├── stores/                     ─── 状态层（10 文件）
    │   ├── index.ts
    │   ├── app.ts
    │   ├── layout.ts
    │   ├── task.ts
    │   ├── env.ts
    │   ├── services.ts
    │   ├── sites.ts
    │   ├── go.ts
    │   ├── backup.ts
    │   └── offline.ts
    ├── modules/                    ─── 业务域
    │   ├── sites/                  （8 文件）
    │   ├── services/               （7 文件）
    │   ├── php/                    （4 文件）
    │   ├── go/                     （3 文件）
    │   ├── backup/                 （2 文件）
    │   ├── offline/                （2 文件）
    │   ├── overview/               （1 文件）
    │   ├── settings/               （5 文件）
    │   └── theme/                  （1 文件）
    ├── components/                 ─── 跨域共享（17 文件）
    │   ├── ui/                     （12 文件）
    │   ├── InlineEdit.vue
    │   ├── DangerModal.vue
    │   ├── TaskDrawer.vue
    │   └── CommandPalette.vue
    ├── layouts/                    （6 文件）
    ├── composables/                （7 文件）
    ├── constants/                  （7 文件）
    ├── i18n/                       （4 文件）
    ├── router/                     （2 文件）
    ├── styles/                     （4 文件）
    ├── types/                      （4 文件）
    └── utils/                      （7 文件）
四、逐文件详细规格
4.1 main.ts
ts
import { createApp } from 'vue'
import { createPinia } from 'pinia'
import App from './App.vue'
import router from './router'
import i18n from './i18n'
import { api } from './api/client'
import { useAppStore } from './stores/app'
import './styles/index.css'

const app = createApp(App)
app.use(createPinia())
app.use(router)
app.use(i18n)

// ★ 顺序：先探测通道，再挂载（原型 boot() 的顺序）
api.probe().then(channel => {
  const appStore = useAppStore()
  appStore.setChannel(channel)
  appStore.initTheme()
  app.mount('#app')
})
关键：client.probe() 必须在 mount 前完成，否则首屏 app.connection 闪烁。

4.2 App.vue
vue
<template>
  <MainLayout>
    <RouterView />
  </MainLayout>
  <TraySim v-if="isBrowser" />
  <ModalRoot />
  <ToastRoot />
  <CommandPalette />
</template>
DOM 层级必须与原型一致：.tray-sim → .tray-menu → .app → .modal-root → .cmd-palette → .toast-root。原型里这些是兄弟节点，不是嵌套。若组件封装导致层级变化，用 <Teleport to="body">。

4.3 api/client.ts
ts
export type Channel = 'wails' | 'http'

interface ApiClient {
  channel: Channel
  probe(): Promise<Channel>
  // 全部 API 方法按域分组
  sites: typeof sitesApi
  services: typeof servicesApi
  // ...
}

let current: ApiClient | null = null

export const api = {
  async probe(): Promise<Channel> {
    if (typeof window !== 'undefined' && (window as any).go?.phpbox) {
      current = buildWails()
      return 'wails'
    }
    current = buildHttp()
    return 'http'
  },
  get(): ApiClient {
    if (!current) throw new Error('api.probe() not called')
    return current
  },
}
不许在 client.ts 里出现 if (import.meta.env.DEV) 之类的分支——通道选择只由 window.go 决定。

4.4 api/events.ts
ts
export interface TaskLogEvent { taskId: string; line: { t: 'ok'|'err'|'meta'|'cmd'|'dim'; s: string } }
export interface TaskDoneEvent { taskId: string; status: 'success'|'failed'; duration: number }

export const events = {
  onTaskLog(cb: (e: TaskLogEvent) => void): () => void { /* ... */ },
  onTaskDone(cb: (e: TaskDoneEvent) => void): () => void { /* ... */ },
}
Wails 通道：runtime.EventsOn('task:log', cb)

HTTP 通道：new EventSource('/api/events').addEventListener('task:log', ...)

返回值是取消订阅函数，useTaskRunner 在 finally 里调用

4.5 stores/app.ts
ts
export const useAppStore = defineStore('app', () => {
  const channel = ref<Channel>('http')
  const theme = ref(localStorage.getItem('phpbox-theme') ?? 'midnight')
  const locale = ref<'zh-CN'|'en-US'>(localStorage.getItem('phpbox-locale') as any ?? 'zh-CN')
  const tray = reactive({ enabled: true, minimizeOnClose: true })
  const connection = ref<'connected'|'disconnected'>('connected')

  function initTheme() {
    document.documentElement.setAttribute('data-theme', theme.value)
  }
  watch(theme, v => {
    document.documentElement.setAttribute('data-theme', v)
    localStorage.setItem('phpbox-theme', v)
  })
  watch(locale, v => {
    document.documentElement.setAttribute('lang', v)
    localStorage.setItem('phpbox-locale', v)
  })

  function setChannel(c: Channel) { channel.value = c }

  return { channel, theme, locale, tray, connection, initTheme, setChannel }
})
主题应用逻辑只在这里——applyTheme(id) 在原型里做的事（setAttribute + localStorage）要严格照搬，不许在 ThemePickerModal 里再写一次。

4.6 stores/layout.ts
照搬原型 layout 对象：

ts
export const LAYOUT_LIMITS = {
  sidebar: { min: 64, max: 380, default: 200 },
  drawer:  { min: 120, max: 600, default: 160 },
}
export const LAYOUT_PRESETS = {
  compact: { sidebar: 200, drawer: 160 },
  default: { sidebar: 232, drawer: 220 },
  wide:    { sidebar: 280, drawer: 320 },
}

export const useLayoutStore = defineStore('layout', () => {
  const sidebarWidth = ref(LAYOUT_PRESETS.compact.sidebar)
  const drawerHeight = ref(LAYOUT_PRESETS.compact.drawer)

  function apply() {
    document.documentElement.style.setProperty('--sidebar-width', sidebarWidth.value + 'px')
    document.documentElement.style.setProperty('--drawer-height', drawerHeight.value + 'px')
  }
  function setSidebar(w: number) { sidebarWidth.value = clamp(w, 64, 380); apply(); persist() }
  function setDrawer(h: number)   { drawerHeight.value = clamp(h, 120, 600); apply(); persist() }
  function reset() { sidebarWidth.value = 200; drawerHeight.value = 160; apply(); persist() }
  function applyPreset(name: keyof typeof LAYOUT_PRESETS) { /* ... */ }

  return { sidebarWidth, drawerHeight, apply, setSidebar, setDrawer, reset, applyPreset }
})
关键：--sidebar-width / --drawer-height 是 CSS 变量，写在 document.documentElement 上，不是组件的 scoped style。原型就是这么做的。

4.7 stores/task.ts（最核心）
ts
export interface TaskState {
  args: string[]
  label: string
  lines: LogLine[]
  status: 'running' | 'success' | 'failed'
  cursor: number
  startedAt: number
}

export const useTaskStore = defineStore('task', () => {
  const current = ref<TaskState | null>(null)
  const expanded = ref(false)

  function start(args: string[], label: string) {
    current.value = { args, label, lines: [], status: 'running', cursor: 0, startedAt: Date.now() }
    expanded.value = true
  }
  function append(line: LogLine) {
    if (!current.value) return
    current.value.lines.push(line)
    current.value.cursor++
  }
  function complete() {
    if (!current.value) return
    current.value.status = 'success'
    current.value.lines.push({ t: 'ok', s: '✓ Done' })   // 或由后端推送
  }
  function fail(err: any) { if (current.value) current.value.status = 'failed' }
  function toggle() { expanded.value = !expanded.value }

  return { current, expanded, start, append, complete, fail, toggle }
})
日志动画是后端推的——原型的 playTask 是模拟；Vue 里由 events.onTaskLog 逐条推。useTaskRunner 订阅 + 分发：

ts
// composables/useTaskRunner.ts
export function useTaskRunner() {
  const task = useTaskStore()
  async function run(args: string[], label: string, meta: TaskMeta) {
    task.start(args, label)
    const handle = await api.get().startTask(args)
    const offLog = events.onTaskLog(e => task.append(e.line))
    const offDone = events.onTaskDone(e => {
      if (e.status === 'success') { task.complete(); dispatchMeta(meta) }
      else task.fail(e)
      offLog(); offDone()
    })
  }
  return { run }
}
dispatchMeta 是纯函数，放在 stores/task.ts 里：

ts
export function dispatchMeta(meta: TaskMeta) {
  const env = useEnvStore(); const services = useServicesStore()
  const sites = useSitesStore(); const offline = useOfflineStore()
  switch (meta.type) {
    case 'install':
      services.addInstalled(meta.kind, meta.version)
      if (meta.port) env.setPort(meta.kind, meta.version, meta.port)
      if (meta.password) env.setPassword(meta.kind, meta.version, meta.password)
      break
    case 'uninstall': services.removeInstalled(meta.kind, meta.version); break
    case 'service-stop': services.stop(meta.kind, meta.version); break
    case 'service-start': services.start(meta.kind, meta.version); break
    case 'update-config': env.setPort(meta.kind, meta.version, +meta.newValue) /* or password */; break
    case 'update-global': env.set(meta.key, meta.newValue); break
    case 'site-add': sites.add(meta); break
    case 'site-port': sites.setPort(meta.domain, meta.newValue); break
    case 'site-remove': sites.remove(meta.domain); break
    case 'rewrite': sites.setRewrite(meta.domain, meta.preset); break
    case 'service-config': /* 已保存到 service-config store */ break
    case 'extensions': services.setExtensions(meta.version, meta.added, meta.removed); break
    case 'offline-prune': offline.remove(meta.svc, meta.ver); break
    case 'restore': /* 整树 reload */ location.reload(); break
    // 注意：站点新增/删除后 320ms 再 render + toast，见原型 applyStateChange
  }
  setTimeout(() => { showToast(t('toast.uiUpdated'), 'ok', 1400) }, 320)
}
4.8 stores/env.ts
ts
export const useEnvStore = defineStore('env', () => {
  const vars = reactive<Record<string, string>>({
    WWW_ROOT: '~/www', NGINX_PORT: '80', NGINX_VERSION: 'alpine',
    MYSQL_DATA_ROOT: '~/mysql-data', PGSQL_DATA_ROOT: '~/pgsql-data',
    GO_PROJECTS_ROOT: '~/www', GO_DEFAULT_VERSION: 'alpine',
    GO_PROXY: 'https://goproxy.cn,direct',
    MYSQL_84_PORT: '3384', MYSQL_84_ROOT_PASSWORD: '123456',
    PGSQL_17_PORT: '5417', PGSQL_17_ROOT_PASSWORD: '123456',
    REDIS_8_PORT: '6379', REDIS_8_ROOT_PASSWORD: 'ba953852137eab6a',
  })
  function set(key: string, value: string) { vars[key] = value }
  function setPort(kind: ServiceKind, version: string, port: number) {
    const key = kind === 'nginx' ? 'NGINX_PORT' : `${kind.toUpperCase()}_${version.replace(/\./g,'')}_PORT`
    vars[key] = String(port)
  }
  function setPassword(kind: ServiceKind, version: string, pw: string) { /* ... */ }
  function get(key: string) { return vars[key] ?? '' }
  return { vars, set, setPort, setPassword, get }
})
初始值一字不差照抄原型 state.env。

4.9 stores/services.ts
ts
const installed = reactive<Record<ServiceKind, string[]>>({
  php: ['8.4','8.3','8.0'], mysql: ['8.4'], pgsql: ['17'],
  redis: ['8'], nginx: ['alpine'],
})
const stopped = reactive<Record<ServiceKind, string[]>>({
  php: [], mysql: [], pgsql: [], redis: [], nginx: [],
})
const phpExtensions = reactive<Record<string, string[]>>({
  '8.4': ['gd','redis','pdo_mysql','mysqli','pgsql','pdo_pgsql','zip','bcmath','intl','opcache','exif','soap','sockets','imagick','xdebug'],
  '8.3': ['gd','redis','pdo_mysql','mysqli','zip','bcmath','opcache','exif','sockets'],
  '8.0': ['gd','redis','pdo_mysql','mysqli','zip','bcmath','opcache'],
})
注意：state.installed.php 排序用 cmpVer（版本降序），新增时保持。

4.10 modules/services/registry.ts（核心数据文件）
ts
export interface ServiceField {
  key: 'port' | 'password'
  labelKey: string
  inline: 'port' | 'password'
}
export interface ServiceMeta {
  kind: ServiceKind
  titleKey: string
  subtitleKey: string
  hintKey: string
  icon: string
  suggested: string[]
  single?: boolean                 // nginx
  fields: ServiceField[]           // ★ 决定 VersionCard 渲染哪些行
  hasExtensions?: boolean          // ★ php 用
}

export const REGISTRY: Record<ServiceKind, ServiceMeta> = {
  php: {
    kind: 'php', icon: '🐘',
    titleKey: 'php.title', subtitleKey: 'php.subtitle', hintKey: 'php.empty.desc',
    suggested: ['8.4','8.3','8.2','8.1','8.0','7.4'],
    fields: [{ key: 'port', labelKey: 'svc.port', inline: 'port' }],
    hasExtensions: true,
  },
  mysql: {
    kind: 'mysql', icon: '🐬',
    titleKey: 'mysql.title', subtitleKey: 'mysql.subtitle', hintKey: 'mysql.empty.desc',
    suggested: ['9.1','8.4','8.0','5.7'],
    fields: [
      { key: 'port', labelKey: 'svc.port', inline: 'port' },
      { key: 'password', labelKey: 'svc.password', inline: 'password' },
    ],
  },
  pgsql: { /* 同 mysql，icon 🐘 */ },
  redis: { /* 同 mysql，icon ⚡ */ },
  nginx: {
    kind: 'nginx', icon: '🌐', single: true,
    titleKey: 'nginx.title', subtitleKey: 'nginx.subtitle', hintKey: 'nginx.empty.desc',
    suggested: ['alpine','1.25'],
    fields: [{ key: 'port', labelKey: 'svc.port', inline: 'port' }],
  },
}
4.11 modules/services/ServiceView.vue
vue
<script setup lang="ts">
const props = defineProps<{ kind: ServiceKind }>()
const meta = REGISTRY[props.kind]
const services = useServicesStore()
const installed = computed(() => services.installed[props.kind])
</script>

<template>
  <div class="view-inner">
    <header class="view-header">
      <div>
        <h1>{{ t(meta.titleKey) }}</h1>
        <p class="view-sub">{{ t(meta.subtitleKey) }}</p>
      </div>
      <div class="header-actions">
        <button class="btn btn-primary" @click="openInstall(meta.kind)">
          <PlusIcon /> {{ t('svc.install') }} {{ t(meta.titleKey) }}
        </button>
      </div>
    </header>

    <div v-if="!installed.length" class="empty">
      <div class="empty-icon">{{ meta.icon }}</div>
      <h2>{{ t('php.empty.title').replace('PHP', t(meta.titleKey)) }}</h2>
      <p>{{ t(meta.hintKey) }}</p>
      <button class="btn btn-primary" @click="openInstall(meta.kind)">
        {{ t('svc.install') }} {{ t(meta.titleKey) }}
      </button>
    </div>

    <div v-else class="grid grid-3">
      <VersionCard v-for="v in installed" :key="v" :kind="kind" :version="v">
        <!-- ★ php 域注入 footer 插槽 -->
        <template v-if="kind === 'php'" #footer>
          <ExtensionsButton :version="v" />
        </template>
      </VersionCard>
    </div>
  </div>
</template>
空态文案注意：原型里 php.empty.title 是 "还没有安装 PHP"，其他服务用 .replace('PHP', t(meta.titleKey)) 替换——照抄这个 trick，不要每个服务单写 key。

4.12 modules/services/components/VersionCard.vue
对照原型 versionCard(kind, version)：

vue
<template>
  <article class="card version-card">
    <div class="version-card-head">
      <span class="version-tag">{{ version }}</span>
      <span v-if="running" class="status-pill pill-ok">
        <span class="pill-dot"></span>{{ t('svc.running') }}
      </span>
      <span v-else class="status-pill pill-off">
        <span class="pill-dot"></span>{{ t('svc.stopped') }}
      </span>
    </div>

    <div>
      <!-- ★ fields 驱动 -->
      <div v-for="f in meta.fields" :key="f.key" class="kv">
        <span class="k">{{ t(f.labelKey) }}</span>
        <InlineEdit
          :inline="f.inline"
          :kind="kind"
          :version="version"
          :field="f.key"
          :value="fieldValue(f.key)"
          @commit="v => commitField(f.key, v)"
        />
      </div>
      <!-- ★ php 的扩展按钮：不用 fields 驱动，用 hasExtensions -->
      <div v-if="meta.hasExtensions" class="kv">
        <span class="k">{{ t('php.extensions') }}</span>
        <button class="btn btn-sm" @click="openExtensions(version)">
          <PlusIcon /> {{ t('php.manageExt') }}
          <span style="opacity:.6;font-family:var(--mono);font-size:11px;margin-left:2px">
            {{ extCount }}
          </span>
        </button>
      </div>
      <div class="kv">
        <span class="k">{{ t('svc.config') }}</span>
        <span class="v">config/{{ kind }}/{{ version }}/</span>
      </div>
    </div>

    <div class="version-card-foot">
      <button class="btn btn-sm" @click="openConfig">
        <FileIcon /> {{ t('svc.manageConfig') }}
        <span style="opacity:.55;font-family:var(--mono);font-size:11px;margin-left:2px">{{ filesCount }}</span>
      </button>
      <button class="btn btn-sm" :class="{ 'btn-primary': !running }" @click="toggleRunning">
        {{ running ? t('svc.stop') : t('svc.start') }}
      </button>
      <button class="btn btn-sm btn-danger" @click="openUninstall">
        {{ t('svc.uninstall') }}
      </button>
    </div>
  </article>
</template>
注意按钮顺序：原型的顺序是 [配置] [启停] [卸载]，其中启停按钮仅在 running=false 时加 btn-primary。

4.13 components/InlineEdit.vue
唯一一个同时处理 port/password/text 的组件，直接从原型 startEdit 移植：

vue
<script setup lang="ts">
const props = defineProps<{
  inline: 'port' | 'password' | 'text'
  kind: string
  version: string
  field: string
  value: string
  validate?: (v: string) => string | null
}>()
const emit = defineEmits<{ commit: [value: string] }>()

const editing = ref(false)
const saving = ref(false)
const draft = ref('')
const original = computed(() => props.value)

function start() {
  if (editing.value || saving.value) return
  draft.value = props.inline === 'password' ? getRealPassword() : props.value
  editing.value = true
  nextTick(() => { const i = inputRef.value; i?.focus(); i?.select() })
}
function commit() {
  const v = draft.value.trim()
  if (v === props.value) { cancel(); return }
  // 校验（原型：port 1-5 位数字，password ≥ 6）
  const err = props.validate?.(v)
  if (err) { toast(err, 'err'); return }
  editing.value = false
  saving.value = true
  emit('commit', v)
  setTimeout(() => { saving.value = false }, 4000)
}
function cancel() { editing.value = false; draft.value = props.value }
</script>

<template>
  <span
    class="inline-edit"
    :class="{ editing, saving, error: hasError }"
    :data-inline="inline"
    tabindex="0"
    :title="t('common.edit')"
    @click.stop="start"
    @keydown.enter.prevent="start"
  >
    <template v-if="!editing">
      <span v-if="inline === 'password'" class="pw">••••••••</span>
      <span v-else>{{ value }}</span>
    </template>
    <input
      v-else
      ref="inputRef"
      :type="inline === 'password' ? pwType : 'text'"
      :class="inputClass"
      :value="draft"
      spellcheck="false"
      autocomplete="off"
      @input="draft = ($event.target as HTMLInputElement).value"
      @keydown.enter.prevent="commit"
      @keydown.escape.prevent="cancel"
      @keydown.stop
      @blur="() => setTimeout(() => { if (editing) cancel() }, 100)"
    />
  </span>
</template>
校验规则（来自原型 startEdit）：

port：/^\d{1,5}$/，否则 toast t('common.invalidPort')

password：newValue.length < 6，否则 toast t('common.shortPassword')

input class 宽度（原型 .inline-edit input.port-input 等）：

port → port-input（width 80px，text-align right）

password → password-input（width 170px）

text → text-input（width 200px）

4.14 components/TaskDrawer.vue
对照原型 .drawer 结构：

vue
<template>
  <section class="drawer" :class="{ expanded: task.expanded }">
    <div class="drawer-resizer" @mousedown="startDrawerResize" />
    <header class="drawer-head">
      <div class="drawer-left">
        <span class="drawer-dot" :class="dotClass" />
        <span class="drawer-label">{{ task.current?.label ?? t('task.none') }}</span>
        <span class="drawer-cmd">{{ cmdText }}</span>
      </div>
      <div class="drawer-right">
        <span class="drawer-status">{{ statusText }}</span>
        <button class="icon-btn" @click="copyLog"><CopyIcon /></button>
        <button class="icon-btn" @click="task.toggle">
          <svg class="drawer-toggle-icon" ... />
        </button>
      </div>
    </header>
    <pre class="drawer-log">{{ logText }}</pre>
  </section>
</template>
关键交互：

drawer-dot 的 class 由 task.current.status 决定：running → running，success → success，failed → failed

statusText：running → t('task.running')；success → t('task.complete', {dur}) 其中 dur = ((Date.now()-startedAt)/1000).toFixed(1)

日志渲染：保留 \n，不用 v-for —— 原型的 <pre> 里 textContent 逐行追加，用 computed(() => lines.map(l => l.s).join('\n'))

每行 class 由 l.t 决定：ok → log-line ok，err → log-line err，meta → log-line meta，cmd → log-line cmd，dim → log-line dim

4.15 components/DangerModal.vue
对照原型 openDangerConfirm：

ts
interface DangerConfirmOpts {
  title: string
  description?: string
  warnings?: (string | { text: string; keep: boolean })[]
  cliPreview?: string
  checkbox?: { label: string }
  confirmLabel: string
  onConfirm: () => void
}
DOM 必须逐字对照：.danger-header（含 .danger-icon ⚠ + 标题 + 描述）→ .modal-body（.danger-list → .danger-check → CLI 预览）→ .modal-foot（取消 + 确认）。确认按钮 disabled 直到勾选。

4.16 components/CommandPalette.vue
CMD_ITEMS 数组照抄原型：

ts
export const CMD_ITEMS = [
  { labelKey: 'nav.sites', kbd: '⌘1', route: 'sites' },
  { labelKey: 'nav.php', kbd: '⌘2', route: 'php' },
  // ... 共 9 条路由
  { labelKey: 'cmd.newSite', kbd: '', action: 'site-add' },
  { labelKey: 'cmd.backup', kbd: '', task: ['backup', 'backup.nowTask'] },
  { labelKey: 'cmd.doctor', kbd: '', task: ['doctor', 'overview.doctorTask'] },
  { labelKey: 'cmd.scanGo', kbd: '', task: ['go,server', 'go.scanTask'] },
  { labelKey: 'cmd.theme', kbd: '', action: 'theme' },
  { labelKey: 'cmd.layoutCompact', kbd: '', layout: 'compact' },
  { labelKey: 'cmd.layoutDefault', kbd: '', layout: 'default' },
  { labelKey: 'cmd.layoutWide', kbd: '', layout: 'wide' },
  { labelKey: 'cmd.layoutReset', kbd: '', layout: 'reset' },
]
Enter 触发第一个结果，Esc 关闭，点遮罩关闭。

4.17 modules/sites/SitesView.vue
对照原型 renderSites：

header + summary（4 个数字）

空态：<div class="empty"> + 🌍 + 空态文案 + 按钮

表格：7 列，列宽严格照抄：22% / 9% / 11% / 22% / 10% / 10% / 16%

col-port 和 col-actions 有特殊 class（sticky）

行渲染交给 SiteTableRow.vue

4.18 modules/sites/components/SiteTableRow.vue
逐字对照原型 siteRow：

域名列：.site-domain 链接 + .favicon（首字母大写）+ .external-icon（hover 才显示）

端口列：PortEdit 组件（原型 startPortEdit）

PHP 列：<select class="php-select">，options = 已安装 ∪ 站点当前版本，降序

根目录列：.path-btn 点击 copy 路径 + toast

健康列：status-pill（up/warn/down 三态）

Hosts 列：已解析 → chip chip-accent；未解析 → btn btn-sm

操作列：.row-actions 含 ⋯（row-menu）+ 删除按钮

PHP 切换交互（原型 #view change 监听）：

ts
async function onPhpChange(e) {
  const sel = e.target
  sel.disabled = true
  sel.options[sel.selectedIndex].textContent = '…'
  site.php = newV
  // 更新 vhost
  await api.sites.switchPhp(domain, newV)
  setTimeout(() => {
    toast(t('sites.switched', { domain, version: newV }), 'ok', 2400)
    render()  // 重渲染整个视图（原型也是全量重渲染）
  }, 900)
}
注意 900ms 延迟——这是原型里的 UX 节奏，不是随机值。

4.19 modules/sites/components/RowActionsMenu.vue
弹出菜单位置计算必须照抄原型 openRowMenu：

ts
const r = anchor.getBoundingClientRect()
const mw = menu.offsetWidth, mh = menu.offsetHeight
let left = r.right - mw
let top = r.bottom + 6
if (left < 8) left = 8
if (left + mw > window.innerWidth - 8) left = window.innerWidth - mw - 8
if (top + mh > window.innerHeight - 8) top = r.top - mh - 6
if (top < 8) top = 8
menu.style.left = left + 'px'
menu.style.top = top + 'px'
必须用 position: fixed（原型 .row-menu 就是 fixed），滚动时关闭（原型有 view.scroll → closeRowMenu）。

菜单项：伪静态配置 + 分隔线 + 修改配置。没有第三项——原型的 row-menu 只有两个 action。

4.20 modules/sites/modals/SiteRewriteModal.vue
对照原型 openRewriteModal：

框架卡片网格：REWRITE_PRESETS 全量（9 个），选中态加 .selected

编辑器：textarea.rw-editor，Tab 键插入 4 空格

预览：默认折叠，showPreview 切换；预览里对 location|try_files|rewrite|if|return|last|break 高亮为 accent，$var 高亮为 ok，#... 高亮为 mute

应用按钮：脏状态检测——selected !== site.rewrite || editorVal !== preset.rule 时启用，文案切换 apply / current

4.21 modules/sites/modals/SiteConfigModal.vue
对照原型 openSiteConfigModal：

单 textarea 编辑 vhost

高度 80vh（style="height:80vh"）

脏状态：editor.value !== original → saveBtn.disabled 反转，#vhost-status 显示 ● unsaved

保存：解析 listen (\d+) 更新 site.port，解析 ^\s*root\s+([^;]+); 更新 site.root

4.22 modules/services/modals/InstallModal.vue
对照原型 openInstallModal：

版本输入 + quick-picks（已装版本加 ✓ 且 selected）

端口建议：suggestPortFor(kind, version)（mysql → 3300+N，pgsql → 5400+major，redis → 6379，nginx → 80）

密码生成：genPassword() 8 字节 → 16 hex

显示/隐藏密码按钮 + 重新生成按钮（原型 #m-toggle-pw / #m-regen-pw）

预览：phpbox {kind} install {version} --port ... --password ****xxxx --ext ...，多行用 \\\n 连接

4.23 modules/services/modals/ServiceConfigModal.vue
对照原型 openConfigModal：

左侧文件列表：每个文件项含图标 + 文件名 + .file-dot（modified 时显示）

右侧编辑器 + 顶部路径 + "Tab 键缩进 4 空格"提示

底部：foot-status（● unsaved · N/M）+ Reset + Cancel + Save

保存：runTask([kind, 'config', 'save', version, '--files', names.join(',')], ...)

Tab 缩进必须用 keydown + preventDefault + 手动插入，不能用 textarea.tabSize。

4.24 modules/php/ExtensionsButton.vue + modals/ExtensionsModal.vue
对照原型 openPhpExtensionsModal：

已启用扩展：.ext-pill.on 含 .ext-pill-main（点击 toggle）+ .ext-pill-del（点击从 enabledList 删除）

推荐扩展：.ext-toggle.on/off 点击 toggle

添加扩展：/^[a-zA-Z0-9._-]+$/ 校验，已存在 → toast toast.extAdded

脏检测：selected 集合 vs originalExts 集合

应用：计算 added/removed，runTask(['php','extension','sync',version,'--ext',...], ...)

4.25 modules/go/GoView.vue + GoGlobalConfigCard.vue
对照原型 renderGo：

全局配置卡：3 个 .kv 含 GO_PROJECTS_ROOT / GO_DEFAULT_VERSION / GO_PROXY，用 InlineEdit（data-kind="go"，meta.type='update-global'）

表格 5 列：Project / Version / Port / Status / Actions

Run / Stop 按钮 → runTask(['go','run',name], ...)

4.26 modules/backup/BackupView.vue
对照原型 renderBackup：

顶部 warning alert（.alert-warn，文案 backup.warning）

表格 5 列：Archive / Size / Time / items / Actions

Actions：Download / Restore / Delete

Download 用 Blob 生成假 gzip 内容（原型 downloadBackup）

Delete → openDeleteBackupModal

Restore → openRestoreModal

4.27 modules/offline/OfflineView.vue
对照原型 renderOffline：

summary 4 项：total / entries / verified / threshold

表格 6 列：svc / ver / size / items / lastVerify / Actions

verify → runTask(['offline','verify',svc,ver], ...)

prune → openOfflinePruneModal

4.28 modules/overview/OverviewView.vue
对照原型 renderOverview：

聚合 services.installed + services.stopped

summary 4 项：instances / running / stopped / lines

表格 5 列：Service / Version / Status / Container / Manage

Manage → router.push('/services/' + kind)

Doctor → runTask(['doctor'], t('overview.doctorTask'))

4.29 modules/settings/SettingsView.vue
对照原型 renderSettings：

分组顺序：Layout → Appearance → Tray → Paths → Go

注意：原型里没有"端口"分组（注释里说"已删除"）

Layout：3 个 preset pick + 2 个 slider

Appearance：lang-picker（zh-CN / en-US）

Tray：2 个 checkbox（minimize / show）

Paths：WWW_ROOT / MYSQL_DATA_ROOT / PGSQL_DATA_ROOT，每个含文件夹浏览按钮

Go：GO_PROJECTS_ROOT（含浏览）/ GO_DEFAULT_VERSION / GO_PROXY

保存：比较 input[data-env-key] 与 state.env，有变化才 toast

4.30 modules/theme/ThemePickerModal.vue
对照原型 openThemePicker：

6 个主题卡片

点击立即应用（app.theme = id）

关闭按钮文案 theme.done

4.31 layouts/Sidebar.vue
对照原型 .sidebar：

.brand：logo（π）+ name（phpbox）+ status（绿点 + app.connected）

.nav：由 NAV 常量驱动，分 section（nav.business / nav.services / nav.ops）

nav-item 有 badge（数量），active 态有 ::before 竖条

.sidebar-foot：Refresh / Lang / Theme 三个 btn-ghost

.sidebar-resizer：在右侧 6px

语言按钮文案：i18n.locale === 'zh-CN' ? 'English' : '中文'

4.32 layouts/SidebarResizer.vue / DrawerResizer.vue
拖拽逻辑照抄原型 setupSidebarResize / setupDrawerResize：

mousedown → 开始

document.mousemove → 更新 layout.sidebarWidth

document.mouseup → 结束

dblclick → 重置到 default（sidebar → 200，drawer → 160）

拖拽时给 #app 加 .resizing class（禁用 transition）

4.33 composables/useLayoutResize.ts
把 sidebar / drawer / modal 三套拖拽合并：

ts
export function useLayoutResize() {
  function setupSidebar(...) { /* 原型 setupSidebarResize */ }
  function setupDrawer(...) { /* 原型 setupDrawerResize */ }
  function setupModal(modalEl: HTMLElement) { /* 原型 setupModalResize */ }
  return { setupSidebar, setupDrawer, setupModal }
}
Modal 拖拽用 .modal-resizer（右下角 16x16），只在 modal 打开时启用。

4.34 composables/useShortcuts.ts
ts
export function useShortcuts() {
  const router = useRouter()
  function onKey(e: KeyboardEvent) {
    const modalOpen = document.querySelector('.modal-root.open')
    const paletteOpen = document.querySelector('.cmd-palette.open')
    if (e.key === 'Escape') {
      if (paletteOpen) return closeCmdPalette()
      if (modalOpen) return closeModal()
      return closeRowMenu()
    }
    const mod = e.ctrlKey || e.metaKey
    if (mod && e.key.toLowerCase() === 'k') { e.preventDefault(); toggleCmdPalette(); return }
    if (mod && e.key.toLowerCase() === 'r') { e.preventDefault(); toast(t('common.refresh'), 'ok', 1200); return }
    if (mod && /^[1-9]$/.test(e.key)) {
      const routes = ['sites','php','mysql','pgsql','redis','nginx','go','backup','overview']
      const idx = parseInt(e.key) - 1
      if (routes[idx]) { e.preventDefault(); router.push('/' + routes[idx]) }
    }
  }
  onMounted(() => document.addEventListener('keydown', onKey))
  onUnmounted(() => document.removeEventListener('keydown', onKey))
}
4.35 i18n/index.ts
必须手写 t()，不依赖 vue-i18n（原型是手写的）：

ts
export const i18n = reactive({
  locale: localStorage.getItem('phpbox-locale') as 'zh-CN' | 'en-US' ?? 'zh-CN',
  t(key: string, params?: Record<string, any>): string {
    const msg = MESSAGES[this.locale]?.[key] || MESSAGES['zh-CN'][key] || key
    if (!params) return msg
    return msg.replace(/\{(\w+)\}/g, (_, k) => params[k] != null ? params[k] : `{${k}}`)
  },
  setLocale(loc: 'zh-CN' | 'en-US') {
    this.locale = loc
    localStorage.setItem('phpbox-locale', loc)
    document.documentElement.setAttribute('lang', loc)
  },
})

export function t(key: string, params?: Record<string, any>) { return i18n.t(key, params) }
MESSAGES 字典照抄原型——一条不许少。

4.36 router/routes.ts
ts
export const routes = [
  { path: '/', redirect: '/sites' },
  { path: '/sites', component: () => import('@/modules/sites/SitesView.vue') },
  { path: '/php', component: () => import('@/modules/php/PhpView.vue') },
  { path: '/mysql', component: () => import('@/modules/services/ServiceView.vue'), props: { kind: 'mysql' } },
  { path: '/pgsql', component: () => import('@/modules/services/ServiceView.vue'), props: { kind: 'pgsql' } },
  { path: '/redis', component: () => import('@/modules/services/ServiceView.vue'), props: { kind: 'redis' } },
  { path: '/nginx', component: () => import('@/modules/services/ServiceView.vue'), props: { kind: 'nginx' } },
  { path: '/go', component: () => import('@/modules/go/GoView.vue') },
  { path: '/backup', component: () => import('@/modules/backup/BackupView.vue') },
  { path: '/offline', component: () => import('@/modules/offline/OfflineView.vue') },
  { path: '/settings', component: () => import('@/modules/settings/SettingsView.vue') },
  { path: '/overview', component: () => import('@/modules/overview/OverviewView.vue') },
]
php 用独立路由（不是 ServiceView kind="php"），因为 PhpView 要包一层 footer slot。

4.37 styles/tokens.css
整段照抄原型 <style> 里的 :root + 6 个 [data-theme=...]，一个字符不改。

4.38 styles/base.css
照抄 *{box-sizing:border-box;margin:0;padding:0} / html,body / body / ::-webkit-scrollbar 段落。

4.39 styles/components.css
照抄原型里除 tokens 和 base 外的所有 CSS：.app / .sidebar / .nav-item / .btn / .card / .table-wrap / .modal-* / .drawer-* / .toast-* / .row-menu-* / .ext-* / .danger-* / .theme-* / .fw-* / .tray-* / .cmd-palette-* / .config-* / .slider-* / 所有 @media 断点。

注意 3 个响应式断点：1280px / 960px / 720px / 1600px / 2200px / max-height:720px——一个不许漏。

4.40 types/task.ts
判别联合，穷举所有 meta type：

ts
export type TaskMeta =
  | { type: 'install'; kind: ServiceKind; version: string; port?: number | null; password?: string | null }
  | { type: 'uninstall'; kind: ServiceKind; version: string }
  | { type: 'service-stop'; kind: ServiceKind; version: string }
  | { type: 'service-start'; kind: ServiceKind; version: string }
  | { type: 'update-config'; kind: ServiceKind; version: string; field: 'port' | 'password'; oldValue: string; newValue: string }
  | { type: 'update-global'; key: string; newValue: string }
  | { type: 'site-add'; domain: string; port: number; php: string; rewrite: RewriteKey; root: string }
  | { type: 'site-vhost'; domain: string }
  | { type: 'site-port'; domain: string; oldValue: number; newValue: number }
  | { type: 'site-remove'; domain: string }
  | { type: 'rewrite'; domain: string; preset: RewriteKey; rule: string }
  | { type: 'service-config'; kind: ServiceKind; version: string; files: string[] }
  | { type: 'extensions'; version: string; added: string[]; removed: string[] }
  | { type: 'restore'; file: string }
  | { type: 'offline-prune'; svc: string; ver: string }
4.41 types/env.ts
ts
export type EnvKey =
  | 'WWW_ROOT' | 'NGINX_PORT' | 'NGINX_VERSION'
  | 'MYSQL_DATA_ROOT' | 'PGSQL_DATA_ROOT'
  | 'GO_PROJECTS_ROOT' | 'GO_DEFAULT_VERSION' | 'GO_PROXY'
  | `MYSQL_${string}_PORT` | `MYSQL_${string}_ROOT_PASSWORD`
  | `PGSQL_${string}_PORT` | `PGSQL_${string}_ROOT_PASSWORD`
  | `REDIS_${string}_PORT` | `REDIS_${string}_ROOT_PASSWORD`
4.42 utils/ 各文件
逐一照抄原型函数：

esc.ts → 原型 esc

version.ts → cmpVer

password.ts → genPassword（8 字节 → hex，共 16 字符）

port.ts → suggestPortFor

site-url.ts → siteUrl

format.ts → 原型里没有显式的 format 函数，但 size/at 是字符串，不用转

validators.ts → port（/^\d{1,5}$/）、password（≥6）、domain（/^[a-zA-Z0-9.-]+$/）、ext（/^[a-zA-Z0-9._-]+$/）

4.43 constants/default-configs.ts
照抄原型 DEFAULT_CONFIGS——php / mysql / pgsql / redis / nginx 五组模板，包括 Dockerfile、my.cnf、postgresql.conf、redis.conf、nginx.conf，一字不改。

4.44 constants/rewrite-presets.ts
照抄原型 REWRITE_PRESETS——9 个框架（none/laravel/thinkphp/yii2/thinkcmf/ci/symfony/wordpress/custom），每个含 icon / nameKey / tag / rule。rule 内的缩进空格数照抄。

4.45 constants/ext-lib.ts
['apcu','memcached','mongodb','amqp','yaml','ssh2','swoole','event','grpc','protobuf','igbinary','msgpack','ds','uv','pthreads']。

五、必须一致的关键数值表
项	值
侧栏默认宽	200px
侧栏拖拽范围	64–380
抽屉默认高	160px
抽屉拖拽范围	120–600
布局预设 compact	200/160
布局预设 default	232/220
布局预设 wide	280/320
弹窗拖拽范围	宽 ≥360，高 ≥240
日志行延迟	cmd 120ms；其他 90 + Math.random()*200
PHP 切换延迟	900ms
Toast 默认 3200ms	err/ok 视类型，task.copyLog 1600ms
任务完成后重渲染延迟	320ms
行内编辑 blur 取消延迟	100ms
行内编辑 saving 释放延迟	4000ms
弹窗动画	popIn .2s cubic-bezier(.2,.9,.3,1.15)
Toast 动画	toastIn .2s ease-out
侧栏折叠点	960px
内容最大宽	1200 / 1400(≥1600) / 1680(≥2200)
内容水平内边距	40 / 48(≥1600) / 60(≥2200) / 24(≤1280) / 20(≤960)
六、给执行 AI 的"防翻车清单"
不许用 <script setup> 之外的写法（例如 Options API）——统一 Composition API

不许把 .btn 等全局类改成 scoped——所有原型 CSS 是全局的，改 scoped 会破坏伪类、动画、媒体查询

不许"顺手改文案"——即使用户觉得"同步状态"改成"刷新"更好，也不许

不许合并原型里的"看起来一样"的模态——例如 SiteConfigModal 和 ServiceConfigModal 结构类似但文案、行为不同（前者单文件 80vh，后者多文件），必须分开

不许省略 tray 模拟——即使在 Wails 下永不显示，浏览器预览也要在

不许在 stores/* 里 import 其他 store 的实例并直接改其 state——必须调用对方的 action

useTaskRunner.run 成功后必须调 dispatchMeta，不许在调用点自己改 store

InlineEdit 的 4s 延迟清除 saving 必须保留——这是 UX 节奏

SiteTableRow 的 PHP 切换 900ms 延迟必须保留

列宽百分比必须照抄——22/9/11/22/10/10/16

七、验证矩阵（每 Phase 完成后手动跑）
场景	期望
首次进入	站点页有 4 个站点，导航 badge 显示 4/3/1/1/1/1/2/3
点侧栏"服务"	主题切换按钮文案 = "切换主题"；语言按钮 = "English"
⌘K	命令面板弹出，输入 "php" 只剩 PHP 相关项
点主题按钮	6 卡片网格，点卡片立即换色，data-theme 属性变
点语言按钮	全站文案变英文，导航 badge 数字不变
拖侧栏	宽度实时变，DBL click 回 200
拖抽屉（展开态）	高度实时变，DBL click 回 160
站点端口 click → 改 8081 → Enter	抽屉展开，日志逐行出现，success 后表格显示 8081，toast ✓ demo.test 端口已更新为 8081
站点 PHP 下拉切到 8.0	下拉禁用 900ms → toast → 重渲染
站点删除 → 勾选 → 确认	抽屉展开，日志 3 行，success 后站点消失，toast
服务（mysql）安装	抽屉 4 步日志，success 后卡片出现，env.MYSQL_84_PORT 更新
服务端口行内编辑为 3385	抽屉日志出现 port 3384 → 3385，成功后 env.MYSQL_84_PORT = 3385
服务卸载 → 勾选 → 确认	抽屉 3 步日志，success 后卡片消失
服务停止	状态 pill 变 off，启停按钮变 primary
服务配置弹窗	左侧 3 文件，改内容后左侧圆点出现，底部 ● unsaved · 1/3，Save 按钮文案变 保存并重载（1 个文件）
伪静态弹窗	点 Laravel 卡片，编辑器内容变 Laravel 规则，Apply 按钮启用
nginx vhost 弹窗	高度 80vh，改 listen 80 为 listen 8080，保存后站点端口列变 8080
PHP 扩展弹窗	点已启用 pill 变 off 态，点 × 从列表移除，输入 swoole 点添加进入列表且 on
备份下载	触发浏览器下载，toast ✓ 已下载到本地：backup-xxx.tar.gz
备份删除	DangerModal，勾选前确认按钮 disabled
设置页调侧栏滑杆	立即生效，toast 已切换到「标准」布局 当匹配预设时
设置页保存	有变化时 toast ✓ 已保存 N 项到 .env，无变化 toast 没有改动
Esc 在弹窗	弹窗关闭（DangerModal 除外——原型里 DangerModal 不许 Esc 关闭）
⌘1-9	路由跳转，侧栏高亮跟随
八、交付物清单
执行 AI 必须产出：

frontend/ 完整可运行工程

frontend/README.md：本地开发 + 构建命令

frontend/docs/PORTING-NOTES.md：本方案中所有"必须照抄"的条款逐条勾选确认

frontend/docs/PROTOTYPE-DIFF.md：如有任何与原型不一致之处，逐条列出并说明理由（默认应该是空的）

frontend/tests/（可选）：InlineEdit / DangerModal / useTaskRunner 的单元测试

九、给执行 AI 的最后一段话
这份方案不是"建议"，是"契约"。原型 HTML 里的每一行 CSS、每一个 setTimeout 延迟、每一条 i18n 文案，都是产品决策的结果，不是随手写的。你的任务是翻译，不是重写。

遇到以下情况请停下来提问，不要自作主张：

原型某处逻辑矛盾（例如 applyStateChange 里 site-add 后 320ms 才 render，但 dispatchMeta 又想立即改 store）

某个 API 端点 Wails 侧还没有实现

某个交互在两个原型副本间不一致

唯一允许的"优化"：把原型里重复的 DOM 结构抽取成组件（这正是本方案做的），但抽取后渲染出的 HTML 必须字节级等价。用 document.body.innerHTML 对比验证。

