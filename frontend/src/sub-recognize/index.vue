<script setup lang="ts">
import { onLoad } from '@dcloudio/uni-app'
import { computed, onUnmounted, ref, watch } from 'vue'

import { fetchBooks } from '@/api/book'
import { fetchEventDetail } from '@/api/event'
import { ApiError } from '@/api/http'
import { commitRecognizeTask, fetchRecognizeTask, uploadRecognizeImage } from '@/api/recognize'
import type {
  BookItem,
  EventItem,
  RecognizeCommitItem,
  RecognizeCommitResult,
  RecognizeItem,
  RecognizeTask,
} from '@/api/types'
import AppNav from '@/components/AppNav.vue'
import BookChips from '@/components/BookChips.vue'
import EventPicker from '@/components/EventPicker.vue'
import { useDictStore } from '@/stores/dict'
import { useContextStore } from '@/stores/context'
import { centToYuanText, formatCent, yuanToCent } from '@/utils/money'
import { formatDate, todayString } from '@/utils/date'
import { go, toast } from '@/utils/nav'

interface ReviewRepay {
  key: number
  enabled: boolean
  eventName: string
  occurredOn: string
  datePrecision: string
  cashText: string
  huopaoText: string
}

interface ReviewRow {
  key: number
  checked: boolean
  seq: number
  name: string
  personId: number
  matchStatus: 'single' | 'none' | 'multiple'
  candidates: { id: number; name: string; relationLabel: string }[]
  amountText: string
  amountUpper: string
  amountLower: number | null
  amountMismatch: boolean
  giftName: string
  itemKind: string
  direction: string
  date: string
  note: string
  paperPage: string
  paperSeq: number
  confidence: number
  confirmed: boolean
  duplicate: boolean
  error: string
  repays: ReviewRepay[]
  repaysEnabled: boolean
}

type Step = 'upload' | 'recognizing' | 'review' | 'done'

const steps = ['上传', '识别', '核对', '完成']
const step = ref<Step>('upload')
const stepIndex = computed(() => ['upload', 'recognizing', 'review', 'done'].indexOf(step.value))

const books = ref<BookItem[]>([])
const bookId = ref(0)
const eventId = ref(0)
const eventName = ref('')
const direction = ref<'in' | 'out'>('in')
const pageNo = ref('')
const pageTotalText = ref('')
const dictStore = useDictStore()
const contextStore = useContextStore()

const file = ref<File | null>(null)
const previewUrl = ref('')
const eventPickerShow = ref(false)
const uploading = ref(false)
const error = ref('')
const task = ref<RecognizeTask | null>(null)
const elapsed = ref(0)
const rows = ref<ReviewRow[]>([])
const saving = ref(false)
const doneResult = ref<RecognizeCommitResult | null>(null)
const itemKindOptions = ['现金', '火炮', '实物礼品']
let pollTimer: ReturnType<typeof setInterval> | null = null
let keySeq = 1

const imageUrl = computed(() => {
  if (!task.value) return previewUrl.value
  return task.value.imageUrl.startsWith('http') ? task.value.imageUrl : `${window.location.origin}${task.value.imageUrl}`
})

const selectedRows = computed(() => rows.value.filter((row) => row.checked))
const selectedSumCent = computed(() =>
  selectedRows.value.reduce((sum, row) => sum + yuanToCent(row.amountText), 0),
)
const pageTotalCent = computed(() => (pageTotalText.value ? yuanToCent(pageTotalText.value) : 0))
const pageConsistent = computed(
  () => pageTotalCent.value > 0 && pageTotalCent.value === selectedSumCent.value,
)

function needsConfirm(row: ReviewRow): boolean {
  if (!row.checked) return false
  if (!row.name.trim()) return true
  if (row.matchStatus === 'multiple' && !row.personId) return true
  if (row.confidence < 0.7 && !row.confirmed) return true
  if (row.amountMismatch && !row.confirmed) return true
  return false
}

const pendingRows = computed(() => rows.value.filter(needsConfirm))
const needPersonRows = computed(
  () => rows.value.filter((row) => row.checked && row.matchStatus === 'multiple' && !row.personId),
)
const canCommit = computed(
  () => selectedRows.value.length > 0 && pendingRows.value.length === 0 && !saving.value,
)

function stopPolling(): void {
  if (pollTimer) {
    clearInterval(pollTimer)
    pollTimer = null
  }
}

function resetFile(): void {
  stopPolling()
  file.value = null
  previewUrl.value = ''
  task.value = null
  rows.value = []
  error.value = ''
  elapsed.value = 0
  doneResult.value = null
  step.value = 'upload'
}

function onDrop(event: DragEvent): void {
  const dropped = event.dataTransfer?.files?.[0]
  if (dropped) {
    acceptFile(dropped)
  }
}

function acceptFile(picked: File): void {
  if (!['image/jpeg', 'image/png'].includes(picked.type)) {
    toast('只支持 jpg、png 图片')
    return
  }
  if (picked.size > 5 * 1024 * 1024) {
    toast('图片不能超过 5MB')
    return
  }
  error.value = ''
  if (previewUrl.value.startsWith('blob:')) {
    URL.revokeObjectURL(previewUrl.value)
  }
  file.value = picked
  previewUrl.value = URL.createObjectURL(picked)
}

