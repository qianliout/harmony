<script setup lang="ts">
import { onLoad, onPullDownRefresh, onReachBottom, onShow } from '@dcloudio/uni-app'
import { computed, ref } from 'vue'

import { deleteBook, fetchBookDetail } from '@/api/book'
import { fetchPendingNotes, fetchRecords } from '@/api/record'
import type { BookItem, RecordFilters, RecordItem } from '@/api/types'
import AppNav from '@/components/AppNav.vue'
import EmptyState from '@/components/EmptyState.vue'
import FilterBar from '@/components/FilterBar.vue'
import FilterSheet from '@/components/FilterSheet.vue'
import Money from '@/components/Money.vue'
import RecordRow from '@/components/RecordRow.vue'
import Skeleton from '@/components/Skeleton.vue'
import SummaryStrip from '@/components/SummaryStrip.vue'
import { usePagedList } from '@/composables/usePagedList'
import { paletteOf } from '@/styles/palette'
import { confirmModal, back, go, toast } from '@/utils/nav'

const bookId = ref(0)
const book = ref<BookItem | null>(null)
const pendingNoteCount = ref(0)
const bookError = ref('')
const showSkeleton = ref(false)
const quick = ref('all')
const keyword = ref('')
const filterShow = ref(false)
const menuShow = ref(false)

const filters = ref<RecordFilters>({
  bookId: 0,
  eventId: 0,
  itemKind: '',
  direction: '',
  status: '',
  from: '',
  to: '',
})

let skeletonTimer: ReturnType<typeof setTimeout> | null = null
let searchTimer: ReturnType<typeof setTimeout> | null = null
let loadingAll = false

const {
  items: records,
  total,
  loading: recordsLoading,
  finished,
  error: recordsError,
  refresh: refreshRecords,
  loadMore,
} = usePagedList<RecordItem>((page, pageSize) => {
  const status = quick.value === 'pending' ? 'pending' : filters.value.status
  const direction = quick.value === 'in' || quick.value === 'out' ? quick.value : filters.value.direction
  return fetchRecords({
    bookId: isMerged.value ? undefined : bookId.value,
    originBookId: isMerged.value ? bookId.value : undefined,
    page,
    pageSize,
    itemKind: filters.value.itemKind,
    direction,
    status,
    from: filters.value.from,
    to: filters.value.to,
    keyword: keyword.value.trim(),
  })
}, 20)

const isMerged = computed(() => book.value?.status === 'merged')
const byItems = computed(() => {
  const groups = new Map<string, { label: string; cent: number }>()
  for (const part of book.value?.byItem ?? []) {
    const current = groups.get(part.itemKind) ?? { label: part.itemKindLabel, cent: 0 }
    current.cent += part.cent
    groups.set(part.itemKind, current)
  }
  return Array.from(groups.values())
})
const spineColor = computed(() => {
  const name = book.value?.categoryColor || 'stone'
  return paletteOf(name).main
})
const categoryColors = computed(() => paletteOf(book.value?.categoryColor || 'stone'))
const hasActiveFilters = computed(
  () =>
    Boolean(
      filters.value.itemKind ||
        filters.value.direction ||
        filters.value.status ||
        filters.value.from ||
        filters.value.to ||
        filters.value.eventId ||
        keyword.value.trim(),
    ) || quick.value !== 'all',
)
const activeFilterCount = computed(() => {
  let count = 0
  if (filters.value.itemKind) count += 1
  if (filters.value.direction) count += 1
  if (filters.value.status) count += 1
  if (filters.value.from || filters.value.to) count += 1
  if (filters.value.eventId) count += 1
  return count
})
const emptyTitle = computed(() =>
  hasActiveFilters.value ? '没有符合条件的记录' : '还没有记录',
)
const emptyDescription = computed(() =>
  hasActiveFilters.value ? '换个筛选条件试试' : '记下第一笔人情往来吧',
)

async function loadBook(): Promise<void> {
  try {
    const detail = await fetchBookDetail(bookId.value)
    book.value = detail
    bookError.value = ''
  } catch (err) {
    bookError.value = err instanceof Error ? err.message : '礼簿加载失败'
  }
}

