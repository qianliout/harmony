import { defineStore } from 'pinia'
import { ref } from 'vue'

/** 记一笔抽屉的预填上下文。 */
export interface EntryContext {
  personId?: number
  eventId?: number
  bookId?: number
  recordId?: number
}

export type WebOverlay =
  | { type: 'entry'; context: EntryContext }
  | { type: 'record'; id: number }
  | { type: 'repay'; originIds: number[] }
  | { type: 'paperAnno'; id: number }
  | { type: 'merge'; sourceId?: number }
  | { type: 'search' }
  | { type: 'notes' }

export const useWebUiStore = defineStore('webUi', () => {
  const overlay = ref<WebOverlay | null>(null)
  const toastMessage = ref('')
  const revision = ref(0)
  let toastTimer: number | undefined

  function openEntry(context: EntryContext = {}): void {
    overlay.value = { type: 'entry', context }
  }

  function openRecord(id: number): void {
    overlay.value = { type: 'record', id }
  }

  function openRepay(originIds: number[]): void {
    if (!originIds.length) return
    overlay.value = { type: 'repay', originIds }
  }

  function openPaperAnno(id: number): void {
    overlay.value = { type: 'paperAnno', id }
  }

  function openMerge(sourceId?: number): void {
    overlay.value = { type: 'merge', sourceId }
  }

  function openSearch(): void {
    overlay.value = { type: 'search' }
  }

  function openNotes(): void {
    overlay.value = { type: 'notes' }
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

  function markChanged(): void {
    revision.value += 1
  }

  return {
    overlay,
    toastMessage,
    revision,
    openEntry,
    openRecord,
    openRepay,
    openPaperAnno,
    openMerge,
    openSearch,
    openNotes,
    close,
    toast,
    markChanged,
  }
})
