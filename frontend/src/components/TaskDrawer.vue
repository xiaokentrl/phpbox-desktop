<script setup lang="ts">
// 任务抽屉：真实 phpbox CLI 输出流（task:log/task:done 事件驱动），完成态可关闭、按行着色、自动滚底。
import { computed, nextTick, ref, watch } from 'vue'
import { t } from '../i18n'
import { state, clearTask } from '../state'

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
</script>

<template>
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
</template>
