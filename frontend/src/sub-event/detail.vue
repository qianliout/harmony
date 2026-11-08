<script setup lang="ts">
import { onLoad, onShow } from '@dcloudio/uni-app'
import { computed, ref } from 'vue'

import { fetchEventDetail } from '@/api/event'
import { fetchRecords } from '@/api/record'
import { fetchEventSummary } from '@/api/view'
import type { EventItem, EventPerson, EventPersonLine, EventSummary } from '@/api/types'
import AppNav from '@/components/AppNav.vue'
import DirTag from '@/components/DirTag.vue'
import EmptyState from '@/components/EmptyState.vue'
import EventSeal from '@/components/EventSeal.vue'
import Money from '@/components/Money.vue'
import PersonAvatar from '@/components/PersonAvatar.vue'
import Skeleton from '@/components/Skeleton.vue'
import SummaryStrip from '@/components/SummaryStrip.vue'
import { parseDateParts } from '@/utils/date'
import { go, toast } from '@/utils/nav'

type TabKey = 'in' | 'out' | 'pending'

interface BookStatItem {
  key: string
  label: string
  direction: string
  cent: number
}

interface BookStat {
  bookName: string
  inCent: number
  outCent: number
  items: BookStatItem[]
}

interface ItemStat {
  key: string
  label: string
  direction: string
  cent: number
}

const eventId = ref(0)
const event = ref<EventItem | null>(null)
const summary = ref<EventSummary | null>(null)
const activeTab = ref<TabKey>('in')
const keyword = ref('')
const expandedPerson = ref(0)
const statsShow = ref(false)
const loadingRepay = ref(0)
const loading = ref(true)
const showSkeleton = ref(false)
const error = ref('')

let skeletonTimer: ReturnType<typeof setTimeout> | null = null
let loadingData = false
let initializedTab = false

const persons = computed(() => summary.value?.persons ?? [])
const inPersons = computed(() => persons.value.filter((person) => person.inCent > 0))
const outPersons = computed(() => persons.value.filter((person) => person.outCent > 0))
const pendingPersons = computed(() => persons.value.filter((person) => person.pendingCount > 0))

const tabs = computed(() => [
  { key: 'in' as TabKey, label: '来礼', count: inPersons.value.length },
  { key: 'out' as TabKey, label: '回礼', count: outPersons.value.length },
  { key: 'pending' as TabKey, label: '未回', count: pendingPersons.value.length },
])

const activePersons = computed(() => {
  if (activeTab.value === 'in') return inPersons.value
  if (activeTab.value === 'out') return outPersons.value
  return pendingPersons.value
})

const filteredPersons = computed(() => {
  const q = keyword.value.trim().toLowerCase()
  if (!q) return activePersons.value
  return activePersons.value.filter((person) =>
    [person.person.name, person.person.relationLabel].some((text) => text?.toLowerCase().includes(q)),
  )
})

const eventDateLabel = computed(() => {
  if (!event.value?.eventDate) return '日期未定'
  const parts = parseDateParts(event.value.eventDate)
  if (!parts) return '日期未定'
  if (event.value.datePrecision === 'year') return `${parts.year}年 约`
  if (event.value.datePrecision === 'month') return `${parts.year}年${parts.month}月 约`
  return `${parts.year}年${parts.month}月${parts.day}日`
})

const tabDirection = computed<'in' | 'out'>(() => (activeTab.value === 'out' ? 'out' : 'in'))

const bookStats = computed<BookStat[]>(() => {
  const map = new Map<string, { stat: BookStat; index: Map<string, BookStatItem> }>()
  for (const person of persons.value) {
    for (const line of person.details) {
      const bookName = line.bookName || '未注明礼簿'
      let bucket = map.get(bookName)
      if (!bucket) {
        bucket = {
          stat: { bookName, inCent: 0, outCent: 0, items: [] },
          index: new Map(),
        }
        map.set(bookName, bucket)
      }
      if (line.direction === 'in') bucket.stat.inCent += line.cent
      else bucket.stat.outCent += line.cent
      const key = `${line.itemKindLabel}-${line.direction}`
      let item = bucket.index.get(key)
      if (!item) {
        item = { key, label: line.itemKindLabel, direction: line.direction, cent: 0 }
        bucket.index.set(key, item)
        bucket.stat.items.push(item)
      }
      item.cent += line.cent
    }
  }
  return Array.from(map.values())
    .map((bucket) => bucket.stat)
    .sort((a, b) => b.inCent + b.outCent - (a.inCent + a.outCent))
})

