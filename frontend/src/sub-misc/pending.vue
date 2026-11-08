<script setup lang="ts">
import { onShow } from '@dcloudio/uni-app'
import { computed, onUnmounted, ref, watch } from 'vue'

import { fetchRecords, updateRecord } from '@/api/record'
import type { RecordItem, PendingList, PendingPerson } from '@/api/types'
import { fetchPending } from '@/api/view'
import AppNav from '@/components/AppNav.vue'
import ConfirmSheet from '@/components/ConfirmSheet.vue'
import EmptyState from '@/components/EmptyState.vue'
import Money from '@/components/Money.vue'
import PersonAvatar from '@/components/PersonAvatar.vue'
import Skeleton from '@/components/Skeleton.vue'
import { openConfirmSheet } from '@/composables/useConfirmSubmit'
import { useContextStore } from '@/stores/context'
import { relativeSince } from '@/utils/date'
import { go, toast } from '@/utils/nav'
import { formatCent } from '@/utils/money'

type Kind = 'owed' | 'owing'
type SortKey = 'oldest' | 'amount' | 'name'

interface TouchLike {
  touches: { clientX: number; clientY: number }[]
}

const contextStore = useContextStore()

const kind = ref<Kind>('owed')
const sort = ref<SortKey>('oldest')
const sortSheet = ref(false)
const data = ref<PendingList>({ persons: [], personCount: 0, totalCount: 0, totalCent: 0 })
const loading = ref(true)
const showSkeleton = ref(false)
const error = ref('')
const markingId = ref(0)
const initialized = ref(false)
let skeletonTimer: ReturnType<typeof setTimeout> | null = null

const openId = ref(0)
const dragId = ref(0)
const dragX = ref(0)
let startX = 0
let startY = 0
let baseX = 0
let horizontal = false
let suppressClick = false

const tabs = computed(() =>
  [
    { code: 'owed' as const, label: '我该回的' },
    { code: 'owing' as const, label: '该回我的' },
  ].filter((tab) => tab.code === 'owed' || contextStore.settings.showOwing),
)

const sortOptions: { code: SortKey; label: string }[] = [
  { code: 'oldest', label: '最久远' },
  { code: 'amount', label: '金额最大' },
  { code: 'name', label: '姓名' },
]

const sortLabel = computed(() => sortOptions.find((option) => option.code === sort.value)?.label ?? '最久远')

const sortedPersons = computed(() => {
  const rows = [...data.value.persons]
  if (sort.value === 'amount') {
    return rows.sort((a, b) => b.totalCent - a.totalCent)
  }
  if (sort.value === 'name') {
    return rows.sort((a, b) => a.person.name.localeCompare(b.person.name, 'zh-Hans-CN'))
  }
  return rows.sort((a, b) => a.earliest.localeCompare(b.earliest))
})

function scheduleSkeleton(): void {
  if (skeletonTimer) clearTimeout(skeletonTimer)
  skeletonTimer = setTimeout(() => {
    if (loading.value && !data.value.persons.length) showSkeleton.value = true
  }, 300)
}

async function load(): Promise<void> {
  loading.value = true
  error.value = ''
  scheduleSkeleton()
  try {
    data.value = await fetchPending(kind.value)
  } catch (err) {
    error.value = err instanceof Error ? err.message : '加载失败'
    data.value = { persons: [], personCount: 0, totalCount: 0, totalCent: 0 }
  } finally {
    loading.value = false
    showSkeleton.value = false
    initialized.value = true
    if (skeletonTimer) clearTimeout(skeletonTimer)
  }
}

onShow(async () => {
  await contextStore.load()
  if (kind.value === 'owing' && !contextStore.settings.showOwing) {
    kind.value = 'owed'
    return
  }
  await load()
})

onUnmounted(() => {
  if (skeletonTimer) clearTimeout(skeletonTimer)
})

watch(kind, () => {
  openId.value = 0
  if (initialized.value) void load()
})

function openPerson(row: PendingPerson): void {
  if (suppressClick) return
  if (openId.value === row.person.id) {
    openId.value = 0
    return
  }
  const query = kind.value === 'owed' ? '&status=pending' : ''
  go(`/sub-person/detail?id=${row.person.id}${query}`)
}

function setSort(value: SortKey): void {
  sort.value = value
  sortSheet.value = false
}

function onTouchStart(id: number, event: TouchLike): void {
  const touch = event.touches[0]
  if (!touch) return
  startX = touch.clientX
  startY = touch.clientY
  baseX = openId.value === id ? -88 : 0
  dragId.value = id
  dragX.value = baseX
  horizontal = false
}

function onTouchMove(id: number, event: TouchLike): void {
  if (dragId.value !== id) return
  const touch = event.touches[0]
  if (!touch) return
  const dx = touch.clientX - startX
  const dy = touch.clientY - startY
  if (!horizontal) {
    if (Math.abs(dx) > 6 && Math.abs(dx) > Math.abs(dy)) {
      horizontal = true
      suppressClick = true
    } else if (Math.abs(dy) > 10) {
      dragId.value = 0
      dragX.value = 0
      return
    } else {
      return
    }
  }
  dragX.value = Math.max(-88, Math.min(0, baseX + dx))
}

