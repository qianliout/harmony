<script setup lang="ts">
import { onShow } from '@dcloudio/uni-app'
import { computed, ref } from 'vue'

import { fetchBooks } from '@/api/book'
import { fetchEventDetail, fetchRecentEvents } from '@/api/event'
import { fetchRecords } from '@/api/record'
import { fetchYears, fetchYearStats } from '@/api/stats'
import type { BookItem, EventItem, PendingList, RecordItem, YearStats } from '@/api/types'
import { fetchPending } from '@/api/view'
import AppNav from '@/components/AppNav.vue'
import BookChips from '@/components/BookChips.vue'
import EmptyState from '@/components/EmptyState.vue'
import EventPicker from '@/components/EventPicker.vue'
import EventSeal from '@/components/EventSeal.vue'
import Money from '@/components/Money.vue'
import RecordRow from '@/components/RecordRow.vue'
import SectionCard from '@/components/SectionCard.vue'
import Skeleton from '@/components/Skeleton.vue'
import { useAuthStore } from '@/stores/auth'
import { useContextStore } from '@/stores/context'
import { paletteOf } from '@/styles/palette'
import { themeVars } from '@/styles/theme'
import { preciseDateLabel, todayString, weekdayOf } from '@/utils/date'
import { go, toast } from '@/utils/nav'

const contextStore = useContextStore()
const auth = useAuthStore()

const greeting = computed(() => {
  const hour = new Date().getHours()
  if (hour < 6) return '夜深了'
  if (hour < 11) return '早上好'
  if (hour < 13) return '中午好'
  if (hour < 18) return '下午好'
  return '晚上好'
})

const todayLabel = computed(() => {
  const today = todayString()
  const [, month, day] = today.split('-').map(Number)
  return `${month}月${day}日 ${weekdayOf(today)}`
})

const years = ref<number[]>([])
const year = ref(new Date().getFullYear())
const yearStats = ref<YearStats | null>(null)
const pending = ref<PendingList | null>(null)
const recentRecords = ref<RecordItem[]>([])
const recentEvents = ref<EventItem[]>([])

const loading = ref(false)
const booted = ref(false)
const statsError = ref('')
const skeleton = ref(false)
const yearSheet = ref(false)

const continuousShow = ref(false)
const eventPicker = ref(false)
const books = ref<BookItem[]>([])
const sheetBookId = ref(0)
const selectedEvent = ref<EventItem | null>(null)

let skeletonTimer: ReturnType<typeof setTimeout> | null = null

const pendingCount = computed(() => pending.value?.totalCount ?? 0)
const pendingCent = computed(() => pending.value?.totalCent ?? 0)
const pendingBadge = computed(() => (pendingCount.value > 99 ? '99+' : String(pendingCount.value || '')))
const yearOptions = computed(() => (years.value.length ? years.value : [new Date().getFullYear()]))
const canStartContinuous = computed(() => Boolean(sheetBookId.value && selectedEvent.value))

function startSkeleton(): void {
  if (booted.value) return
  if (skeletonTimer) clearTimeout(skeletonTimer)
  skeletonTimer = setTimeout(() => {
    skeleton.value = true
  }, 300)
}

function stopSkeleton(): void {
  if (skeletonTimer) {
    clearTimeout(skeletonTimer)
    skeletonTimer = null
  }
  skeleton.value = false
}

async function loadYearStats(target: number): Promise<void> {
  statsError.value = ''
  try {
    yearStats.value = await fetchYearStats(target)
  } catch (err) {
    yearStats.value = null
    statsError.value = err instanceof Error ? err.message : '加载失败'
    toast(statsError.value)
  }
}

async function loadHome(): Promise<void> {
  if (loading.value) return
  loading.value = true
  startSkeleton()
  loadRecents()
  try {
    const [yearList, pendingList] = await Promise.all([fetchYears(), fetchPending('owed')])
    years.value = [...(yearList ?? [])].sort((a, b) => b - a)
    pending.value = pendingList
    const current = new Date().getFullYear()
    if (!years.value.includes(year.value)) {
      year.value = years.value.length ? years.value[0] : current
    }
    await loadYearStats(year.value)
  } catch (err) {
    statsError.value = err instanceof Error ? err.message : '加载失败'
    toast(statsError.value)
  } finally {
    loading.value = false
    booted.value = true
    stopSkeleton()
  }
}