const itemStats = computed<ItemStat[]>(() => {
  const map = new Map<string, ItemStat>()
  for (const person of persons.value) {
    for (const line of person.details) {
      const key = `${line.itemKindLabel}-${line.direction}`
      let item = map.get(key)
      if (!item) {
        item = { key, label: line.itemKindLabel, direction: line.direction, cent: 0 }
        map.set(key, item)
      }
      item.cent += line.cent
    }
  }
  return Array.from(map.values()).sort((a, b) => b.cent - a.cent)
})

function detailsOf(person: EventPerson): EventPersonLine[] {
  return person.details.filter((line) => line.direction === tabDirection.value)
}

function booksOf(person: EventPerson): number {
  return new Set(detailsOf(person).map((line) => line.bookName)).size
}

function recordsOf(person: EventPerson): number {
  return detailsOf(person).reduce((sum, line) => sum + line.recordIds.length, 0)
}

function personMeta(person: EventPerson): string {
  let text = `${recordsOf(person)} 笔 · ${booksOf(person)} 个礼簿`
  if (person.pendingCount) text += ` · 未回礼 ${person.pendingCount} 笔`
  return text
}

function togglePerson(person: EventPerson): void {
  if (detailsOf(person).length <= 1) return
  expandedPerson.value = expandedPerson.value === person.person.id ? 0 : person.person.id
}

function switchTab(tab: TabKey): void {
  activeTab.value = tab
  expandedPerson.value = 0
}

function onKeywordInput(event: Event): void {
  keyword.value = (event as unknown as { detail: { value: string } }).detail.value
}

function clearKeyword(): void {
  keyword.value = ''
}

async function load(): Promise<void> {
  if (!eventId.value || loadingData) return
  loadingData = true
  const firstLoad = !event.value
  if (firstLoad) loading.value = true
  error.value = ''
  if (firstLoad) {
    if (skeletonTimer) clearTimeout(skeletonTimer)
    skeletonTimer = setTimeout(() => {
      showSkeleton.value = true
    }, 300)
  }
  try {
    const [detail, summaryData] = await Promise.all([
      fetchEventDetail(eventId.value),
      fetchEventSummary(eventId.value),
    ])
    event.value = detail
    summary.value = summaryData
    if (!initializedTab) {
      const preferred = detail.host === 'self' ? 'in' : 'out'
      const hasPeople = summaryData.persons.length > 0
      if (!hasPeople) {
        activeTab.value = preferred
      } else if (preferred === 'in' && summaryData.persons.some((person) => person.inCent > 0)) {
        activeTab.value = 'in'
      } else if (preferred === 'out' && summaryData.persons.some((person) => person.outCent > 0)) {
        activeTab.value = 'out'
      } else if (summaryData.persons.some((person) => person.inCent > 0)) {
        activeTab.value = 'in'
      } else if (summaryData.persons.some((person) => person.outCent > 0)) {
        activeTab.value = 'out'
      } else {
        activeTab.value = 'pending'
      }
      initializedTab = true
    }
  } catch (err) {
    error.value = err instanceof Error ? err.message : '加载失败'
  } finally {
    if (skeletonTimer) clearTimeout(skeletonTimer)
    skeletonTimer = null
    showSkeleton.value = false
    loading.value = false
    loadingData = false
  }
}

function retry(): void {
  load()
}

async function fetchAllPendingIds(personId: number): Promise<number[]> {
  const ids: number[] = []
  let page = 1
  for (let guard = 0; guard < 20; guard += 1) {
    const result = await fetchRecords({
      eventId: eventId.value,
      personId,
      direction: 'in',
      status: 'pending',
      page,
      pageSize: 100,
    })
    ids.push(...result.items.map((item) => item.id))
    if (result.items.length < 100 || ids.length >= result.total) break
    page += 1
  }
  return ids
}

