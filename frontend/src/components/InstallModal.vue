<script setup lang="ts">
// 安装弹窗（五服务线复用）：版本快选 + 实时命令预览；nginx 单实例无版本参数。
import { computed, ref, watch } from 'vue'
import { t } from '../i18n'
import { state, closeModal, toastBus } from '../state'
import { dispatchTask } from '../api/task'
import { loadContainers } from '../api/data'

const installVer = ref('')
const installModal = computed(() => state.modal?.kind === 'install' ? state.modal : null)
watch(() => state.modal?.kind, () => { // 打开安装弹窗时重置输入（单实例预填首个建议版本）
  const m = installModal.value
  if (m) installVer.value = m.single ? (m.suggested[0] ?? '') : ''
})
function pickInstallVer(v: string) { installVer.value = v }
const installCmdPreview = computed(() => {
  const m = installModal.value
  if (!m) return ''
  const v = installVer.value.trim()
  // nginx install 不带版本（版本由 .env 的 NGINX_VERSION 决定）；其余线 install <版本>
  return m.svc === 'nginx' ? 'phpbox nginx install' : v ? `phpbox ${m.svc} install ${v}` : `phpbox ${m.svc} install <版本>`
})
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
      if (v && !((state.installed as Record<string, string[]>)[m.svc] ?? []).includes(v)) {
        ((state.installed as Record<string, string[]>)[m.svc] ??= []).push(v)
      }
      loadContainers()
    },
  })
}
</script>

<template>
  <div v-if="installModal" class="modal">
    <div class="modal-head"><h3>{{ t('svc.install') }} {{ installModal.title }}</h3><p>{{ t('mod.installDesc') }}</p></div>
    <div class="modal-body">
      <div class="field">
        <label>{{ t('mod.version') }}</label>
        <input type="text" v-model="installVer" :placeholder="installModal.suggested[0] || ''" :disabled="installModal.single" autocomplete="off">
        <div class="quick-picks">
          <button v-for="v in installModal.suggested" :key="v" class="pick" :class="{ selected: installVer === v }" @click="pickInstallVer(v)">{{ v }}</button>
        </div>
      </div>
      <div class="field"><label>{{ t('mod.willRun') }}</label>
        <div class="cmd-preview"><span class="prompt">$ </span>{{ installCmdPreview }}</div>
      </div>
    </div>
    <div class="modal-foot">
      <button class="btn" @click="closeModal()">{{ t('mod.cancel') }}</button>
      <button class="btn btn-primary" @click="confirmInstall()">{{ t('mod.install') }}</button>
    </div>
  </div>
</template>
