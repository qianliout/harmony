import { onMounted, onUnmounted } from 'vue'

export interface WebHotkeyHandlers {
  entry: () => void
  escape: () => void
}

function isTypingTarget(target: EventTarget | null): boolean {
  const element = target as HTMLElement | null
  if (!element) return false
  const tag = element.tagName
  return tag === 'INPUT' || tag === 'TEXTAREA' || tag === 'SELECT' || element.isContentEditable
}

/** 全局快捷键：N 记一笔，Esc 关闭浮层。只在 H5 壳层注册。 */
export function useWebHotkeys(handlers: WebHotkeyHandlers, isActive?: () => boolean): void {
  function onKeydown(event: KeyboardEvent): void {
    if (isActive && !isActive()) return
    if (event.key === 'Escape') {
      handlers.escape()
      return
    }
    const modifier = event.metaKey || event.ctrlKey || event.altKey
    if (isTypingTarget(event.target) || modifier) return
    if (event.key.toLowerCase() === 'n') {
      event.preventDefault()
      handlers.entry()
    }
  }

  onMounted(() => window.addEventListener('keydown', onKeydown))
  onUnmounted(() => window.removeEventListener('keydown', onKeydown))
}
