<script setup lang="ts">
import { onShow, onUnload } from '@dcloudio/uni-app'
import { computed, ref } from 'vue'

import { fetchYearStats, fetchYears } from '@/api/stats'
import type { BookCategoryStat, BookStat, EventTypeStat, MonthStat, YearStats } from '@/api/types'
import AppNav from '@/components/AppNav.vue'
import EmptyState from '@/components/EmptyState.vue'
import EventSeal from '@/components/EventSeal.vue'
import Money from '@/components/Money.vue'
import PersonAvatar from '@/components/PersonAvatar.vue'
import SectionCard from '@/components/SectionCard.vue'
import Skeleton from '@/components/Skeleton.vue'
import { useDictStore } from '@/stores/dict'
import { paletteOf } from '@/styles/palette'
import { go, toast } from '@/utils/nav'

const currentYear = new Date().getFullYear()

const dict = useDictStore()
const years = ref<number[]>([])
const year = ref(currentYear)
const stats = ref<YearStats | null>(null)
const loading = ref(false)
const error = ref('')
const showSkeleton = ref(false)
const selectedMonth = ref(0)
const yearSheet = ref(false)

let skeletonTimer: ReturnType<typeof setTimeout> | null = null
let initialized = false
let statsSeq = 0

const yearIndex = computed(() => years.value.indexOf(year.value))
const canPrev = computed(() => yearIndex.value > 0)
const canNext = computed(() => yearIndex.value >= 0 && yearIndex.value < years.value.length - 1)
const yearOptions = computed(() => [...years.value].reverse())

const monthly = computed<MonthStat[]>(() => {
  const map = new Map<number, MonthStat>()
  for (const item of stats.value?.monthly ?? []) {
    map.set(item.month, item)
  }
  return Array.from({ length: 12 }, (_, index) => map.get(index + 1) ?? {
    month: index + 1,
    inCent: 0,
    outCent: 0,
    recordCount: 0,
  })
})

const monthMax = computed(() => {
  let max = 1
  for (const item of monthly.value) {
    max = Math.max(max, item.inCent, item.outCent)
  }
  return max
})

const categoryRows = computed<BookCategoryStat[]>(() =>
  [...(stats.value?.byBookCategory ?? [])].sort((a, b) => b.cent - a.cent),
)
const categoryMax = computed(() => {
  let max = 1
  for (const item of categoryRows.value) {
    max = Math.max(max, item.cent)
  }
  return max
})

const eventMax = computed(() => {
  let max = 1
  for (const item of stats.value?.byEventType ?? []) {
    max = Math.max(max, item.cent)
  }
  return max
})

const selectedMonthData = computed<MonthStat | null>(
  () => monthly.value.find((item) => item.month === selectedMonth.value) ?? null,
)

const topPersons = computed(() => stats.value?.topPersons ?? [])
const bookStats = computed(() => stats.value?.byBook ?? [])
const eventTypeStats = computed(() =>
  [...(stats.value?.byEventType ?? [])].sort((a, b) => b.cent - a.cent),
)

function startSkeleton(): void {
  stopSkeleton()
  skeletonTimer = setTimeout(() => {
    if (loading.value) showSkeleton.value = true
  }, 300)
}

function stopSkeleton(): void {
  if (skeletonTimer) {
    clearTimeout(skeletonTimer)
    skeletonTimer = null
  }
  showSkeleton.value = false
}

async function loadStats(targetYear: number, silent = false): Promise<void> {
  const seq = ++statsSeq
  loading.value = true
  error.value = ''
  if (!silent) startSkeleton()
  try {
    const data = await fetchYearStats(targetYear)
    if (seq !== statsSeq) return
    stats.value = data
    selectedMonth.value = 0
  } catch (err) {
    if (seq !== statsSeq) return
    error.value = err instanceof Error ? err.message : '统计加载失败'
    toast(error.value)
  } finally {
    if (seq === statsSeq) {
      loading.value = false
      stopSkeleton()
    }
  }
}

