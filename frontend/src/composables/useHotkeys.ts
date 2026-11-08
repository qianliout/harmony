// 识别核对页的键盘快捷键，只在 H5 宽屏核对中启用。
// 整个文件只在 H5 编译，小程序产物中不出现 keydown 监听。
// #ifdef H5
import { onHide, onShow, onUnload } from '@dcloudio/uni-app'

export interface HotkeyHandlers {
  /** ↑ 上一行（不在输入框中时） */
  up?: () => void
  /** ↓ 下一行（不在输入框中时） */
  down?: () => void
  /** Enter：不在编辑时开始编辑姓名；编辑中保存并跳到下一行同一列。参数表示当前是否在输入框中。 */
  enter?: (editing: boolean) => void
  /** Tab / Shift+Tab：在当前行字段间跳转。参数 shift 表示反向。 */
  tab?: (shift: boolean) => void
  /** Esc：取消当前单元格编辑。 */
  escape?: () => void
  /** Space：勾选/取消勾选当前行（不在输入框中时）。 */
  space?: () => void
  /** A：在当前行下方插入一行。 */
  addRow?: () => void
  /** R：原图顺时针旋转 90 度。 */
  rotate?: () => void
  zoomIn?: () => void
  zoomOut?: () => void
  zoomFit?: () => void
  /** Ctrl+Enter / ⌘+Enter：写入。 */
  commit?: () => void
}

function isEditingTarget(target: EventTarget | null): boolean {
  const element = target as HTMLElement | null
  if (!element) return false
  const tag = element.tagName
  return tag === 'INPUT' || tag === 'TEXTAREA' || tag === 'SELECT' || element.isContentEditable
}

/**
 * 在页面显示期间监听 window 的 keydown，隐藏或卸载时注销。
 * enabled 用于把快捷键限制在宽屏核对步骤；输入框内的普通字符不截获，避免影响打字。
 */
export function useHotkeys(handlers: HotkeyHandlers, enabled: () => boolean): void {
  let bound = false

  function onKeydown(event: KeyboardEvent): void {
    if (!enabled()) return
    if (event.defaultPrevented) return

    const modifier = event.ctrlKey || event.metaKey
    if (modifier && event.key === 'Enter') {
      if (handlers.commit) {
        event.preventDefault()
        handlers.commit()
      }
      return
    }
    if (modifier || event.altKey) return

    const editing = isEditingTarget(event.target)
    const key = event.key

    if (key === 'ArrowUp') {
      if (editing || !handlers.up) return
      event.preventDefault()
      handlers.up()
      return
    }
    if (key === 'ArrowDown') {
      if (editing || !handlers.down) return
      event.preventDefault()
      handlers.down()
      return
    }
    if (key === 'Enter') {
      if (!handlers.enter) return
      event.preventDefault()
      handlers.enter(editing)
      return
    }
    if (key === 'Tab') {
      if (!handlers.tab) return
      event.preventDefault()
      handlers.tab(event.shiftKey)
      return
    }
    if (key === 'Escape') {
      if (!editing || !handlers.escape) return
      event.preventDefault()
      handlers.escape()
      return
    }
    if (editing) return

    if (key === ' ' || key === 'Spacebar') {
      if (!handlers.space) return
      event.preventDefault()
      handlers.space()
      return
    }
    if (key === 'a' || key === 'A') {
      if (!handlers.addRow) return
      event.preventDefault()
      handlers.addRow()
      return
    }
    if (key === 'r' || key === 'R') {
      if (!handlers.rotate) return
      event.preventDefault()
      handlers.rotate()
      return
    }
    if (key === '+' || key === '=') {
      if (!handlers.zoomIn) return
      event.preventDefault()
      handlers.zoomIn()
      return
    }
    if (key === '-' || key === '_') {
      if (!handlers.zoomOut) return
      event.preventDefault()
      handlers.zoomOut()
      return
    }
    if (key === '0') {
      if (!handlers.zoomFit) return
      event.preventDefault()
      handlers.zoomFit()
    }
  }

  function bind(): void {
    if (bound || typeof window === 'undefined') return
    window.addEventListener('keydown', onKeydown)
    bound = true
  }

  function unbind(): void {
    if (!bound || typeof window === 'undefined') return
    window.removeEventListener('keydown', onKeydown)
    bound = false
  }

  onShow(bind)
  onHide(unbind)
  onUnload(unbind)
}
// #endif
