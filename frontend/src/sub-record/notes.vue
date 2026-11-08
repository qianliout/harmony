<script setup lang="ts">
import { onLoad } from '@dcloudio/uni-app'
import { computed, onUnmounted, ref } from 'vue'

import { fetchBooks } from '@/api/book'
import { fetchPendingNotes, parsePaperNote, repayRecords } from '@/api/record'
import type { BookItem, PaperDraft, RecordItem } from '@/api/types'
import AppNav from '@/components/AppNav.vue'
import ConfirmSheet from '@/components/ConfirmSheet.vue'
import EmptyState from '@/components/EmptyState.vue'
import PaperRepaySheet from '@/components/PaperRepaySheet.vue'
import Skeleton from '@/components/Skeleton.vue'
import { openConfirmSheet } from '@/composables/useConfirmSubmit'
import { preciseDateLabel } from '@/utils/date'
import { toast } from '@/utils/nav'
import { formatCent } from '@/utils/money'

interface NoteItem {
  record: RecordItem
  drafts: PaperDraft[]
  onlyMark: boolean
  parsing: boolean
  parseError: string
  complete: boolean
  processing: boolean
  error: string
}

const items = ref<NoteItem[]>([])
const books = ref<BookItem[]>([])
const bookId = ref(0)
const loading = ref(true)
const showSkeleton = ref(false)
const error = ref('')
const sheetShow = ref(false)
const activeRecord = ref<RecordItem | null>(null)
const batch = ref({ running: false, done: 0, total: 0 })
let skeletonTimer: ReturnType<typeof setTimeout> | null = null

const pendingCount = computed(() => items.value.length)
const completeCount = computed(() => items.value.filter((item) => item.complete).length)
const batchPercent = computed(() =>
  batch.value.total ? Math.round((batch.value.done / batch.value.total) * 100) : 0,
)

function draftCent(draft: PaperDraft): number {
  return (draft.items ?? []).reduce((sum, item) => sum + item.amountCent, 0)
}

function draftComplete(draft: PaperDraft): boolean {
  return Boolean(draft.event?.name) && Boolean(draft.occurredOn) && draftCent(draft) > 0
}

function previewText(item: NoteItem): string {
  if (item.onlyMark) return '只有已回礼记号'
  const first = item.drafts[0]
  if (!first) return '未能解析出回礼'
  const parts: string[] = []
  if (item.drafts.length > 1) parts.push(`${item.drafts.length} 次`)
  parts.push(preciseDateLabel(first.occurredOn, first.datePrecision || 'day') || '时间待补')
  parts.push(first.event?.name || '事项待补')
  const cent = item.drafts.reduce((sum, draft) => sum + draftCent(draft), 0)
  parts.push(cent > 0 ? formatCent(cent) : '金额待补')
  return parts.join(' · ')
}

function parseErrorLabel(): string {
  return '解析失败，可点“补录”手动填写'
}

async function parseItem(item: NoteItem): Promise<void> {
  item.parsing = true
  item.parseError = ''
  try {
    const result = await parsePaperNote(item.record.paperNote, item.record.personId)
    item.onlyMark = result.onlyMark
    item.drafts = result.repays ?? []
    item.complete =
      !result.onlyMark &&
      item.drafts.length > 0 &&
      item.drafts.every((draft) => draftComplete(draft))
  } catch (err) {
    item.parseError = err instanceof Error ? err.message : parseErrorLabel()
    item.complete = false
  } finally {
    item.parsing = false
  }
}

async function load(): Promise<void> {
  loading.value = true
  error.value = ''
  if (skeletonTimer) clearTimeout(skeletonTimer)
  skeletonTimer = setTimeout(() => {
    if (loading.value && !items.value.length) showSkeleton.value = true
  }, 300)
  try {
    const records = await fetchPendingNotes(bookId.value)
    items.value = records.map((record) => ({
      record,
      drafts: [],
      onlyMark: false,
      parsing: true,
      parseError: '',
      complete: false,
      processing: false,
      error: '',
    }))
    await Promise.all(items.value.map((item) => parseItem(item)))
  } catch (err) {
    error.value = err instanceof Error ? err.message : '加载失败'
    items.value = []
  } finally {
    loading.value = false
    showSkeleton.value = false
    if (skeletonTimer) clearTimeout(skeletonTimer)
  }
}

async function loadBooks(): Promise<void> {
  try {
    books.value = await fetchBooks('normal')
  } catch {
    books.value = []
  }
}

