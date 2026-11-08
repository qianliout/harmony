import { defineStore } from 'pinia'
import { computed, ref } from 'vue'

import { fetchDicts } from '@/api/dict'
import type { DictItem } from '@/api/types'

export const useDictStore = defineStore('dict', () => {
  const categories = ref<DictItem[]>([])
  const itemKinds = ref<DictItem[]>([])
  const loaded = ref(false)
  const loading = ref(false)

  const categoryMap = computed(() =>
    Object.fromEntries(categories.value.map((item) => [item.code, item])) as Record<string, DictItem>,
  )
  const itemKindMap = computed(() =>
    Object.fromEntries(itemKinds.value.map((item) => [item.code, item])) as Record<string, DictItem>,
  )

  function categoryLabel(code: string): string {
    return categoryMap.value[code]?.label ?? code
  }

  function categoryColor(code: string): string {
    return categoryMap.value[code]?.color ?? 'stone'
  }

  function itemKindLabel(code: string): string {
    return itemKindMap.value[code]?.label ?? code
  }

  async function ensure(force = false): Promise<void> {
    if (loaded.value && !force) return
    if (loading.value) return
    loading.value = true
    try {
      const [categoryItems, kindItems] = await Promise.all([
        fetchDicts('book_category'),
        fetchDicts('item_kind'),
      ])
      categories.value = categoryItems
      itemKinds.value = kindItems
      loaded.value = true
    } finally {
      loading.value = false
    }
  }

  return {
    categories,
    itemKinds,
    loaded,
    loading,
    categoryMap,
    itemKindMap,
    categoryLabel,
    categoryColor,
    itemKindLabel,
    ensure,
  }
})
