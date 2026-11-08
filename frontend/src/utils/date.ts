// 日期展示与精度：精度不是 day 的日期按该月/该年第一天存储，展示时带“约”。
import type { PaperDraft } from '@/api/types'

const WEEKDAYS = ['周日', '周一', '周二', '周三', '周四', '周五', '周六']

export function parseDateParts(value: string): { year: number; month: number; day: number } | null {
  if (!value) {
    return null
  }
  const [y, m, d] = value.split('-').map(Number)
  if (!y || !m || !d) {
    return null
  }
  return { year: y, month: m, day: d }
}

export function todayString(): string {
  const now = new Date()
  return `${now.getFullYear()}-${pad(now.getMonth() + 1)}-${pad(now.getDate())}`
}

export function pad(value: number): string {
  return String(value).padStart(2, '0')
}

export function formatDate(value: string, precision = 'day'): string {
  const parts = parseDateParts(value)
  if (!parts) {
    return ''
  }
  const currentYear = new Date().getFullYear()
  const yearPrefix = parts.year === currentYear ? '' : `${parts.year}年`
  if (precision === 'year') {
    return `${parts.year}年`
  }
  if (precision === 'month') {
    return `${yearPrefix}${parts.month}月`
  }
  return `${yearPrefix}${parts.month}月${parts.day}日`
}

/** 精度不是 day 时加“约”。 */
export function preciseDateLabel(value: string, precision: string): string {
  const label = formatDate(value, precision)
  if (!label) {
    return ''
  }
  return precision === 'day' ? label : `${label} 约`
}

export function formatGroupTitle(value: string): string {
  const parts = parseDateParts(value)
  if (!parts) {
    return ''
  }
  return `${parts.year} · ${parts.month}月${parts.day}日`
}

export function formatFullDate(value: string): string {
  const parts = parseDateParts(value)
  if (!parts) {
    return ''
  }
  return `${parts.year}年${parts.month}月${parts.day}日`
}

export function weekdayOf(value: string): string {
  const parts = parseDateParts(value)
  if (!parts) {
    return ''
  }
  return WEEKDAYS[new Date(parts.year, parts.month - 1, parts.day).getDay()]
}

export function relativeSince(value: string): string {
  const parts = parseDateParts(value)
  if (!parts) {
    return ''
  }
  const now = new Date()
  const then = new Date(parts.year, parts.month - 1, parts.day)
  const months = (now.getFullYear() - then.getFullYear()) * 12 + (now.getMonth() - then.getMonth())
  if (months < 1) {
    return '本月'
  }
  if (months < 12) {
    return `${months} 个月前`
  }
  const years = Math.floor(months / 12)
  return `${years} 年前`
}

export function yearOf(value: string): number {
  return parseDateParts(value)?.year ?? 0
}

/** 把解析草稿的时间转成表单用的年/月/日三件套。 */
export function draftDateParts(draft: PaperDraft | null): { year: number; month: number; day: number } {
  const parts = parseDateParts(draft?.occurredOn ?? '')
  return { year: parts?.year ?? new Date().getFullYear(), month: parts?.month ?? 0, day: parts?.day ?? 0 }
}
