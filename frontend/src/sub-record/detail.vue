<script setup lang="ts">
import { onLoad, onShow } from '@dcloudio/uni-app'
import { computed, ref } from 'vue'

import { fetchRecordDetail, unlinkRepay, updateRecord } from '@/api/record'
import type { RecordDetail, RecordItem } from '@/api/types'
import AppNav from '@/components/AppNav.vue'
import ConfirmSheet from '@/components/ConfirmSheet.vue'
import DirTag from '@/components/DirTag.vue'
import ListCell from '@/components/ListCell.vue'
import Money from '@/components/Money.vue'
import PaperRepaySheet from '@/components/PaperRepaySheet.vue'
import Skeleton from '@/components/Skeleton.vue'
import StatusPill from '@/components/StatusPill.vue'
import { openConfirmSheet } from '@/composables/useConfirmSubmit'
import { relations } from '@/styles/palette'
import { centToChinese } from '@/utils/chinese-amount'
import { formatDate, preciseDateLabel } from '@/utils/date'
import { go, toast } from '@/utils/nav'
import { formatCent } from '@/utils/money'

const recordId = ref(0)
const detail = ref<RecordDetail | null>(null)
const loading = ref(true)
const showSkeleton = ref(false)
const error = ref('')
const statusSheet = ref(false)
const repaySheetShow = ref(false)
const noteExpanded = ref(false)
const updating = ref(false)
let skeletonTimer: ReturnType<typeof setTimeout> | null = null
let loadedOnce = false

const statusOptions = [
  { code: 'pending', label: '待还', desc: '收礼后尚未还礼' },
  { code: 'repaid', label: '已还', desc: '已经还过这份人情' },
  { code: 'none', label: '无需还', desc: '不必再还，不再出现在待还清单' },
] as const

const SOURCE_LABELS: Record<string, string> = {
  manual: '手动录入',
  import: '导入',
}

const directionLabel = computed(() => (detail.value?.direction === 'in' ? '收礼' : '送礼'))
const relationLabel = computed(() => {
  const code = detail.value?.personRelation ?? ''
  return relations.find((item) => item.code === code)?.label ?? '其他'
})
const personValue = computed(() =>
  detail.value ? `${detail.value.personName} · ${relationLabel.value}` : '',
)
const moneyDirection = computed<'in' | 'out'>(() => (detail.value?.direction === 'in' ? 'in' : 'out'))
const bookValue = computed(() => {
  const item = detail.value
  if (!item) return ''
  if (!item.originBookName) return item.bookName
  return `${item.bookName}（原：${item.originBookName}）`
})
const paperLabel = computed(() => {
  const item = detail.value
  if (!item) return ''
  const parts: string[] = []
  if (item.paperPage) parts.push(`第 ${item.paperPage} 页`)
  if (item.paperSeq) parts.push(`第 ${item.paperSeq} 行`)
  return parts.join(' · ')
})
const dateLabel = computed(() =>
  detail.value ? preciseDateLabel(detail.value.occurredOn, detail.value.datePrecision) : '',
)
const sourceLabel = computed(() => SOURCE_LABELS[detail.value?.source ?? ''] ?? '网页录入')
const canRepay = computed(() => detail.value?.direction === 'in' && detail.value?.status === 'pending')
const repayActionLabel = computed(() => (detail.value?.paperNote ? '补录纸本还礼' : '还礼'))
const chainTitle = computed(() => (detail.value?.direction === 'in' ? '被以下还礼关联' : '还的是以下收礼'))
const chainRecords = computed<RecordItem[]>(() => {
  const item = detail.value
  if (!item) return []
  return item.direction === 'in' ? item.repayRecords : item.originRecords
})

function scheduleSkeleton(): void {
  if (skeletonTimer) clearTimeout(skeletonTimer)
  skeletonTimer = setTimeout(() => {
    if (loading.value && !detail.value) showSkeleton.value = true
  }, 300)
}

async function load(): Promise<void> {
  if (!recordId.value) return
  loading.value = true
  error.value = ''
  scheduleSkeleton()
  try {
    detail.value = await fetchRecordDetail(recordId.value)
    loadedOnce = true
  } catch (err) {
    error.value = err instanceof Error ? err.message : '加载失败'
    detail.value = null
  } finally {
    loading.value = false
    showSkeleton.value = false
    if (skeletonTimer) clearTimeout(skeletonTimer)
  }
}

onLoad((query) => {
  recordId.value = Number((query as Record<string, string> | undefined)?.id ?? 0)
  void load()
})

onShow(() => {
  if (loadedOnce) void load()
})

