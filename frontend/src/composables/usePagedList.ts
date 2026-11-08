import { ref, type Ref } from 'vue'

import type { Page } from '@/api/types'

export interface PagedList<T> {
  items: Ref<T[]>
  total: Ref<number>
  loading: Ref<boolean>
  finished: Ref<boolean>
  error: Ref<string>
  load: (reset?: boolean) => Promise<void>
  refresh: () => Promise<void>
  loadMore: () => Promise<void>
}

export function usePagedList<T>(
  fetchPage: (page: number, pageSize: number) => Promise<Page<T>>,
  pageSize = 20,
): PagedList<T> {
  const items = ref<T[]>([]) as Ref<T[]>
  const total = ref(0)
  const loading = ref(false)
  const finished = ref(false)
  const error = ref('')
  let page = 1

  async function load(reset = false): Promise<void> {
    if (loading.value) return
    if (reset) {
      page = 1
      finished.value = false
    }
    if (finished.value) return
    loading.value = true
    error.value = ''
    try {
      const result = await fetchPage(page, pageSize)
      items.value = reset ? result.items : [...items.value, ...result.items]
      total.value = result.total
      finished.value = items.value.length >= result.total || result.items.length < pageSize
      page += 1
    } catch (err) {
      error.value = err instanceof Error ? err.message : '加载失败'
    } finally {
      loading.value = false
    }
  }

  async function refresh(): Promise<void> {
    loading.value = false
    await load(true)
  }

  async function loadMore(): Promise<void> {
    await load(false)
  }

  return { items, total, loading, finished, error, load, refresh, loadMore }
}
