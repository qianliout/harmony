<script setup lang="ts">
import { onLoad, onShow } from '@dcloudio/uni-app'
import { computed, ref } from 'vue'

import { fetchBooks } from '@/api/book'
import { fetchEvents } from '@/api/event'
import type { BookItem, EventItem } from '@/api/types'
import AppNav from '@/components/AppNav.vue'
import EmptyState from '@/components/EmptyState.vue'
import EventSeal from '@/components/EventSeal.vue'
import FabButton from '@/components/FabButton.vue'
import Money from '@/components/Money.vue'
import Skeleton from '@/components/Skeleton.vue'
import { eventTypes, paletteOf } from '@/styles/palette'
import { pad, preciseDateLabel, yearOf } from '@/utils/date'
import { go } from '@/utils/nav'

type Tab = 'book' | 'event'

const TAB_KEY = 'harmony:book-tab'

const tab = ref<Tab>('book')
const books = ref<BookItem[]>([])
const mergedCount = ref(0)
const events = ref<EventItem[]>([])
const loading = ref(false)
const loaded = ref(false)
const error = ref('')
const showSkeleton = ref(false)
const keyword = ref('')
const typeFilter = ref('')
const hostFilter = ref('')
const yearFilter = ref(0)

let skeletonTimer: ReturnType<typeof setTimeout> | null = null

const searchPlaceholder = computed(() => (tab.value === 'book' ? '搜索礼簿名称、类别' : '搜索事项名称、地点'))

const filteredBooks = computed(() => {
  const q = keyword.value.trim().toLowerCase()
  if (!q) return books.value
  return books.value.filter((book) =>
    [book.name, book.categoryLabel, book.description].some((text) => text?.toLowerCase().includes(q)),
  )
})

const yearOptions = computed(() => {
  const years = new Set<number>()
  events.value.forEach((event) => {
    const year = yearOf(event.eventDate)
    if (year) years.add(year)
  })
  return Array.from(years).sort((a, b) => b - a)
})

const filteredEvents = computed(() => {
  const q = keyword.value.trim().toLowerCase()
  return events.value.filter((event) => {
    if (typeFilter.value && event.type !== typeFilter.value) return false
    if (hostFilter.value && event.host !== hostFilter.value) return false
    if (yearFilter.value && yearOf(event.eventDate) !== yearFilter.value) return false
    if (!q) return true
    return [event.name, event.location, event.hostPersonName, event.typeLabel].some((text) =>
      text?.toLowerCase().includes(q),
    )
  })
})

const groupedEvents = computed(() => {
  const sorted = [...filteredEvents.value].sort((a, b) => {
    const dateA = a.eventDate || ''
    const dateB = b.eventDate || ''
    if (dateA === dateB) return b.id - a.id
    return dateB.localeCompare(dateA)
  })
  const groups: { year: number; label: string; items: EventItem[] }[] = []
  for (const event of sorted) {
    const year = yearOf(event.eventDate)
    const label = year ? `${year} 年` : '未标日期'
    let group = groups.find((item) => item.year === year)
    if (!group) {
      group = { year, label, items: [] }
      groups.push(group)
    }
    group.items.push(event)
  }
  return groups.sort((a, b) => {
    if (a.year === 0) return 1
    if (b.year === 0) return -1
    return b.year - a.year
  })
})

const hasEvents = computed(() => groupedEvents.value.length > 0)
const hasActiveEventFilter = computed(() => Boolean(typeFilter.value || hostFilter.value || yearFilter.value))

function spineColor(book: BookItem): string {
  return paletteOf(book.categoryColor || 'stone').main
}

function categoryColors(book: BookItem): { main: string; light: string } {
  return paletteOf(book.categoryColor || 'stone')
}

