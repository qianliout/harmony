<script setup lang="ts">
import { onLoad, onUnload } from '@dcloudio/uni-app'
import { computed, ref } from 'vue'

import { fetchBooks, mergeBooks, previewMerge } from '@/api/book'
import type { BookItem, MergePreview } from '@/api/types'
import AppNav from '@/components/AppNav.vue'
import EmptyState from '@/components/EmptyState.vue'
import Skeleton from '@/components/Skeleton.vue'
import { paletteOf } from '@/styles/palette'
import { go, toast } from '@/utils/nav'

const sourceIds = ref<number[]>([])
const targetId = ref(0)
const books = ref<BookItem[]>([])
const step = ref(1)
const loading = ref(true)
const showSkeleton = ref(false)
const error = ref('')
const preview = ref<MergePreview | null>(null)
const previewing = ref(false)
const merging = ref(false)
const countdown = ref(2)

let skeletonTimer: ReturnType<typeof setTimeout> | null = null
let countdownTimer: ReturnType<typeof setInterval> | null = null
let navigateTimer: ReturnType<typeof setTimeout> | null = null

const steps = [
  { index: 1, label: '选来源' },
  { index: 2, label: '选目标' },
  { index: 3, label: '确认' },
]

const sourceBooks = computed(() => books.value.filter((book) => sourceIds.value.includes(book.id)))
const targetBook = computed(() => books.value.find((book) => book.id === targetId.value) || null)
const canNextFromSource = computed(() => sourceIds.value.length > 0)
const canNextFromTarget = computed(() => Boolean(targetId.value) && !sourceIds.value.includes(targetId.value))
const confirmText = computed(() => {
  const count = preview.value?.recordCount ?? 0
  const suffix = countdown.value > 0 ? `（${countdown.value}）` : ''
  return merging.value ? '正在合并…' : `合并 ${count} 条记录${suffix}`
})

function bookColor(book: BookItem): string {
  return paletteOf(book.categoryColor || 'stone').main
}

function categoryColors(book: BookItem): { main: string; light: string } {
  return paletteOf(book.categoryColor || 'stone')
}

function createdYear(book: BookItem): number {
  return book.createdAt ? new Date(book.createdAt).getFullYear() : 0
}

async function load(): Promise<void> {
  loading.value = true
  error.value = ''
  if (skeletonTimer) clearTimeout(skeletonTimer)
  skeletonTimer = setTimeout(() => {
    showSkeleton.value = true
  }, 300)
  try {
    books.value = await fetchBooks('normal')
    sourceIds.value = sourceIds.value.filter((id) => books.value.some((book) => book.id === id))
    if (sourceIds.value.length && books.value.length) {
      step.value = 2
    }
  } catch (err) {
    error.value = err instanceof Error ? err.message : '加载礼簿失败'
  } finally {
    if (skeletonTimer) clearTimeout(skeletonTimer)
    skeletonTimer = null
    showSkeleton.value = false
    loading.value = false
  }
}

function retry(): void {
  load()
}

function toggleSource(id: number): void {
  const index = sourceIds.value.indexOf(id)
  if (index >= 0) {
    sourceIds.value.splice(index, 1)
    return
  }
  sourceIds.value.push(id)
  if (targetId.value === id) {
    targetId.value = 0
  }
}

function chooseTarget(book: BookItem): void {
  if (sourceIds.value.includes(book.id)) {
    toast('这本礼簿已经选为来源')
    return
  }
  targetId.value = book.id
}

function toTargetStep(): void {
  if (!canNextFromSource.value) {
    toast('请先选择要合并的礼簿')
    return
  }
  step.value = 2
}

function stopCountdown(): void {
  if (countdownTimer) {
    clearInterval(countdownTimer)
    countdownTimer = null
  }
}

function startCountdown(): void {
  stopCountdown()
  countdown.value = 2
  countdownTimer = setInterval(() => {
    countdown.value -= 1
    if (countdown.value <= 0) {
      stopCountdown()
      countdown.value = 0
    }
  }, 1000)
}

async function toPreviewStep(): Promise<void> {
  if (previewing.value) return
  if (!canNextFromTarget.value) {
    toast('请选择合并后的目标礼簿')
    return
  }
  previewing.value = true
  try {
    preview.value = await previewMerge([...sourceIds.value], targetId.value)
    step.value = 3
    startCountdown()
  } catch (err) {
    toast(err instanceof Error ? err.message : '生成合并摘要失败')
  } finally {
    previewing.value = false
  }
}

function previousStep(): void {
  if (step.value === 3) {
    stopCountdown()
    countdown.value = 0
    step.value = 2
    return
  }
  if (step.value === 2) {
    step.value = 1
  }
}

