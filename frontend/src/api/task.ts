// 任务链路桥接：Wails 环境订阅 Go Runner 事件并调用 spawn 绑定；
// 浏览器环境（vite dev 纯预览）降级为模拟步骤（规约 §13 显式降级）。
import { Events } from '@wailsio/runtime'
import { RunTask } from '../../bindings/github.com/xiaokentrl/phpbox-desktop/internal/bindings/runner'
import { state, startTask, pushTaskLine, finishTask, taskRunning, runTask } from '../state'
import type { Step, TaskLine } from '../state'

// 桌面环境探测：@wailsio/runtime 在任何环境都会自建 window._wails（invoke/clientId），
// 只有原生 Wails 窗口才由 Go 侧注入 _wails.flags（runtime.Core，WindowLoadFinished 时）。
// 因此必须惰性探测（函数），且启动期调用要等 wails:runtime-config-ready 事件。
export function inWails(): boolean {
  return typeof window !== 'undefined'
    && !!(window as any)._wails
    && !!(window as any)._wails.flags
}

// 启动期回调：桌面环境等 Core 注入完成后再执行；浏览器环境事件永不触发（§13 降级：不执行）
export function onWailsReady(cb: () => void): void {
  if (inWails()) { cb(); return }
  window.addEventListener('wails:runtime-config-ready', () => {
    if (inWails()) cb()
  }, { once: true })
}

export interface TaskOptions { doneMsg?: string; onDone?: () => void; fallback?: Step[] }

// task:done 回调需要拿到发起方的 doneMsg/onDone —— 模块级登记（单队列，同时最多一个任务）
let taskDoneOpts: TaskOptions = {}

// Go 侧 TaskEvent{Line, Cls}：Cls 为 'ok'|'err'|''（runner.go classify）
function evLine(data: any): TaskLine {
  const cls = data?.cls
  const c: TaskLine['c'] = cls === 'ok' || cls === 'err' ? cls : cls === '' ? '' : 'dim'
  return { t: String(data?.line ?? ''), c }
}

let subscribed = false
export function initTaskEvents() {
  if (subscribed || !inWails()) return
  subscribed = true
  Events.On('task:log', (ev) => {
    const l = evLine(ev.data)
    pushTaskLine(l.t, l.c)
  })
  Events.On('task:done', (ev) => {
    const task = state.task
    if (!task || task.phase !== 'running') return
    const result = String((ev.data as any)?.line ?? '')
    if (result === 'success') finishTask('success', taskDoneOpts)
    else if (result === 'failed') finishTask('failed')
    // 未知结果：保持 running，由 RunTask promise 的 then/catch 兜底
  })
}

// 发起任务：Wails 内走真实 spawn，浏览器降级为模拟步骤
export function dispatchTask(label: string, cli: string, args: string[], opts: TaskOptions = {}): boolean {
  if (taskRunning()) return false
  initTaskEvents()

  if (!inWails()) {
    return runTask(label, cli, opts.fallback ?? [], { doneMsg: opts.doneMsg, onDone: opts.onDone })
  }

  if (!startTask(label, cli)) return false
  taskDoneOpts = { doneMsg: opts.doneMsg, onDone: opts.onDone }
  RunTask(args)
    .then(() => { if (taskRunning()) finishTask('success', taskDoneOpts) })
    .catch((e: unknown) => {
      if (!taskRunning()) return
      pushTaskLine('[ERR] ' + String(e), 'err')
      finishTask('failed')
    })
  return true
}