function onTouchEnd(id: number): void {
  if (dragId.value !== id) return
  openId.value = dragX.value < -40 ? id : 0
  dragX.value = openId.value === id ? -88 : 0
  dragId.value = 0
  if (horizontal) {
    setTimeout(() => {
      suppressClick = false
    }, 80)
  }
}

function onRowLongPress(row: PendingPerson): void {
  suppressClick = true
  setTimeout(() => {
    suppressClick = false
  }, 600)
  void markAll(row)
}

function handleTouchStart(id: number, event: Event): void {
  onTouchStart(id, event as unknown as TouchLike)
}

function handleTouchMove(id: number, event: Event): void {
  onTouchMove(id, event as unknown as TouchLike)
}

function rowStyle(id: number): Record<string, string> {
  if (dragId.value === id) {
    return { transform: `translateX(${dragX.value}px)` }
  }
  return { transform: openId.value === id ? 'translateX(-88px)' : 'translateX(0)' }
}

async function collectPending(personId: number): Promise<RecordItem[]> {
  const direction = kind.value === 'owed' ? 'in' : 'out'
  const collected: RecordItem[] = []
  let page = 1
  for (;;) {
    const result = await fetchRecords({
      personId,
      status: 'pending',
      direction,
      page,
      pageSize: 100,
    })
    collected.push(...result.items)
    if (collected.length >= result.total || !result.items.length) break
    page += 1
  }
  return collected
}

async function markAll(row: PendingPerson): Promise<void> {
  if (markingId.value) return
  openId.value = 0
  dragX.value = 0
  const ok = await openConfirmSheet({
    kind: 'pending-none',
    title: '全部不用回',
    message: `${row.person.name} 的 ${row.count} 笔未回礼，合计 ${formatCent(row.totalCent)}，标记后不再出现在回礼清单。`,
    conflicts: [{ id: row.person.id, summary: `${row.count} 笔 · ${formatCent(row.totalCent)}` }],
    confirmText: '全部不用回',
  })
  if (!ok) return
  openId.value = 0
  markingId.value = row.person.id
  try {
    const records = await collectPending(row.person.id)
    for (const record of records) {
      await updateRecord(record.id, {
        personId: record.personId,
        eventId: record.eventId,
        bookId: record.bookId,
        direction: record.direction,
        itemKind: record.itemKind,
        amountCent: record.amountCent,
        giftName: record.giftName,
        giftValueCent: record.giftValueCent,
        occurredOn: record.occurredOn,
        datePrecision: record.datePrecision,
        status: 'none',
        remark: record.remark,
        paperNote: record.paperNote,
        paperPage: record.paperPage,
        paperSeq: record.paperSeq,
      })
    }
    toast(`已标记 ${records.length} 笔为不用回`, 'success')
    await load()
  } catch (err) {
    toast(err instanceof Error ? err.message : '操作失败')
  } finally {
    markingId.value = 0
  }
}

function emptyText(): { title: string; description: string } {
  if (kind.value === 'owed') {
    return { title: '人情都回完了', description: '有新的收礼时会出现在这里' }
  }
  return { title: '暂时没有该回我的', description: '送出去的礼对方还没回时会出现在这里' }
}
</script>

<template>
  <view class="screen pending">
    <AppNav title="回礼清单" />

    <view class="page">
      <view class="tabs">
        <text
          v-for="tab in tabs"
          :key="tab.code"
          class="tabs__item"
          :class="{ 'tabs__item--on': kind === tab.code }"
          @click="kind = tab.code"
        >
          {{ tab.label }}
        </text>
      </view>

      <view class="summary">
        <text class="summary__text">
          共 {{ data.personCount }} 人 · {{ data.totalCount }} 笔 ·
        </text>
        <Money :cent="data.totalCent" :direction="kind === 'owed' ? 'none' : 'out'" :tone="kind === 'owed' ? 'pending' : undefined" size="m" />
      </view>

      <view class="sort" @click="sortSheet = true">
        <text class="sort__text">排序：{{ sortLabel }}</text>
        <text class="sort__caret">⌄</text>
      </view>
    </view>

    <Skeleton v-if="loading && showSkeleton" />
    <view v-else-if="loading" class="loading-tip">正在加载…</view>

    <view v-else-if="error" class="error-state">
      <text class="error-state__text">{{ error }}</text>
      <view class="btn btn--secondary error-state__btn" @click="load">重试</view>
    </view>

    <view v-else-if="sortedPersons.length" class="page">
      <view class="rows">
        <view v-for="row in sortedPersons" :key="row.person.id" class="prow">
          <view class="prow__action" @click="markAll(row)">
            <text>{{ markingId === row.person.id ? '处理中…' : '全部不用回' }}</text>
          </view>
          <view
            class="prow__body"
            :style="rowStyle(row.person.id)"
            @touchstart="handleTouchStart(row.person.id, $event)"
            @touchmove="handleTouchMove(row.person.id, $event)"
            @touchend="onTouchEnd(row.person.id)"
            @touchcancel="onTouchEnd(row.person.id)"
            @longpress="onRowLongPress(row)"
            @click="openPerson(row)"
          >
            <view v-if="kind === 'owed'" class="prow__bar" />
            <PersonAvatar :id="row.person.id" :name="row.person.name" :size="40" />
            <view class="prow__main">
              <view class="prow__line">
                <text class="prow__name">{{ row.person.name }}</text>
                <text class="prow__relation">{{ row.person.relationLabel }}</text>
              </view>
              <text class="prow__meta">{{ row.count }} 笔 · 最早 {{ relativeSince(row.earliest) }}</text>
            </view>
            <Money :cent="row.totalCent" :direction="kind === 'owed' ? 'none' : 'out'" :tone="kind === 'owed' ? 'pending' : undefined" size="m" />
          </view>
        </view>
      </view>
      <text class="pending__hint">左滑或长按可批量标记不用回</text>
    </view>

    <EmptyState
      v-else
      :title="emptyText().title"
      :description="emptyText().description"
    />

    <wd-popup v-model="sortSheet" position="bottom" :safe-area-inset-bottom="true" custom-style="border-radius: 20px 20px 0 0;">
      <view class="sort-sheet">
        <text class="sort-sheet__title">排序</text>
        <view
          v-for="option in sortOptions"
          :key="option.code"
          class="sort-sheet__row"
          :class="{ 'sort-sheet__row--on': sort === option.code }"
          @click="setSort(option.code)"
        >
          <text>{{ option.label }}</text>
          <text v-if="sort === option.code" class="sort-sheet__check">✓</text>
        </view>
      </view>
    </wd-popup>

    <ConfirmSheet />
  </view>