async function confirmMerge(): Promise<void> {
  if (!preview.value || countdown.value > 0 || merging.value) return
  merging.value = true
  try {
    await mergeBooks([...sourceIds.value], targetId.value)
    toast('已合并', 'success')
    navigateTimer = setTimeout(() => {
      uni.setStorageSync('harmony:book-tab', 'book')
      uni.switchTab({ url: '/pages/book/list' })
    }, 700)
  } catch (err) {
    toast(err instanceof Error ? err.message : '合并失败')
  } finally {
    merging.value = false
  }
}

function createBook(): void {
  go('/sub-book/edit')
}

onLoad((query) => {
  const sourceId = Number(query?.sourceId ?? 0)
  if (sourceId) {
    sourceIds.value = [sourceId]
  }
  load()
})

onUnload(() => {
  stopCountdown()
  if (skeletonTimer) clearTimeout(skeletonTimer)
  if (navigateTimer) clearTimeout(navigateTimer)
})
</script>

<template>
  <view class="screen screen--merge">
    <AppNav title="合并礼簿" />

    <view class="page">
      <view class="steps">
        <view
          v-for="item in steps"
          :key="item.index"
          class="steps__item"
          :class="{ 'steps__item--on': step === item.index, 'steps__item--done': step > item.index }"
        >
          <view class="steps__dot">
            <text>{{ step > item.index ? '✓' : item.index }}</text>
          </view>
          <text class="steps__label">{{ item.label }}</text>
        </view>
      </view>

      <Skeleton v-if="showSkeleton" />

      <view v-else-if="error" class="merge-state">
        <EmptyState title="没能加载礼簿" :description="error" action-text="重试" @action="retry" />
      </view>

      <view v-else-if="!loading && !books.length" class="merge-state">
        <EmptyState title="还没有可合并的礼簿" description="先建两本礼簿，再来合并" action-text="新建礼簿" @action="createBook" />
      </view>

      <template v-else-if="!loading">
        <template v-if="step === 1">
          <text class="hint">选择要合并进来的礼簿，可以多选。</text>
          <view class="list">
            <view
              v-for="book in books"
              :key="book.id"
              class="book-row card"
              :class="{ 'book-row--on': sourceIds.includes(book.id) }"
              @click="toggleSource(book.id)"
            >
              <view class="check" :class="{ 'check--on': sourceIds.includes(book.id) }">
                <text v-if="sourceIds.includes(book.id)">✓</text>
              </view>
              <view class="book-row__spine" :style="{ background: bookColor(book) }" />
              <view class="book-row__main">
                <view class="book-row__line">
                  <text class="book-row__name">{{ book.name }}</text>
                  <text class="book-row__category" :style="{ background: categoryColors(book).light, color: categoryColors(book).main }">
                    {{ book.categoryLabel }}
                  </text>
                </view>
                <text class="book-row__meta">{{ book.recordCount }} 笔 · 建于 {{ createdYear(book) }} 年</text>
              </view>
            </view>
          </view>
        </template>

        <template v-else-if="step === 2">
          <text class="hint">选择合并后的目标礼簿，已选为来源的不能再选。</text>
          <view class="list">
            <view
              v-for="book in books"
              :key="book.id"
              class="book-row card"
              :class="{
                'book-row--on': targetId === book.id,
                'book-row--disabled': sourceIds.includes(book.id),
              }"
              @click="chooseTarget(book)"
            >
              <view class="radio" :class="{ 'radio--on': targetId === book.id }">
                <view v-if="targetId === book.id" class="radio__dot" />
              </view>
              <view class="book-row__spine" :style="{ background: bookColor(book) }" />
              <view class="book-row__main">
                <view class="book-row__line">
                  <text class="book-row__name">{{ book.name }}</text>
                  <text class="book-row__category" :style="{ background: categoryColors(book).light, color: categoryColors(book).main }">
                    {{ book.categoryLabel }}
                  </text>
                  <text v-if="sourceIds.includes(book.id)" class="book-row__flag">来源</text>
                </view>
                <text class="book-row__meta">{{ book.recordCount }} 笔 · 建于 {{ createdYear(book) }} 年</text>
              </view>
            </view>
          </view>
        </template>

        <template v-else>
          <template v-if="preview">
            <view class="preview-head">
              <text class="preview-head__title">合并摘要</text>
              <text class="preview-head__caption">核对一下，合并后不可撤销</text>
            </view>

            <view class="flow card">
              <view class="flow__sources">
                <view v-for="book in sourceBooks" :key="book.id" class="flow__source">
                  <view class="flow__spine" :style="{ background: bookColor(book) }" />
                  <text class="flow__name">{{ book.name }}</text>
                  <text class="flow__count">{{ book.recordCount }} 笔</text>
                </view>
              </view>
              <text class="flow__arrow">↓</text>
              <view v-if="targetBook" class="flow__target">
                <view class="flow__spine" :style="{ background: bookColor(targetBook) }" />
                <view class="flow__target-main">
                  <text class="flow__name">{{ targetBook.name }}</text>
                  <text class="flow__count">{{ targetBook.categoryLabel }} · 原有 {{ targetBook.recordCount }} 笔</text>
                </view>
              </view>
            </view>

            <view class="preview-stats card">
              <view class="preview-stats__cell">
                <text class="preview-stats__label">迁移记录</text>
                <text class="preview-stats__value">{{ preview.recordCount }} <text class="preview-stats__unit">笔</text></text>
              </view>
              <view class="preview-stats__cell">
                <text class="preview-stats__label">涉及名目</text>
                <text class="preview-stats__value">{{ preview.itemCounts.length }} <text class="preview-stats__unit">种</text></text>
              </view>
              <view class="preview-stats__cell">
                <text class="preview-stats__label">来源礼簿</text>
                <text class="preview-stats__value">{{ sourceBooks.length }} <text class="preview-stats__unit">本</text></text>
              </view>
            </view>

            <view v-if="preview.crossCategory" class="cross">
              <text class="cross__title">⚠ 跨类别合并</text>
              <text class="cross__text">记录保留原名目：火炮仍计为火炮，统计不会混在一起。</text>
            </view>

            <view v-if="preview.itemCounts.length" class="item-counts card">
              <view v-for="item in preview.itemCounts" :key="item.itemKind" class="item-counts__row">
                <text class="item-counts__name">{{ item.itemKindLabel }}</text>
                <text class="item-counts__value">{{ item.recordCount }} 笔</text>
              </view>
            </view>

            <text class="note">
              合并后，来源礼簿不再出现在列表中，可在「已合并的礼簿」中查看；每条记录都会保留原礼簿信息。合并不可撤销。
            </text>
          </template>
        </template>
      </template>
    </view>

    <view v-if="!loading && !error && books.length" class="bottom-bar">
      <template v-if="step === 1">
        <view
          class="btn btn--primary bottom-bar__btn"
          :class="{ 'btn--disabled': !canNextFromSource }"
          @click="toTargetStep"
        >
          下一步
        </view>
      </template>
      <template v-else-if="step === 2">
        <view class="btn btn--secondary bottom-bar__btn" @click="previousStep">上一步</view>
        <view
          class="btn btn--primary bottom-bar__btn bottom-bar__btn--main"
          :class="{ 'btn--disabled': !canNextFromTarget || previewing }"
          @click="toPreviewStep"
        >
          {{ previewing ? '生成摘要…' : '下一步' }}
        </view>
      </template>
      <template v-else>
        <view class="btn btn--secondary bottom-bar__btn" @click="previousStep">上一步</view>
        <view
          class="btn btn--danger bottom-bar__btn bottom-bar__btn--main"
          :class="{ 'btn--disabled': countdown > 0 || merging }"
          @click="confirmMerge"
        >
          {{ confirmText }}
        </view>
      </template>
    </view>
  </view>