onLoad(async (query) => {
  const parsed = (query ?? {}) as Record<string, string>
  bookId.value = Number(parsed.bookId ?? 0)
  await Promise.all([loadBooks(), load()])
})

onUnmounted(() => {
  if (skeletonTimer) clearTimeout(skeletonTimer)
})

function switchBook(id: number): void {
  bookId.value = id
  void load()
}

function skip(item: NoteItem): void {
  const index = items.value.indexOf(item)
  if (index < 0) return
  items.value.splice(index, 1)
  items.value.push(item)
}

function openSheet(item: NoteItem): void {
  activeRecord.value = item.record
  sheetShow.value = true
}

function onSaved(count: number, cent: number): void {
  const record = activeRecord.value
  if (record) {
    items.value = items.value.filter((item) => item.record.id !== record.id)
  }
  activeRecord.value = null
  if (count > 0) {
    toast(`已补录 ${count} 笔回礼 · ${formatCent(cent)}`, 'success')
  } else {
    toast('已标记为已回礼', 'success')
  }
}

async function repayDrafts(item: NoteItem): Promise<void> {
  for (const draft of item.drafts) {
    await repayRecords({
      originIds: [item.record.id],
      markRepaid: true,
      event: draft.event,
      occurredOn: draft.occurredOn,
      datePrecision: draft.datePrecision || 'day',
      paperNote: item.record.paperNote,
      items: draft.items,
    })
  }
}

async function batchRepay(): Promise<void> {
  if (batch.value.running) return
  const targets = items.value.filter((item) => item.complete)
  if (!targets.length) {
    toast('没有解析完整的条目')
    return
  }
  const totalCent = targets.reduce(
    (sum, item) => sum + item.drafts.reduce((draftSum, draft) => draftSum + draftCent(draft), 0),
    0,
  )
  const preview = targets.slice(0, 10).map((item) => ({
    id: item.record.id,
    summary: `${item.record.personName} · ${previewText(item)}`,
  }))
  if (targets.length > 10) {
    preview.push({ id: 0, summary: `…共 ${targets.length} 条，先展示前 10 条` })
  }
  const ok = await openConfirmSheet({
    kind: 'batch-repay',
    title: `一键补录 ${targets.length} 条`,
    message: `合计 ${formatCent(totalCent)}，将逐条生成回礼记录并把原记录标为已回礼。`,
    conflicts: preview,
    confirmText: `补录 ${targets.length} 条`,
  })
  if (!ok) return

  batch.value = { running: true, done: 0, total: targets.length }
  const settled: number[] = []
  for (const item of targets) {
    item.processing = true
    item.error = ''
    try {
      await repayDrafts(item)
      settled.push(item.record.id)
    } catch (err) {
      item.error = err instanceof Error ? err.message : '补录失败'
    } finally {
      item.processing = false
      batch.value = { ...batch.value, done: batch.value.done + 1 }
    }
  }
  batch.value.running = false
  items.value = items.value.filter((item) => !settled.includes(item.record.id))
  const failed = targets.length - settled.length
  if (failed > 0) {
    toast(`已补录 ${settled.length} 条，${failed} 条失败`)
  } else {
    toast(`已补录 ${settled.length} 条回礼`, 'success')
  }
}
</script>