async function loadPendingNotes(): Promise<void> {
  try {
    const notes = await fetchPendingNotes(bookId.value)
    pendingNoteCount.value = notes.length
  } catch {
    pendingNoteCount.value = 0
  }
}

async function loadAll(): Promise<void> {
  if (!bookId.value || loadingAll) return
  loadingAll = true
  if (!book.value) {
    if (skeletonTimer) clearTimeout(skeletonTimer)
    skeletonTimer = setTimeout(() => {
      showSkeleton.value = true
    }, 300)
  }
  try {
    await loadBook()
    if (bookError.value) {
      return
    }
    await Promise.all([loadPendingNotes(), refreshRecords()])
  } finally {
    if (skeletonTimer) clearTimeout(skeletonTimer)
    skeletonTimer = null
    showSkeleton.value = false
    loadingAll = false
  }
}

function retry(): void {
  loadAll()
}

function onQuickChange(value: string): void {
  quick.value = value
  refreshRecords()
}

function refreshList(): void {
  refreshRecords()
}

function onKeywordInput(event: Event): void {
  keyword.value = (event as unknown as { detail: { value: string } }).detail.value
  if (searchTimer) clearTimeout(searchTimer)
  searchTimer = setTimeout(() => {
    refreshRecords()
  }, 300)
}

function clearKeyword(): void {
  keyword.value = ''
  refreshRecords()
}

function applyFilters(next: RecordFilters): void {
  filters.value = { ...next, bookId: bookId.value }
  quick.value = 'all'
  refreshRecords()
}

function resetFilters(): void {
  filters.value = { bookId: bookId.value, eventId: 0, itemKind: '', direction: '', status: '', from: '', to: '' }
  quick.value = 'all'
  keyword.value = ''
  refreshRecords()
}

function openPending(): void {
  quick.value = 'pending'
  refreshRecords()
}

function clearAllFilters(): void {
  resetFilters()
}

function openRecord(record: RecordItem): void {
  go(`/sub-record/detail?id=${record.id}`)
}

function openMergedTarget(): void {
  if (book.value?.mergedIntoId) {
    go(`/sub-book/detail?id=${book.value.mergedIntoId}`)
    return
  }
  go('/sub-book/merged')
}

function openMenu(): void {
  menuShow.value = true
}

function closeMenu(): void {
  menuShow.value = false
}

function menuTo(url: string): void {
  menuShow.value = false
  go(url)
}

async function removeBook(): Promise<void> {
  menuShow.value = false
  const ok = await confirmModal('删除后不可恢复，礼簿里的记录也会一起删除。', '删除礼簿')
  if (!ok) return
  try {
    await deleteBook(bookId.value)
    toast('已删除', 'success')
    back()
  } catch (err) {
    toast(err instanceof Error ? err.message : '删除失败')
  }
}

function addRecord(): void {
  go(`/sub-record/form?bookId=${bookId.value}`)
}

function continuousEntry(): void {
  go(`/sub-record/continuous?bookId=${bookId.value}`)
}

onLoad((query) => {
  bookId.value = Number(query?.id ?? 0)
})

onShow(() => {
  if (!bookId.value) return
  loadAll()
})

onReachBottom(() => {
  if (!finished.value) loadMore()
})

onPullDownRefresh(async () => {
  try {
    await loadAll()
  } finally {
    uni.stopPullDownRefresh()
  }
})
</script>