async function init(): Promise<void> {
  error.value = ''
  let dataYears: number[] = []
  try {
    const list = await fetchYears()
    dataYears = [...list].sort((a, b) => a - b)
  } catch (err) {
    toast(err instanceof Error ? err.message : '年份加载失败')
  }
  years.value = Array.from(new Set([...dataYears, currentYear])).sort((a, b) => a - b)
  year.value = dataYears.includes(currentYear)
    ? currentYear
    : (dataYears[dataYears.length - 1] ?? currentYear)
  await loadStats(year.value)
}

function step(delta: number): void {
  const index = yearIndex.value + delta
  if (index < 0 || index >= years.value.length) return
  year.value = years.value[index]
  void loadStats(year.value)
}

function pickYear(value: number): void {
  year.value = value
  yearSheet.value = false
  void loadStats(value)
}

function selectMonth(month: number): void {
  selectedMonth.value = selectedMonth.value === month ? 0 : month
}

function barHeight(cent: number): string {
  if (cent <= 0) return '0%'
  return `${Math.max((cent / monthMax.value) * 100, 3)}%`
}

function categoryWidth(cent: number): string {
  if (cent <= 0) return '0%'
  return `${Math.max((cent / categoryMax.value) * 100, 4)}%`
}

function typeWidth(cent: number): string {
  if (cent <= 0) return '0%'
  return `${Math.max((cent / eventMax.value) * 100, 4)}%`
}

function bookColor(book: BookStat): string {
  return paletteOf(dict.categoryColor(book.book.category)).main
}

const yearRange = computed(() => ({
  from: `${year.value}-01-01`,
  to: `${year.value}-12-31`,
}))

function openCategoryRow(item: BookCategoryStat): void {
  const params = `bookCategory=${encodeURIComponent(item.category)}&from=${yearRange.value.from}&to=${yearRange.value.to}`
  go(`/sub-record/list?${params}`)
}

function openEventTypeRow(item: EventTypeStat): void {
  const params = `eventType=${encodeURIComponent(item.type)}&from=${yearRange.value.from}&to=${yearRange.value.to}`
  go(`/sub-record/list?${params}`)
}

function openPerson(id: number): void {
  go(`/sub-person/detail?id=${id}`)
}

function openBook(id: number): void {
  go(`/sub-book/detail?id=${id}`)
}

onShow(() => {
  dict.ensure().catch(() => undefined)
  if (!initialized) {
    initialized = true
    void init()
    return
  }
  void loadStats(year.value, true)
})

onUnload(stopSkeleton)
</script>

