// 轻量 i18n（阶段 0）：词典 + t()；Vue 前端全量化时迁移 vue-i18n（规约 §11）
type Dict = Record<string, string>

const ZH: Dict = {
  'nav.business': '业务', 'nav.services': '服务', 'nav.ops': '运维',
  'nav.sites': '站点', 'nav.php': 'PHP', 'nav.mysql': 'MySQL', 'nav.pgsql': 'PostgreSQL',
  'nav.redis': 'Redis', 'nav.nginx': 'Nginx', 'nav.go': 'Go',
  'nav.backup': '备份恢复', 'nav.offline': '离线缓存', 'nav.settings': '设置', 'nav.overview': '总览',
  'brand.connected': '引擎已连接',
  'foot.palette': '命令面板', 'foot.sync': '同步状态', 'foot.theme': '切换主题',
  'sites.title': '站点',
  'sites.sub': '点击域名打开 · 行内切换 PHP 版本',
  'sites.add': '新建站点',
  'sites.empty.title': '还没有站点',
  'sites.empty.desc': '创建第一个站点，就能用浏览器访问本地项目。',
  'sum.sites': '站点', 'sum.php': 'PHP 版本', 'sum.healthy': '健康', 'sum.port': 'Nginx 端口',
  'th.domain': '域名', 'th.php': 'PHP 版本', 'th.root': '根目录', 'th.health': '健康', 'th.hosts': 'HOSTS',
  'hosts.add': '加 hosts', 'hosts.parsed': '已解析',
  'health.up': '正常', 'health.warn': '降级', 'health.down': '未响应',
  'common.delete': '删除', 'common.cancel': '取消', 'common.save': '保存修改',
  'toast.copied': '已复制', 'toast.switching': '切换中…',
  'service.port': '端口', 'service.password': '密码', 'service.config': '配置', 'service.container': '容器',
  'service.running': '运行中', 'service.exts': '扩展',
  'btn.exts': '扩展', 'btn.uninstall': '卸载', 'btn.reveal': '显示密码', 'btn.copyDsn': '复制连接',
  'btn.reload': '重载配置', 'btn.refresh': '刷新',
  'svc.php.sub': '多版本 PHP-FPM 运行时', 'svc.mysql.sub': '关系型数据库服务',
  'svc.pgsql.sub': '对象关系型数据库', 'svc.redis.sub': '内存缓存与数据结构服务',
  'svc.nginx.sub': '反向代理与站点入口',
  'svc.install': '安装',
  'overview.title': '总览', 'overview.sub': '所有服务实例一览 · 数据来自 Docker Engine API',
  'overview.diag': '环境诊断', 'ov.instances': '服务实例', 'ov.running': '运行中',
  'th.image': '镜像', 'th.state': '状态',
}
const EN: Dict = {
  'nav.business': 'Business', 'nav.services': 'Services', 'nav.ops': 'Operations',
  'nav.sites': 'Sites', 'nav.php': 'PHP', 'nav.mysql': 'MySQL', 'nav.pgsql': 'PostgreSQL',
  'nav.redis': 'Redis', 'nav.nginx': 'Nginx', 'nav.go': 'Go',
  'nav.backup': 'Backups', 'nav.offline': 'Offline Cache', 'nav.settings': 'Settings', 'nav.overview': 'Overview',
  'brand.connected': 'Engine connected',
  'foot.palette': 'Command Palette', 'foot.sync': 'Sync State', 'foot.theme': 'Theme',
  'sites.title': 'Sites',
  'sites.sub': 'Click a domain to open · switch PHP version inline',
  'sites.add': 'New Site',
  'sites.empty.title': 'No sites yet',
  'sites.empty.desc': 'Create your first site to reach local projects from the browser.',
  'sum.sites': 'Sites', 'sum.php': 'PHP versions', 'sum.healthy': 'Healthy', 'sum.port': 'Nginx port',
  'th.domain': 'Domain', 'th.php': 'PHP version', 'th.root': 'Root', 'th.health': 'Health', 'th.hosts': 'HOSTS',
  'hosts.add': 'Add hosts', 'hosts.parsed': 'Resolved',
  'health.up': 'Up', 'health.warn': 'Degraded', 'health.down': 'Down',
  'common.delete': 'Delete', 'common.cancel': 'Cancel', 'common.save': 'Save Changes',
  'toast.copied': 'Copied', 'toast.switching': 'Switching…',
  'service.port': 'Port', 'service.password': 'Password', 'service.config': 'Config', 'service.container': 'Container',
  'service.running': 'Running', 'service.exts': 'Extensions',
  'btn.exts': 'Extensions', 'btn.uninstall': 'Uninstall', 'btn.reveal': 'Reveal password', 'btn.copyDsn': 'Copy DSN',
  'btn.reload': 'Reload config', 'btn.refresh': 'Refresh',
  'svc.php.sub': 'Multi-version PHP-FPM runtime', 'svc.mysql.sub': 'Relational database service',
  'svc.pgsql.sub': 'Object-relational database', 'svc.redis.sub': 'In-memory cache & data structures',
  'svc.nginx.sub': 'Reverse proxy & site entry',
  'svc.install': 'Install',
  'overview.title': 'Overview', 'overview.sub': 'All service instances · live from the Docker Engine API',
  'overview.diag': 'Diagnostics', 'ov.instances': 'Instances', 'ov.running': 'Running',
  'th.image': 'Image', 'th.state': 'State',
}
const DICTS: Record<string, Dict> = { 'zh-CN': ZH, 'en-US': EN }

export type Locale = 'zh-CN' | 'en-US'
export const LOCALE_NAMES: Record<Locale, string> = { 'zh-CN': '简体中文', 'en-US': 'English (US)' }

function detect(): Locale {
  try {
    const saved = localStorage.getItem('phpbox-locale')
    if (saved === 'zh-CN' || saved === 'en-US') return saved
  } catch { /* 隐私模式 */ }
  if ((navigator.language || '').startsWith('en')) return 'en-US'
  return 'zh-CN'
}

import { ref } from 'vue'
export const locale = ref(detect() as Locale)

export function t(key: string): string {
  return DICTS[locale.value][key] ?? ZH[key] ?? key
}

export function setAppLocale(l: Locale) { setLocale(l) }
export function setLocale(l: Locale) {
  locale.value = l
  try {
    localStorage.setItem('phpbox-locale', l)
    document.documentElement.lang = l
  } catch { /* 隐私模式下忽略 */ }
}