async function loadRecents(): Promise<void> {
  try {
    const page = await fetchRecords({ pageSize: 5 })
    recentRecords.value = page.items
  } catch (err) {
    recentRecords.value = []
    toast(err instanceof Error ? err.message : '加载失败')
  }
  try {
    recentEvents.value = await fetchRecentEvents(6)
  } catch (err) {
    recentEvents.value = []
    toast(err instanceof Error ? err.message : '加载失败')
  }
}

async function chooseYear(value: number): Promise<void> {
  yearSheet.value = false
  if (value === year.value && yearStats.value) return
  year.value = value
  await loadYearStats(value)
}

async function retryStats(): Promise<void> {
  await loadYearStats(year.value)
}

function gateStyle(name: string): Record<string, string> {
  const colors = paletteOf(name)
  return { background: colors.light, color: colors.main }
}

async function openContinuous(): Promise<void> {
  continuousShow.value = true
  selectedEvent.value = null
  try {
    await contextStore.load()
    if (!books.value.length) {
      books.value = await fetchBooks('normal')
    }
    const lastBookId = contextStore.settings.lastBookId
    sheetBookId.value = books.value.some((book) => book.id === lastBookId)
      ? lastBookId
      : books.value[0]?.id ?? 0
    const lastEventId = contextStore.settings.lastEventId
    if (lastEventId) {
      selectedEvent.value = await fetchEventDetail(lastEventId).catch(() => null)
    }
  } catch (err) {
    toast(err instanceof Error ? err.message : '加载礼簿失败')
  }
}

function onEventPicked(event: EventItem): void {
  selectedEvent.value = event
}

function createBook(): void {
  continuousShow.value = false
  go('/sub-book/edit')
}

async function startContinuous(): Promise<void> {
  if (!canStartContinuous.value) {
    toast('先选好礼簿和事项')
    return
  }
  await contextStore.rememberContext(sheetBookId.value, selectedEvent.value?.id ?? 0)
  continuousShow.value = false
  go(`/sub-record/continuous?bookId=${sheetBookId.value}&eventId=${selectedEvent.value?.id ?? 0}`)
}

function openRecord(record: RecordItem): void {
  go(`/sub-record/detail?id=${record.id}`)
}

function openEvent(event: EventItem): void {
  if (!event.id) return
  go(`/sub-event/detail?id=${event.id}`)
}

function openPending(): void {
  go('/sub-misc/pending')
}

function openEventsTab(): void {
  uni.setStorageSync('harmony:book-tab', 'event')
  go('/pages/book/list?tab=event')
}

onShow(() => {
  auth.loadMe().catch(() => undefined)
  loadHome()
})
</script>

