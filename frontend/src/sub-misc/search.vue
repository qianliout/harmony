<script setup lang="ts">
import { onLoad, onReachBottom, onUnload } from '@dcloudio/uni-app'
import { computed, nextTick, ref } from 'vue'

import { searchAll, searchRecords } from '@/api/search'
import type {
  RecordItem,
  SearchBookHit,
  SearchEventHit,
  SearchPersonHit,
  SearchRecordHit,
  SearchResult,
} from '@/api/types'
import AppNav from '@/components/AppNav.vue'
import EmptyState from '@/components/EmptyState.vue'
import EventSeal from '@/components/EventSeal.vue'
import PersonAvatar from '@/components/PersonAvatar.vue'
import RecordRow from '@/components/RecordRow.vue'
import Skeleton from '@/components/Skeleton.vue'
import { usePagedList } from '@/composables/usePagedList'
import { useDictStore } from '@/stores/dict'
import { paletteOf } from '@/styles/palette'
import { formatDate } from '@/utils/date'
import { formatCent, yuanToCent } from '@/utils/money'
import { back as goBack, go, toast } from '@/utils/nav'

type GroupKey = 'persons' | 'events' | 'books'

const RECENT_KEY = 'harmony.search.recent'
const TRY_KEYWORDS = ['zs', '500', '11年', 'p1-10']

const dict = useDictStore()
const keyword = ref('')
const inputFocus = ref(false)
const loading = ref(false)
const error = ref('')
const result = ref<SearchResult | null>(null)
const mode = ref<'results' | 'records'>('results')
const activeSearch = ref('')
const recent = ref<string[]>([])
const expanded = ref<Record<GroupKey, boolean>>({ persons: false, events: false, books: false })

let searchTimer: ReturnType<typeof setTimeout> | null = null
let focusTimer: ReturnType<typeof setTimeout> | null = null
let searchSeq = 0

const {
  items: recordItems,
  total: recordTotal,
  loading: recordLoading,
  finished: recordFinished,
  error: recordError,
  load: loadRecordPage,
  loadMore: loadMoreRecords,
} = usePagedList<SearchRecordHit>(
  (page, pageSize) => searchRecords(activeSearch.value, page, pageSize),
  20,
)

const highlightTerm = computed(() => {
  const data = result.value
  if (!data) return ''
  switch (data.intent) {
    case 'text':
      return data.query
    case 'year':
    case 'year_amount':
      return String(completeYear(data.query.match(/\d{2,4}/)?.[0] ?? ''))
    default:
      return ''
  }
})

const resultTotal = computed(() => {
  const data = result.value
  if (!data) return 0
  return data.persons.total + data.events.total + data.books.total + data.records.total
})

const personGroup = computed(() => result.value?.persons ?? { total: 0, items: [] as SearchPersonHit[] })
const eventGroup = computed(() => result.value?.events ?? { total: 0, items: [] as SearchEventHit[] })
const bookGroup = computed(() => result.value?.books ?? { total: 0, items: [] as SearchBookHit[] })
const recordHits = computed(() => result.value?.records.items ?? [])
const recordsResultTotal = computed(() => result.value?.records.total ?? 0)
const recordGroupVisible = computed(() => recordsResultTotal.value > 0)

const personList = computed(() => visibleItems(personGroup.value.items, expanded.value.persons))
const eventList = computed(() => visibleItems(eventGroup.value.items, expanded.value.events))
const bookList = computed(() => visibleItems(bookGroup.value.items, expanded.value.books))

const intentText = computed(() => {
  const data = result.value
  if (!data) return ''
  const query = data.query
  const amountMatch = query.match(/(\d+(?:\.\d+)?)/)
  const amountLabel = amountMatch ? formatCent(yuanToCent(amountMatch[1])) : query
  switch (data.intent) {
    case 'amount':
      return `按金额 ${amountLabel} 搜索`
    case 'amount_range': {
      const range = query.match(/(\d+(?:\.\d+)?)\s*(?:-|~|～|—|至)\s*(\d+(?:\.\d+)?)/)
      if (!range) return `按金额区间 ${query} 搜索`
      return `按金额 ${formatCent(yuanToCent(range[1]))}–${formatCent(yuanToCent(range[2]))} 搜索`
    }
    case 'year': {
      const year = completeYear(query.match(/\d{2,4}/)?.[0] ?? '')
      return year ? `按年份 ${year} 年搜索` : `按年份 ${query} 搜索`
    }
    case 'year_amount': {
      const year = completeYear(query.match(/\d{2,4}/)?.[0] ?? '')
      return `按年份 ${year} 年 · 金额 ${amountLabel} 搜索`
    }
    case 'page': {
      const page = query.replace(/^第/, '').replace(/页$/, '').toUpperCase()
      return `按纸本页码 ${page} 搜索`
    }
    default:
      return `搜索“${query}”`
  }
})

