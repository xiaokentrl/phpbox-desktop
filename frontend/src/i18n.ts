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
  'common.delete': '删除', 'common.cancel': '取消', 'common.save': '保存修改', 'common.copy': '复制路径',
  'toast.copied': '已复制', 'toast.switching': '切换中…', 'toast.demoRollback': '演示：切换失败已回滚',
  'task.running': '进行中…', 'task.success': '成功 ✓', 'task.failed': '失败',
  'task.close': '关闭', 'task.busy': '已有任务进行中，请稍候',
  'task.done': '任务完成', 'task.doneOf': '任务完成',
  'task.diag.label': '环境诊断', 'task.diag.done': '诊断完成',
  'bk.title': '备份恢复', 'bk.sub': '打包 .env、config/、离线缓存与数据卷 · 归档保存在 ~/phpbox/backups/',
  'bk.now': '立即备份', 'bk.warn': '备份前提示：', 'bk.warnBody': '将暂停 MySQL / PostgreSQL / Redis 服务，备份完成后自动重启。Go 镜像不包含在备份内。',
  'th.archive': '归档文件', 'th.size': '大小', 'th.time': '时间',
  'bk.restore': '恢复', 'bk.download': '下载', 'bk.delete': '删除',
  'bk.restoreTitle': '恢复备份', 'bk.restoreDesc': '将覆盖归档中记录的所有路径，服务会停止后恢复。',
  'bk.warn.stopSvc': '停止 MySQL / Redis 容器以恢复数据',
  'bk.warn.overwrite': '覆盖 .env、config/、离线缓存、站点目录与数据目录',
  'bk.warn.reload': '恢复后自动重载 Nginx（若在运行）',
  'bk.restoreCheck': '我了解当前环境会被归档内容覆盖，且无法撤销',
  'bk.confirmFile': '请输入文件名以确认：', 'bk.confirmFilePh': '粘贴或输入归档文件名',
  'bk.warn.noImages': 'Docker 镜像不包含在备份内', 'bk.warn.goReinstall': '恢复后需重新执行 phpbox go install',
  'bk.confirmRestore': '确认恢复',
  'bk.deleteTitle': '删除备份归档', 'bk.deleteDesc': '删除后无法恢复该归档',
  'bk.warn.deleteFile': '从磁盘删除该 .tar.gz 文件',
  'bk.warn.envKeep': '当前运行环境不受影响',
  'bk.deleteCheck': '我了解该归档会被永久删除',
  'bk.confirmDelete': '删除归档',
  'bk.empty': '还没有备份归档', 'bk.emptyDesc': '点击「立即备份」创建第一份完整环境快照。',
  'bk.task.now': '创建备份', 'bk.task.restore': '恢复备份', 'bk.task.delete': '删除归档',
  'bk.done.now': '备份完成', 'bk.done.restore': '恢复完成', 'bk.done.delete': '归档已删除',
  'bk.loadErr': '备份列表加载失败',
  'svc.hint.php': '安装一个 PHP 版本，就能开始创建站点。', 'svc.hint.mysql': '安装 MySQL，为你的应用准备数据库。',
  'svc.hint.pgsql': '安装 PostgreSQL，支持 pgvector 等扩展。', 'svc.hint.redis': '安装 Redis，为应用提供缓存与会话存储。',
  'svc.hint.nginx': '安装 Nginx，站点才能通过域名访问。',
  'svc.empty.title': '还没有安装 {name}', 'svc.installed': '已安装',
  'svc.card.port': '端口', 'svc.card.exts': '扩展', 'svc.card.extsCount': '{n} 个', 'svc.card.password': '密码',
  'svc.card.config': '配置', 'svc.card.configPath': 'config/{kind}/{ver}/',
  'svc.card.running': '运行中', 'svc.card.cmd': '命令',
  'mod.install': '安装', 'mod.cancel': '取消', 'mod.version': '版本', 'mod.willRun': '将执行',
  'mod.err.version': '请输入或选择版本',
  'mod.installDesc': '选择版本，将自动生成配置并启动容器',
  'mod.uninstallTitle': '卸载 {name} {ver}', 'mod.uninstallDesc': '会停止并移除该版本的服务容器与配置。',
  'mod.warn.container': '移除容器 phpbox-{kind}-{ver}',
  'mod.warn.config': '移除 config/{kind}/{ver}/',
  'mod.warn.dataKeep': '数据目录默认保留', 'mod.warn.offlineKeep': '离线缓存默认保留',
  'mod.warn.sourceKeep': '站点源码不会被删除',
  'mod.uninstallCheck': '我了解卸载后需重新执行 phpbox {kind} install {ver} 才能恢复服务',
  'mod.confirmInput': '请输入版本号以确认：', 'mod.confirmPlaceholder': '输入 {ver}',
  'mod.confirmUninstall': '确认卸载',
  'mod.purge': '同时删除数据目录（--purge）——', 'mod.purgeIrreversible': '数据不可恢复',
  'mod.warn.uninstallCmd': '卸载将停止并移除 Nginx 容器',
  'mod.warn.configKeep': '站点配置保留于 config/nginx/',
  'mod.uninstallNginxCheck': '我了解卸载后需重新执行 phpbox nginx install 才能恢复服务',
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
  'common.delete': 'Delete', 'common.cancel': 'Cancel', 'common.save': 'Save Changes', 'common.copy': 'Copy path',
  'toast.copied': 'Copied', 'toast.switching': 'Switching…', 'toast.demoRollback': 'Demo: switch failed, rolled back',
  'task.running': 'Running…', 'task.success': 'Success ✓', 'task.failed': 'Failed',
  'task.close': 'Close', 'task.busy': 'A task is already running, please wait',
  'task.done': 'Task complete', 'task.doneOf': 'Task complete',
  'task.diag.label': 'Diagnostics', 'task.diag.done': 'Diagnostics complete',
  'bk.title': 'Backups', 'bk.sub': 'Archives .env, config/, offline cache and volumes · stored in ~/phpbox/backups/',
  'bk.now': 'Backup now', 'bk.warn': 'Before backing up:', 'bk.warnBody': 'MySQL / PostgreSQL / Redis will be paused and restarted afterwards. Go images are not included.',
  'th.archive': 'Archive', 'th.size': 'Size', 'th.time': 'Time',
  'bk.restore': 'Restore', 'bk.download': 'Download', 'bk.delete': 'Delete',
  'bk.restoreTitle': 'Restore backup', 'bk.restoreDesc': 'Overwrites all paths stored in the archive; services are stopped then restored.',
  'bk.warn.stopSvc': 'Stops MySQL / Redis containers to restore data',
  'bk.warn.overwrite': 'Overwrites .env, config/, offline cache, site dirs and data dirs',
  'bk.warn.reload': 'Nginx is reloaded automatically afterwards (if running)',
  'bk.restoreCheck': 'I understand the current environment is overwritten and this cannot be undone',
  'bk.confirmFile': 'Type the file name to confirm:', 'bk.confirmFilePh': 'Paste or type the archive file name',
  'bk.warn.noImages': 'Docker images are not included in the archive', 'bk.warn.goReinstall': 'Run phpbox go install again after restoring',
  'bk.confirmRestore': 'Restore',
  'bk.deleteTitle': 'Delete backup archive', 'bk.deleteDesc': 'This archive cannot be recovered once deleted',
  'bk.warn.deleteFile': 'Removes the .tar.gz file from disk',
  'bk.warn.envKeep': 'The running environment is not affected',
  'bk.deleteCheck': 'I understand this archive is permanently deleted',
  'bk.confirmDelete': 'Delete archive',
  'bk.empty': 'No backups yet', 'bk.emptyDesc': 'Click "Backup now" to create the first full environment snapshot.',
  'bk.task.now': 'Create backup', 'bk.task.restore': 'Restore backup', 'bk.task.delete': 'Delete archive',
  'bk.done.now': 'Backup complete', 'bk.done.restore': 'Restore complete', 'bk.done.delete': 'Archive deleted',
  'bk.loadErr': 'Failed to load backup list',
  'svc.hint.php': 'Install a PHP version to start creating sites.', 'svc.hint.mysql': 'Install MySQL for your applications.',
  'svc.hint.pgsql': 'Install PostgreSQL, with pgvector and more.', 'svc.hint.redis': 'Install Redis for caching and sessions.',
  'svc.hint.nginx': 'Install Nginx so sites are reachable by domain.',
  'svc.empty.title': '{name} not installed yet', 'svc.installed': 'Installed',
  'svc.card.port': 'Port', 'svc.card.exts': 'Extensions', 'svc.card.extsCount': '{n}', 'svc.card.password': 'Password',
  'svc.card.config': 'Config', 'svc.card.configPath': 'config/{kind}/{ver}/',
  'svc.card.running': 'Running', 'svc.card.cmd': 'Command',
  'mod.install': 'Install', 'mod.cancel': 'Cancel', 'mod.version': 'Version', 'mod.willRun': 'Will run',
  'mod.err.version': 'Enter or pick a version',
  'mod.installDesc': 'Pick a version; config is generated and the container started automatically',
  'mod.uninstallTitle': 'Uninstall {name} {ver}', 'mod.uninstallDesc': 'Stops and removes the service container and its config.',
  'mod.warn.container': 'Removes container phpbox-{kind}-{ver}',
  'mod.warn.config': 'Removes config/{kind}/{ver}/',
  'mod.warn.dataKeep': 'Data directory is kept by default', 'mod.warn.offlineKeep': 'Offline cache is kept by default',
  'mod.warn.sourceKeep': 'Site source code is NOT deleted',
  'mod.uninstallCheck': 'I understand reinstalling requires phpbox {kind} install {ver}',
  'mod.confirmInput': 'Type the version to confirm:', 'mod.confirmPlaceholder': 'Type {ver}',
  'mod.confirmUninstall': 'Uninstall',
  'mod.purge': 'Also delete the data directory (--purge) — ', 'mod.purgeIrreversible': 'irreversible',
  'mod.warn.uninstallCmd': 'Stops and removes the Nginx container',
  'mod.warn.configKeep': 'Site configs are kept in config/nginx/',
  'mod.uninstallNginxCheck': 'I understand reinstalling requires phpbox nginx install',
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

export function t(key: string, vars?: Record<string, string | number>): string {
  let s = DICTS[locale.value][key] ?? ZH[key] ?? key
  if (vars) for (const [k, v] of Object.entries(vars)) s = s.split(`{${k}}`).join(String(v))
  return s
}

export function setAppLocale(l: Locale) { setLocale(l) }
export function setLocale(l: Locale) {
  locale.value = l
  try {
    localStorage.setItem('phpbox-locale', l)
    document.documentElement.lang = l
  } catch { /* 隐私模式下忽略 */ }
}