async function rotateImage(): Promise<void> {
  if (!file.value) return
  const bitmap = await createImageBitmap(file.value)
  const canvas = document.createElement('canvas')
  canvas.width = bitmap.height
  canvas.height = bitmap.width
  const ctx = canvas.getContext('2d')
  if (!ctx) return
  ctx.translate(canvas.width / 2, canvas.height / 2)
  ctx.rotate(Math.PI / 2)
  ctx.drawImage(bitmap, -bitmap.width / 2, -bitmap.height / 2)
  const dataUrl = canvas.toDataURL('image/jpeg', 0.92)
  const blob = await (await fetch(dataUrl)).blob()
  const rotated = new File([blob], file.value.name, { type: 'image/jpeg' })
  acceptFile(rotated)
}

async function startRecognize(): Promise<void> {
  if (!file.value) {
    toast('先选择一张礼簿照片')
    return
  }
  if (!bookId.value) {
    toast('先选择这页属于哪本礼簿')
    return
  }
  if (!eventId.value) {
    toast('先选择这页属于哪个事项')
    openEventPicker()
    return
  }
  uploading.value = true
  error.value = ''
  try {
    const created = await uploadRecognizeImage(file.value)
    task.value = created
    step.value = 'recognizing'
    elapsed.value = 0
    startPolling(created.id)
  } catch (err) {
    error.value = err instanceof Error ? err.message : '上传失败'
  } finally {
    uploading.value = false
  }
}

function startPolling(id: number): void {
  stopPolling()
  const started = Date.now()
  pollTimer = setInterval(async () => {
    elapsed.value = Math.floor((Date.now() - started) / 1000)
    if (elapsed.value >= 60) {
      stopPolling()
      error.value = '识别超时，请重试或改为手动录入'
      return
    }
    try {
      const current = await fetchRecognizeTask(id)
      task.value = current
      if (current.status === 'succeeded') {
        stopPolling()
        buildRows(current)
        step.value = 'review'
      } else if (current.status === 'failed') {
        stopPolling()
        error.value = current.errorMessage || '识别失败，请重试'
      }
    } catch (err) {
      stopPolling()
      error.value = err instanceof Error ? err.message : '获取识别结果失败'
    }
  }, 1500)
}

function newRepay(draft: { event?: { name?: string } | null; occurredOn?: string; datePrecision?: string; items?: { itemKind: string; amountCent: number }[] }): ReviewRepay {
  const items = draft.items ?? []
  return {
    key: keySeq++,
    enabled: true,
    eventName: draft.event?.name ?? '',
    occurredOn: draft.occurredOn || todayString(),
    datePrecision: draft.datePrecision || 'day',
    cashText: centToYuanText(items.find((item) => item.itemKind === 'cash')?.amountCent ?? 0),
    huopaoText: centToYuanText(items.find((item) => item.itemKind === 'huopao')?.amountCent ?? 0),
  }
}

function toRow(item: RecognizeItem, index: number, current: RecognizeTask): ReviewRow {
  const repays = (item.repayDrafts ?? []).map(newRepay)
  return {
    key: keySeq++,
    checked: !item.duplicateOf,
    seq: item.seq || index + 1,
    name: item.name,
    personId: item.match?.status === 'single' ? item.match.personId : 0,
    matchStatus: item.match?.status ?? 'none',
    candidates: (item.match?.candidates ?? []).map((candidate) => ({
      id: candidate.id,
      name: candidate.name,
      relationLabel: candidate.relationLabel,
    })),
    amountText: centToYuanText(Math.round(item.amount * 100)),
    amountUpper: item.amountUpper,
    amountLower: item.amountLower,
    amountMismatch: item.amountMismatch,
    giftName: item.giftName ?? '',
    itemKind: item.itemKind || 'cash',
    direction: item.direction || direction.value,
    date: item.date && /^\d{4}-\d{2}-\d{2}$/.test(item.date) ? item.date : todayString(),
    note: item.note ?? '',
    paperPage: current.page || pageNo.value,
    paperSeq: item.seq || index + 1,
    confidence: item.confidence ?? 0,
    confirmed: false,
    duplicate: Boolean(item.duplicateOf),
    error: '',
    repays,
    repaysEnabled: repays.length > 0,
  }
}

function buildRows(current: RecognizeTask): void {
  if (current.page && !pageNo.value) {
    pageNo.value = current.page
  }
  if (current.pageTotalCent > 0 && !pageTotalText.value) {
    pageTotalText.value = centToYuanText(current.pageTotalCent)
  }
  rows.value = current.items.map((item, index) => toRow(item, index, current))
}

function blankRow(): ReviewRow {
  return {
    key: keySeq++,
    checked: true,
    seq: rows.value.length + 1,
    name: '',
    personId: 0,
    matchStatus: 'none',
    candidates: [],
    amountText: '',
    amountUpper: '',
    amountLower: null,
    amountMismatch: false,
    giftName: '',
    itemKind: 'cash',
    direction: direction.value,
    date: todayString(),
    note: '',
    paperPage: pageNo.value || task.value?.page || '',
    paperSeq: rows.value.length + 1,
    confidence: 1,
    confirmed: true,
    duplicate: false,
    error: '',
    repays: [],
    repaysEnabled: false,
  }
}

function addRow(): void {
  rows.value.push(blankRow())
}

