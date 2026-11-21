import { computed, reactive, ref, watch } from 'vue'

import { deleteRecord, fetchRecords, fetchRecordSummary, type RecordFilter } from '@/api/record'
import type { Direction, RecordItem, RecordSummary } from '@/api/types'
import { useWebUiStore } from '@/stores/web'

const PAGE_SIZE = 20

/** 页面上的筛选输入；空值表示不限。 */
export interface RecordFilterState {
  bookId: number
  direction: string
  from: string
  to: string
  keyword: string
}

function emptyFilter(): RecordFilterState {
  return { bookId: 0, direction: '', from: '', to: '', keyword: '' }
}

/** 总账、礼簿详情、亲友详情共用的记录列表：同一套筛选同时驱动列表与合计，任何写操作后自动刷新。 */
export function useRecordList(fixed: () => RecordFilter = () => ({})) {
  const ui = useWebUiStore()
  const filter = reactive<RecordFilterState>(emptyFilter())
  const page = ref(1)
  const items = ref<RecordItem[]>([])
  const total = ref(0)
  const summary = ref<RecordSummary>({ count: 0, inCent: 0, outCent: 0 })
  const loading = ref(false)
  const error = ref('')
  let seq = 0

  const query = computed<RecordFilter>(() => ({
    bookId: filter.bookId || undefined,
    direction: (filter.direction as Direction) || undefined,
    from: filter.from || undefined,
    to: filter.to || undefined,
    keyword: filter.keyword.trim() || undefined,
    ...fixed(),
  }))
  const pageCount = computed(() => Math.max(1, Math.ceil(total.value / PAGE_SIZE)))
  const hasFilters = computed(() =>
    Boolean(filter.bookId || filter.direction || filter.from || filter.to || filter.keyword.trim()),
  )

  async function reload(): Promise<void> {
    const current = ++seq
    loading.value = true
    error.value = ''
    try {
      const [list, sum] = await Promise.all([
        fetchRecords({ ...query.value, page: page.value, pageSize: PAGE_SIZE }),
        fetchRecordSummary(query.value),
      ])
      if (current !== seq) return
      items.value = list.items
      total.value = list.total
      summary.value = sum
    } catch (err) {
      if (current === seq) error.value = err instanceof Error ? err.message : '加载失败'
    } finally {
      if (current === seq) loading.value = false
    }
  }

  function reset(): void {
    Object.assign(filter, emptyFilter())
  }

  async function remove(record: RecordItem): Promise<void> {
    if (!window.confirm(`删除 ${record.personDisplayName} 的这条记录？此操作不能撤销。`)) return
    try {
      await deleteRecord(record.id)
      ui.toast('已删除')
      ui.markChanged()
    } catch (err) {
      ui.toast(err instanceof Error ? err.message : '删除失败')
    }
  }

  // 筛选变化回到第一页；页码或查询变化都重新加载。
  watch(
    () => ({ ...filter }),
    () => {
      page.value = 1
    },
  )
  watch([query, page], () => void reload(), { deep: true })
  watch(
    () => ui.revision,
    () => void reload(),
  )

  return { filter, page, items, total, summary, loading, error, pageCount, hasFilters, reload, reset, remove }
}
