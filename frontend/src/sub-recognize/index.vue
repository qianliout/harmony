<script setup lang="ts">
// 识别流程编排：上传 → 识别 → 核对 → 完成。
// 只保留步骤与上下文状态；核对行状态在 useReviewState，区域渲染拆到 components/。
import { onLoad } from '@dcloudio/uni-app'
import { computed, nextTick, onMounted, onUnmounted, ref, watch } from 'vue'

import { fetchBooks } from '@/api/book'
import { fetchEventDetail } from '@/api/event'
import { ApiError } from '@/api/http'
import { commitRecognizeTask, fetchRecognizeTask, uploadRecognizeImage } from '@/api/recognize'
import type { BookItem, EventItem, RecognizeCommitResult, RecognizeTask } from '@/api/types'
import AppNav from '@/components/AppNav.vue'
import EventPicker from '@/components/EventPicker.vue'
import { useHotkeys } from '@/composables/useHotkeys'
import { useWideLayout } from '@/composables/useWideLayout'
import { useDictStore } from '@/stores/dict'
import { useContextStore } from '@/stores/context'
import { centToYuanText, formatCent, yuanToCent } from '@/utils/money'
import { go, toast } from '@/utils/nav'

import ImageViewer from './components/ImageViewer.vue'
import PageCheckBar from './components/PageCheckBar.vue'
import RecognizeDone from './components/RecognizeDone.vue'
import RecognizeProgress from './components/RecognizeProgress.vue'
import RecognizeUpload from './components/RecognizeUpload.vue'
import ReviewList from './components/ReviewList.vue'
import ReviewTable from './components/ReviewTable.vue'
import { REVIEW_FIELDS, useReviewState, type ReviewField, type ReviewRow } from './useReviewState'

type Step = 'upload' | 'recognizing' | 'review' | 'done'

interface ItemError {
  index: number
  message: string
}

const steps = ['上传', '识别', '核对', '完成']
const POLL_INTERVAL_MS = 1500
const POLL_LIMIT_SECONDS = 100

const step = ref<Step>('upload')
const stepIndex = computed(() => ['upload', 'recognizing', 'review', 'done'].indexOf(step.value))
const { isWide } = useWideLayout()
const review = useReviewState()
const dictStore = useDictStore()
const contextStore = useContextStore()

const books = ref<BookItem[]>([])
const bookId = ref(0)
const eventId = ref(0)
const eventName = ref('')
const direction = ref<'in' | 'out'>('in')
const pageNo = ref('')
const pageTotalText = ref('')

const eventPickerShow = ref(false)
const file = ref<File | null>(null)
const previewUrl = ref('')
const uploading = ref(false)
const error = ref('')
const task = ref<RecognizeTask | null>(null)
const elapsed = ref(0)
const saving = ref(false)
const doneResult = ref<RecognizeCommitResult | null>(null)
const readonly = ref(false)
const resumeLoading = ref(false)
const resumeError = ref('')
const shortcutsShow = ref(false)
const showImageViewer = ref(false)

const viewerRef = ref<InstanceType<typeof ImageViewer> | null>(null)

let pollTimer: ReturnType<typeof setInterval> | null = null
let pollToken = 0
let pollBusy = false
let pollStarted = 0
let flowToken = 0

const imageUrl = computed(() => {
  if (step.value === 'review' || step.value === 'done') {
    return task.value?.imageUrl || previewUrl.value
  }
  return previewUrl.value || task.value?.imageUrl || ''
})
const pageTotalCent = computed(() => (pageTotalText.value ? yuanToCent(pageTotalText.value) : 0))
const pageConsistent = computed(
  () => pageTotalCent.value > 0 && pageTotalCent.value === review.selectedSumCent.value,
)

const commitBlockReason = computed(() => {
  if (readonly.value) return '只读查看，不能写入'
  if (saving.value) return '正在写入…'
  if (!task.value) return '没有可写入的识别任务'
  if (!review.selectedRows.value.length) return '请至少勾选一条'
  if (review.needPersonRows.value.length) return `还有 ${review.needPersonRows.value.length} 条需要选择亲友`
  if (review.pendingRows.value.length) return `还有 ${review.pendingRows.value.length} 行待确认`
  if (!bookId.value) return '先选择礼簿'
  if (!eventId.value) return '先选择事项'
  return ''
})
const canCommit = computed(() => !commitBlockReason.value)
const hasUnresolved = computed(
  () => review.pendingRows.value.length > 0 || review.needPersonRows.value.length > 0,
)

