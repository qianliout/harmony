<script setup lang="ts">
import { onLoad, onPullDownRefresh, onReachBottom, onShow } from '@dcloudio/uni-app'
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'

import { fetchRecords } from '@/api/record'
import type { RecordListParams } from '@/api/record'
import type { RecordFilters, RecordItem } from '@/api/types'
import AppNav from '@/components/AppNav.vue'
import EmptyState from '@/components/EmptyState.vue'
import FilterBar from '@/components/FilterBar.vue'
import FilterSheet from '@/components/FilterSheet.vue'
import RecordRow from '@/components/RecordRow.vue'
import Skeleton from '@/components/Skeleton.vue'
import { usePagedList } from '@/composables/usePagedList'
import { go, toast } from '@/utils/nav'

const keyword = ref('')
const quick = ref('all')
function emptyFilters(): RecordFilters {
  return {
    bookCategory: '',
    eventType: '',
    personId: 0,
    bookId: 0,
    eventId: 0,
    itemKind: '',
    direction: '',
    status: '',
    from: '',
    to: '',
  }
}
const filters = ref<RecordFilters>(emptyFilters())
const filterShow = ref(false)
const showSkeleton = ref(false)
const initialized = ref(false)
let skeletonTimer: ReturnType<typeof setTimeout> | null = null
let searchTimer: ReturnType<typeof setTimeout> | null = null

function queryFor(): RecordListParams {
  const status = quick.value === 'pending' ? 'pending' : filters.value.status
  const direction =
    quick.value === 'pending' || quick.value === 'in'
      ? 'in'
      : quick.value === 'out'
        ? 'out'
        : filters.value.direction
  return {
    keyword: keyword.value.trim() || undefined,
    bookCategory: filters.value.bookCategory || undefined,
    eventType: filters.value.eventType || undefined,
    personId: filters.value.personId || undefined,
    bookId: filters.value.bookId || undefined,
    eventId: filters.value.eventId || undefined,
    itemKind: filters.value.itemKind || undefined,
    status: status || undefined,
    direction: direction || undefined,
    from: filters.value.from || undefined,
    to: filters.value.to || undefined,
  }
}

const list = usePagedList<RecordItem>(
  (page, pageSize) => fetchRecords({ ...queryFor(), page, pageSize }),
  20,
)

const activeFilterCount = computed(
  () =>
    [
      filters.value.bookCategory,
      filters.value.eventType,
      filters.value.personId,
      filters.value.bookId,
      filters.value.eventId,
      filters.value.itemKind,
      filters.value.direction,
      filters.value.status,
      filters.value.from,
      filters.value.to,
    ].filter(Boolean).length,
)

const hasAnyFilter = computed(
  () => Boolean(keyword.value.trim()) || quick.value !== 'all' || activeFilterCount.value > 0,
)

function startSkeletonTimer(): void {
  if (skeletonTimer) clearTimeout(skeletonTimer)
  skeletonTimer = setTimeout(() => {
    if (list.loading.value && !list.items.value.length) showSkeleton.value = true
  }, 300)
}

async function reload(): Promise<void> {
  startSkeletonTimer()
  await list.refresh()
}

watch(
  () => list.loading.value,
  (loading) => {
    if (loading) {
      startSkeletonTimer()
    } else {
      showSkeleton.value = false
      initialized.value = true
      if (skeletonTimer) clearTimeout(skeletonTimer)
    }
  },
)

let suppressQuickWatch = false

watch(quick, () => {
  if (suppressQuickWatch) {
    suppressQuickWatch = false
    return
  }
  filters.value = { ...filters.value, status: '', direction: '' }
  void reload()
})

function onKeywordInput(event: Event): void {
  keyword.value = (event as unknown as { detail: { value: string } }).detail.value
  if (searchTimer) clearTimeout(searchTimer)
  searchTimer = setTimeout(() => {
    void reload()
  }, 300)
}

function clearKeyword(): void {
  if (!keyword.value) return
  keyword.value = ''
  void reload()
}

function applyFilters(value: RecordFilters): void {
  const previousQuick = quick.value
  filters.value = { ...value }
  if ((value.status || value.direction) && quick.value !== 'all') {
    suppressQuickWatch = true
    quick.value = 'all'
  }
  if (quick.value === previousQuick) void reload()
}