<template>
  <view class="screen">
    <AppNav title="统计" />

    <view class="page">
      <view class="yearbar">
        <text class="yearbar__btn" :class="{ 'yearbar__btn--off': !canPrev }" @click="step(-1)">‹</text>
        <text class="yearbar__year" @click="yearSheet = years.length > 0">
          {{ year }}<text class="yearbar__unit"> 年</text>
        </text>
        <text class="yearbar__btn" :class="{ 'yearbar__btn--off': !canNext }" @click="step(1)">›</text>
      </view>

      <Skeleton v-if="showSkeleton" type="list" />

      <view v-else-if="error && !stats" class="state">
        <text class="empty-tip">{{ error }}</text>
        <view class="btn btn--secondary btn--small" @click="init">重新加载</view>
      </view>

      <EmptyState
        v-else-if="!stats"
        title="还没有统计"
        description="先记一笔，再来看看这一年的往来"
        action-text="去记一笔"
        @action="go('/sub-record/form')"
      />

      <EmptyState
        v-else-if="stats.recordCount === 0"
        title="这一年还没有记录"
        description="换个年份看看，或者先记一笔"
        action-text="去记一笔"
        @action="go('/sub-record/form')"
      />

      <template v-else>
        <view class="card summary">
          <view class="summary__row">
            <view class="summary__col">
              <text class="summary__label">总收</text>
              <Money :cent="stats.inCent" direction="in" size="l" compact />
            </view>
            <view class="summary__col">
              <text class="summary__label">总送</text>
              <Money :cent="stats.outCent" direction="out" size="l" compact />
            </view>
          </view>
          <view class="summary__meta">
            <text class="summary__label">差额</text>
            <Money :cent="stats.netCent" size="s" sign />
            <text class="summary__dot">·</text>
            <text class="summary__meta-text">{{ stats.recordCount }} 笔</text>
            <text class="summary__dot">·</text>
            <text class="summary__meta-text">{{ stats.personCount }} 人</text>
          </view>
        </view>

        <SectionCard title="月度往来">
          <view class="chart">
            <view class="chart__legend">
              <view class="chart__legend-item">
                <view class="legend-dot legend-dot--in" />
                <text>收</text>
              </view>
              <view class="chart__legend-item">
                <view class="legend-dot legend-dot--out" />
                <text>送</text>
              </view>
            </view>
            <view class="chart__months">
              <view
                v-for="item in monthly"
                :key="item.month"
                class="chart__month"
                @click="selectMonth(item.month)"
              >
                <view class="chart__half chart__half--top">
                  <view
                    v-if="item.inCent > 0"
                    class="chart__bar chart__bar--in"
                    :style="{ height: barHeight(item.inCent) }"
                  />
                </view>
                <view class="chart__axis" :class="{ 'chart__axis--on': selectedMonth === item.month }" />
                <view class="chart__half chart__half--bottom">
                  <view
                    v-if="item.outCent > 0"
                    class="chart__bar chart__bar--out"
                    :style="{ height: barHeight(item.outCent) }"
                  />
                </view>
                <text class="chart__label" :class="{ 'chart__label--on': selectedMonth === item.month }">
                  {{ item.month }}
                </text>
              </view>
            </view>
            <view class="chart__detail">
              <template v-if="selectedMonthData">
                <text class="chart__detail-title">{{ selectedMonthData.month }} 月</text>
                <view class="chart__detail-item">
                  <text class="caption">收</text>
                  <Money :cent="selectedMonthData.inCent" direction="in" size="s" />
                </view>
                <view class="chart__detail-item">
                  <text class="caption">送</text>
                  <Money :cent="selectedMonthData.outCent" direction="out" size="s" />
                </view>
                <text class="caption">{{ selectedMonthData.recordCount }} 笔</text>
              </template>
              <text v-else class="caption">点击柱子查看当月明细</text>
            </view>
          </view>
        </SectionCard>

        <SectionCard title="按礼簿类别">
          <view v-if="!categoryRows.length" class="empty-tip">这一年还没有礼簿数据</view>
          <view
            v-for="item in categoryRows"
            :key="item.category"
            class="bar-row bar-row--tap"
            @click="openCategoryRow(item)"
          >
            <view class="bar-row__label">
              <text class="bar-row__name">{{ item.label }}</text>
            </view>
            <view class="bar-row__track">
              <view
                class="bar-row__fill"
                :style="{ width: categoryWidth(item.cent), background: paletteOf(dict.categoryColor(item.category)).main }"
              />
            </view>
            <Money :cent="item.cent" size="s" compact />
          </view>
        </SectionCard>

        <SectionCard title="按事项类型">
          <view v-if="!eventTypeStats.length" class="empty-tip">这一年还没有事项数据</view>
          <view v-for="item in eventTypeStats" :key="item.type" class="type-row" @click="openEventTypeRow(item)">
            <EventSeal :type="item.type" size="s" />
            <view class="type-row__main">
              <view class="type-row__head">
                <text class="type-row__name">{{ item.typeLabel }}</text>
                <text class="caption">{{ item.eventCount }} 场 · {{ item.recordCount }} 笔</text>
              </view>
              <view class="type-row__track">
                <view class="type-row__fill" :style="{ width: typeWidth(item.cent) }" />
              </view>
            </view>
            <Money :cent="item.cent" size="s" compact />
          </view>
        </SectionCard>

        <SectionCard title="往来最多的亲友">
          <view v-if="!topPersons.length" class="empty-tip">这一年还没有亲友往来</view>
          <view
            v-for="(item, index) in topPersons"
            :key="item.person.id"
            class="person-row"
            @click="openPerson(item.person.id)"
          >
            <text class="person-row__rank" :class="{ 'person-row__rank--top': index < 3 }">{{ index + 1 }}</text>
            <PersonAvatar :id="item.person.id" :name="item.person.name" :size="36" />
            <view class="person-row__main">
              <view class="person-row__line">
                <text class="person-row__name">{{ item.person.name }}</text>
                <text class="pill-env">{{ item.person.relationLabel }}</text>
              </view>
              <text class="caption">{{ item.recordCount }} 笔往来</text>
            </view>
            <view class="person-row__money">
              <Money :cent="item.inCent" direction="in" size="s" />
              <Money :cent="item.outCent" direction="out" size="s" />
            </view>
            <text class="row-arrow">›</text>
          </view>
        </SectionCard>

        <SectionCard title="礼簿统计">
          <view v-if="!bookStats.length" class="empty-tip">这一年还没有礼簿数据</view>
          <view v-for="item in bookStats" :key="item.book.id" class="book-row" @click="openBook(item.book.id)">
            <view class="book-row__dot" :style="{ background: bookColor(item) }" />
            <view class="book-row__main">
              <text class="book-row__name">{{ item.book.name }}</text>
              <text class="caption">{{ item.book.categoryLabel }} · {{ item.recordCount }} 笔</text>
            </view>
            <view class="book-row__money">
              <Money :cent="item.inCent" direction="in" size="s" />
              <Money :cent="item.outCent" direction="out" size="s" />
            </view>
            <text class="row-arrow">›</text>
          </view>
        </SectionCard>

        <text class="caption foot">只统计 {{ year }} 年内发生的记录</text>
      </template>
    </view>

    <wd-popup
      v-model="yearSheet"
      position="bottom"
      :safe-area-inset-bottom="true"
      custom-style="border-radius: 20px 20px 0 0;"
    >
      <view class="years">
        <text class="years__title">选择年份</text>
        <scroll-view scroll-y class="years__list">
          <view
            v-for="item in yearOptions"
            :key="item"
            class="years__item"
            :class="{ 'years__item--on': item === year }"
            @click="pickYear(item)"
          >
            <text>{{ item }} 年</text>
            <text v-if="item === year" class="years__check">✓</text>
          </view>
        </scroll-view>
      </view>
    </wd-popup>
  </view>