function insertAfter(index: number): void {
  const row = blankRow()
  row.seq = rows.value[index].seq + 1
  row.paperSeq = rows.value[index].paperSeq + 1
  rows.value.splice(index + 1, 0, row)
}

function removeRow(index: number): void {
  rows.value.splice(index, 1)
}

function toggleRow(row: ReviewRow): void {
  row.checked = !row.checked
}

function toggleAll(): void {
  const allChecked = rows.value.every((row) => row.checked)
  rows.value.forEach((row) => {
    row.checked = !allChecked
  })
}

function openFileDialog(): void {
  // uni-app 会把模板里的 input 编译成 uni-input，file 类型要动态创建原生元素。
  const input = document.createElement('input')
  input.type = 'file'
  input.accept = 'image/*'
  input.setAttribute('capture', 'environment')
  input.onchange = () => {
    const picked = input.files?.[0]
    if (picked) acceptFile(picked)
  }
  input.click()
}

function onItemKindChange(row: ReviewRow, event: Event): void {
  const index = Number((event as unknown as { detail: { value: string } }).detail.value)
  row.itemKind = ['cash', 'huopao', 'gift'][index] ?? 'cash'
}

function onRepayEventInput(repay: ReviewRepay, event: Event): void {
  repay.eventName = (event as unknown as { detail: { value: string } }).detail.value
}

function onRepayDateChange(repay: ReviewRepay, event: Event): void {
  repay.occurredOn = (event as unknown as { detail: { value: string } }).detail.value
}

function onRepayCashInput(repay: ReviewRepay, event: Event): void {
  repay.cashText = (event as unknown as { detail: { value: string } }).detail.value
}

function onRepayHuopaoInput(repay: ReviewRepay, event: Event): void {
  repay.huopaoText = (event as unknown as { detail: { value: string } }).detail.value
}

function onNameInput(row: ReviewRow, event: Event): void {
  row.name = (event as unknown as { detail: { value: string } }).detail.value
  row.personId = 0
  row.matchStatus = 'none'
  row.candidates = []
}

function onAmountInput(row: ReviewRow, event: Event): void {
  row.amountText = (event as unknown as { detail: { value: string } }).detail.value
}

function onNoteInput(row: ReviewRow, event: Event): void {
  row.note = (event as unknown as { detail: { value: string } }).detail.value
}

function onDateChange(row: ReviewRow, event: Event): void {
  row.date = (event as unknown as { detail: { value: string } }).detail.value
}

function chooseCandidate(row: ReviewRow, candidate: { id: number; name: string }): void {
  row.personId = candidate.id
  row.matchStatus = 'single'
}

function confirmRow(row: ReviewRow): void {
  row.confirmed = true
}

function useLower(row: ReviewRow): void {
  if (row.amountLower === null) return
  row.amountText = centToYuanText(Math.round(row.amountLower * 100))
  row.amountMismatch = false
  row.confirmed = true
}

function rowMenu(index: number): void {
  uni.showActionSheet({
    itemList: ['在下方插入一行', '删除这一行'],
    success: (res) => {
      if (res.tapIndex === 0) {
        insertAfter(index)
      } else {
        removeRow(index)
      }
    },
  })
}

function batchApply(): void {
  const targets = selectedRows.value
  if (!targets.length) {
    toast('先勾选要批量修改的行')
    return
  }
  uni.showActionSheet({
    itemList: ['方向：收礼', '方向：送礼', '名目：现金', '名目：火炮', '名目：实物礼品'],
    success: (res) => {
      if (res.tapIndex === 0) targets.forEach((row) => (row.direction = 'in'))
      if (res.tapIndex === 1) targets.forEach((row) => (row.direction = 'out'))
      if (res.tapIndex === 2) targets.forEach((row) => (row.itemKind = 'cash'))
      if (res.tapIndex === 3) targets.forEach((row) => (row.itemKind = 'huopao'))
      if (res.tapIndex === 4) targets.forEach((row) => (row.itemKind = 'gift'))
    },
  })
}

function chooseBookBatch(): void {
  if (!books.value.length) return
  uni.showActionSheet({
    itemList: books.value.map((book) => book.name),
    success: (res) => {
      const chosen = books.value[res.tapIndex]
      if (chosen) {
        bookId.value = chosen.id
        selectedRows.value.forEach((row) => {
          row.paperPage = row.paperPage || pageNo.value
        })
      }
    },
  })
}

function repayItems(row: ReviewRepay): { itemKind: string; amountCent: number }[] {
  return [
    { itemKind: 'cash', amountCent: yuanToCent(row.cashText) },
    { itemKind: 'huopao', amountCent: yuanToCent(row.huopaoText) },
  ].filter((item) => item.amountCent > 0)
}