<template>
  <view class="screen" :class="{ 'screen--with-actions': !isMerged }">
    <AppNav title="礼簿详情" :right="isMerged ? '' : '⋯'" @right="openMenu" />

    <Skeleton v-if="showSkeleton" />

    <view v-else-if="bookError && !book" class="page">
      <EmptyState title="没能打开这本礼簿" :description="bookError" action-text="重试" @action="retry" />
    </view>

    <template v-else-if="book">
      <view v-if="isMerged" class="merged-banner" @click="openMergedTarget">
        <text class="merged-banner__text">已合并到「{{ book.mergedIntoName }}」</text>
        <text class="merged-banner__link">查看 ›</text>
      </view>

      <view class="page">
        <view class="hero card">
          <view class="hero__spine" :style="{ background: spineColor }" />
          <view class="hero__body">
            <text class="hero__name">{{ book.name }}</text>
            <view class="hero__meta">
              <text
                class="hero__category"
                :style="{ background: categoryColors.light, color: categoryColors.main }"
              >
                {{ book.categoryLabel }}
              </text>
              <text v-if="book.description" class="hero__desc">{{ book.description }}</text>
            </view>
            <text v-if="book.defaultEventName" class="hero__default">默认事项 · {{ book.defaultEventName }}</text>
          </view>
        </view>

        <view v-if="!isMerged" class="summary">
          <SummaryStrip
            :in-cent="book.inCent"
            :out-cent="book.outCent"
            :pending-count="book.pendingCount"
            @pending="openPending"
          />
        </view>

        <view v-if="!isMerged && byItems.length" class="by-item">
          <scroll-view scroll-x class="by-item__scroll">
            <view class="by-item__inner">
              <text class="by-item__label">名目</text>
              <view v-for="part in byItems" :key="part.label" class="by-item__part">
                <text class="by-item__name">{{ part.label }}</text>
                <Money :cent="part.cent" direction="none" size="s" compact />
              </view>
            </view>
          </scroll-view>
        </view>

        <text v-if="isMerged" class="merged-hint">以下记录原先记在这本礼簿上，合并后归属目标礼簿</text>

        <view class="search-box search">
          <text class="search-box__icon">⌕</text>
          <input
            class="search-box__input"
            :value="keyword"
            placeholder="在本礼簿中搜索"
            confirm-type="search"
            @input="onKeywordInput"
          />
          <text v-if="keyword" class="search__clear" @click="clearKeyword">×</text>
        </view>

        <FilterBar :quick="quick" :count="activeFilterCount" @update:quick="onQuickChange" @open="filterShow = true" />

        <Skeleton v-if="recordsLoading && !records.length" />

        <view v-else-if="recordsError && !records.length" class="list-empty">
          <EmptyState title="没能加载记录" :description="recordsError" action-text="重试" @action="refreshList" />
        </view>

        <view v-else-if="!records.length" class="list-empty">
          <EmptyState
            v-if="isMerged"
            title="这本礼簿已经合并"
            description="原本的记录都迁到了目标礼簿"
            :action-text="book.mergedIntoId ? '查看目标礼簿' : ''"
            @action="openMergedTarget"
          />
          <EmptyState
            v-else
            :title="emptyTitle"
            :description="emptyDescription"
            :action-text="hasActiveFilters ? '清除筛选' : '记一笔'"
            @action="hasActiveFilters ? clearAllFilters() : addRecord()"
          />
        </view>

        <view v-else class="rows card">
          <RecordRow
            v-for="record in records"
            :key="record.id"
            :record="record"
            mode="book"
            @tap="openRecord"
          />
        </view>

        <view v-if="records.length" class="list-foot">
          <text v-if="recordsLoading" class="loading-tip">加载中…</text>
          <text v-else-if="finished" class="loading-tip">没有更多了</text>
          <text v-if="recordsError" class="list-foot__error" @click="refreshList">{{ recordsError }}，点击重试</text>
        </view>

        <text v-if="total && records.length" class="list-total">共 {{ total }} 笔</text>
      </view>
    </template>

    <view v-if="book && !isMerged" class="bottom-bar">
      <view class="btn btn--secondary bottom-bar__btn" @click="addRecord">记一笔</view>
      <view class="btn btn--primary bottom-bar__btn bottom-bar__btn--main" @click="continuousEntry">连续录入</view>
    </view>

    <wd-popup v-model="menuShow" position="bottom" :safe-area-inset-bottom="true" custom-style="border-radius: 20px 20px 0 0;">
      <view class="menu">
        <view class="menu__handle" />
        <view class="menu__item" @click="menuTo(`/sub-book/edit?id=${bookId}`)">编辑礼簿</view>
        <view class="menu__item" @click="menuTo(`/sub-book/merge?sourceId=${bookId}`)">合并到其他礼簿</view>
        <view class="menu__item" @click="menuTo(`/sub-book/pages?bookId=${bookId}`)">按页核对</view>
        <view class="menu__item" @click="menuTo(`/sub-record/notes?bookId=${bookId}`)">
          整理纸本标注<text v-if="pendingNoteCount" class="menu__badge">{{ pendingNoteCount }}</text>
        </view>
        <view class="menu__item menu__item--danger" @click="removeBook">删除礼簿</view>
        <view class="menu__cancel" @click="closeMenu">取消</view>
      </view>
    </wd-popup>

    <FilterSheet
      v-model:show="filterShow"
      :value="filters"
      :count="total"
      @apply="applyFilters"
      @reset="resetFilters"
    />
  </view>