async function setStatus(status: string): Promise<void> {
  const item = detail.value
  if (!item || updating.value) return
  updating.value = true
  statusSheet.value = false
  try {
    await updateRecord(item.id, {
      personId: item.personId,
      eventId: item.eventId,
      bookId: item.bookId,
      direction: item.direction,
      itemKind: item.itemKind,
      amountCent: item.amountCent,
      giftName: item.giftName,
      giftValueCent: item.giftValueCent,
      occurredOn: item.occurredOn,
      datePrecision: item.datePrecision,
      status,
      remark: item.remark,
      paperNote: item.paperNote,
      paperPage: item.paperPage,
      paperSeq: item.paperSeq,
    })
    detail.value = { ...item, status: status as RecordItem['status'] }
    toast('状态已更新', 'success')
  } catch (err) {
    toast(err instanceof Error ? err.message : '修改失败')
  } finally {
    updating.value = false
  }
}

function openChain(record: RecordItem): void {
  if (record.id === recordId.value) return
  go(`/sub-record/detail?id=${record.id}`)
}

async function unlink(record: RecordItem): Promise<void> {
  const item = detail.value
  if (!item) return
  const ok = await openConfirmSheet({
    kind: 'unlink-repay',
    title: '解除还礼关联',
    message: '解除后，这条还礼记录与原收礼记录不再关联，记录本身不会被删除。',
    conflicts: [{ id: record.id, summary: `${record.personName} · ${formatCent(record.totalCent)} · ${chainDate(record)}` }],
    confirmText: '解除关联',
  })
  if (!ok) return
  try {
    if (item.direction === 'in') {
      await unlinkRepay(record.id, item.id)
    } else {
      await unlinkRepay(item.id, record.id)
    }
    toast('已解除关联', 'success')
    await load()
  } catch (err) {
    toast(err instanceof Error ? err.message : '解除失败')
  }
}

function edit(): void {
  go(`/sub-record/form?mode=edit&id=${recordId.value}`)
}

function repay(): void {
  const item = detail.value
  if (!item) return
  if (item.paperNote) {
    repaySheetShow.value = true
    return
  }
  go(`/sub-record/form?mode=repay&originIds=${item.id}`)
}

function onRepaySaved(count: number, cent: number): void {
  if (count > 0) {
    toast(`已补录 ${count} 笔还礼 · ${formatCent(cent)}`, 'success')
  } else {
    toast('已标记为已还', 'success')
  }
  void load()
}

function chainDate(record: RecordItem): string {
  return formatDate(record.occurredOn, record.datePrecision)
}
</script>

<template>
  <view class="screen detail">
    <AppNav title="记录详情" />

    <Skeleton v-if="loading && showSkeleton" />

    <view v-else-if="loading && !detail" class="loading-tip">正在加载…</view>

    <view v-else-if="error" class="error-state">
      <text class="error-state__text">{{ error }}</text>
      <view class="btn btn--secondary error-state__btn" @click="load">重试</view>
    </view>

    <template v-else-if="detail">
      <view class="page">
        <view class="hero">
          <text class="hero__direction">{{ directionLabel }}</text>
          <Money :cent="detail.totalCent" :direction="moneyDirection" size="display" />
          <text class="hero__chinese">{{ centToChinese(detail.totalCent) }}</text>
          <view class="hero__status" @click="statusSheet = true">
            <StatusPill :status="detail.status" clickable />
            <text class="hero__status-caret">⌄</text>
          </view>
          <text v-if="detail.giftName" class="hero__gift">
            礼品 {{ detail.giftName }}<text v-if="detail.giftValueCent"> · 估 {{ formatCent(detail.giftValueCent) }}</text>
          </text>
        </view>
      </view>

      <view class="page">
        <view class="form-card info">
          <ListCell title="亲友" :value="personValue" :arrow="false" />
          <ListCell title="事项" :value="detail.eventName" :arrow="false" />
          <ListCell title="礼簿" :value="bookValue" :arrow="false" />
          <ListCell title="名目" :value="detail.itemKindLabel" :arrow="false" />
          <ListCell title="日期" :value="dateLabel" :arrow="false" />
          <ListCell v-if="detail.remark" title="备注" :value="detail.remark" :arrow="false" />
          <ListCell v-if="paperLabel" title="纸本" :value="paperLabel" :arrow="false" />
          <ListCell title="来源" :value="sourceLabel" :arrow="false" />
          <view v-if="detail.paperNote" class="note-cell" @click="noteExpanded = !noteExpanded">
            <text class="note-cell__label">标注</text>
            <text class="note-cell__text" :class="{ 'note-cell__text--full': noteExpanded }">
              {{ detail.paperNote }}
            </text>
          </view>
        </view>
      </view>

      <view class="page">
        <view class="chain">
          <view class="chain__head">
            <text class="chain__title">还礼链路</text>
            <text class="chain__sub">{{ chainTitle }}</text>
          </view>
          <view v-if="chainRecords.length" class="chain__list card">
            <view v-for="record in chainRecords" :key="record.id" class="chain__row">
              <view class="chain__main" @click="openChain(record)">
                <text class="chain__date">{{ chainDate(record) }}</text>
                <DirTag :direction="record.direction" />
                <Money :cent="record.totalCent" :direction="record.direction" size="s" />
                <text class="chain__event">{{ record.eventName }}</text>
                <text class="chain__arrow">›</text>
              </view>
              <text class="chain__unlink" @click="unlink(record)">解除关联</text>
            </view>
          </view>
          <view v-else class="chain__empty card">
            <text class="chain__empty-text">{{ detail.direction === 'in' ? '还没有还礼记录' : '没有关联的收礼记录' }}</text>
          </view>
        </view>
      </view>

      <view class="detail__spacer" />

      <view class="bottom-bar">
        <view class="btn btn--secondary bottom-bar__btn" @click="edit">编辑</view>
        <view v-if="canRepay" class="btn btn--primary bottom-bar__btn" @click="repay">
          {{ repayActionLabel }}
        </view>
      </view>
    </template>

    <wd-popup v-model="statusSheet" position="bottom" :safe-area-inset-bottom="true" custom-style="border-radius: 20px 20px 0 0;">
      <view class="status-sheet">
        <text class="status-sheet__title">修改状态</text>
        <view
          v-for="option in statusOptions"
          :key="option.code"
          class="status-sheet__row"
          :class="{ 'status-sheet__row--on': detail?.status === option.code }"
          @click="setStatus(option.code)"
        >
          <view class="status-sheet__main">
            <text class="status-sheet__label">{{ option.label }}</text>
            <text class="status-sheet__desc">{{ option.desc }}</text>
          </view>
          <text v-if="detail?.status === option.code" class="status-sheet__check">✓</text>
        </view>
      </view>
    </wd-popup>

    <PaperRepaySheet
      v-model:show="repaySheetShow"
      :record="detail"
      @saved="onRepaySaved"
    />

    <ConfirmSheet />
  </view>
