<script setup lang="ts">
// 服务线通用底座：版本卡片网格（真实容器状态）+ 安装弹窗触发 + 卸载三条件危险确认。
// php/mysql/pgsql/redis/nginx 五个模块复用；主按钮差异点经 #primary slot 注入（PHP=扩展，默认=命令复制）。
import { computed } from 'vue'
import { t } from '../i18n'
import { state, openInstall, openDanger, taskRunning } from '../state'
import { dispatchTask } from '../api/task'
import { loadContainers } from '../api/data'
import { copyCmd } from '../utils'
import type { ServiceMeta } from './service-base'

const props = defineProps<{ meta: ServiceMeta }>()

const versions = computed(() => (state.installed as Record<string, string[]>)[props.meta.svc] ?? [])

// 容器名规则（get_container_name）：php84 / mysql84 / redis8 / pg17 / nginx（前缀+去点版本）
function containerNameFor(svc: string, ver: string): string {
  if (svc === 'nginx') return 'nginx'
  if (svc === 'pgsql') return `pg${ver.replace(/\./g, '')}`
  return `${svc}${ver.replace(/\./g, '')}`
}
function containerStateFor(ver: string): string {
  const want = containerNameFor(props.meta.svc, ver)
  const c = state.containers.find(x => x.Name.replace(/^\//, '') === want)
  return c?.State ?? ''
}

function openInstallModal() {
  openInstall({ svc: props.meta.svc, title: t('nav.' + props.meta.svc), suggested: props.meta.suggested, single: props.meta.single })
}

function openUninstallModal(ver: string) {
  const svc = props.meta.svc
  const isNginx = svc === 'nginx'
  openDanger({
    title: t('mod.uninstallTitle', { name: t('nav.' + svc), ver }),
    description: t('mod.uninstallDesc'),
    warnings: isNginx ? [
      { text: t('mod.warn.uninstallCmd') },
      { text: t('mod.warn.configKeep'), keep: true },
    ] : [
      { text: t('mod.warn.container', { kind: svc, ver }) },
      { text: t('mod.warn.config', { kind: svc, ver }) },
      { text: t('mod.warn.dataKeep'), keep: true },
      { text: t('mod.warn.offlineKeep'), keep: true },
    ],
    checkboxLabel: isNginx ? t('mod.uninstallNginxCheck') : t('mod.uninstallCheck', { kind: svc, ver }),
    inputLabel: t('mod.confirmInput'),
    expect: ver,
    placeholder: t('mod.confirmPlaceholder', { ver }),
    cliPreview: isNginx ? 'phpbox nginx uninstall' : `phpbox ${svc} uninstall ${ver}`,
    confirmLabel: t('mod.confirmUninstall'),
    purge: isNginx ? undefined : { label: t('mod.purge') },
    onConfirm: (purge) => {
      const args = isNginx ? ['nginx', 'uninstall'] : [svc, 'uninstall', ver, ...(purge ? ['--purge'] : [])]
      const label = `${t('mod.confirmUninstall')}·${t('nav.' + svc)} ${ver}`
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
</script>

<template>
  <div>
    <header class="view-header">
      <div><h1>{{ t('nav.' + meta.svc) }}</h1><p class="view-sub">{{ t('svc.' + meta.svc + '.sub') }}</p></div>
      <div class="header-actions">
        <button class="btn btn-primary" :disabled="taskRunning()" @click="openInstallModal()">+ {{ t('svc.install') }} {{ t('nav.' + meta.svc) }}</button>
      </div>
    </header>
    <div v-if="versions.length === 0" class="empty">
      <div class="empty-icon">{{ meta.icon }}</div>
      <h2>{{ t('svc.empty.title', { name: t('nav.' + meta.svc) }) }}</h2>
      <p>{{ t('svc.hint.' + meta.svc) }}</p>
      <button class="btn btn-primary" @click="openInstallModal()">{{ t('svc.install') }} {{ t('nav.' + meta.svc) }}</button>
    </div>
    <div v-else class="grid grid-3">
      <article v-for="v in versions" :key="v" class="card version-card">
        <div class="version-card-head">
          <span class="version-tag">{{ v }}</span>
          <span class="status-pill" :class="containerStateFor(v) === 'running' ? 'pill-ok' : 'pill-off'">
            <span class="pill-dot"></span>{{ containerStateFor(v) === 'running' ? t('svc.card.running') : t('health.down') }}
          </span>
        </div>
        <div>
          <div v-if="meta.portMap?.[v]" class="kv"><span class="k">{{ t('svc.card.port') }}</span><span class="v">{{ meta.portMap[v] }}</span></div>
          <div v-if="meta.hasPassword" class="kv"><span class="k">{{ t('svc.card.password') }}</span><span class="v">••••••••</span></div>
          <div class="kv"><span class="k">{{ t('svc.card.config') }}</span><span class="v">{{ t('svc.card.configPath', { kind: meta.svc, ver: v }) }}</span></div>
        </div>
        <div class="version-card-foot">
          <slot name="primary" :ver="v">
            <button class="btn btn-sm" @click="copyCmd(`phpbox ${meta.svc} list`)">{{ t('svc.card.cmd') }}</button>
          </slot>
          <button class="btn btn-sm btn-danger" @click="openUninstallModal(v)">{{ t('btn.uninstall') }}</button>
        </div>
      </article>
    </div>
  </div>
</template>