function visibleItems<T>(items: T[], isExpanded: boolean): T[] {
  return isExpanded ? items : items.slice(0, 5)
}

function completeYear(text: string): number {
  const value = Number(text)
  if (!value) return 0
  if (text.length >= 4) return value
  const current = new Date().getFullYear()
  return value <= current % 100 ? 2000 + value : 1900 + value
}

function loadRecent(): string[] {
  try {
    const stored = uni.getStorageSync(RECENT_KEY)
    if (!Array.isArray(stored)) return []
    return stored.filter((item): item is string => typeof item === 'string').slice(0, 10)
  } catch {
    return []
  }
}

function remember(query: string): void {
  const text = query.trim()
  if (!text) return
  recent.value = [text, ...recent.value.filter((item) => item !== text)].slice(0, 10)
  try {
    uni.setStorageSync(RECENT_KEY, recent.value)
  } catch {
    // 本地存储不可用时不影响搜索。
  }
}

function clearRecent(): void {
  recent.value = []
  try {
    uni.removeStorageSync(RECENT_KEY)
  } catch {
    // 忽略存储异常。
  }
}

function cancelTimer(): void {
  if (searchTimer) {
    clearTimeout(searchTimer)
    searchTimer = null
  }
}

function onInput(event: Event): void {
  keyword.value = (event as unknown as { detail: { value: string } }).detail.value
  mode.value = 'results'
  cancelTimer()
  if (!keyword.value.trim()) {
    searchSeq += 1
    result.value = null
    activeSearch.value = ''
    loading.value = false
    error.value = ''
    return
  }
  searchTimer = setTimeout(() => {
    void runSearch()
  }, 300)
}

async function runSearch(): Promise<void> {
  const query = keyword.value.trim()
  if (!query) return
  cancelTimer()
  const seq = ++searchSeq
  loading.value = true
  error.value = ''
  try {
    const data = await searchAll(query, 5)
    if (seq !== searchSeq) return
    result.value = data
    activeSearch.value = query
    expanded.value = { persons: false, events: false, books: false }
  } catch (err) {
    if (seq !== searchSeq) return
    result.value = null
    error.value = err instanceof Error ? err.message : '搜索失败'
    toast(error.value)
  } finally {
    if (seq === searchSeq) loading.value = false
  }
}

function submitSearch(): void {
  const query = keyword.value.trim()
  if (!query) return
  remember(query)
  void runSearch()
}

function useKeyword(text: string): void {
  keyword.value = text
  remember(text)
  void runSearch()
}

function clearKeyword(): void {
  keyword.value = ''
  mode.value = 'results'
  cancelTimer()
  searchSeq += 1
  result.value = null
  activeSearch.value = ''
  loading.value = false
  error.value = ''
  refocus()
}

function refocus(): void {
  inputFocus.value = false
  nextTick(() => {
    inputFocus.value = true
  })
}

async function expandGroup(key: GroupKey): Promise<void> {
  const data = result.value
  if (!data) return
  if (expanded.value[key]) {
    expanded.value = { ...expanded.value, [key]: false }
    return
  }
  const cap = Math.min(data[key].total, 20)
  if (data[key].items.length < cap) {
    const query = data.query
    try {
      const fresh = await searchAll(query, 20)
      if (result.value?.query !== query) return
      if (key === 'persons') result.value = { ...data, persons: fresh.persons }
      if (key === 'events') result.value = { ...data, events: fresh.events }
      if (key === 'books') result.value = { ...data, books: fresh.books }
    } catch (err) {
      toast(err instanceof Error ? err.message : '加载失败')
      return
    }
  }
  expanded.value = { ...expanded.value, [key]: true }
}

function segments(text: string): { text: string; hit: boolean }[] {
  const term = highlightTerm.value.trim()
  if (!text || !term) return [{ text, hit: false }]
  const lower = text.toLowerCase()
  const needle = term.toLowerCase()
  const parts: { text: string; hit: boolean }[] = []
  let cursor = 0
  while (cursor < text.length) {
    const index = lower.indexOf(needle, cursor)
    if (index < 0) break
    if (index > cursor) parts.push({ text: text.slice(cursor, index), hit: false })
    parts.push({ text: text.slice(index, index + needle.length), hit: true })
    cursor = index + needle.length
  }
  if (!parts.length) return [{ text, hit: false }]
  if (cursor < text.length) parts.push({ text: text.slice(cursor), hit: false })
  return parts
}

