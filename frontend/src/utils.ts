// 跨模块共享工具（格式化与剪贴板降级）
import { toastBus } from './state'
import { t } from './i18n'

// 版本降序比较器（cmpVer 语义对齐原型：8.4 > 8.2 > 8.0 > 7.4；非数字 tag 如 alpine 沉底）
export function cmpVerDesc(a: string, b: string): number {
  const na = a.match(/\d+(\.\d+)*/)
  const nb = b.match(/\d+(\.\d+)*/)
  if (na && nb) {
    const pa = na[0].split('.').map(Number), pb = nb[0].split('.').map(Number)
    for (let i = 0; i < Math.max(pa.length, pb.length); i++) {
      const d = (pb[i] ?? 0) - (pa[i] ?? 0)
      if (d) return d
    }
    return 0
  }
  if (na) return -1
  if (nb) return 1
  return a.localeCompare(b)
}

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