async function repay(person: EventPerson): Promise<void> {
  if (loadingRepay.value) return
  loadingRepay.value = person.person.id
  try {
    const ids = await fetchAllPendingIds(person.person.id)
    if (!ids.length) {
      toast('没有未回礼的记录')
      return
    }
    go(`/sub-record/form?mode=repay&originIds=${ids.join(',')}`)
  } catch (err) {
    toast(err instanceof Error ? err.message : '打开回礼失败')
  } finally {
    loadingRepay.value = 0
  }
}

function addRecord(): void {
  go(`/sub-record/form?eventId=${eventId.value}`)
}

function openStats(): void {
  statsShow.value = true
}

function goEdit(): void {
  go(`/sub-event/edit?id=${eventId.value}`)
}

onLoad((query) => {
  eventId.value = Number(query?.id ?? 0)
})

onShow(() => {
  if (!eventId.value) return
  if (loading.value && loadingData) return
  load()
})
</script>

<template>
  <view class="screen screen--with-actions">
    <AppNav title="事项详情" right="编辑" @right="goEdit" />

    <Skeleton v-if="showSkeleton" />

    <view v-else-if="error && !event" class="page">
      <EmptyState title="没能打开这个事项" :description="error" action-text="重试" @action="retry" />
    </view>

    <template v-else-if="!loading && event">
      <view class="page">
        <view class="event-head">
          <EventSeal :type="event.type" size="l" />
          <view class="event-head__main">
            <text class="event-head__name">{{ event.name }}</text>
            <text class="event-head__meta">{{ eventDateLabel }} · {{ event.hostLabel }}</text>
            <text v-if="event.location" class="event-head__meta">{{ event.location }}</text>
            <text v-if="event.hostPersonName" class="event-head__meta">主办人 · {{ event.hostPersonName }}</text>
          </view>
        </view>

        <view class="summary">
          <SummaryStrip
            :in-cent="event.inCent"
            :out-cent="event.outCent"
            :pending-count="event.pendingCount"
            @pending="switchTab('pending')"
          />
        </view>

        <view class="tabs">
          <view
            v-for="tab in tabs"
            :key="tab.key"
            class="tabs__item"
            :class="{ 'tabs__item--on': activeTab === tab.key }"
            @click="switchTab(tab.key)"
          >
            <text>{{ tab.label }}</text>
            <text class="tabs__count">{{ tab.count }}</text>
          </view>
        </view>

        <view class="search-box search">
          <text class="search-box__icon">⌕</text>
          <input
            class="search-box__input"
            :value="keyword"
            placeholder="在本事项中找人"
            confirm-type="search"
            @input="onKeywordInput"
          />
          <text v-if="keyword" class="search__clear" @click="clearKeyword">×</text>
        </view>

        <view v-if="!persons.length" class="state">
          <EmptyState title="还没有记录" description="记下第一笔人情往来吧" action-text="记一笔" @action="addRecord" />
        </view>

        <view v-else-if="!filteredPersons.length" class="state">
          <EmptyState
            :title="keyword ? `没找到「${keyword}」` : '这里还没有人'"
            :description="keyword ? '换个关键词试试' : '换个分段看看'"
            :action-text="keyword ? '清除搜索' : ''"
            @action="clearKeyword"
          />
        </view>

        <view v-else class="person-list card">
          <view v-for="person in filteredPersons" :key="person.person.id" class="person-block">
            <view class="person-row" @click="togglePerson(person)">
              <PersonAvatar :id="person.person.id" :name="person.person.name" :size="40" />
              <view class="person-main">
                <view class="person-head">
                  <text class="person-name">{{ person.person.name }}</text>
                  <text class="person-relation">{{ person.person.relationLabel }}</text>
                </view>
                <text class="person-meta">{{ personMeta(person) }}</text>
              </view>
              <view class="person-right">
                <view v-if="person.inCent" class="person-amount">
                  <text class="person-amount__label">收</text>
                  <Money :cent="person.inCent" direction="in" size="m" />
                </view>
                <view v-if="person.outCent" class="person-amount">
                  <text class="person-amount__label">送</text>
                  <Money :cent="person.outCent" direction="out" size="m" />
                </view>
                <view
                  v-if="activeTab === 'pending' && person.pendingCount"
                  class="repay-btn"
                  @click.stop="repay(person)"
                >
                  {{ loadingRepay === person.person.id ? '…' : '回礼' }}
                </view>
                <text v-else-if="detailsOf(person).length > 1" class="person-expand">
                  {{ expandedPerson === person.person.id ? '收起' : '展开' }}
                </text>
              </view>
            </view>

            <view v-if="expandedPerson === person.person.id" class="person-detail">
              <view v-for="line in detailsOf(person)" :key="`${line.bookName}-${line.itemKindLabel}-${line.direction}`" class="detail-line">
                <view class="detail-line__left">
                  <text class="detail-line__book">{{ line.bookName || '未注明礼簿' }}</text>
                  <text class="detail-line__sep">·</text>
                  <text class="detail-line__kind">{{ line.itemKindLabel }}</text>
                  <DirTag :direction="line.direction" />
                </view>
                <Money :cent="line.cent" :direction="line.direction === 'in' ? 'in' : 'out'" size="s" />
              </view>
            </view>
          </view>
        </view>
      </view>
    </template>

    <view v-if="event" class="bottom-bar">
      <view class="btn btn--secondary bottom-bar__btn" @click="openStats">按礼簿统计</view>
      <view class="btn btn--primary bottom-bar__btn bottom-bar__btn--main" @click="addRecord">记一笔</view>
    </view>

    <wd-popup v-model="statsShow" position="bottom" :safe-area-inset-bottom="true" custom-style="border-radius: 20px 20px 0 0;">
      <view class="stats">
        <view class="stats__head">
          <text class="stats__title">按礼簿统计</text>
          <text class="stats__close" @click="statsShow = false">关闭</text>
        </view>
        <scroll-view scroll-y class="stats__body">
          <text class="stats__section">按名目</text>
          <view class="stats__items">
            <view v-for="item in itemStats" :key="item.key" class="stats__item">
              <view class="stats__item-left">
                <DirTag :direction="item.direction" />
                <text class="stats__item-label">{{ item.label }}</text>
              </view>
              <Money :cent="item.cent" :direction="item.direction === 'in' ? 'in' : 'out'" size="s" />
            </view>
            <text v-if="!itemStats.length" class="stats__empty">还没有记录</text>
          </view>

          <text class="stats__section">按礼簿</text>
          <view v-for="group in bookStats" :key="group.bookName" class="stats__book">
            <view class="stats__book-head">
              <text class="stats__book-name">{{ group.bookName }}</text>
              <view class="stats__book-amounts">
                <Money v-if="group.inCent" :cent="group.inCent" direction="in" size="s" />
                <Money v-if="group.outCent" :cent="group.outCent" direction="out" size="s" />
              </view>
            </view>
            <view v-for="item in group.items" :key="item.key" class="stats__book-item">
              <view class="stats__item-left">
                <DirTag :direction="item.direction" />
                <text class="stats__item-label">{{ item.label }}</text>
              </view>
              <Money :cent="item.cent" :direction="item.direction === 'in' ? 'in' : 'out'" size="s" />
            </view>
          </view>
        </scroll-view>
      </view>
    </wd-popup>
  </view>
