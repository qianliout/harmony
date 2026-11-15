import { computed, reactive } from 'vue'

import type { RecordListParams } from '@/api/record'

export interface RecordFilterState {
  bookCategory: string
  eventType: string
  personKeyword: string
  year: string
  direction: string
  status: string
  itemKind: string
  page: number
}

export function emptyRecordFilter(): RecordFilterState {
  return {
    bookCategory: '',
    eventType: '',
    personKeyword: '',
    year: '',
    direction: '',
    status: '',
    itemKind: '',
    page: 1,
  }
}

/** 筛选状态与 /records 参数的单向映射；页面只改状态，分页在筛选变化时回到第一页。 */
export function useRecordQuery(initial: Partial<RecordFilterState> = {}) {
  const filters = reactive<RecordFilterState>({ ...emptyRecordFilter(), ...initial })

  const params = computed<RecordListParams>(() => ({
    bookCategory: filters.bookCategory || undefined,
    eventType: filters.eventType || undefined,
    personKeyword: filters.personKeyword.trim() || undefined,
    direction: filters.direction || undefined,
    status: filters.status || undefined,
    itemKind: filters.itemKind || undefined,
    from: filters.year ? `${filters.year}-01-01` : undefined,
    to: filters.year ? `${filters.year}-12-31` : undefined,
    page: filters.page,
    pageSize: 20,
  }))

  const hasFilters = computed(() =>
    Boolean(
      filters.bookCategory ||
        filters.eventType ||
        filters.personKeyword.trim() ||
        filters.year ||
        filters.direction ||
        filters.status ||
        filters.itemKind,
    ),
  )

  function reset(): void {
    Object.assign(filters, emptyRecordFilter())
  }

  function changePage(page: number): void {
    filters.page = page
  }

  return { filters, params, hasFilters, reset, changePage }
}