</template>

<style lang="scss" scoped>
.yearbar {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 18px;
  padding: 6px 0 14px;
}

.yearbar__btn {
  width: 32px;
  height: 32px;
  text-align: center;
  font-size: 22px;
  line-height: 30px;
  color: var(--c-ink-2);
}

.yearbar__btn--off {
  color: var(--c-ink-4);
}

.yearbar__year {
  min-width: 108px;
  text-align: center;
  font-family: $font-num;
  font-size: 22px;
  line-height: 30px;
  font-weight: 600;
  color: var(--c-ink-1);
}

.yearbar__unit {
  font-family: $font-sans;
  font-size: 14px;
  font-weight: 400;
  color: var(--c-ink-3);
}

.summary {
  margin-top: 2px;
  padding: 16px;
}

.summary__row {
  display: flex;
  gap: 12px;
}

.summary__col {
  flex: 1;
  min-width: 0;
}

.summary__label {
  display: block;
  margin-bottom: 2px;
  font-size: 12px;
  color: var(--c-ink-3);
}

.summary__meta {
  display: flex;
  align-items: baseline;
  gap: 6px;
  margin-top: 12px;
  padding-top: 10px;
  box-shadow: inset 0 1px 0 var(--c-divider);
}

.summary__meta .summary__label {
  display: inline;
  margin-bottom: 0;
}

.summary__meta-text {
  font-size: 13px;
  color: var(--c-ink-2);
}

.summary__dot {
  font-size: 12px;
  color: var(--c-ink-4);
}

.chart {
  padding: 14px 14px 12px;
}

.chart__legend {
  display: flex;
  justify-content: flex-end;
  gap: 14px;
  margin-bottom: 6px;
}

.chart__legend-item {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  font-size: 11px;
  color: var(--c-ink-3);
}

.legend-dot {
  width: 8px;
  height: 8px;
  border-radius: var(--r-full);
}

.legend-dot--in {
  background: var(--c-in);
}

.legend-dot--out {
  background: var(--c-out);
}

.chart__months {
  display: flex;
  align-items: stretch;
  gap: 2px;
}

.chart__month {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
}

.chart__half {
  position: relative;
  height: 68px;
  display: flex;
  justify-content: center;
}

.chart__half--top {
  align-items: flex-end;
}

.chart__half--bottom {
  align-items: flex-start;
}

