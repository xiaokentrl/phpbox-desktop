<script setup lang="ts">
// 阶段 0 POC 视图：容器列表（真实数据经 Wails 绑定）
// 由谁触发：应用启动自动加载 / 刷新按钮
// 干什么：经绑定层调用引擎 ListContainers
// 成功后：表格展示容器名/镜像/状态；失败显示错误与重试
import { ref, onMounted } from 'vue'
import { ListContainers } from '../bindings/github.com/xiaokentrl/phpbox-desktop/internal/bindings/docker'
import type { ContainerSummary } from '../bindings/github.com/xiaokentrl/phpbox-desktop/internal/engine/docker/models'

const items = ref<ContainerSummary[]>([])
const error = ref('')
const loading = ref(true)

async function load() {
  loading.value = true
  error.value = ''
  try {
    items.value = (await ListContainers()) ?? []
  } catch (e) {
    error.value = String(e)
  } finally {
    loading.value = false
  }
}

onMounted(load)
</script>

<template>
  <main class="app">
    <header class="head">
      <h1>phpbox Desktop</h1>
      <button class="btn" :disabled="loading" @click="load">刷新</button>
    </header>

    <p v-if="error" class="err">[ERR] {{ error }}</p>
    <p v-else-if="loading" class="dim">加载中…</p>

    <table v-else class="tbl">
      <thead>
        <tr><th>容器</th><th>镜像</th><th>状态</th></tr>
      </thead>
      <tbody>
        <tr v-for="c in items" :key="c.Name">
          <td class="mono">{{ c.Name }}</td>
          <td class="mono dim">{{ c.Image }}</td>
          <td>
            <span class="pill" :class="c.State === 'running' ? 'ok' : 'off'">
              <span class="pdot" />{{ c.State }}
            </span>
          </td>
        </tr>
      </tbody>
    </table>
  </main>
</template>

<style scoped>
.app { max-width: 860px; margin: 0 auto; padding: 32px 28px; }
.head { display: flex; align-items: center; justify-content: space-between; margin-bottom: 20px; }
h1 { font-size: 22px; font-weight: 600; letter-spacing: -.3px; }
.btn { padding: 7px 16px; border-radius: 8px; border: 1px solid var(--border); background: var(--surface-2); color: var(--text); font-size: 13px; }
.btn:hover:not(:disabled) { background: var(--surface-3); }
.tbl { width: 100%; border-collapse: collapse; font-size: 13.5px; }
.tbl th { text-align: left; padding: 9px 14px; font-size: 11px; text-transform: uppercase; letter-spacing: .6px; color: var(--text-mute); background: var(--surface-2); }
.tbl td { padding: 10px 14px; border-bottom: 1px solid var(--border-2); }
.pill { display: inline-flex; align-items: center; gap: 5px; padding: 2px 9px; border-radius: 20px; font-size: 11.5px; }
.pill.ok { background: var(--ok-bg); color: var(--ok); }
.pill.off { background: var(--surface-2); color: var(--text-mute); }
.pdot { width: 5px; height: 5px; border-radius: 50%; background: currentColor; }
.mono { font-family: var(--mono); font-size: 12.5px; }
.dim { color: var(--text-dim); }
.err { color: var(--danger); }
</style>