</template>

<style lang="scss" scoped>
.screen--with-actions {
  padding-bottom: calc(88px + env(safe-area-inset-bottom));
}

.merged-banner {
  display: flex;
  align-items: center;
  gap: 6px;
  margin: 8px var(--gutter) 0;
  padding: 10px 14px;
  border-radius: var(--r-s);
  background: var(--c-pending-light);
  color: var(--c-pending);
  font-size: 13px;
  line-height: 18px;
}

.merged-banner__text {
  flex: 1;
  min-width: 0;
}

.merged-banner__link {
  flex: none;
  font-weight: 600;
}

.hero {
  position: relative;
  display: flex;
  overflow: hidden;
  margin-top: 8px;
}

.hero__spine {
  width: 6px;
  flex: none;
}

.hero__body {
  flex: 1;
  min-width: 0;
  padding: 14px 16px 14px 14px;
}

.hero__name {
  display: block;
  font-size: 22px;
  line-height: 30px;
  font-weight: 600;
  color: var(--c-ink-1);
}

.hero__meta {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-top: 5px;
}

.hero__category {
  flex: none;
  height: 20px;
  padding: 0 7px;
  border-radius: var(--r-xs);
  font-size: 12px;
  line-height: 20px;
}

.hero__desc {
  flex: 1;
  min-width: 0;
  font-size: 13px;
  line-height: 18px;
  color: var(--c-ink-3);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.hero__default {
  display: block;
  margin-top: 4px;
  font-size: 12px;
  color: var(--c-ink-3);
}

.summary {
  margin-top: var(--card-gap);
}

.merged-hint {
  display: block;
  margin-top: 10px;
  padding: 8px 12px;
  border-radius: var(--r-s);
  background: var(--c-fill);
  font-size: 12px;
  color: var(--c-ink-2);
}

.by-item {
  margin-top: 10px;
}

.by-item__scroll {
  width: 100%;
  white-space: nowrap;
}

.by-item__inner {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 0 2px;
}

.by-item__label {
  flex: none;
  font-size: 12px;
  color: var(--c-ink-3);
}

.by-item__part {
  display: flex;
  align-items: baseline;
  gap: 4px;
  flex: none;
}

.by-item__name {
  font-size: 12px;
  color: var(--c-ink-2);
}

.search {
  margin-top: 12px;
}

.search__clear {
  padding: 0 2px;
  font-size: 18px;
  line-height: 1;
  color: var(--c-ink-3);
}

.list-empty {
  padding-top: 12px;
}

.rows {
  margin-top: 4px;
  overflow: hidden;
}

.list-foot {
  padding: 14px 0 4px;
  text-align: center;
}

.list-foot__error {
  display: block;
  font-size: 13px;
  color: var(--c-danger);
}

.list-total {
  display: block;
  padding-bottom: 8px;
  text-align: center;
  font-size: 12px;
  color: var(--c-ink-3);
}

.menu {
  padding: 8px 0 calc(8px + env(safe-area-inset-bottom));
}

.menu__handle {
  width: 36px;
  height: 4px;
  margin: 4px auto 8px;
  border-radius: var(--r-full);
  background: var(--c-line);
}

.menu__item {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 52px;
  font-size: 16px;
  color: var(--c-ink-1);
  box-shadow: inset 0 -1px 0 var(--c-divider);
}

.menu__item--danger {
  color: var(--c-danger);
}

.menu__badge {
  margin-left: 4px;
  padding: 0 6px;
  height: 18px;
  border-radius: var(--r-full);
  background: var(--c-pending-light);
  color: var(--c-pending);
  font-size: 11px;
  line-height: 18px;
}

.menu__cancel {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 52px;
  margin-top: 8px;
  color: var(--c-ink-2);
  font-size: 16px;
}
</style>