<template>
  <wd-config-provider :theme-vars="themeVars">
    <view class="screen screen--tab">
      <AppNav :back="false" title="" />

      <view class="page">
        <view class="hello">
          <view class="hello__main">
            <text class="hello__greeting">{{ greeting }}</text>
            <text class="hello__date">{{ todayLabel }}</text>
          </view>
          <view class="hello__search" @click="go('/sub-misc/search')">
            <text>⌕</text>
          </view>
        </view>

        <Skeleton v-if="skeleton" type="list" />
        <view v-else-if="loading && !booted" class="boot-hold" />

        <template v-else>
          <view class="stat">
            <text class="stat__watermark">礼</text>
            <view class="stat__head" @click="yearSheet = true">
              <text class="stat__year">{{ year }} 年人情往来</text>
              <text class="stat__caret">▾</text>
            </view>
            <view class="stat__nums">
              <view class="stat__num">
                <text class="stat__label">收礼</text>
                <Money :cent="yearStats?.inCent ?? 0" direction="in" size="display" compact />
              </view>
              <view class="stat__num">
                <text class="stat__label">送礼</text>
                <Money :cent="yearStats?.outCent ?? 0" direction="out" size="display" compact />
              </view>
            </view>
            <view class="stat__line" />
            <view class="stat__foot" @click="openPending">
              <text class="stat__foot-label">未回礼 {{ pendingCount }} 笔</text>
              <text class="stat__foot-label"> · </text>
              <Money :cent="pendingCent" size="s" />
              <text class="stat__foot-arrow">›</text>
            </view>
          </view>

          <view v-if="statsError" class="stat-error">
            <text>{{ statsError }}</text>
            <text class="stat-error__retry" @click="retryStats">重试</text>
          </view>

          <view class="actions">
            <view class="btn btn--primary actions__main" @click="go('/sub-record/form')">＋ 记一笔</view>
            <view class="btn btn--secondary actions__sub" @click="openContinuous">连续录入</view>
          </view>

          <view class="gates">
            <view class="gate" @click="openPending">
              <view class="gate__icon" :style="gateStyle('amber')">
                <text>回</text>
                <text v-if="pendingCount" class="gate__badge">{{ pendingBadge }}</text>
              </view>
              <text class="gate__label">未回礼</text>
            </view>
            <view class="gate" @click="openEventsTab">
              <view class="gate__icon" :style="gateStyle('cinnabar')"><text>喜</text></view>
              <text class="gate__label">事项</text>
            </view>
            <view class="gate" @click="go('/sub-misc/stats')">
              <view class="gate__icon" :style="gateStyle('teal')"><text>统</text></view>
              <text class="gate__label">统计</text>
            </view>
            <!-- #ifdef H5 -->
            <view v-if="auth.isAdmin" class="gate" @click="go('/sub-recognize/index')">
              <view class="gate__icon" :style="gateStyle('indigo')"><text>识</text></view>
              <text class="gate__label">识别礼簿</text>
            </view>
            <!-- #endif -->
          </view>

          <SectionCard title="最近记录" link-text="全部" @link="go('/sub-record/list')">
            <RecordRow
              v-for="record in recentRecords"
              :key="record.id"
              :record="record"
              mode="global"
              @tap="openRecord"
            />
            <EmptyState
              v-if="!recentRecords.length"
              title="还没有记录"
              description="记下第一笔人情往来吧"
              action-text="记一笔"
              @action="go('/sub-record/form')"
            />
          </SectionCard>

          <view class="events">
            <view class="events__head">
              <text class="events__title">最近事项</text>
              <text class="events__link" @click="openEventsTab">全部 ›</text>
            </view>
            <scroll-view v-if="recentEvents.length" scroll-x class="events__scroll">
              <view class="events__row">
                <view v-for="event in recentEvents" :key="event.id" class="event-card card" @click="openEvent(event)">
                  <EventSeal :type="event.type" size="m" />
                  <text class="event-card__name">{{ event.name }}</text>
                  <text class="event-card__date">
                    {{ preciseDateLabel(event.eventDate, event.datePrecision) || '日期待补' }}
                  </text>
                  <view class="event-card__sum">
                    <text class="event-card__label">{{ event.inCent > 0 || event.outCent === 0 ? '收' : '送' }}</text>
                    <Money
                      :cent="event.inCent > 0 || event.outCent === 0 ? event.inCent : event.outCent"
                      :direction="event.inCent > 0 || event.outCent === 0 ? 'in' : 'out'"
                      size="s"
                    />
                  </view>
                </view>
              </view>
            </scroll-view>
            <view v-else class="empty-tip">还没有事项</view>
          </view>
        </template>
      </view>

      <wd-popup
        v-model="yearSheet"
        position="bottom"
        :safe-area-inset-bottom="true"
        custom-style="border-radius: 20px 20px 0 0;"
      >
        <view class="ysheet">
          <view class="ysheet__head">
            <text class="ysheet__title">选择年份</text>
            <text class="ysheet__close" @click="yearSheet = false">关闭</text>
          </view>
          <view class="ysheet__chips">
            <text
              v-for="option in yearOptions"
              :key="option"
              class="chip"
              :class="{ 'chip--on': option === year }"
              @click="chooseYear(option)"
            >
              {{ option }} 年
            </text>
          </view>
        </view>
      </wd-popup>

      <wd-popup
        v-model="continuousShow"
        position="bottom"
        :safe-area-inset-bottom="true"
        custom-style="border-radius: 20px 20px 0 0;"
      >
        <view class="csheet">
          <view class="csheet__head">
            <text class="csheet__title">连续录入</text>
            <text class="csheet__close" @click="continuousShow = false">取消</text>
          </view>
          <text class="csheet__label">礼簿</text>
          <view v-if="books.length" class="csheet__books">
            <BookChips v-model="sheetBookId" :books="books" @create="createBook" />
          </view>
          <view v-else class="csheet__hint" @click="createBook">还没有礼簿，先去新建一本 ›</view>
          <view class="field csheet__event" @click="eventPicker = true">
            <text class="field__label">事项</text>
            <text class="field__value" :class="{ 'csheet__placeholder': !selectedEvent }">
              {{ selectedEvent?.name ?? '选择事项' }}
            </text>
            <text class="field__arrow">›</text>
          </view>
          <view
            class="btn btn--primary csheet__go"
            :class="{ 'btn--disabled': !canStartContinuous }"
            @click="startContinuous"
          >
            开始录入
          </view>
        </view>
      </wd-popup>

      <EventPicker v-model:show="eventPicker" @select="onEventPicked" />
    </view>
  </wd-config-provider>