const footerText = computed(() => {
  if (readonly.value) return '只读查看 · 该任务已写入'
  if (saving.value) return '正在写入…'
  if (review.needPersonRows.value.length) return `还有 ${review.needPersonRows.value.length} 条需要选择亲友`
  if (review.pendingRows.value.length) return `还有 ${review.pendingRows.value.length} 行待确认`
  if (!review.selectedRows.value.length) return '未勾选任何行'
  return `已选 ${review.selectedRows.value.length} 条 · ${formatCent(review.selectedSumCent.value)}`
})

function stopPolling(): void {
  pollToken += 1
  pollBusy = false
  if (pollTimer) {
    clearInterval(pollTimer)
    pollTimer = null
  }
}

function applyTaskContext(current: RecognizeTask): void {
  if (current.page && !pageNo.value) {
    pageNo.value = current.page
  }
  if (current.pageTotalCent > 0 && !pageTotalText.value) {
    pageTotalText.value = centToYuanText(current.pageTotalCent)
  }
}

function openTask(current: RecognizeTask): void {
  applyTaskContext(current)
  review.loadTask(current, { page: pageNo.value, direction: direction.value })
  readonly.value = current.committed || readonly.value
  step.value = 'review'
}

async function pollOnce(id: number, token: number): Promise<void> {
  if (pollBusy) return
  pollBusy = true
  try {
    const current = await fetchRecognizeTask(id)
    if (token !== pollToken) return
    task.value = current
    if (current.status === 'succeeded') {
      stopPolling()
      openTask(current)
    } else if (current.status === 'failed') {
      stopPolling()
      error.value = current.errorMessage || '识别失败，请重试'
    }
  } catch (err) {
    if (token === pollToken) {
      stopPolling()
      error.value = err instanceof Error ? err.message : '获取识别结果失败'
    }
  } finally {
    if (token === pollToken) {
      pollBusy = false
    }
  }
}

function startPolling(id: number, offset = 0): void {
  stopPolling()
  const token = pollToken
  pollStarted = Date.now() - offset * 1000
  elapsed.value = offset
  error.value = ''
  step.value = 'recognizing'
  const tick = (): void => {
    if (token !== pollToken) return
    elapsed.value = Math.floor((Date.now() - pollStarted) / 1000)
    if (elapsed.value >= POLL_LIMIT_SECONDS) {
      stopPolling()
      error.value = '识别超过 100 秒仍未完成，请重试或改为手动录入'
      return
    }
    void pollOnce(id, token)
  }
  pollTimer = setInterval(tick, POLL_INTERVAL_MS)
  tick()
}

async function resumeTask(id: number, onlyRead: boolean): Promise<void> {
  resumeLoading.value = true
  resumeError.value = ''
  readonly.value = onlyRead
  try {
    const current = await fetchRecognizeTask(id)
    task.value = current
    applyTaskContext(current)
    if (current.status === 'succeeded') {
      openTask(current)
    } else if (current.status === 'failed') {
      step.value = 'recognizing'
      error.value = current.errorMessage || '识别失败，请重试'
    } else {
      startPolling(current.id)
    }
  } catch (err) {
    resumeError.value = err instanceof Error ? err.message : '加载识别任务失败'
  } finally {
    resumeLoading.value = false
  }
}

function onFileChange(payload: { file: File | null; previewUrl: string }): void {
  if (previewUrl.value.startsWith('blob:') && previewUrl.value !== payload.previewUrl) {
    URL.revokeObjectURL(previewUrl.value)
  }
  file.value = payload.file
  previewUrl.value = payload.previewUrl
}

