// 跨模块共享工具（格式化与剪贴板降级）
import { toastBus } from './state'
import { t } from './i18n'

export function fmtSize(n: number): string {
  if (n >= 1024 ** 3) return (n / 1024 ** 3).toFixed(1) + ' GB'
  if (n >= 1024 ** 2) return (n / 1024 ** 2).toFixed(0) + ' MB'
  if (n >= 1024) return (n / 1024).toFixed(0) + ' KB'
  return n + ' B'
}

export function fmtTime(iso: string): string {
  const d = new Date(iso)
  return isNaN(d.getTime()) ? iso : d.toLocaleString()
}

export function copyCmd(cmd: string) {
  if (navigator.clipboard?.writeText) {
    navigator.clipboard.writeText(cmd).then(() => toastBus(t('toast.copied'), 'ok'), () => toastBus(cmd, 'info', 4000))
  } else toastBus(cmd, 'info', 4000) // 非 https/localhost 环境降级为展示
}
