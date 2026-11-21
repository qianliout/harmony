// 日期展示：记录日期固定为 YYYY-MM-DD，只精确到日。

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

export function pad(value: number): string {
  return String(value).padStart(2, '0')
}

export function todayString(): string {
  const now = new Date()
  return `${now.getFullYear()}-${pad(now.getMonth() + 1)}-${pad(now.getDate())}`
}

/** 今年省略年份：8月20日；往年带年份：2011年8月20日。 */
export function formatDate(value: string): string {
  const parts = parseDateParts(value)
  if (!parts) {
    return ''
  }
  const yearPrefix = parts.year === new Date().getFullYear() ? '' : `${parts.year}年`
  return `${yearPrefix}${parts.month}月${parts.day}日`
}

export function formatFullDate(value: string): string {
  const parts = parseDateParts(value)
  if (!parts) {
    return ''
  }
  return `${parts.year}年${parts.month}月${parts.day}日`
}

export function relativeSince(value: string): string {
  const parts = parseDateParts(value)
  if (!parts) {
    return ''
  }
  const now = new Date()
  const months = (now.getFullYear() - parts.year) * 12 + (now.getMonth() + 1 - parts.month)
  if (months < 1) {
    return '本月'
  }
  if (months < 12) {
    return `${months} 个月前`
  }
  return `${Math.floor(months / 12)} 年前`
}