async function commit(): Promise<void> {
  if (!task.value || saving.value) return
  if (pendingRows.value.length > 0) {
    toast('还有待确认的行')
    return
  }
  if (!selectedRows.value.length) {
    toast('请至少勾选一条')
    return
  }
  if (needPersonRows.value.length > 0) {
    toast('还有行需要选择亲友')
    return
  }
  const items: RecognizeCommitItem[] = selectedRows.value.map((row) => ({
    personId: row.personId || undefined,
    personName: row.personId ? undefined : row.name.trim(),
    eventId: eventId.value,
    bookId: bookId.value,
    itemKind: row.itemKind,
    direction: row.direction,
    amountCent: yuanToCent(row.amountText),
    giftName: row.giftName,
    giftValueCent: 0,
    occurredOn: row.date,
    datePrecision: 'day',
    status: 'pending',
    remark: '',
    paperNote: row.note,
    paperPage: row.paperPage || pageNo.value,
    paperSeq: row.paperSeq,
    repays: row.repaysEnabled
      ? row.repays
          .filter((repay) => repay.enabled && repayItems(repay).length > 0)
          .map((repay) => ({
            event: repay.eventName.trim()
              ? { id: 0, name: repay.eventName.trim(), type: 'other', date: repay.occurredOn, datePrecision: repay.datePrecision }
              : null,
            occurredOn: repay.occurredOn,
            datePrecision: repay.datePrecision,
            items: repayItems(repay).map((item) => ({
              bookId: 0,
              itemKind: item.itemKind,
              amountCent: item.amountCent,
              giftName: '',
              giftValueCent: 0,
            })),
            remark: '',
          }))
      : [],
  }))
  rows.value.forEach((row) => {
    row.error = ''
  })
  saving.value = true
  try {
    const result = await commitRecognizeTask(
      task.value.id,
      items,
      pageNo.value,
      pageTotalCent.value > 0 ? pageTotalCent.value : null,
      bookId.value,
    )
    doneResult.value = result
    step.value = 'done'
  } catch (err) {
    if (err instanceof ApiError) {
      const custom = err.custom as { itemErrors?: { index: number; message: string }[] } | null
      custom?.itemErrors?.forEach((itemError) => {
        const row = rows.value[itemError.index]
        if (row) row.error = itemError.message
      })
      toast(custom?.itemErrors?.[0]?.message ?? err.message)
    } else {
      toast(err instanceof Error ? err.message : '写入失败')
    }
  } finally {
    saving.value = false
  }
}

function nextPageLabel(value: string): string {
  return value.replace(/(\d+)(?!.*\d)/, (match) => String(Number(match) + 1))
}

function continueNext(): void {
  const nextPage = pageNo.value ? nextPageLabel(pageNo.value) : ''
  resetFile()
  pageNo.value = nextPage
  pageTotalText.value = ''
}

function openEventPicker(): void {
  eventPickerShow.value = true
}

watch(bookId, (id) => {
  const chosen = books.value.find((book) => book.id === id)
  if (chosen?.defaultEventId && !eventId.value) {
    eventId.value = chosen.defaultEventId
    eventName.value = chosen.defaultEventName
  }
})

function onEventSelect(event: EventItem): void {
  eventId.value = event.id
  eventName.value = event.name
}

function toManualEntry(): void {
  go(`/sub-record/continuous?bookId=${bookId.value}&eventId=${eventId.value}&direction=${direction.value}`)
}

function previewImage(): void {
  if (!imageUrl.value) return
  uni.previewImage({ urls: [imageUrl.value] })
}

function onPageNoInput(event: Event): void {
  pageNo.value = (event as unknown as { detail: { value: string } }).detail.value
}

function onPageTotalInput(event: Event): void {
  pageTotalText.value = (event as unknown as { detail: { value: string } }).detail.value
}

onLoad((query) => {
  if (query?.bookId) bookId.value = Number(query.bookId)
  if (query?.eventId) {
    eventId.value = Number(query.eventId)
    fetchEventDetail(eventId.value).then((event) => {
      eventName.value = event.name
    }).catch(() => undefined)
  }
  if (query?.direction === 'out') direction.value = 'out'
  if (query?.page) pageNo.value = query.page
})

Promise.all([fetchBooks('normal'), dictStore.ensure(), contextStore.load()])
  .then(([loadedBooks]) => {
    books.value = loadedBooks
    if (!bookId.value) {
      const last = contextStore.settings.lastBookId
      bookId.value = loadedBooks.some((book) => book.id === last) ? last : loadedBooks[0]?.id ?? 0
    }
  })
  .catch(() => undefined)

onUnmounted(stopPolling)
</script>