</template>

<style lang="scss" scoped>
.screen--with-actions {
  padding-bottom: calc(88px + env(safe-area-inset-bottom));
}

.event-head {
  display: flex;
  align-items: flex-start;
  gap: 14px;
  padding: 6px 2px 4px;
}

.event-head__main {
  flex: 1;
  min-width: 0;
}

.event-head__name {
  display: block;
  font-size: 22px;
  line-height: 30px;
  font-weight: 600;
  color: var(--c-ink-1);
}

.event-head__meta {
  display: block;
  margin-top: 3px;
  font-size: 13px;
  line-height: 18px;
  color: var(--c-ink-3);
}

.summary {
  margin-top: var(--card-gap);
}

.tabs {
  display: flex;
  gap: 8px;
  margin-top: 14px;
}

.tabs__item {
  display: flex;
  align-items: center;
  gap: 4px;
  height: 34px;
  padding: 0 14px;
  border-radius: var(--r-s);
  background: var(--c-surface);
  color: var(--c-ink-2);
  font-size: 14px;
  box-shadow: inset 0 0 0 1px var(--c-line);
}

.tabs__item--on {
  background: var(--c-ink-1);
  color: var(--c-surface);
  box-shadow: none;
  font-weight: 600;
}

.tabs__count {
  font-family: $font-num;
  font-size: 12px;
  opacity: 0.8;
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

.state {
  padding-top: 16px;
}

.person-list {
  margin-top: 10px;
  overflow: hidden;
}

.person-block + .person-block {
  box-shadow: inset 0 1px 0 var(--c-divider);
}

.person-row {
  display: flex;
  align-items: flex-start;
  gap: 12px;
  padding: 12px 16px;
}

.person-main {
  flex: 1;
  min-width: 0;
  padding-top: 1px;
}

.person-head {
  display: flex;
  align-items: center;
  gap: 6px;
}

.person-name {
  font-size: 15px;
  line-height: 22px;
  font-weight: 500;
  color: var(--c-ink-1);
}

.person-relation {
  height: 20px;
  padding: 0 6px;
  border-radius: var(--r-xs);
  background: var(--c-fill);
  color: var(--c-ink-2);
  font-size: 12px;
  line-height: 20px;
}

.person-meta {
  display: block;
  margin-top: 3px;
  font-size: 12px;
  line-height: 17px;
  color: var(--c-ink-3);
}

.person-right {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  gap: 3px;
  flex: none;
}

.person-amount {
  display: flex;
  align-items: baseline;
  gap: 3px;
}

.person-amount__label {
  font-size: 11px;
  color: var(--c-ink-3);
}

.person-expand {
  font-size: 12px;
  color: var(--c-ink-3);
}

.repay-btn {
  height: 28px;
  padding: 0 12px;
  border-radius: var(--r-full);
  background: var(--c-primary-light);
  color: var(--c-primary);
  font-size: 13px;
  line-height: 28px;
  font-weight: 500;
}

.person-detail {
  padding: 2px 16px 12px 68px;
  background: var(--c-surface);
}

.detail-line {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  min-height: 30px;
  font-size: 13px;
}

.detail-line__left {
  display: flex;
  align-items: center;
  gap: 4px;
  min-width: 0;
}

.detail-line__book {
  color: var(--c-ink-2);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.detail-line__sep {
  color: var(--c-ink-4);
}

.detail-line__kind {
  flex: none;
  color: var(--c-ink-3);
}

.stats {
  display: flex;
  flex-direction: column;
  max-height: 82vh;
}

.stats__head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 16px 16px 8px;
}

.stats__title {
  font-size: 17px;
  font-weight: 600;
}

.stats__close {
  font-size: 14px;
  color: var(--c-ink-3);
}

.stats__body {
  height: 58vh;
  padding: 0 16px calc(16px + env(safe-area-inset-bottom));
  box-sizing: border-box;
}

.stats__section {
  display: block;
  margin: 10px 0 6px;
  font-size: 13px;
  color: var(--c-ink-3);
}

.stats__items {
  border-radius: var(--r-s);
  background: var(--c-fill);
  padding: 2px 12px;
}

.stats__item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  min-height: 40px;
}

.stats__item + .stats__item {
  box-shadow: inset 0 1px 0 var(--c-divider);
}

.stats__item-left {
  display: flex;
  align-items: center;
  gap: 6px;
}

.stats__item-label {
  font-size: 14px;
  color: var(--c-ink-2);
}

.stats__book {
  margin-bottom: 10px;
  padding: 10px 12px;
  border-radius: var(--r-s);
  background: var(--c-fill);
}

.stats__book-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  min-height: 30px;
}

.stats__book-name {
  font-size: 14px;
  font-weight: 600;
  color: var(--c-ink-1);
}

.stats__book-amounts {
  display: flex;
  align-items: baseline;
  gap: 8px;
}

.stats__book-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  min-height: 34px;
  padding-left: 4px;
}

.stats__empty {
  display: block;
  padding: 14px 0;
  text-align: center;
  font-size: 13px;
  color: var(--c-ink-3);
}
</style>