async function startRecognize(picked: File): Promise<void> {
  if (uploading.value) return
  if (!bookId.value) {
    toast('先选择这页属于哪本礼簿')
    return
  }
  if (!eventId.value) {
    toast('先选择这页属于哪个事项')
    eventPickerShow.value = true
    return
  }
  const token = ++flowToken
  uploading.value = true
  error.value = ''
  step.value = 'recognizing'
  elapsed.value = 0
  try {
    const created = await uploadRecognizeImage(picked)
    if (token !== flowToken) return
    file.value = picked
    task.value = created
    readonly.value = false
    review.reset()
    startPolling(created.id)
  } catch (err) {
    if (token !== flowToken) return
    step.value = 'upload'
    error.value = err instanceof Error ? err.message : '上传失败'
  } finally {
    uploading.value = false
  }
}

function resetToUpload(): void {
  flowToken += 1
  stopPolling()
  if (previewUrl.value.startsWith('blob:')) {
    URL.revokeObjectURL(previewUrl.value)
  }
  previewUrl.value = ''
  file.value = null
  task.value = null
  error.value = ''
  elapsed.value = 0
  doneResult.value = null
  readonly.value = false
  review.reset()
  step.value = 'upload'
}

function retryRecognize(): void {
  if (file.value) {
    void startRecognize(file.value)
    return
  }
  if (task.value && task.value.status !== 'failed') {
    startPolling(task.value.id)
    return
  }
  resetToUpload()
}

function restartRecognize(): void {
  if (saving.value) return
  const picked = file.value
  if (!picked) {
    resetToUpload()
    return
  }
  void startRecognize(picked)
}

function toManualEntry(): void {
  go(`/sub-record/continuous?bookId=${bookId.value}&eventId=${eventId.value}&direction=${direction.value}`)
}

function onEventSelect(event: EventItem): void {
  eventId.value = event.id
  eventName.value = event.name
}

function openEventPicker(): void {
  eventPickerShow.value = true
}

function rowMenu(row: ReviewRow): void {
  uni.showActionSheet({
    itemList: ['在下方插入一行', '删除这一行'],
    success: (res) => {
      if (res.tapIndex === 0) {
        review.insertAfter(row, { page: pageNo.value, direction: direction.value })
      } else {
        review.removeRow(row)
      }
    },
  })
}

function batchApply(): void {
  if (!review.selectedRows.value.length) {
    toast('先勾选要批量修改的行')
    return
  }
  uni.showActionSheet({
    itemList: ['方向：收礼', '方向：送礼', '名目：现金', '名目：火炮', '名目：实物礼品'],
    success: (res) => {
      if (res.tapIndex === 0) review.applyBatch('direction', 'in')
      if (res.tapIndex === 1) review.applyBatch('direction', 'out')
      if (res.tapIndex === 2) review.applyBatch('itemKind', 'cash')
      if (res.tapIndex === 3) review.applyBatch('itemKind', 'huopao')
      if (res.tapIndex === 4) review.applyBatch('itemKind', 'gift')
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
      }
    },
  })
}

function readItemErrors(custom: Record<string, unknown> | null | undefined): ItemError[] {
  const raw = custom?.itemErrors
  if (!Array.isArray(raw)) return []
  const out: ItemError[] = []
  raw.forEach((entry) => {
    if (!entry || typeof entry !== 'object') return
    if (!('index' in entry) || !('message' in entry)) return
    const index = Number(entry.index)
    const message = typeof entry.message === 'string' ? entry.message : ''
    if (Number.isFinite(index) && message) {
      out.push({ index, message })
    }
  })
  return out
}

function locateRow(row: ReviewRow): void {
  review.activate(row)
  nextTick(() => {
    document.querySelector(`[data-row-key="${row.key}"]`)?.scrollIntoView({ block: 'center', behavior: 'smooth' })
    cellElement(row.key, 'name')?.focus()
  })
}

function locateFirstUnresolved(): void {
  const row = review.firstUnresolved.value
  if (row) locateRow(row)
}

