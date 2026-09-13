<script setup lang="ts">
// 危险确认弹窗（§8.2 三条件）：警告清单 + 认知勾选 + 输入匹配解锁；可选 --purge 勾选。
// 被 卸载服务/恢复备份/删备份/删站点/清离线缓存 5 个模块复用，状态全部来自 state.modal 单例。
import { computed, ref, watch } from 'vue'
import { t } from '../i18n'
import { state, closeModal } from '../state'

const dcInput = ref(''), dcChecked = ref(false), dcPurge = ref(false)
const dcModal = computed(() => state.modal?.kind === 'danger' ? state.modal : null)
const dcMatched = computed(() => dcModal.value ? dcInput.value.trim() === dcModal.value.expect : false)
const dcReady = computed(() => !!(dcMatched.value && dcChecked.value))
watch(() => state.modal?.kind, (k) => {
  if (k === 'danger') { dcInput.value = ''; dcChecked.value = false; dcPurge.value = false }
})
function dcConfirm() {
  const m = dcModal.value
  if (!m || !dcReady.value) return
  closeModal()
  m.onConfirm(dcPurge.value)
}
</script>

<template>
  <div v-if="dcModal" class="modal">
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
</template>