function eventDateLabel(value: string): string {
  return value ? formatDate(value.slice(0, 10), 'day') : '日期未定'
}

function bookColor(code: string): string {
  return paletteOf(dict.categoryColor(code)).main
}

function openPerson(id: number): void {
  remember(activeSearch.value || keyword.value)
  go(`/sub-person/detail?id=${id}`)
}

function openEvent(id: number): void {
  remember(activeSearch.value || keyword.value)
  go(`/sub-event/detail?id=${id}`)
}

function openBook(id: number): void {
  remember(activeSearch.value || keyword.value)
  go(`/sub-book/detail?id=${id}`)
}

function openRecord(record: { id: number }): void {
  remember(activeSearch.value || keyword.value)
  go(`/sub-record/detail?id=${record.id}`)
}

function openRecords(): void {
  const data = result.value
  if (!data) return
  remember(data.query)
  activeSearch.value = data.query
  mode.value = 'records'
  recordItems.value = []
  recordTotal.value = 0
  void loadRecordPage(true)
}

function backToResults(): void {
  mode.value = 'results'
}

function rowRecord(hit: SearchRecordHit): RecordItem {
  if (hit.matchField === 'paperNote') {
    return { ...hit, paperNote: '' }
  }
  return hit
}

function matchTag(hit: SearchRecordHit): string {
  if (hit.matchField === 'paperNote') return '注'
  if (hit.matchField === 'giftName') return '礼品'
  if (hit.matchField === 'remark') return '备注'
  return ''
}

function matchText(hit: SearchRecordHit): string {
  if (hit.matchField === 'paperNote') return hit.paperNote
  if (hit.matchField === 'giftName') return hit.giftName
  if (hit.matchField === 'remark') return hit.remark
  return ''
}

onLoad((query) => {
  recent.value = loadRecent()
  if (query?.q) {
    keyword.value = String(query.q)
    void runSearch()
  }
  focusTimer = setTimeout(() => {
    inputFocus.value = true
  }, 120)
  dict.ensure().catch(() => undefined)
})

onReachBottom(() => {
  if (mode.value === 'records') {
    void loadMoreRecords()
  }
})

onUnload(() => {
  cancelTimer()
  if (focusTimer) {
    clearTimeout(focusTimer)
    focusTimer = null
  }
})
</script>