.chart__bar {
  width: 10px;
  border-radius: 3px 3px 0 0;
}

.chart__bar--in {
  background: var(--c-in);
}

.chart__bar--out {
  background: var(--c-out);
  border-radius: 0 0 3px 3px;
}

.chart__axis {
  height: 1px;
  background: var(--c-line);
}

.chart__axis--on {
  background: var(--c-ink-3);
}

.chart__label {
  margin-top: 4px;
  text-align: center;
  font-size: 10px;
  line-height: 14px;
  color: var(--c-ink-3);
}

.chart__label--on {
  color: var(--c-ink-1);
  font-weight: 600;
}

.chart__detail {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 10px;
  min-height: 40px;
  margin-top: 8px;
  padding: 8px 10px;
  border-radius: var(--r-s);
  background: var(--c-fill);
}

.chart__detail-title {
  font-size: 13px;
  font-weight: 600;
  color: var(--c-ink-1);
}

.chart__detail-item {
  display: inline-flex;
  align-items: baseline;
  gap: 4px;
}

.bar-row {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 9px 14px;
}

.bar-row + .bar-row {
  box-shadow: inset 0 1px 0 var(--c-divider);
}

.bar-row__label {
  display: flex;
  align-items: center;
  gap: 5px;
  width: 96px;
  flex: none;
}

.bar-row__name {
  font-size: 13px;
  color: var(--c-ink-1);
}

.bar-row__track {
  flex: 1;
  height: 10px;
  border-radius: var(--r-full);
  background: var(--c-fill);
  overflow: hidden;
}

.bar-row__fill {
  height: 100%;
  border-radius: var(--r-full);
}

.type-row {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 10px 14px;
}

.type-row + .type-row,
.person-row + .person-row,
.book-row + .book-row {
  box-shadow: inset 0 1px 0 var(--c-divider);
}

.type-row__main {
  flex: 1;
  min-width: 0;
}

.type-row__head {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: 8px;
  margin-bottom: 5px;
}

.type-row__name {
  font-size: 13px;
  font-weight: 500;
  color: var(--c-ink-1);
}

.type-row__track {
  height: 8px;
  border-radius: var(--r-full);
  background: var(--c-fill);
  overflow: hidden;
}

.type-row__fill {
  height: 100%;
  border-radius: var(--r-full);
  background: var(--c-primary);
}

.person-row {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px 14px;
}

.person-row__rank {
  width: 16px;
  flex: none;
  text-align: center;
  font-family: $font-num;
  font-size: 13px;
  color: var(--c-ink-3);
}

.person-row__rank--top {
  color: var(--c-primary);
  font-weight: 600;
}

.person-row__main {
  flex: 1;
  min-width: 0;
}

.person-row__line {
  display: flex;
  align-items: center;
  gap: 6px;
}

.person-row__name {
  font-size: 14px;
  font-weight: 500;
  color: var(--c-ink-1);
}

.person-row__money {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  gap: 1px;
}

.book-row {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 11px 14px;
}

.book-row__dot {
  width: 10px;
  height: 10px;
  border-radius: var(--r-full);
  flex: none;
}

.book-row__main {
  flex: 1;
  min-width: 0;
}

.book-row__name {
  display: block;
  font-size: 14px;
  font-weight: 500;
  color: var(--c-ink-1);
}

.book-row__money {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  gap: 1px;
}

.row-arrow {
  flex: none;
  font-size: 17px;
  color: var(--c-ink-4);
}

.foot {
  display: block;
  margin-top: 16px;
  text-align: center;
}

.state {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 6px;
  padding-top: 24px;
}

.years {
  padding: 18px 16px calc(12px + env(safe-area-inset-bottom));
}

.years__title {
  display: block;
  font-size: 17px;
  font-weight: 600;
  color: var(--c-ink-1);
}

.years__list {
  max-height: 50vh;
  margin-top: 10px;
}

.years__item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  height: 48px;
  padding: 0 4px;
  font-size: 15px;
  color: var(--c-ink-1);
}

.years__item + .years__item {
  box-shadow: inset 0 1px 0 var(--c-divider);
}

.years__item--on {
  color: var(--c-primary);
  font-weight: 600;
}

.years__check {
  color: var(--c-primary);
}
</style>