function resetFilters(): void {
  const previousQuick = quick.value
  keyword.value = ''
  filters.value = emptyFilters()
  if (quick.value !== 'all') {
    suppressQuickWatch = true
    quick.value = 'all'
  }
  if (quick.value === previousQuick) void reload()
}

function openRecord(record: RecordItem): void {
  go(`/sub-record/detail?id=${record.id}`)
}

function createRecord(): void {
  go('/sub-record/form?mode=create')
}

onLoad((query) => {
  const parsed = (query ?? {}) as Record<string, string>
  const bookId = Number(parsed.bookId ?? 0)
  const personId = Number(parsed.personId ?? 0)
  filters.value = {
    ...filters.value,
    bookCategory: parsed.bookCategory ?? '',
    eventType: parsed.eventType ?? '',
    from: parsed.from ?? '',
    to: parsed.to ?? '',
    bookId,
    personId,
  }
})

onMounted(() => {
  void reload()
})

onUnmounted(() => {
  if (skeletonTimer) clearTimeout(skeletonTimer)
  if (searchTimer) clearTimeout(searchTimer)
})

onShow(() => {
  if (initialized.value) void reload()
})

onPullDownRefresh(async () => {
  try {
    await reload()
  } catch (err) {
    toast(err instanceof Error ? err.message : '刷新失败')
  } finally {
    uni.stopPullDownRefresh()
  }
})

onReachBottom(() => {
  if (!list.loading.value && !list.finished.value) void list.loadMore()
})
</script>

<template>
  <view class="screen list">
    <AppNav title="全部记录" />

    <view class="page">
      <view class="search-box list__search">
        <text class="search-box__icon">⌕</text>
        <input
          class="search-box__input"
          :value="keyword"
          placeholder="搜亲友、事项、备注、礼品"
          confirm-type="search"
          @input="onKeywordInput"
          @confirm="reload"
        />
        <text v-if="keyword" class="list__clear" @click="clearKeyword">清空</text>
      </view>

      <FilterBar
        v-model:quick="quick"
        :count="activeFilterCount"
        @open="filterShow = true"
      />
    </view>

    <Skeleton v-if="showSkeleton" />

    <view v-else-if="list.error.value && !list.items.value.length" class="error-state">
      <text class="error-state__text">{{ list.error.value }}</text>
      <view class="btn btn--secondary error-state__btn" @click="reload">重试</view>
    </view>

    <template v-else-if="list.items.value.length">
      <view class="page list__rows">
        <view class="list__card">
          <RecordRow
            v-for="record in list.items.value"
            :key="record.id"
            :record="record"
            mode="global"
            @tap="openRecord"
          />
        </view>
        <view class="list__footer">
          <text v-if="list.loading.value" class="caption">正在加载…</text>
          <text v-else-if="list.finished.value" class="caption">
            共 {{ list.total.value }} 条记录<text v-if="hasAnyFilter">（已筛选）</text>
          </text>
          <text v-else class="caption" @click="list.loadMore">上拉加载更多</text>
        </view>
      </view>
    </template>

    <EmptyState
      v-else
      :title="hasAnyFilter ? '没有符合条件的记录' : '还没有记录'"
      :description="hasAnyFilter ? '试试调整搜索或筛选条件' : '记下第一笔人情往来吧'"
      :action-text="hasAnyFilter ? '清除筛选' : '记一笔'"
      @action="hasAnyFilter ? resetFilters() : createRecord()"
    />

    <FilterSheet
      v-model:show="filterShow"
      :value="filters"
      :count="list.total.value"
      @apply="applyFilters"
      @reset="resetFilters"
    />
  </view>
</template>

<style lang="scss" scoped>
.list {
  padding-bottom: calc(24px + env(safe-area-inset-bottom));
}

.list__search {
  margin-top: 4px;
}

.list__clear {
  flex: none;
  font-size: 12px;
  color: var(--c-ink-3);
}

.list__rows {
  margin-top: 4px;
}

.list__card {
  border-radius: var(--r-m);
  background: var(--c-surface);
  box-shadow: var(--shadow-card);
  overflow: hidden;
}

.list__footer {
  padding: 14px 0 8px;
  text-align: center;
}

.error-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 64px 24px;
}

.error-state__text {
  font-size: 14px;
  color: var(--c-ink-2);
}

.error-state__btn {
  margin-top: 16px;
  padding: 0 24px;
}
</style>
