/**
 * uni-app 的 input / picker change 事件把值放在 event.detail.value 里，但 Events 类型没有这个形状。
 * 这里做一次运行时收窄，避免每个组件各写一遍不安全的断言。
 */
export function inputValue(event: Event): string {
  if (!('detail' in event) || typeof event.detail !== 'object' || event.detail === null) {
    return ''
  }
  if (!('value' in event.detail)) {
    return ''
  }
  const value = event.detail.value
  return typeof value === 'string' || typeof value === 'number' ? String(value) : ''
}
