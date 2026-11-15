import { onMounted, onUnmounted } from 'vue'

export interface WebHotkeyHandlers {
  search: () => void
  entry: () => void
  escape: () => void
}

function isTypingTarget(target: EventTarget | null): boolean {
  const element = target as HTMLElement | null
  if (!element) return false
  const tag = element.tagName
  return tag === 'INPUT' || tag === 'TEXTAREA' || tag === 'SELECT' || element.isContentEditable
}

/** 全局快捷键：⌘K / Ctrl+K 或 / 搜索，N 记一笔，Esc 关闭浮层。只在 H5 壳层注册。 */
export function useWebHotkeys(handlers: WebHotkeyHandlers, isActive?: () => boolean): void {
  function onKeydown(event: KeyboardEvent): void {
    if (isActive && !isActive()) return
    const modifier = event.metaKey || event.ctrlKey
    if (modifier && event.key.toLowerCase() === 'k') {
      event.preventDefault()
      handlers.search()
      return
    }
    if (event.key === 'Escape') {
      handlers.escape()
      return
    }
    if (isTypingTarget(event.target) || modifier || event.altKey) return
    if (event.key === '/') {
      event.preventDefault()
      handlers.search()
      return
    }
    if (event.key.toLowerCase() === 'n') {
      event.preventDefault()
      handlers.entry()
    }
  }

  onMounted(() => window.addEventListener('keydown', onKeydown))
  onUnmounted(() => window.removeEventListener('keydown', onKeydown))
}