</template>

<style lang="scss" scoped>
.pending {
  padding-bottom: calc(24px + env(safe-area-inset-bottom));
}

.tabs {
  display: flex;
  gap: 8px;
  margin-top: 4px;
  padding: 3px;
  border-radius: 12px;
  background: var(--c-fill);
}

.tabs__item {
  flex: 1;
  height: 38px;
  border-radius: 10px;
  font-size: 15px;
  line-height: 38px;
  text-align: center;
  color: var(--c-ink-2);
}

.tabs__item--on {
  background: var(--c-surface);
  color: var(--c-ink-1);
  font-weight: 600;
  box-shadow: var(--shadow-card);
}

.summary {
  display: flex;
  align-items: baseline;
  gap: 4px;
  margin-top: 16px;
}

.summary__text {
  font-size: 14px;
  color: var(--c-ink-2);
}

.sort {
  display: flex;
  align-items: center;
  gap: 4px;
  margin-top: 10px;
  font-size: 13px;
  color: var(--c-ink-3);
}

.sort__caret {
  font-size: 12px;
}

.rows {
  margin-top: 8px;
  border-radius: var(--r-m);
  background: var(--c-surface);
  box-shadow: var(--shadow-card);
  overflow: hidden;
}

.prow {
  position: relative;
  overflow: hidden;
}

.prow + .prow {
  box-shadow: inset 0 1px 0 var(--c-divider);
}

.prow__action {
  position: absolute;
  right: 0;
  top: 0;
  bottom: 0;
  width: 88px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--c-pending-light);
  color: var(--c-pending);
  font-size: 12px;
  text-align: center;
  padding: 0 8px;
  box-sizing: border-box;
}

.prow__body {
  position: relative;
  display: flex;
  align-items: center;
  gap: 12px;
  min-height: 64px;
  padding: 12px 16px;
  background: var(--c-surface);
  transition: transform 0.2s ease;
}

.prow__bar {
  position: absolute;
  left: 0;
  top: 12px;
  bottom: 12px;
  width: 3px;
  border-radius: 2px;
  background: var(--c-pending);
}

.prow__main {
  flex: 1;
  min-width: 0;
}

.prow__line {
  display: flex;
  align-items: center;
  gap: 6px;
}

.prow__name {
  font-size: 15px;
  font-weight: 500;
  color: var(--c-ink-1);
}

.prow__relation {
  padding: 1px 6px;
  border-radius: var(--r-xs);
  background: var(--c-fill);
  font-size: 11px;
  color: var(--c-ink-2);
}

.prow__meta {
  display: block;
  margin-top: 2px;
  font-size: 12px;
  color: var(--c-ink-3);
}

.pending__hint {
  display: block;
  margin-top: 12px;
  font-size: 12px;
  color: var(--c-ink-3);
  text-align: center;
}

.sort-sheet {
  padding: 18px 16px calc(20px + env(safe-area-inset-bottom));
}

.sort-sheet__title {
  display: block;
  margin-bottom: 8px;
  font-size: 16px;
  font-weight: 600;
}

.sort-sheet__row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  min-height: 52px;
  font-size: 15px;
  color: var(--c-ink-1);
  box-shadow: inset 0 -1px 0 var(--c-divider);
}

.sort-sheet__row--on {
  color: var(--c-primary);
  font-weight: 600;
}

.sort-sheet__check {
  color: var(--c-primary);
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