<template>
  <view class="screen notes">
    <AppNav title="整理纸本标注" />

    <view class="page">
      <view class="notes__head">
        <text class="notes__count">{{ pendingCount }} 条待整理</text>
        <text v-if="pendingCount" class="notes__complete">{{ completeCount }} 条解析完整</text>
      </view>

      <scroll-view v-if="books.length" scroll-x class="books">
        <view class="books__inner">
          <text class="chip chip--flat" :class="{ 'chip--flat-on': bookId === 0 }" @click="switchBook(0)">
            全部礼簿
          </text>
          <text
            v-for="book in books"
            :key="book.id"
            class="chip chip--flat"
            :class="{ 'chip--flat-on': bookId === book.id }"
            @click="switchBook(book.id)"
          >
            {{ book.name }}
          </text>
        </view>
      </scroll-view>

      <view v-if="batch.running" class="batch-progress">
        <view class="batch-progress__bar">
          <view class="batch-progress__fill" :style="{ width: `${batchPercent}%` }" />
        </view>
        <text class="batch-progress__text">{{ batch.done }} / {{ batch.total }}</text>
      </view>
    </view>

    <Skeleton v-if="loading && showSkeleton" />
    <view v-else-if="loading" class="loading-tip">正在解析标注…</view>

    <view v-else-if="error" class="error-state">
      <text class="error-state__text">{{ error }}</text>
      <view class="btn btn--secondary error-state__btn" @click="load">重试</view>
    </view>

    <template v-else-if="items.length">
      <view class="page">
        <view class="note-list">
          <view
            v-for="item in items"
            :key="item.record.id"
            class="note-card"
            :class="{ 'note-card--processing': item.processing }"
          >
            <view class="note-card__head">
              <text class="note-card__person">{{ item.record.personName }}</text>
              <text class="note-card__meta">
                收 {{ formatCent(item.record.totalCent) }}<text v-if="item.record.paperPage"> · 第 {{ item.record.paperPage }} 页</text>
              </text>
            </view>

            <text class="note-card__raw">{{ item.record.paperNote }}</text>

            <view v-if="item.parsing" class="note-card__parse">
              <text class="caption">正在解析…</text>
            </view>
            <view v-else class="note-card__parse">
              <text class="note-card__preview" :class="item.complete ? 'note-card__preview--ok' : 'note-card__preview--warn'">
                → {{ item.parseError || previewText(item) }}<text v-if="item.complete"> ✓</text>
              </text>
            </view>

            <text v-if="item.error" class="note-card__error">{{ item.error }}</text>

            <view class="note-card__actions">
              <text class="note-card__skip" @click="skip(item)">跳过</text>
              <view class="btn btn--secondary btn--small note-card__btn" @click="openSheet(item)">补录</view>
            </view>
          </view>
        </view>
      </view>

      <view class="notes__spacer" />
      <view class="bottom-bar">
        <view
          class="btn btn--primary bottom-bar__btn"
          :class="{ 'btn--disabled': batch.running || !completeCount }"
          @click="batchRepay"
        >
          {{ batch.running ? `正在补录 ${batch.done} / ${batch.total}` : `一键补录 ${completeCount} 条解析完整的` }}
        </view>
      </view>
    </template>

    <EmptyState
      v-else
      title="没有待整理的标注"
      description="连续录入时抄下的手写标注会出现在这里"
    />

    <PaperRepaySheet v-model:show="sheetShow" :record="activeRecord" @saved="onSaved" />
    <ConfirmSheet />
  </view>
</template>

<style lang="scss" scoped>
.notes {
  padding-bottom: calc(84px + env(safe-area-inset-bottom));
}

.notes__head {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  margin-top: 4px;
}

.notes__count {
  font-size: 17px;
  font-weight: 600;
  color: var(--c-ink-1);
}

.notes__complete {
  font-size: 12px;
  color: var(--c-ink-3);
}

.books {
  margin-top: 10px;
  white-space: nowrap;
}

.books__inner {
  display: flex;
  gap: 8px;
}

.batch-progress {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-top: 12px;
}

.batch-progress__bar {
  flex: 1;
  height: 6px;
  border-radius: var(--r-full);
  background: var(--c-fill);
  overflow: hidden;
}

.batch-progress__fill {
  height: 100%;
  background: var(--c-primary);
  transition: width 0.3s ease;
}

.batch-progress__text {
  font-family: $font-num;
  font-size: 12px;
  color: var(--c-ink-2);
}

.note-list {
  margin-top: 12px;
}

.note-card {
  margin-bottom: 12px;
  padding: 14px;
  border-radius: var(--r-m);
  background: var(--c-surface);
  box-shadow: var(--shadow-card);
  transition: opacity 0.2s;
}

.note-card--processing {
  opacity: 0.6;
}

.note-card__head {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: 8px;
}

.note-card__person {
  font-size: 15px;
  font-weight: 500;
  color: var(--c-ink-1);
}

.note-card__meta {
  font-size: 12px;
  color: var(--c-ink-3);
}

.note-card__raw {
  display: block;
  margin-top: 8px;
  padding: 10px 12px;
  border-radius: var(--r-s);
  background: var(--c-fill);
  font-family: $font-serif;
  font-size: 14px;
  line-height: 22px;
  color: var(--c-ink-1);
}

.note-card__parse {
  margin-top: 8px;
}

.note-card__preview {
  font-size: 13px;
  line-height: 19px;
}

.note-card__preview--ok {
  color: var(--c-repaid);
}

.note-card__preview--warn {
  color: var(--c-pending);
}

.note-card__error {
  display: block;
  margin-top: 6px;
  font-size: 12px;
  color: var(--c-danger);
}

.note-card__actions {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 12px;
  margin-top: 12px;
}

.note-card__skip {
  font-size: 13px;
  color: var(--c-ink-3);
}

.note-card__btn {
  padding: 0 20px;
}

.notes__spacer {
  height: 8px;
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