<template>
  <view class="screen">
    <AppNav title="搜索" right="取消" @right="goBack" />

    <view class="searchbar page">
      <view class="search-box">
        <text class="search-box__icon">⌕</text>
        <input
          class="search-box__input"
          :value="keyword"
          :focus="inputFocus"
          type="text"
          confirm-type="search"
          placeholder="搜亲友、事项、礼簿、备注"
          @input="onInput"
          @confirm="submitSearch"
        />
        <text v-if="keyword" class="searchbar__clear" @click="clearKeyword">×</text>
      </view>
    </view>

    <view v-if="mode === 'records'" class="page">
      <view class="listhead">
        <text class="listhead__back" @click="backToResults">‹ 返回搜索结果</text>
        <text class="listhead__meta">{{ intentText }} · {{ recordTotal }} 条</text>
      </view>

      <Skeleton v-if="recordLoading && !recordItems.length" type="list" />

      <view v-else-if="recordError" class="state">
        <text class="empty-tip">{{ recordError }}</text>
        <view class="btn btn--secondary btn--small state__retry" @click="loadRecordPage(true)">重新加载</view>
      </view>

      <EmptyState
        v-else-if="!recordItems.length"
        title="没有找到记录"
        description="换个关键词，或者试试金额、年份、页码"
      />

      <view v-else class="card records">
        <view v-for="hit in recordItems" :key="hit.id" class="record-item">
          <RecordRow :record="rowRecord(hit)" @tap="openRecord" />
          <view v-if="matchText(hit)" class="hit-note">
            <text class="hit-note__tag">{{ matchTag(hit) }}</text>
            <text class="hit-note__text">
              <text v-for="(part, index) in segments(matchText(hit))" :key="index" :class="{ hl: part.hit }">{{ part.text }}</text>
            </text>
          </view>
        </view>
        <text v-if="recordFinished" class="loading-tip">没有更多了</text>
        <text v-else-if="recordLoading" class="loading-tip">加载中…</text>
      </view>
    </view>

    <template v-else>
      <view v-if="!keyword.trim()" class="page intro">
        <view class="intro__block">
          <view class="intro__head">
            <text class="intro__title">最近搜索</text>
            <text v-if="recent.length" class="intro__clear" @click="clearRecent">清除</text>
          </view>
          <view v-if="recent.length" class="chips">
            <text v-for="item in recent" :key="item" class="chip chip--flat" @click="useKeyword(item)">
              {{ item }}
            </text>
          </view>
          <text v-else class="caption">搜过的关键词会留在这里，最多 10 条</text>
        </view>

        <view class="intro__block">
          <view class="intro__head">
            <text class="intro__title">试试</text>
          </view>
          <view class="chips">
            <text v-for="item in TRY_KEYWORDS" :key="item" class="chip chip--flat" @click="useKeyword(item)">
              {{ item }}
            </text>
          </view>
          <text class="caption intro__hint">姓名、拼音首字母、金额、年份、纸本页码都能搜</text>
        </view>
      </view>

      <view v-else class="page results">
        <view v-if="result" class="intent">
          <text class="intent__text">{{ intentText }}</text>
          <text class="intent__count">· {{ resultTotal }} 条</text>
        </view>

        <text v-if="loading" class="loading-tip">正在找…</text>

        <view v-if="error" class="state">
          <text class="empty-tip">{{ error }}</text>
          <view class="btn btn--secondary btn--small state__retry" @click="runSearch()">重试</view>
        </view>

        <EmptyState
          v-else-if="result && resultTotal === 0"
          title="没有找到"
          description="换个关键词，或者试试金额、年份、页码"
        />

        <template v-else-if="result">
          <view v-if="personGroup.total" class="card group">
            <view class="group__head">
              <text class="group__title">亲友</text>
              <text class="group__more" @click="expandGroup('persons')">
                {{ expanded.persons ? '收起' : `全部 ${personGroup.total} ›` }}
              </text>
            </view>
            <view
              v-for="person in personList"
              :key="person.id"
              class="hit-row"
              @click="openPerson(person.id)"
            >
              <PersonAvatar :id="person.id" :name="person.name" :size="36" />
              <view class="hit-row__main">
                <text class="hit-row__name">
                  <text v-for="(part, index) in segments(person.name)" :key="index" :class="{ hl: part.hit }">{{ part.text }}</text>
                </text>
                <text class="caption">{{ person.relationLabel }} · {{ person.recordCount }} 笔</text>
              </view>
              <text class="hit-row__arrow">›</text>
            </view>
            <text v-if="expanded.persons && personGroup.total > personList.length" class="group__note">
              只显示前 20 个，试试更具体的关键词
            </text>
          </view>

          <view v-if="eventGroup.total" class="card group">
            <view class="group__head">
              <text class="group__title">事项</text>
              <text class="group__more" @click="expandGroup('events')">
                {{ expanded.events ? '收起' : `全部 ${eventGroup.total} ›` }}
              </text>
            </view>
            <view
              v-for="event in eventList"
              :key="event.id"
              class="hit-row"
              @click="openEvent(event.id)"
            >
              <EventSeal :type="event.type" size="s" />
              <view class="hit-row__main">
                <text class="hit-row__name">
                  <text v-for="(part, index) in segments(event.name)" :key="index" :class="{ hl: part.hit }">{{ part.text }}</text>
                </text>
                <text class="caption">{{ event.typeLabel }} · {{ eventDateLabel(event.eventDate) }}</text>
              </view>
              <text class="hit-row__arrow">›</text>
            </view>
            <text v-if="expanded.events && eventGroup.total > eventList.length" class="group__note">
              只显示前 20 个，试试更具体的关键词
            </text>
          </view>

          <view v-if="bookGroup.total" class="card group">
            <view class="group__head">
              <text class="group__title">礼簿</text>
              <text class="group__more" @click="expandGroup('books')">
                {{ expanded.books ? '收起' : `全部 ${bookGroup.total} ›` }}
              </text>
            </view>
            <view v-for="book in bookList" :key="book.id" class="hit-row" @click="openBook(book.id)">
              <view class="book-dot" :style="{ background: bookColor(book.category) }" />
              <view class="hit-row__main">
                <text class="hit-row__name">
                  <text v-for="(part, index) in segments(book.name)" :key="index" :class="{ hl: part.hit }">{{ part.text }}</text>
                </text>
                <text class="caption">{{ book.categoryLabel }} · {{ book.recordCount }} 笔</text>
              </view>
              <text class="hit-row__arrow">›</text>
            </view>
            <text v-if="expanded.books && bookGroup.total > bookList.length" class="group__note">
              只显示前 20 个，试试更具体的关键词
            </text>
          </view>

          <view v-if="recordGroupVisible" class="card group records">
            <view class="group__head">
              <text class="group__title">记录（备注、礼品）</text>
              <text class="group__more" @click="openRecords">全部 {{ recordsResultTotal }} ›</text>
            </view>
            <view v-for="hit in recordHits" :key="hit.id" class="record-item">
              <RecordRow :record="rowRecord(hit)" @tap="openRecord" />
              <view v-if="matchText(hit)" class="hit-note">
                <text class="hit-note__tag">{{ matchTag(hit) }}</text>
                <text class="hit-note__text">
                  <text v-for="(part, index) in segments(matchText(hit))" :key="index" :class="{ hl: part.hit }">{{ part.text }}</text>
                </text>
              </view>
            </view>
          </view>
        </template>
      </view>
    </template>
  </view>
