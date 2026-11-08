/** 毫秒时间戳转本机可读时间；0 或空值显示占位符。 */
export function formatTimestamp(ms: number): string {
  if (!ms) {
    return '-'
  }
  const d = new Date(ms)
  const pad = (n: number) => String(n).padStart(2, '0')
  const date = `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`
  const time = `${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`
  return `${date} ${time}`
}
