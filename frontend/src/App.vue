<script setup lang="ts">
// 应用壳 + 视图路由（阶段 0：内联视图；§22.1 晋升制——复用时抽组件）
import { computed, onMounted, ref, watch } from 'vue'
import { t, locale, setAppLocale, type Locale } from './i18n'
import { state, setRoute, setTheme, initTheme, runTask, toastBus, type Route, type ContainerRow } from './state'
import { ListContainers } from '../bindings/github.com/xiaokentrl/phpbox-desktop/internal/bindings/docker'
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
  if (id === 'offline') return 10
  const list = (state.installed as Record<string, string[]>)[id]
  return list ? list.length || null : null
}
const svcLabel = (id: string) => t('nav.' + id)

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
onMounted(() => { initTheme(); loadContainers() })

// ── 任务抽屉 ──
const drawerCollapsed = ref(false)

// ── 事件委托（站点切换等）──
function onSiteSwitch(e: Event, domain: string) {
  const sel = e.target as HTMLSelectElement
  const site = state.sites.find(s => s.domain === domain)
  if (!site || site.php === sel.value) return
  const old = site.php; site.php = sel.value; sel.disabled = true
  toastBus(t('toast.switching'), 'info', 1500)
  setTimeout(() => {
    site.php = old; sel.disabled = false; sel.value = old
    toastBus('演示：切换失败已回滚', 'err', 3000)
  }, 1500)
}
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
        <button class="btn-ghost" @click="setAppLocale(locale === 'zh-CN' ? 'en-US' : 'zh-CN')">
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

          <!-- ═══ 站点（默认首屏）═══ -->
          <template v-if="state.route === 'sites'">
            <header class="view-header">
              <div><h1>{{ t('sites.title') }}</h1><p class="view-sub">{{ t('sites.sub') }}</p></div>
            </header>
            <div v-if="state.sites.length === 0" class="empty">
              <div class="empty-icon">🌍</div><h2>{{ t('sites.empty.title') }}</h2><p>{{ t('sites.empty.desc') }}</p>
            </div>
            <div v-else class="summary">
              <div class="summary-item"><div class="summary-num">{{ state.sites.length }}</div><div class="summary-label">{{ t('sum.sites') }}</div></div>
              <div class="summary-item"><div class="summary-num" style="color:var(--ok)">{{ state.sites.filter(s=>s.health==='up').length }}</div><div class="summary-label">{{ t('sum.healthy') }}</div></div>
              <div class="summary-item"><div class="summary-num">{{ state.env.NGINX_PORT }}</div><div class="summary-label">{{ t('sum.port') }}</div></div>
            </div>
            <div v-if="state.sites.length" class="table-wrap"><table>
              <thead><tr><th>{{ t('th.domain') }}</th><th>{{ t('th.php') }}</th><th>{{ t('th.health') }}</th><th>{{ t('th.hosts') }}</th></tr></thead>
              <tbody><tr v-for="s in state.sites" :key="s.domain">
                <td><a class="site-domain" :href="'http://'+s.domain" target="_blank" rel="noopener"><span class="favicon">{{ s.domain[0].toUpperCase() }}</span>{{ s.domain }}</a></td>
                <td><select class="php-select" :value="s.php" @change="onSiteSwitch($event, s.domain)">
                  <option v-for="v in ['8.4','8.2','8.0','7.4']" :key="v" :value="v" :selected="v===s.php">{{ v }}</option>
                </select></td>
                <td><span class="mono dim">{{ s.root }}</span></td>
                <td><span class="status-pill" :class="s.health==='up'?'pill-ok':s.health==='warn'?'pill-warn':'pill-err'"><span class="pill-dot"></span>{{ s.health==='up'?'正常':s.health==='warn'?'降级':'未响应' }}</span></td>
                <td><span class="chip" :class="s.hosts?'chip-accent':''">{{ s.hosts ? '已解析' : '未解析' }}</span></td>
              </tr></tbody></table></div>
          </template>

          <!-- ═══ 总览（真实 Docker 数据）═══ -->
          <template v-else-if="state.route === 'overview'">
            <header class="view-header"><div><h1>{{ t('overview.title') }}</h1><p class="view-sub">{{ t('overview.sub') }}</p></div>
              <div class="header-actions"><button class="btn" @click="loadContainers">{{ t('btn.refresh') }}</button></div></header>
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

          <!-- ═══ 服务线（通用卡片视图）═══ -->
          <template v-else-if="['php','mysql','pgsql','redis','nginx'].includes(state.route)">
            <header class="view-header"><div><h1>{{ svcLabel(state.route) }}</h1><p class="view-sub">{{ t('svc.'+state.route+'.sub') }}</p></div></header>
            <div class="empty"><div class="empty-icon">📦</div><h2>{{ svcLabel(state.route) }}</h2><p>Service management view — full port next slice.</p></div>
          </template>

          <!-- ═══ Go / 备份 / 离线 / 设置（占位）═══ -->
          <template v-else-if="state.route === 'go'">
            <header class="view-header"><div><h1>Go</h1><p class="view-sub">go.mod auto-discovery</p></div></header>
            <div class="empty"><div class="empty-icon">🐹</div><h2>Go Projects</h2><p>Projects under ~/www with go.mod are auto-discovered.</p></div>
          </template>
          <template v-else-if="state.route === 'backup'">
            <header class="view-header"><div><h1>Backups</h1><p class="view-sub">Archives downloadable to local machine</p></div></header>
            <div class="empty"><div class="empty-icon">📦</div><h2>Backups</h2><p>Backup management — full port next slice.</p></div>
          </template>
          <template v-else-if="state.route === 'offline'">
            <header class="view-header"><div><h1>Offline Cache</h1><p class="view-sub">Zero-network installs rely on this</p></div></header>
            <div class="summary"><div class="summary-item"><div class="summary-num">1.3 GB</div><div class="summary-label">Total</div></div><div class="summary-item"><div class="summary-num">10</div><div class="summary-label">Entries</div></div></div>
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
      <!-- 任务抽屉 -->
      <section class="drawer" :class="{ collapsed: drawerCollapsed }" v-if="state.task">
        <header class="drawer-head">
          <div class="drawer-left">
            <span class="drawer-dot" :class="state.task.phase"></span>
            <span class="drawer-label">{{ state.task.label }}</span>
            <span class="drawer-cmd">$ {{ state.task.cli }}</span>
          </div>
          <div class="drawer-right">
            <span class="drawer-status">{{ state.task.phase === 'running' ? '进行中…' : state.task.phase === 'success' ? '成功 ✓' : '失败' }}</span>
            <button class="icon-btn" @click="drawerCollapsed = !drawerCollapsed">{{ drawerCollapsed ? '▲' : '▼' }}</button>
          </div>
        </header>
        <pre class="drawer-log">{{ state.task.lines.map(l => l.t).join('\n') }}</pre>
      </section>
    </main>
    <div class="toast-root">
      <div v-for="x in toasts" :key="x.id" class="toast" :class="x.kind">{{ x.msg }}</div>
    </div>
  </div>
</template>