<template>
  <view class="screen">
    <AppNav title="识别礼簿" />
    <view class="page">
      <view class="steps">
        <view v-for="(label, index) in steps" :key="label" class="steps__item" :class="{ 'steps__item--on': index === stepIndex, 'steps__item--done': index < stepIndex }">
          <text class="steps__dot">{{ index < stepIndex ? '✓' : index + 1 }}</text>
          <text>{{ label }}</text>
        </view>
      </view>

      <template v-if="step === 'upload'">
        <view
          class="drop"
          :class="{ 'drop--filled': previewUrl }"
          @dragover.prevent
          @drop.prevent="onDrop"
          @click="openFileDialog"
        >
          <image v-if="previewUrl" class="drop__image" :src="previewUrl" mode="aspectFit" />
          <template v-else>
            <text class="drop__seal">簿</text>
            <text class="drop__title">拍照或选择礼簿照片</text>
            <text class="drop__hint">支持 jpg / png，不超过 5MB</text>
          </template>
        </view>
        <view v-if="previewUrl" class="drop-actions">
          <text class="drop-actions__btn" @click="rotateImage">旋转</text>
          <text class="drop-actions__btn" @click="openFileDialog">重选</text>
          <text class="drop-actions__btn" @click="resetFile">移除</text>
        </view>

        <view class="card context">
          <view class="field">
            <text class="field__label">礼簿</text>
            <view class="field__value">
              <BookChips v-model="bookId" :books="books" @create="go('/sub-book/edit')" />
            </view>
          </view>
          <view class="field" @click="openEventPicker">
            <text class="field__label">事项</text>
            <text class="field__value">{{ eventName || '选填，礼簿默认事项可用' }}</text>
            <text class="field__arrow">›</text>
          </view>
          <view class="field">
            <text class="field__label">方向</text>
            <view class="segment">
              <text class="segment__item" :class="{ 'segment__item--on': direction === 'in' }" @click="direction = 'in'">收礼</text>
              <text class="segment__item" :class="{ 'segment__item--on': direction === 'out' }" @click="direction = 'out'">送礼</text>
            </view>
          </view>
          <view class="field">
            <text class="field__label">页码</text>
            <input class="field__input" :value="pageNo" placeholder="如 1-10" @input="onPageNoInput" />
          </view>
          <view class="field">
            <text class="field__label">本页合计</text>
            <input class="field__input" type="digit" :value="pageTotalText" placeholder="纸本页脚写的合计" @input="onPageTotalInput" />
          </view>
        </view>

        <text v-if="error" class="error">{{ error }}</text>
        <view class="btn btn--primary start" :class="{ 'btn--disabled': uploading }" @click="startRecognize">
          {{ uploading ? '上传中…' : '开始识别' }}
        </view>
        <text class="hint">识别结果只是草稿，核对确认后才会写入账目。</text>
      </template>

      <template v-else-if="step === 'recognizing'">
        <view class="scan">
          <image v-if="previewUrl || imageUrl" class="scan__image" :src="previewUrl || imageUrl" mode="aspectFit" />
          <view v-if="!error" class="scan__line" />
        </view>
        <view v-if="!error" class="recognizing">
          <text class="recognizing__title">{{ elapsed < 5 ? '正在识别…' : elapsed < 20 ? '字迹较多，再等一下…' : '快好了…' }}</text>
          <text class="recognizing__hint">已等待 {{ elapsed }} 秒</text>
          <view class="btn btn--secondary" @click="resetFile">取消</view>
        </view>
        <view v-else class="failed">
          <text class="failed__title">识别没有完成</text>
          <text class="failed__message">{{ error }}</text>
          <view class="failed__actions">
            <view class="btn btn--secondary failed__btn" @click="toManualEntry">改为手动录入</view>
            <view class="btn btn--primary failed__btn" @click="resetFile">重试</view>
          </view>
        </view>
      </template>

      <template v-else-if="step === 'review'">
        <view class="review-head">
          <image class="review-head__thumb" :src="imageUrl" mode="aspectFill" @click="previewImage" />
          <view class="review-head__main" @click="openEventPicker">
            <text class="review-head__title">识别出 {{ rows.length }} 条</text>
            <text class="review-head__meta">{{ books.find((book) => book.id === bookId)?.name }}{{ eventName ? ` · ${eventName}` : ' · 点此选择事项' }}</text>
            <text class="review-head__meta">{{ direction === 'in' ? '收礼' : '送礼' }}{{ pageNo ? ` · 第 ${pageNo} 页` : '' }}</text>
          </view>
          <text class="review-head__zoom" @click="previewImage">原图</text>
        </view>

        <view class="checkbar" :class="{ 'checkbar--ok': pageConsistent, 'checkbar--warn': pageTotalCent > 0 && !pageConsistent }">
          <text v-if="pageTotalCent > 0">
            纸本合计 {{ formatCent(pageTotalCent) }} · 已选 {{ formatCent(selectedSumCent) }}
            {{ pageConsistent ? '✓' : `差 ${formatCent(selectedSumCent - pageTotalCent)}` }}
          </text>
          <text v-else>未填本页合计</text>
          <text v-if="pendingRows.length" class="checkbar__pending">还有 {{ pendingRows.length }} 行待确认 ›</text>
        </view>

        <view class="toolbar">
          <text class="toolbar__all" @click="toggleAll">☑ 全选</text>
          <text class="toolbar__btn" @click="batchApply">批量</text>
          <text class="toolbar__btn" @click="chooseBookBatch">礼簿</text>
        </view>

        <view class="rows">
          <view
            v-for="(row, index) in rows"
            :key="row.key"
            class="rrow"
            :class="{ 'rrow--off': !row.checked, 'rrow--pending': needsConfirm(row), 'rrow--error': row.error }"
          >
            <view class="rrow__top">
              <view class="cb" :class="{ 'cb--on': row.checked }" @click="toggleRow(row)">
                <text v-if="row.checked">✓</text>
              </view>
              <text class="rrow__seq">{{ row.seq }}</text>
              <input class="rrow__name" :value="row.name" placeholder="姓名" @input="onNameInput(row, $event)" />
              <text v-if="row.matchStatus === 'multiple'" class="match match--multi">多个同名</text>
              <text v-else-if="row.matchStatus === 'single'" class="match match--ok">已匹配</text>
              <text v-else class="match match--new">将新建</text>
              <text v-if="row.duplicate" class="badge">可能重复</text>
              <text v-if="needsConfirm(row)" class="badge badge--warn">待确认</text>
            </view>

            <view v-if="row.candidates.length && row.matchStatus === 'multiple'" class="candidates">
              <text
                v-for="candidate in row.candidates"
                :key="candidate.id"
                class="chip chip--flat"
                :class="{ 'chip--flat-on': row.personId === candidate.id }"
                @click="chooseCandidate(row, candidate)"
              >
                {{ candidate.name }} · {{ candidate.relationLabel }}
              </text>
            </view>

            <view class="rrow__fields">
              <input class="rrow__amount" type="digit" :value="row.amountText" placeholder="金额" @input="onAmountInput(row, $event)" />
              <picker
                mode="selector"
                :range="itemKindOptions"
                :value="row.itemKind === 'huopao' ? 1 : row.itemKind === 'gift' ? 2 : 0"
                @change="onItemKindChange(row, $event)"
              >
                <text class="rrow__select">{{ dictStore.itemKindLabel(row.itemKind) }}</text>
              </picker>
              <picker mode="date" :value="row.date" @change="onDateChange(row, $event)">
                <text class="rrow__select">{{ formatDate(row.date) }}</text>
              </picker>
            </view>

            <view v-if="row.amountMismatch" class="mismatch">
              <text class="mismatch__text">大写 {{ row.amountUpper }} · 小写 {{ row.amountLower ?? '-' }}</text>
              <text class="mismatch__pick" @click="row.confirmed = true">按大写</text>
              <text class="mismatch__pick" @click="useLower(row)">按小写</text>
            </view>

            <input class="rrow__note" :value="row.note" placeholder="纸本标注原文（选填）" @input="onNoteInput(row, $event)" />

            <view v-if="row.repays.length" class="repays">
              <view class="repays__head">
                <text class="repays__title">标注里的还礼</text>
                <view class="switch" :class="{ 'switch--on': row.repaysEnabled }" @click="row.repaysEnabled = !row.repaysEnabled">
                  <view class="switch__dot" />
                </view>
                <text class="repays__hint">一并补录</text>
              </view>
              <view v-for="(repay, repayIndex) in row.repays" :key="repay.key" class="repay">
                <text class="repay__no">{{ repayIndex + 1 }}</text>
                <input
                  class="repay__event"
                  :value="repay.eventName"
                  placeholder="对方事项"
                  @input="onRepayEventInput(repay, $event)"
                />
                <picker mode="date" :value="repay.occurredOn" @change="onRepayDateChange(repay, $event)">
                  <text class="repay__date">{{ formatDate(repay.occurredOn) }}</text>
                </picker>
                <input
                  class="repay__amount"
                  type="digit"
                  :value="repay.cashText"
                  placeholder="现金"
                  @input="onRepayCashInput(repay, $event)"
                />
                <input
                  class="repay__amount"
                  type="digit"
                  :value="repay.huopaoText"
                  placeholder="火炮"
                  @input="onRepayHuopaoInput(repay, $event)"
                />
              </view>
            </view>

            <view class="rrow__actions">
              <text v-if="needsConfirm(row)" class="rrow__confirm" @click="confirmRow(row)">确认无误</text>
              <text class="rrow__menu" @click="rowMenu(index)">⋯</text>
            </view>
            <text v-if="row.error" class="rrow__error">{{ row.error }}</text>
          </view>
        </view>

        <view class="addrow" @click="addRow">＋ 添加一行（照片不全或漏识别时手动补录）</view>
        <view class="review-spacer" />
      </template>

      <template v-else>
        <view class="done">
          <view class="done__seal">✓</view>
          <text class="done__title">已写入 {{ doneResult?.count ?? 0 }} 条记录</text>
          <text class="done__sum">合计 {{ formatCent(doneResult?.sumCent ?? 0) }}</text>
          <text class="done__meta">新建亲友 {{ doneResult?.newPersons ?? 0 }} 人 · 补录还礼 {{ doneResult?.repayCount ?? 0 }} 笔</text>
          <view class="done__actions">
            <view class="btn btn--secondary done__btn" @click="go(`/sub-record/list?bookId=${bookId}`)">查看这些记录</view>
            <view class="btn btn--primary done__btn" @click="continueNext">继续识别下一页</view>
          </view>
        </view>
      </template>
    </view>

    <view v-if="step === 'review'" class="bottom-bar">
      <text v-if="pendingRows.length" class="footer-hint">还有 {{ pendingRows.length }} 行待确认</text>
      <text v-else-if="needPersonRows.length" class="footer-hint">还有 {{ needPersonRows.length }} 条需要选择亲友</text>
      <text v-else class="footer-hint">已选 {{ selectedRows.length }} 条</text>
      <view class="btn btn--primary write-btn" :class="{ 'btn--disabled': !canCommit }" @click="commit">
        {{ saving ? '写入中…' : `写入 ${selectedRows.length} 条 · ${formatCent(selectedSumCent)}` }}
      </view>
    </view>

    <EventPicker v-model:show="eventPickerShow" @select="onEventSelect" />
  </view>