</template>

<style lang="scss" scoped>
.screen--merge {
  padding-bottom: calc(88px + env(safe-area-inset-bottom));
}

.steps {
  display: flex;
  align-items: center;
  padding: 8px 0 4px;
}

.steps__item {
  position: relative;
  display: flex;
  flex: 1;
  align-items: center;
  gap: 6px;
}

.steps__item::after {
  content: '';
  flex: 1;
  height: 1px;
  margin-right: 8px;
  background: var(--c-line);
}

.steps__item:last-child::after {
  display: none;
}

.steps__dot {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 22px;
  height: 22px;
  border-radius: var(--r-full);
  background: var(--c-fill);
  color: var(--c-ink-3);
  font-size: 12px;
  flex: none;
}

.steps__label {
  font-size: 13px;
  color: var(--c-ink-3);
  white-space: nowrap;
}

.steps__item--on .steps__dot {
  background: var(--c-primary);
  color: var(--c-surface);
}

.steps__item--on .steps__label {
  color: var(--c-ink-1);
  font-weight: 600;
}

.steps__item--done .steps__dot {
  background: var(--c-primary-light);
  color: var(--c-primary);
}

.steps__item--done .steps__label {
  color: var(--c-ink-2);
}

.hint {
  display: block;
  margin: 8px 2px 12px;
  font-size: 13px;
  line-height: 18px;
  color: var(--c-ink-3);
}

.merge-state {
  padding-top: 20px;
}

.list {
  padding-bottom: 4px;
}

