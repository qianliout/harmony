import { defineStore } from 'pinia'
import { ref } from 'vue'

import { fetchSettings, updateSettings } from '@/api/setting'
import type { Settings } from '@/api/types'

const DEFAULTS: Settings = {
  showOwing: true,
  lastBookId: 0,
  lastEventId: 0,
  quickAmounts: [10000, 20000, 50000, 60000, 80000, 100000],
}

export const useContextStore = defineStore('context', () => {
  const settings = ref<Settings>({ ...DEFAULTS })
  const loaded = ref(false)

  async function load(force = false): Promise<void> {
    if (loaded.value && !force) return
    try {
      settings.value = await fetchSettings()
    } catch {
      settings.value = { ...DEFAULTS }
    } finally {
      loaded.value = true
    }
  }

  async function save(patch: Partial<Settings>): Promise<void> {
    settings.value = { ...settings.value, ...patch }
    try {
      settings.value = await updateSettings(patch)
    } catch {
      // 失败时保留本地值，下一次进入页面会重新从后端读取。
    }
  }

  async function rememberContext(bookId: number, eventId: number): Promise<void> {
    const patch: Partial<Settings> = {}
    if (bookId) patch.lastBookId = bookId
    if (eventId) patch.lastEventId = eventId
    if (Object.keys(patch).length) {
      await save(patch)
    }
  }

  return { settings, loaded, load, save, rememberContext }
})