function timestampDate(ms: number): string {
  if (!ms) return ''
  const date = new Date(ms)
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}`
}

function eventDateLabel(event: EventItem): string {
  return event.eventDate ? preciseDateLabel(event.eventDate, event.datePrecision) : '日期未定'
}

function eventMeta(event: EventItem): string {
  let text = eventDateLabel(event)
  if (event.location) text += ` · ${event.location}`
  return text
}

function setTab(next: Tab): void {
  if (tab.value === next) return
  tab.value = next
  keyword.value = ''
}

function onTapTab(next: Tab): void {
  setTab(next)
  uni.setStorageSync(TAB_KEY, next)
}

function onKeywordInput(event: Event): void {
  keyword.value = (event as unknown as { detail: { value: string } }).detail.value
}

function clearKeyword(): void {
  keyword.value = ''
}

function chooseType(code: string): void {
  typeFilter.value = typeFilter.value === code ? '' : code
}

function chooseHost(code: string): void {
  hostFilter.value = hostFilter.value === code ? '' : code
}

function chooseYear(year: number): void {
  yearFilter.value = yearFilter.value === year ? 0 : year
}

function clearEventFilters(): void {
  typeFilter.value = ''
  hostFilter.value = ''
  yearFilter.value = 0
}

function resetEventFilters(): void {
  clearEventFilters()
  keyword.value = ''
}

async function fetchAllEvents(): Promise<EventItem[]> {
  const all: EventItem[] = []
  let page = 1
  for (let guard = 0; guard < 20; guard += 1) {
    const result = await fetchEvents({ page, pageSize: 100 })
    all.push(...result.items)
    if (result.items.length < 100 || all.length >= result.total) break
    page += 1
  }
  return all
}

async function loadData(): Promise<void> {
  if (loading.value) return
  loading.value = true
  error.value = ''
  if (!loaded.value) {
    if (skeletonTimer) clearTimeout(skeletonTimer)
    skeletonTimer = setTimeout(() => {
      showSkeleton.value = true
    }, 300)
  }
  try {
    const [normalBooks, mergedBooks, allEvents] = await Promise.all([
      fetchBooks('normal'),
      fetchBooks('merged'),
      fetchAllEvents(),
    ])
    books.value = normalBooks
    mergedCount.value = mergedBooks.length
    events.value = allEvents
    loaded.value = true
  } catch (err) {
    error.value = err instanceof Error ? err.message : '加载失败'
  } finally {
    if (skeletonTimer) clearTimeout(skeletonTimer)
    skeletonTimer = null
    showSkeleton.value = false
    loading.value = false
  }
}

function retry(): void {
  loadData()
}

function openBook(book: BookItem): void {
  go(`/sub-book/detail?id=${book.id}`)
}

function openEvent(event: EventItem): void {
  go(`/sub-event/detail?id=${event.id}`)
}

function openMerged(): void {
  go('/sub-book/merged')
}

function createNew(): void {
  go(tab.value === 'book' ? '/sub-book/edit' : '/sub-event/edit')
}

onLoad((query) => {
  if (query?.tab === 'event') {
    tab.value = 'event'
    uni.setStorageSync(TAB_KEY, 'event')
    return
  }
  if (query?.tab === 'book') {
    tab.value = 'book'
    uni.setStorageSync(TAB_KEY, 'book')
    return
  }
  const stored = uni.getStorageSync(TAB_KEY)
  if (stored === 'event' || stored === 'book') {
    tab.value = stored
  }
})

onShow(() => {
  const stored = uni.getStorageSync(TAB_KEY)
  if ((stored === 'book' || stored === 'event') && stored !== tab.value) {
    tab.value = stored
    keyword.value = ''
  }
  loadData()
})
</script>

<template>
  <view class="screen screen--tab">
    <AppNav :back="false" title="" />

    <view class="page">
      <view class="segmented top-segmented">
        <view class="segmented__item" :class="{ 'segmented__item--on': tab === 'book' }" @click="onTapTab('book')">礼簿</view>
        <view class="segmented__item" :class="{ 'segmented__item--on': tab === 'event' }" @click="onTapTab('event')">事项</view>
      </view>

      <view class="search-box search">
        <text class="search-box__icon">⌕</text>
        <input
          class="search-box__input"
          :value="keyword"
          :placeholder="searchPlaceholder"
          confirm-type="search"
          @input="onKeywordInput"
        />
        <text v-if="keyword" class="search__clear" @click="clearKeyword">×</text>
      </view>
    </view>

    <Skeleton v-if="showSkeleton" />

    <view v-else-if="error && !loaded" class="page">
      <EmptyState title="没能加载礼簿" :description="error" action-text="重试" @action="retry" />
    </view>

    <template v-else-if="tab === 'book'">
      <view class="page book-list">
        <view v-if="!filteredBooks.length" class="book-list__empty">
          <EmptyState
            :title="keyword ? '没找到这本礼簿' : '还没有礼簿'"
            :description="keyword ? '换个关键词试试' : '礼簿按用途分本，收礼送礼各记一处'"
            :action-text="keyword ? '清除搜索' : '新建礼簿'"
            @action="keyword ? clearKeyword() : go('/sub-book/edit')"
          />
        </view>

        <view
          v-for="book in filteredBooks"
          :key="book.id"
          class="book-card"
          @click="openBook(book)"
        >
          <view class="book-card__spine" :style="{ background: spineColor(book) }" />
          <view class="book-card__body">
            <view class="book-card__head">
              <text class="book-card__name">{{ book.name }}</text>
              <text
                class="book-card__category"
                :style="{ background: categoryColors(book).light, color: categoryColors(book).main }"
              >
                {{ book.categoryLabel }}
              </text>
            </view>
            <view class="book-card__amounts">
              <view class="book-card__amount">
                <text class="book-card__amount-label">收</text>
                <Money :cent="book.inCent" direction="in" size="m" compact />
              </view>
              <view class="book-card__amount">
                <text class="book-card__amount-label">送</text>
                <Money :cent="book.outCent" direction="out" size="m" compact />
              </view>
            </view>
            <text class="book-card__meta">{{ book.recordCount }} 笔 · 建于 {{ timestampDate(book.createdAt) }}</text>
          </view>
        </view>

        <view class="merged-link" @click="openMerged">
          <text>已合并的礼簿（{{ mergedCount }}）</text>
          <text class="merged-link__arrow">›</text>
        </view>
      </view>
    </template>

    <template v-else>
      <view class="event-filters">
        <scroll-view scroll-x class="chips-row">
          <view class="chips-row__inner">
            <text class="chip" :class="{ 'chip--on': !typeFilter }" @click="typeFilter = ''">全部</text>
            <text
              v-for="type in eventTypes"
              :key="type.code"
              class="chip"
              :class="{ 'chip--on': typeFilter === type.code }"
              @click="chooseType(type.code)"
            >
              {{ type.label }}
            </text>
          </view>
        </scroll-view>
        <scroll-view scroll-x class="chips-row">
          <view class="chips-row__inner">
            <text class="chip chip--flat" :class="{ 'chip--flat-on': !hostFilter }" @click="hostFilter = ''">全部主办</text>
            <text class="chip chip--flat" :class="{ 'chip--flat-on': hostFilter === 'self' }" @click="chooseHost('self')">自家办</text>
            <text class="chip chip--flat" :class="{ 'chip--flat-on': hostFilter === 'other' }" @click="chooseHost('other')">别人家办</text>
            <text class="chip chip--flat" :class="{ 'chip--flat-on': !yearFilter }" @click="yearFilter = 0">全部年份</text>
            <text
              v-for="year in yearOptions"
              :key="year"
              class="chip chip--flat"
              :class="{ 'chip--flat-on': yearFilter === year }"
              @click="chooseYear(year)"
            >
              {{ year }}
            </text>
          </view>
        </scroll-view>
      </view>

      <view class="page event-list">
        <view v-if="!hasEvents" class="event-list__empty">
          <EmptyState
            :title="keyword || hasActiveEventFilter ? '没有符合条件的事项' : '还没有事项'"
            :description="keyword || hasActiveEventFilter ? '换一组条件试试' : '办酒、满月、乔迁，都算一个事项'"
            :action-text="keyword || hasActiveEventFilter ? '清除筛选' : '新建事项'"
            @action="keyword || hasActiveEventFilter ? resetEventFilters() : go('/sub-event/edit')"
          />
        </view>

        <template v-for="group in groupedEvents" :key="group.year">
          <view class="year-head">
            <text class="year-head__text">{{ group.label }}</text>
            <text class="year-head__count">{{ group.items.length }} 个</text>
          </view>
          <view
            v-for="event in group.items"
            :key="event.id"
            class="event-item card"
            @click="openEvent(event)"
          >
            <EventSeal :type="event.type" size="m" />
            <view class="event-item__main">
              <view class="event-item__line">
                <text class="event-item__name">{{ event.name }}</text>
                <text class="event-item__host">{{ event.hostLabel }}</text>
              </view>
              <text class="event-item__meta">{{ eventMeta(event) }}</text>
              <view class="event-item__stats">
                <text class="event-item__meta">{{ event.personCount }} 人 · 收</text>
                <Money :cent="event.inCent" direction="in" size="s" compact />
                <text class="event-item__meta">· 送</text>
                <Money :cent="event.outCent" direction="out" size="s" compact />
              </view>
            </view>
          </view>
        </template>
      </view>
    </template>

    <FabButton @click="createNew" />
  </view>
</template>

<style lang="scss" scoped>
.top-segmented {
  margin-top: 4px;
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

.book-list {
  padding-top: 12px;
}

.book-list__empty {
  padding-top: 24px;
}

.book-card {
  position: relative;
  display: flex;
  min-height: 96px;
  margin-bottom: 12px;
  border-radius: var(--r-m);
  background: var(--c-surface);
  box-shadow: var(--shadow-card);
  overflow: hidden;
}

.book-card__spine {
  width: 6px;
  flex: none;
}

.book-card__body {
  flex: 1;
  min-width: 0;
  padding: 13px 16px 12px;
}

.book-card__head {
  display: flex;
  align-items: center;
  gap: 8px;
}

.book-card__name {
  flex: 1;
  min-width: 0;
  font-size: 17px;
  line-height: 24px;
  font-weight: 600;
  color: var(--c-ink-1);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.book-card__category {
  flex: none;
  height: 20px;
  padding: 0 7px;
  border-radius: var(--r-xs);
  font-size: 12px;
  line-height: 20px;
}

.book-card__amounts {
  display: flex;
  align-items: baseline;
  gap: 16px;
  margin-top: 6px;
}

.book-card__amount {
  display: flex;
  align-items: baseline;
  gap: 4px;
}

.book-card__amount-label {
  font-size: 12px;
  color: var(--c-ink-3);
}

.book-card__meta {
  display: block;
  margin-top: 4px;
  font-size: 12px;
  line-height: 16px;
  color: var(--c-ink-3);
}

.merged-link {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 2px;
  padding: 18px 0 8px;
  font-size: 13px;
  color: var(--c-ink-3);
}

.merged-link__arrow {
  font-size: 16px;
  color: var(--c-ink-4);
}

.event-filters {
  padding: 10px 0 0;
}

.chips-row {
  width: 100%;
  white-space: nowrap;
}

.chips-row + .chips-row {
  margin-top: 8px;
}

.chips-row__inner {
  display: flex;
  gap: 8px;
  padding: 0 var(--gutter);
}

.event-list {
  padding-top: 4px;
}

.event-list__empty {
  padding-top: 24px;
}

.year-head {
  position: sticky;
  top: 0;
  z-index: 2;
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  padding: 12px 2px 8px;
  background: var(--c-bg);
}

.year-head__text {
  font-size: 15px;
  line-height: 22px;
  font-weight: 600;
  color: var(--c-ink-2);
}

.year-head__count {
  font-size: 12px;
  color: var(--c-ink-3);
}

.event-item {
  display: flex;
  align-items: flex-start;
  gap: 12px;
  padding: 12px 16px;
  margin-bottom: 10px;
}

.event-item__main {
  flex: 1;
  min-width: 0;
}

.event-item__line {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
}

.event-item__name {
  font-size: 15px;
  line-height: 22px;
  font-weight: 600;
  color: var(--c-ink-1);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.event-item__host {
  flex: none;
  height: 20px;
  padding: 0 6px;
  border-radius: var(--r-xs);
  background: var(--c-fill);
  color: var(--c-ink-2);
  font-size: 12px;
  line-height: 20px;
}

.event-item__meta {
  font-size: 13px;
  line-height: 18px;
  color: var(--c-ink-3);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.event-item__stats {
  display: flex;
  align-items: baseline;
  gap: 3px;
  margin-top: 2px;
  font-size: 13px;
  color: var(--c-ink-3);
}
</style>