.book-row {
  position: relative;
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 12px 16px;
  margin-bottom: 10px;
  border-radius: var(--r-m);
  overflow: hidden;
}

.book-row--disabled {
  opacity: 0.45;
}

.check {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 22px;
  height: 22px;
  border: 1.5px solid var(--c-ink-4);
  border-radius: var(--r-xs);
  color: var(--c-surface);
  font-size: 13px;
  flex: none;
}

.check--on {
  background: var(--c-primary);
  border-color: var(--c-primary);
}

.radio {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 22px;
  height: 22px;
  border: 1.5px solid var(--c-ink-4);
  border-radius: var(--r-full);
  flex: none;
}

.radio--on {
  border-color: var(--c-primary);
}

.radio__dot {
  width: 12px;
  height: 12px;
  border-radius: var(--r-full);
  background: var(--c-primary);
}

.book-row__spine {
  width: 5px;
  height: 38px;
  border-radius: 3px;
  flex: none;
}

.book-row__main {
  flex: 1;
  min-width: 0;
}

.book-row__line {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
}

.book-row__name {
  font-size: 15px;
  line-height: 22px;
  font-weight: 600;
  color: var(--c-ink-1);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.book-row__category {
  flex: none;
  height: 20px;
  padding: 0 6px;
  border-radius: var(--r-xs);
  font-size: 12px;
  line-height: 20px;
}

.book-row__flag {
  flex: none;
  font-size: 12px;
  color: var(--c-primary);
}

.book-row__meta {
  display: block;
  margin-top: 2px;
  font-size: 12px;
  color: var(--c-ink-3);
}

.preview-head {
  padding: 8px 2px 12px;
}

.preview-head__title {
  font-size: 17px;
  line-height: 24px;
  font-weight: 600;
}

.preview-head__caption {
  display: block;
  margin-top: 2px;
  font-size: 12px;
  color: var(--c-ink-3);
}

.flow {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 16px;
}

.flow__sources {
  display: flex;
  flex-wrap: wrap;
  justify-content: center;
  gap: 8px;
  width: 100%;
}

.flow__source,
.flow__target {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
  padding: 8px 12px;
  border-radius: var(--r-s);
  background: var(--c-fill);
}

.flow__source {
  max-width: 100%;
}

.flow__target {
  width: 100%;
  box-sizing: border-box;
  background: var(--c-primary-light);
}

.flow__spine {
  width: 4px;
  height: 24px;
  border-radius: 2px;
  flex: none;
}

.flow__target-main {
  flex: 1;
  min-width: 0;
}

.flow__name {
  font-size: 14px;
  font-weight: 600;
  color: var(--c-ink-1);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.flow__count {
  font-size: 12px;
  color: var(--c-ink-3);
  flex: none;
}

.flow__arrow {
  padding: 4px 0;
  font-size: 18px;
  color: var(--c-ink-4);
}

.preview-stats {
  display: flex;
  margin-top: var(--card-gap);
  padding: 14px 0;
}

.preview-stats__cell {
  flex: 1;
  text-align: center;
  position: relative;
}

.preview-stats__cell + .preview-stats__cell::before {
  content: '';
  position: absolute;
  left: 0;
  top: 4px;
  bottom: 4px;
  width: 1px;
  background: var(--c-divider);
}

.preview-stats__label {
  display: block;
  font-size: 12px;
  color: var(--c-ink-3);
}

.preview-stats__value {
  display: block;
  margin-top: 2px;
  font-family: $font-num;
  font-size: 22px;
  line-height: 28px;
  font-weight: 600;
  color: var(--c-ink-1);
}

.preview-stats__unit {
  font-size: 12px;
  color: var(--c-ink-3);
}

.cross {
  display: flex;
  flex-direction: column;
  gap: 4px;
  margin-top: var(--card-gap);
  padding: 12px 14px;
  border-radius: var(--r-m);
  background: var(--c-pending-light);
}

.cross__title {
  font-size: 14px;
  line-height: 20px;
  font-weight: 600;
  color: var(--c-pending);
}

.cross__text {
  font-size: 12px;
  line-height: 18px;
  color: var(--c-pending);
}

.item-counts {
  margin-top: var(--card-gap);
  padding: 4px 16px;
}

.item-counts__row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  height: 44px;
  font-size: 14px;
}

.item-counts__row + .item-counts__row {
  box-shadow: inset 0 1px 0 var(--c-divider);
}

.item-counts__name {
  color: var(--c-ink-2);
}

.item-counts__value {
  color: var(--c-ink-3);
  font-size: 13px;
}

.note {
  display: block;
  padding: 16px 2px 8px;
  font-size: 12px;
  line-height: 19px;
  color: var(--c-ink-3);
}
</style>