</template>

<style lang="scss" scoped>
.detail {
  padding-bottom: calc(84px + env(safe-area-inset-bottom));
}

.hero {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 20px 0 8px;
}

.hero__direction {
  font-size: 13px;
  color: var(--c-ink-3);
}

.hero__chinese {
  margin-top: 4px;
  font-family: $font-serif;
  font-size: 15px;
  color: var(--c-ink-3);
}

.hero__status {
  display: flex;
  align-items: center;
  gap: 4px;
  margin-top: 10px;
}

.hero__status-caret {
  font-size: 12px;
  color: var(--c-ink-3);
}

.hero__gift {
  margin-top: 8px;
  font-size: 13px;
  color: var(--c-ink-2);
}

.info {
  margin-top: 12px;
}

.note-cell {
  display: flex;
  align-items: flex-start;
  gap: 10px;
  padding: 12px 16px;
  box-shadow: inset 0 1px 0 var(--c-divider);
}

.note-cell__label {
  width: 60px;
  flex: none;
  font-size: 15px;
  color: var(--c-ink-1);
}

.note-cell__text {
  flex: 1;
  min-width: 0;
  font-family: $font-serif;
  font-size: 14px;
  line-height: 22px;
  color: var(--c-ink-2);
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.note-cell__text--full {
  display: block;
  overflow: visible;
}

.chain {
  margin-top: 22px;
}

.chain__head {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  margin: 0 2px 10px;
}

.chain__title {
  font-size: 17px;
  font-weight: 600;
}

.chain__sub {
  font-size: 12px;
  color: var(--c-ink-3);
}

.chain__list {
  overflow: hidden;
}

.chain__row {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 14px;
}

.chain__row + .chain__row {
  box-shadow: inset 0 1px 0 var(--c-divider);
}

.chain__main {
  flex: 1;
  min-width: 0;
  display: flex;
  align-items: center;
  gap: 6px;
}

.chain__date {
  font-size: 12px;
  color: var(--c-ink-3);
  flex: none;
}

.chain__event {
  flex: 1;
  min-width: 0;
  font-size: 13px;
  color: var(--c-ink-2);
  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
}

.chain__arrow {
  color: var(--c-ink-4);
}

.chain__unlink {
  flex: none;
  font-size: 12px;
  color: var(--c-ink-3);
  padding: 4px 0 4px 8px;
}

.chain__empty {
  padding: 14px 16px;
}

.chain__empty-text {
  font-size: 13px;
  color: var(--c-ink-3);
}

.detail__spacer {
  height: 20px;
}

.status-sheet {
  padding: 18px 16px calc(20px + env(safe-area-inset-bottom));
}

.status-sheet__title {
  display: block;
  font-size: 16px;
  font-weight: 600;
  margin-bottom: 8px;
}

.status-sheet__row {
  display: flex;
  align-items: center;
  gap: 12px;
  min-height: 56px;
  box-shadow: inset 0 -1px 0 var(--c-divider);
}

.status-sheet__main {
  flex: 1;
}

.status-sheet__label {
  display: block;
  font-size: 15px;
  color: var(--c-ink-1);
}

.status-sheet__row--on .status-sheet__label {
  color: var(--c-primary);
  font-weight: 600;
}

.status-sheet__desc {
  display: block;
  margin-top: 2px;
  font-size: 12px;
  color: var(--c-ink-3);
}

.status-sheet__check {
  color: var(--c-primary);
  font-size: 16px;
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