</template>

<style lang="scss" scoped>
.screen {
  padding-bottom: calc(96px + env(safe-area-inset-bottom));
}

.steps {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 4px 0 12px;
  font-size: 12px;
  color: var(--c-ink-3);
}

.steps__item {
  display: flex;
  align-items: center;
  gap: 5px;
}

.steps__dot {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 20px;
  height: 20px;
  border-radius: var(--r-full);
  background: var(--c-fill);
  font-size: 11px;
  font-family: $font-num;
}

.steps__item--done .steps__dot {
  background: var(--c-repaid);
  color: var(--c-surface);
}

.steps__item--on {
  color: var(--c-ink-1);
  font-weight: 600;
}

.steps__item--on .steps__dot {
  background: var(--c-primary);
  color: var(--c-surface);
}

.drop {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  height: 220px;
  border: 1.5px dashed var(--c-line);
  border-radius: var(--r-m);
  background: var(--c-surface);
  overflow: hidden;
}

.drop--filled {
  border-style: solid;
  padding: 0;
}

.drop__image {
  width: 100%;
  height: 100%;
}

.drop__seal {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 56px;
  height: 56px;
  border: 1px solid var(--c-none);
  border-radius: 14px;
  color: var(--c-none);
  font-family: $font-serif;
  font-size: 26px;
  opacity: 0.6;
}