</template>

<style lang="scss" scoped>
.boot-hold {
  height: 320px;
}

.hello {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 4px 2px 16px;
}

.hello__main {
  flex: 1;
  min-width: 0;
}

.hello__greeting {
  display: block;
  font-family: $font-serif;
  font-size: 22px;
  line-height: 30px;
  color: var(--c-ink-1);
}

.hello__date {
  display: block;
  margin-top: 2px;
  font-size: 13px;
  color: var(--c-ink-3);
}

.hello__search {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 34px;
  height: 34px;
  border-radius: var(--r-full);
  background: var(--c-surface);
  color: var(--c-ink-2);
  box-shadow: var(--shadow-card);
  font-size: 18px;
}

.stat {
  position: relative;
  overflow: hidden;
  padding: 16px 18px 12px;
  border-radius: var(--r-l);
  background: linear-gradient(135deg, var(--c-primary) 0%, var(--c-primary-dark) 100%);
  box-shadow: var(--shadow-float);
  /* 卡片上的金额统一用白字。 */
  --c-in: var(--c-surface);
  --c-out: var(--c-surface);
  --c-ink-1: var(--c-surface);
  --c-pending: var(--c-surface);
}

.stat__watermark {
  position: absolute;
  right: -14px;
  bottom: -46px;
  font-family: $font-serif;
  font-size: 120px;
  line-height: 1;
  font-weight: 700;
  color: var(--c-surface);
  opacity: 0.08;
  pointer-events: none;
}

.stat__head {
  position: relative;
  display: flex;
  align-items: center;
  gap: 4px;
}

.stat__year {
  font-size: 13px;
  line-height: 18px;
  color: var(--c-surface);
  opacity: 0.78;
}

.stat__caret {
  font-size: 11px;
  color: var(--c-surface);
  opacity: 0.78;
}

.stat__nums {
  position: relative;
  display: flex;
  gap: 40px;
  margin-top: 10px;
}

.stat__num {
  min-width: 0;
}

.stat__label {
  display: block;
  margin-bottom: 2px;
  font-size: 12px;
  color: var(--c-surface);
  opacity: 0.75;
}

.stat__line {
  position: relative;
  height: 1px;
  margin: 12px 0 10px;
  background: var(--c-surface);
  opacity: 0.18;
}

.stat__foot {
  position: relative;
  display: flex;
  align-items: center;
  gap: 2px;
  min-height: 24px;
}