async function commit(): Promise<void> {
  if (saving.value) return
  if (!task.value) return
  if (commitBlockReason.value) {
    toast(commitBlockReason.value)
    locateFirstUnresolved()
    return
  }
  if (!eventId.value) {
    toast('先选择这页属于哪个事项')
    eventPickerShow.value = true
    return
  }
  const selected = review.selectedRows.value
  const items = review.buildCommitItems({ eventId: eventId.value, bookId: bookId.value, page: pageNo.value })
  review.rows.value.forEach((row) => {
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
    readonly.value = true
    step.value = 'done'
  } catch (err) {
    if (err instanceof ApiError) {
      const itemErrors = readItemErrors(err.custom)
      itemErrors.forEach((itemError) => {
        const row = selected[itemError.index]
        if (row) row.error = itemError.message
      })
      const first = itemErrors[0] ? selected[itemErrors[0].index] : undefined
      if (first) locateRow(first)
      toast(itemErrors[0]?.message ?? err.message)
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
  resetToUpload()
  pageNo.value = nextPage
  pageTotalText.value = ''
}

function viewRecords(): void {
  go(`/sub-record/list?bookId=${bookId.value}`)
}

function cellElement(key: number, field: ReviewField): HTMLElement | null {
  const wrapper = document.querySelector<HTMLElement>(`[data-cell="${key}:${field}"]`)
  if (!wrapper) return null
  if (wrapper instanceof HTMLInputElement) return wrapper
  return wrapper.querySelector('input')
}

function focusedCell(): { key: number; field: ReviewField } | null {
  const active = document.activeElement as HTMLElement | null
  const holder = active?.closest<HTMLElement>('[data-cell]')
  if (!holder) return null
  const [keyText, fieldText] = (holder.dataset.cell ?? '').split(':')
  const field = REVIEW_FIELDS.find((item) => item === fieldText)
  if (!field) return null
  return { key: Number(keyText), field }
}

let editSnapshot: { key: number; field: ReviewField; value: string } | null = null

/** 记录单元格获得焦点时的值，Esc 取消编辑时恢复。 */
function onFocusIn(event: FocusEvent): void {
  const target = event.target as HTMLElement | null
  const holder = target?.closest<HTMLElement>('[data-cell]')
  if (!holder || !(target instanceof HTMLInputElement)) return
  const [keyText, fieldText] = (holder.dataset.cell ?? '').split(':')
  const field = REVIEW_FIELDS.find((item) => item === fieldText)
  if (!field) return
  editSnapshot = { key: Number(keyText), field, value: target.value }
}

function cancelEditing(): void {
  const cell = focusedCell()
  if (cell && editSnapshot && editSnapshot.key === cell.key && editSnapshot.field === cell.field) {
    const row = review.rows.value.find((item) => item.key === cell.key)
    if (row) review.restoreField(row, cell.field, editSnapshot.value)
  }
  const active = document.activeElement as HTMLElement | null
  active?.blur()
}

function focusCell(row: ReviewRow, field: ReviewField): void {
  review.activate(row)
  nextTick(() => {
    const input = cellElement(row.key, field)
    input?.focus()
    if (input instanceof HTMLInputElement) input.select()
    document.querySelector(`[data-row-key="${row.key}"]`)?.scrollIntoView({ block: 'nearest' })
  })
}

useHotkeys(
  {
    up: () => {
      const row = review.moveActive(-1)
      if (row) document.querySelector(`[data-row-key="${row.key}"]`)?.scrollIntoView({ block: 'nearest' })
    },
    down: () => {
      const row = review.moveActive(1)
      if (row) document.querySelector(`[data-row-key="${row.key}"]`)?.scrollIntoView({ block: 'nearest' })
    },
    enter: (editing) => {
      if (!editing) {
        const row = review.activeRow.value ?? review.rows.value[0]
        if (row) focusCell(row, 'name')
        return
      }
      const cell = focusedCell()
      const field = cell?.field ?? 'name'
      const row = review.moveActive(1)
      if (row) focusCell(row, field)
    },
    tab: (shift) => {
      const cell = focusedCell()
      const row = cell ? review.rows.value.find((item) => item.key === cell.key) : review.activeRow.value
      if (!row) return
      const index = cell ? REVIEW_FIELDS.indexOf(cell.field) : 0
      const next = (index + (shift ? -1 : 1) + REVIEW_FIELDS.length) % REVIEW_FIELDS.length
      focusCell(row, REVIEW_FIELDS[next])
    },
    escape: cancelEditing,
    space: () => {
      if (readonly.value) return
      const row = review.activeRow.value
      if (row) review.toggleRow(row)
    },
    addRow: () => {
      if (readonly.value) return
      const row = review.insertAfterActive({ page: pageNo.value, direction: direction.value })
      focusCell(row, 'name')
    },
    rotate: () => viewerRef.value?.rotate(),
    zoomIn: () => viewerRef.value?.zoomIn(),
    zoomOut: () => viewerRef.value?.zoomOut(),
    zoomFit: () => viewerRef.value?.fit(),
    commit: () => {
      void commit()
    },
  },
  () => isWide.value && step.value === 'review' && !readonly.value && !eventPickerShow.value && !shortcutsShow.value,
)

watch(bookId, (id) => {
  const chosen = books.value.find((book) => book.id === id)
  if (chosen?.defaultEventId && !eventId.value) {
    eventId.value = chosen.defaultEventId
    eventName.value = chosen.defaultEventName
  }
})

onLoad((query) => {
  if (query?.bookId) bookId.value = Number(query.bookId)
  if (query?.eventId) {
    eventId.value = Number(query.eventId)
    fetchEventDetail(eventId.value)
      .then((event) => {
        eventName.value = event.name
      })
      .catch(() => undefined)
  }
  if (query?.direction === 'out') direction.value = 'out'
  if (query?.page) pageNo.value = query.page
  const taskId = Number(query?.taskId ?? 0)
  if (taskId > 0) {
    void resumeTask(taskId, String(query?.readonly ?? '') === '1')
  }
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

onMounted(() => {
  window.addEventListener('focusin', onFocusIn)
})

onUnmounted(() => {
  window.removeEventListener('focusin', onFocusIn)
  stopPolling()
})
</script>

<template>
  <view class="screen" :class="{ 'screen--wide': isWide }">
    <AppNav title="识别礼簿" right="历史" @right="go('/sub-recognize/history')" />

    <view class="page" :class="{ 'page--wide': isWide }">
      <view class="steps">
        <view
          v-for="(label, index) in steps"
          :key="label"
          class="steps__item"
          :class="{ 'steps__item--on': index === stepIndex, 'steps__item--done': index < stepIndex }"
        >
          <text class="steps__dot">{{ index < stepIndex ? '✓' : index + 1 }}</text>
          <text>{{ label }}</text>
        </view>
      </view>

      <text v-if="resumeError" class="error">{{ resumeError }}</text>
      <text v-if="resumeLoading" class="loading-tip">正在加载识别任务…</text>

      <template v-if="step === 'upload'">
        <RecognizeUpload
          :books="books"
          :book-id="bookId"
          :event-name="eventName"
          :direction="direction"
          :page-no="pageNo"
          :page-total-text="pageTotalText"
          :uploading="uploading"
          :error="error"
          :wide="isWide"
          @update:book-id="bookId = $event"
          @update:direction="direction = $event"
          @update:page-no="pageNo = $event"
          @update:page-total-text="pageTotalText = $event"
          @pick-event="openEventPicker"
          @create-book="go('/sub-book/edit')"
          @file-change="onFileChange"
          @start="startRecognize"
        />
      </template>

      <template v-else-if="step === 'recognizing'">
        <RecognizeProgress
          :image-url="imageUrl"
          :elapsed="elapsed"
          :error="error"
          @cancel="resetToUpload"
          @retry="retryRecognize"
          @manual="toManualEntry"
        />
      </template>

      <template v-else-if="step === 'review'">
        <view v-if="isWide" class="wide">
          <view class="wide__left">
            <ImageViewer ref="viewerRef" :src="imageUrl" :task-id="task?.id" />
          </view>
          <view class="wide__right">
            <view class="wide__context">
              <view class="wide__title-row" @click="openEventPicker">
                <text class="wide__title">{{ books.find((book) => book.id === bookId)?.name || '未选礼簿' }}</text>
                <text class="wide__meta">
                  {{ eventName || '未选事项 · 点此选择' }} · {{ direction === 'in' ? '收礼' : '送礼' }}{{ pageNo ? ` · 第 ${pageNo} 页` : '' }}
                </text>
              </view>
              <text v-if="readonly" class="wide__note">
                只读查看：该任务已写入，记录明细请在礼簿中查看。
              </text>
              <PageCheckBar
                :review="review"
                :page-total-cent="pageTotalCent"
                :page-consistent="pageConsistent"
                :readonly="readonly"
                @locate="locateFirstUnresolved"
              />
            </view>
            <ReviewTable :review="review" :readonly="readonly" :page="pageNo" :direction="direction" />
          </view>
        </view>

        <template v-else>
          <view class="review-head">
            <image class="review-head__thumb" :src="imageUrl" mode="aspectFill" @click="showImageViewer = true" />
            <view class="review-head__main" @click="openEventPicker">
              <text class="review-head__title">识别出 {{ review.rows.value.length }} 条</text>
              <text class="review-head__meta">
                {{ books.find((book) => book.id === bookId)?.name }}{{ eventName ? ` · ${eventName}` : ' · 点此选择事项' }}
              </text>
              <text class="review-head__meta">{{ direction === 'in' ? '收礼' : '送礼' }}{{ pageNo ? ` · 第 ${pageNo} 页` : '' }}</text>
            </view>
            <text class="review-head__zoom" @click="showImageViewer = true">原图</text>
          </view>

          <text v-if="readonly" class="readonly-note">只读查看：该任务已写入，记录明细请在礼簿中查看。</text>

          <PageCheckBar
            :review="review"
            :page-total-cent="pageTotalCent"
            :page-consistent="pageConsistent"
            :readonly="readonly"
            @locate="locateFirstUnresolved"
          />

          <ReviewList
            :review="review"
            :readonly="readonly"
            :page="pageNo"
            :direction="direction"
            @menu="rowMenu"
            @batch="batchApply"
            @choose-book="chooseBookBatch"
          />
          <view class="review-spacer" />
        </template>
      </template>

      <template v-else>
        <RecognizeDone :result="doneResult" @view="viewRecords" @next="continueNext" />
      </template>
    </view>

    <view v-if="step === 'review'" class="bottom-bar" :class="{ 'bottom-bar--wide': isWide }">
      <view class="footer-summary">
        <text
          class="footer-hint"
          :class="{ 'footer-hint--action': hasUnresolved && !readonly }"
          @click="locateFirstUnresolved"
        >
          {{ footerText }}
        </text>
        <text v-if="isWide" class="shortcut-link" @click="shortcutsShow = true">快捷键</text>
        <text v-if="!canCommit && !readonly" class="footer-reason">{{ commitBlockReason }}</text>
      </view>
      <template v-if="!readonly">
        <view class="btn btn--secondary retry-btn" @click="restartRecognize">重新识别</view>
        <view class="btn btn--primary write-btn" :class="{ 'btn--disabled': !canCommit }" @click="commit">
          {{ saving ? '写入中…' : `写入 ${review.selectedRows.value.length} 条 · ${formatCent(review.selectedSumCent.value)}` }}
        </view>
      </template>
      <view v-else class="btn btn--secondary write-btn" @click="go(`/sub-record/list?bookId=${bookId}`)">
        查看礼簿记录
      </view>
    </view>

    <ImageViewer
      v-if="showImageViewer"
      overlay
      :src="imageUrl"
      :task-id="task?.id"
      @close="showImageViewer = false"
    />

    <view v-if="shortcutsShow" class="shortcut-mask" @click="shortcutsShow = false">
      <view class="shortcut-panel" @click.stop>
        <text class="shortcut-title">键盘快捷键</text>
        <view class="shortcut-row"><text class="shortcut-key">↑ ↓</text><text>在行之间移动当前行</text></view>
        <view class="shortcut-row"><text class="shortcut-key">Enter</text><text>编辑姓名；编辑中保存并跳到下一行</text></view>
        <view class="shortcut-row"><text class="shortcut-key">Tab / Shift+Tab</text><text>在当前行的字段间跳转</text></view>
        <view class="shortcut-row"><text class="shortcut-key">Esc</text><text>取消当前单元格编辑</text></view>
        <view class="shortcut-row"><text class="shortcut-key">Space</text><text>勾选或取消当前行</text></view>
        <view class="shortcut-row"><text class="shortcut-key">A</text><text>在当前行下方插入一行</text></view>
        <view class="shortcut-row"><text class="shortcut-key">R</text><text>原图顺时针旋转 90 度</text></view>
        <view class="shortcut-row"><text class="shortcut-key">＋ － 0</text><text>原图放大、缩小、适应窗口</text></view>
        <view class="shortcut-row"><text class="shortcut-key">Ctrl/⌘ + Enter</text><text>写入</text></view>
        <text class="shortcut-close" @click="shortcutsShow = false">知道了</text>
      </view>
    </view>

    <EventPicker v-model:show="eventPickerShow" @select="onEventSelect" />
  </view>
</template>

<style lang="scss" scoped>
.screen {
  padding-bottom: calc(96px + env(safe-area-inset-bottom));
}

.screen--wide {
  padding-bottom: calc(76px + env(safe-area-inset-bottom));
}

.page--wide {
  padding-left: 24px;
  padding-right: 24px;
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

.error {
  display: block;
  margin: 4px 0 10px;
  font-size: 13px;
  color: var(--c-danger);
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

.readonly-note {
  display: block;
  margin-bottom: 8px;
  font-size: 12px;
  color: var(--c-pending);
}

.review-spacer {
  height: 40px;
}

.wide {
  display: flex;
  align-items: flex-start;
  gap: 20px;
}

.wide__left {
  position: sticky;
  top: 12px;
  flex: 0 0 45%;
  max-width: 45%;
  height: calc(100vh - 160px);
  min-height: 360px;
}

.wide__right {
  flex: 1;
  min-width: 0;
  padding-bottom: 24px;
}

.wide__context {
  position: sticky;
  top: 0;
  z-index: 6;
  padding: 6px 0 8px;
  background: var(--c-bg);
}

.wide__title-row {
  display: flex;
  align-items: baseline;
  gap: 12px;
  margin-bottom: 8px;
  cursor: pointer;
}

.wide__title {
  font-size: 16px;
  font-weight: 600;
}

.wide__meta {
  font-size: 12px;
  color: var(--c-ink-3);
}

.wide__note {
  display: block;
  margin-bottom: 8px;
  font-size: 12px;
  color: var(--c-pending);
}

.footer-summary {
  display: flex;
  flex-direction: column;
  justify-content: center;
  min-width: 0;
  flex: 1;
  gap: 1px;
}

.footer-hint {
  font-size: 12px;
  color: var(--c-pending);
}

.footer-hint--action {
  text-decoration: underline;
  cursor: pointer;
}

.footer-reason {
  font-size: 11px;
  color: var(--c-ink-3);
}

.shortcut-link {
  font-size: 11px;
  color: var(--c-primary);
  cursor: pointer;
}

.retry-btn {
  flex: none;
  width: 104px;
  height: 44px;
  font-size: 14px;
}

.write-btn {
  flex: 1;
  max-width: 260px;
  height: 44px;
  font-size: 15px;
}

/* 覆盖 base.scss 中手机画布下 430px 的吸底条，宽屏时贴满 1280 内容区。 */
.bottom-bar--wide {
  left: 50%;
  right: auto;
  width: 100%;
  max-width: 1280px;
  transform: translateX(-50%);
}

.shortcut-mask {
  position: fixed;
  inset: 0;
  z-index: 70;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--c-mask);
}

.shortcut-panel {
  width: 420px;
  max-width: calc(100vw - 40px);
  padding: 18px 20px;
  border-radius: var(--r-l);
  background: var(--c-surface);
  box-shadow: var(--shadow-float);
}

.shortcut-title {
  display: block;
  margin-bottom: 12px;
  font-size: 16px;
  font-weight: 600;
}

.shortcut-row {
  display: flex;
  align-items: baseline;
  gap: 12px;
  padding: 5px 0;
  font-size: 13px;
  color: var(--c-ink-2);
  border-top: 1px solid var(--c-divider);
}

.shortcut-key {
  flex: 0 0 128px;
  font-family: $font-num;
  font-size: 12px;
  color: var(--c-ink-1);
}

.shortcut-close {
  display: block;
  margin-top: 14px;
  font-size: 14px;
  color: var(--c-primary);
  text-align: center;
}
</style>