.drop__title {
  margin-top: 12px;
  font-size: 15px;
  font-weight: 500;
}

.drop__hint {
  margin-top: 4px;
  font-size: 12px;
  color: var(--c-ink-3);
}

.drop-actions {
  display: flex;
  justify-content: flex-end;
  gap: 16px;
  padding: 8px 4px 0;
}

.drop-actions__btn {
  font-size: 13px;
  color: var(--c-ink-2);
}

.hidden {
  display: none;
}

.context {
  margin-top: 12px;
  overflow: hidden;
}

.context .field {
  padding: 0 12px;
}

.segment {
  display: flex;
  gap: 4px;
  padding: 2px;
  border-radius: var(--r-s);
  background: var(--c-fill);
}

.segment__item {
  padding: 6px 14px;
  border-radius: 8px;
  font-size: 13px;
  color: var(--c-ink-2);
}

.segment__item--on {
  background: var(--c-surface);
  color: var(--c-ink-1);
  font-weight: 500;
}

.start {
  margin-top: 16px;
}

.hint {
  display: block;
  margin-top: 10px;
  font-size: 12px;
  color: var(--c-ink-3);
  text-align: center;
}

.error {
  display: block;
  margin-top: 10px;
  font-size: 13px;
  color: var(--c-danger);
}

.scan {
  position: relative;
  height: 260px;
  border-radius: var(--r-m);
  background: var(--c-ink-1);
  overflow: hidden;
}

.scan__image {
  width: 100%;
  height: 100%;
}

.scan__line {
  position: absolute;
  left: 0;
  right: 0;
  top: 0;
  height: 2px;
  background: var(--c-primary);
  box-shadow: 0 0 12px var(--c-primary);
  animation: scan 1.6s linear infinite;
}

@keyframes scan {
  0% {
    top: 0;
  }
  100% {
    top: 100%;
  }
}

.recognizing,
.failed {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 20px 0;
}

.recognizing__title,
.failed__title {
  font-size: 17px;
  font-weight: 600;
}

.recognizing__hint,
.failed__message {
  margin: 6px 0 16px;
  font-size: 13px;
  color: var(--c-ink-3);
  text-align: center;
}

.failed__actions {
  display: flex;
  gap: 10px;
  width: 100%;
}

.failed__btn {
  flex: 1;
}

.review-head {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 4px 0 10px;
}

.review-head__thumb {
  width: 72px;
  height: 72px;
  border-radius: var(--r-s);
  background: var(--c-fill);
}

.review-head__main {
  flex: 1;
  min-width: 0;
}

.review-head__title {
  display: block;
  font-size: 16px;
  font-weight: 600;
}

.review-head__meta {
  display: block;
  margin-top: 2px;
  font-size: 12px;
  color: var(--c-ink-3);
}

.review-head__zoom {
  font-size: 13px;
  color: var(--c-primary);
}

.checkbar {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 9px 12px;
  border-radius: var(--r-s);
  background: var(--c-fill);
  font-size: 12px;
  color: var(--c-ink-2);
}

.checkbar--ok {
  background: var(--c-repaid-light);
  color: #2f5e35;
}

.checkbar--warn {
  background: var(--c-pending-light);
  color: #7a4e07;
}

.checkbar__pending {
  margin-left: auto;
  font-weight: 500;
  white-space: nowrap;
}

.toolbar {
  display: flex;
  align-items: center;
  gap: 14px;
  padding: 10px 0;
  font-size: 13px;
  color: var(--c-ink-2);
}

.toolbar__all {
  color: var(--c-ink-1);
  font-weight: 500;
}

.toolbar__btn {
  color: var(--c-primary);
}

.rrow {
  margin-bottom: 10px;
  padding: 12px;
  border-radius: var(--r-m);
  background: var(--c-surface);
  box-shadow: var(--shadow-card);
  border: 1px solid transparent;
}

.rrow--off {
  opacity: 0.55;
}

.rrow--pending {
  border-color: var(--c-pending);
}

.rrow--error {
  border-color: var(--c-danger);
}

.rrow__top {
  display: flex;
  align-items: center;
  gap: 8px;
}

.cb {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 18px;
  height: 18px;
  border: 1.5px solid var(--c-ink-4);
  border-radius: 5px;
  color: var(--c-surface);
  font-size: 11px;
  flex: none;
}