.stat__foot-label {
  font-size: 13px;
  color: var(--c-surface);
  opacity: 0.85;
}

.stat__foot-arrow {
  margin-left: auto;
  font-size: 18px;
  color: var(--c-surface);
  opacity: 0.7;
}

.stat-error {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-top: 8px;
  padding: 0 4px;
  font-size: 12px;
  color: var(--c-danger);
}

.stat-error__retry {
  color: var(--c-primary);
}

.actions {
  display: flex;
  gap: 10px;
  margin-top: 14px;
}

.actions__main {
  flex: 2;
}

.actions__sub {
  flex: 1;
  font-size: 15px;
}

.gates {
  display: flex;
  gap: 8px;
  margin-top: 16px;
}

.gate {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 6px;
  padding: 10px 4px;
  border-radius: var(--r-m);
  background: var(--c-surface);
  box-shadow: var(--shadow-card);
}

.gate__icon {
  position: relative;
  display: flex;
  align-items: center;
  justify-content: center;
  width: 40px;
  height: 40px;
  border-radius: var(--r-full);
  font-family: $font-serif;
  font-size: 20px;
  font-weight: 600;
}

.gate__badge {
  position: absolute;
  top: -5px;
  right: -7px;
  min-width: 16px;
  height: 16px;
  padding: 0 4px;
  border-radius: var(--r-full);
  border: 1.5px solid var(--c-surface);
  background: var(--c-primary);
  color: var(--c-surface);
  font-family: $font-num;
  font-size: 10px;
  line-height: 13px;
  text-align: center;
  box-sizing: border-box;
}

.gate__label {
  font-size: 12px;
  color: var(--c-ink-2);
}

.events {
  margin-top: 20px;
}

.events__head {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  margin: 0 2px 10px;
}

.events__title {
  font-size: 17px;
  line-height: 24px;
  font-weight: 600;
  color: var(--c-ink-1);
}

.events__link {
  font-size: 13px;
  color: var(--c-ink-3);
}

.events__scroll {
  width: 100%;
  height: 152px;
  white-space: nowrap;
}

.events__row {
  display: flex;
  gap: 10px;
  padding: 2px 2px 6px;
}

.event-card {
  flex: none;
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 4px;
  width: 140px;
  padding: 12px;
  box-sizing: border-box;
}

.event-card__name {
  width: 100%;
  margin-top: 4px;
  font-size: 14px;
  line-height: 20px;
  font-weight: 600;
  color: var(--c-ink-1);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.event-card__date {
  font-size: 12px;
  color: var(--c-ink-3);
}

.event-card__sum {
  display: flex;
  align-items: baseline;
  gap: 3px;
  margin-top: 2px;
}

.event-card__label {
  font-size: 11px;
  color: var(--c-ink-3);
}

.ysheet {
  padding: 16px 16px calc(20px + env(safe-area-inset-bottom));
}

.ysheet__head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 14px;
}

.ysheet__title {
  font-size: 17px;
  font-weight: 600;
}

.ysheet__close {
  font-size: 14px;
  color: var(--c-ink-3);
}

.ysheet__chips {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.csheet {
  padding: 16px 16px calc(20px + env(safe-area-inset-bottom));
}

.csheet__head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 14px;
}

.csheet__title {
  font-size: 17px;
  font-weight: 600;
}

.csheet__close {
  font-size: 14px;
  color: var(--c-ink-3);
}

.csheet__label {
  display: block;
  margin-bottom: 8px;
  font-size: 13px;
  color: var(--c-ink-3);
}

.csheet__books {
  margin-bottom: 12px;
}

.csheet__hint {
  padding: 8px 0 14px;
  font-size: 13px;
  color: var(--c-primary);
}

.csheet__event {
  margin-bottom: 16px;
  padding: 0 12px;
  border-radius: var(--r-s);
  background: var(--c-fill);
}

.csheet__placeholder {
  color: var(--c-ink-3);
}

.csheet__go {
  width: 100%;
}
</style>
