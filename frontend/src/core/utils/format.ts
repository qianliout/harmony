import dayjs from 'dayjs'

/** 毫秒时间戳转本机可读时间；0 或空值显示占位符。 */
export function formatTimestamp(ms: number): string {
  if (!ms) {
    return '-'
  }
  return dayjs(ms).format('YYYY-MM-DD HH:mm:ss')
}