.cb--on {
  background: var(--c-primary);
  border-color: var(--c-primary);
}

.rrow__seq {
  width: 22px;
  font-family: $font-num;
  font-size: 12px;
  color: var(--c-ink-3);
  flex: none;
}

.rrow__name {
  flex: 1;
  min-width: 0;
  height: 32px;
  padding: 0 8px;
  border-radius: 8px;
  background: var(--c-fill);
  font-size: 14px;
}

.match {
  flex: none;
  font-size: 11px;
  white-space: nowrap;
}

.match--ok {
  color: var(--c-repaid);
}

.match--new {
  color: #3a63a0;
}

.match--multi {
  color: var(--c-pending);
}

.badge {
  flex: none;
  padding: 1px 5px;
  border-radius: 5px;
  background: var(--c-fill);
  font-size: 10px;
  color: var(--c-ink-2);
}

.badge--warn {
  background: var(--c-pending);
  color: var(--c-surface);
}

.candidates {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  margin-top: 8px;
  padding-left: 48px;
}

.rrow__fields {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-top: 8px;
  padding-left: 48px;
}

.rrow__amount {
  width: 84px;
  height: 32px;
  padding: 0 8px;
  border-radius: 8px;
  background: var(--c-fill);
  font-family: $font-num;
  font-size: 15px;
  text-align: right;
}

.rrow__select {
  display: inline-flex;
  align-items: center;
  height: 32px;
  padding: 0 8px;
  border-radius: 8px;
  background: var(--c-fill);
  font-size: 12px;
  color: var(--c-ink-2);
  white-space: nowrap;
}

.mismatch {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-top: 6px;
  padding-left: 48px;
  font-size: 11px;
}

.mismatch__text {
  color: var(--c-pending);
}

.mismatch__pick {
  color: var(--c-primary);
  font-weight: 500;
}

.rrow__note {
  height: 32px;
  margin-top: 8px;
  padding: 0 8px;
  border-radius: 8px;
  background: var(--c-fill);
  font-family: $font-serif;
  font-size: 13px;
}

.repays {
  margin-top: 10px;
  padding: 8px 10px;
  border-radius: var(--r-s);
  background: #fcfaf7;
}

.repays__head {
  display: flex;
  align-items: center;
  gap: 8px;
}

.repays__title {
  font-size: 12px;
  font-weight: 500;
  color: var(--c-ink-2);
}

.repays__hint {
  font-size: 11px;
  color: var(--c-ink-3);
}

.switch {
  position: relative;
  width: 34px;
  height: 20px;
  border-radius: var(--r-full);
  background: var(--c-ink-4);
}

.switch--on {
  background: var(--c-repaid);
}

.switch__dot {
  position: absolute;
  top: 2px;
  left: 2px;
  width: 16px;
  height: 16px;
  border-radius: var(--r-full);
  background: var(--c-surface);
  transition: transform 0.2s;
}

.switch--on .switch__dot {
  transform: translateX(14px);
}

.repay {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-top: 6px;
}

.repay__no {
  width: 16px;
  font-family: $font-num;
  font-size: 11px;
  color: var(--c-ink-3);
}

.repay__event {
  flex: 1;
  min-width: 0;
  height: 28px;
  padding: 0 6px;
  border-radius: 6px;
  background: var(--c-surface);
  font-size: 12px;
}

.repay__date {
  font-size: 11px;
  color: var(--c-ink-2);
  white-space: nowrap;
}

.repay__amount {
  width: 58px;
  height: 28px;
  padding: 0 6px;
  border-radius: 6px;
  background: var(--c-surface);
  font-family: $font-num;
  font-size: 12px;
  text-align: right;
}

.rrow__actions {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 14px;
  margin-top: 8px;
}

.rrow__confirm {
  font-size: 12px;
  color: var(--c-pending);
  font-weight: 500;
}

.rrow__menu {
  font-size: 18px;
  color: var(--c-ink-3);
  padding: 0 4px;
}

.rrow__error {
  display: block;
  margin-top: 4px;
  font-size: 12px;
  color: var(--c-danger);
}

.addrow {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 38px;
  margin-top: 8px;
  border: 1.5px dashed var(--c-line);
  border-radius: var(--r-s);
  font-size: 13px;
  color: var(--c-ink-2);
}

.review-spacer {
  height: 40px;
}

.footer-hint {
  flex: none;
  font-size: 12px;
  color: var(--c-pending);
}

.write-btn {
  flex: 1;
  height: 44px;
  font-size: 15px;
}

.done {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 56px 16px;
}

.done__seal {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 64px;
  height: 64px;
  border-radius: var(--r-full);
  background: var(--c-repaid-light);
  color: var(--c-repaid);
  font-size: 30px;
}

.done__title {
  margin-top: 16px;
  font-size: 20px;
  font-weight: 600;
}

.done__sum {
  margin-top: 6px;
  font-family: $font-num;
  font-size: 17px;
  color: var(--c-ink-2);
}

.done__meta {
  margin-top: 6px;
  font-size: 13px;
  color: var(--c-ink-3);
}

.done__actions {
  display: flex;
  gap: 10px;
  width: 100%;
  margin-top: 28px;
}

.done__btn {
  flex: 1;
}
</style>
