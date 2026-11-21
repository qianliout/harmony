import { defineStore } from 'pinia'
import { ref } from 'vue'

/** 记一笔抽屉的预填上下文；recordId 为正时是编辑。 */
export interface EntryContext {
  personId?: number
  bookId?: number
  recordId?: number
}

export type WebOverlay =
  | { type: 'entry'; context: EntryContext }
  | { type: 'record'; id: number }

export const useWebUiStore = defineStore('webUi', () => {
  const overlay = ref<WebOverlay | null>(null)
  const toastMessage = ref('')
  const revision = ref(0)
  let toastTimer: ReturnType<typeof setTimeout> | undefined

  function openEntry(context: EntryContext = {}): void {
    overlay.value = { type: 'entry', context }
  }

  function openRecord(id: number): void {
    overlay.value = { type: 'record', id }
  }

  function close(): void {
    overlay.value = null
  }

  function toast(message: string): void {
    toastMessage.value = message
    clearTimeout(toastTimer)
    toastTimer = setTimeout(() => {
      toastMessage.value = ''
      toastTimer = undefined
    }, 2600)
  }

  /** 任意写操作后递增，页面据此重新加载。 */
  function markChanged(): void {
    revision.value += 1
  }

  return { overlay, toastMessage, revision, openEntry, openRecord, close, toast, markChanged }
})