</template>

<style lang="scss" scoped>
.searchbar {
  padding-top: 4px;
  padding-bottom: 8px;
}

.searchbar__clear {
  flex: none;
  width: 24px;
  text-align: center;
  font-size: 20px;
  line-height: 1;
  color: var(--c-ink-3);
}

.intro {
  padding-top: 12px;
}

.intro__block + .intro__block {
  margin-top: 28px;
}

.intro__head {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  margin-bottom: 10px;
}

.intro__title {
  font-size: 15px;
  font-weight: 600;
  color: var(--c-ink-1);
}

.intro__clear {
  font-size: 13px;
  color: var(--c-ink-3);
}

.intro__hint {
  display: block;
  margin-top: 10px;
}

.chips {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.intent {
  padding: 10px 2px 4px;
}

.intent__text {
  font-size: 14px;
  font-weight: 500;
  color: var(--c-ink-1);
}

.intent__count {
  margin-left: 4px;
  font-size: 13px;
  color: var(--c-ink-3);
}

.results {
  padding-bottom: 24px;
}

.group {
  margin-top: 12px;
  overflow: hidden;
}

.group__head {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  padding: 14px 16px 8px;
}

.group__title {
  font-size: 15px;
  font-weight: 600;
  color: var(--c-ink-1);
}

.group__more {
  font-size: 13px;
  color: var(--c-ink-3);
}

.group__note {
  display: block;
  padding: 0 16px 10px;
  font-size: 12px;
  line-height: 16px;
  color: var(--c-ink-3);
}

.hit-row {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 10px 16px;
}

.hit-row + .hit-row {
  box-shadow: inset 0 1px 0 var(--c-divider);
}

.hit-row__main {
  flex: 1;
  min-width: 0;
}

.hit-row__name {
  display: block;
  font-size: 15px;
  line-height: 22px;
  color: var(--c-ink-1);
}

.hit-row__arrow {
  flex: none;
  font-size: 18px;
  color: var(--c-ink-4);
}

.book-dot {
  flex: none;
  width: 12px;
  height: 12px;
  margin-left: 8px;
  border-radius: var(--r-full);
}

.hl {
  color: var(--c-primary);
  font-weight: 600;
}

.record-item + .record-item {
  box-shadow: inset 0 1px 0 var(--c-divider);
}

.hit-note {
  display: flex;
  align-items: flex-start;
  gap: 6px;
  padding: 0 16px 10px 68px;
  background: var(--c-surface);
}

.hit-note__tag {
  flex: none;
  height: 18px;
  margin-top: 1px;
  padding: 0 5px;
  border-radius: var(--r-xs);
  background: var(--c-fill);
  color: var(--c-ink-3);
  font-size: 11px;
  line-height: 18px;
}

.hit-note__text {
  font-size: 13px;
  line-height: 20px;
  color: var(--c-ink-2);
  word-break: break-all;
}

.listhead {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  padding: 8px 0 12px;
}

.listhead__back {
  font-size: 14px;
  color: var(--c-primary);
}

.listhead__meta {
  font-size: 13px;
  color: var(--c-ink-3);
}

.records {
  overflow: hidden;
}

.state {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding-top: 8px;
}

.state__retry {
  margin-top: 4px;
}
</style>
