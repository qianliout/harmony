<script setup lang="ts">
// H5 桌面识别页：上传 → 识别中 → 核对 → 完成。
// 只做流程编排与桌面布局；上传、进度、完成、原图与核对行状态全部复用 sub-recognize 的共享实现。
import { computed, nextTick, onMounted, onUnmounted, ref, watch } from 'vue'

import { fetchBooks } from '@/api/book'
import { fetchEventDetail, fetchEvents, fetchRecentEvents } from '@/api/event'
import { ApiError } from '@/api/http'
import { commitRecognizeTask, fetchRecognizeTask, uploadRecognizeImage } from '@/api/recognize'
import type { BookItem, EventItem, RecognizeCommitResult, RecognizeTask } from '@/api/types'
import { useAuthStore } from '@/stores/auth'
import { useWebUiStore } from '@/stores/web'
import ImageViewer from '@/sub-recognize/components/ImageViewer.vue'
import PageCheckBar from '@/sub-recognize/components/PageCheckBar.vue'
import RecognizeDone from '@/sub-recognize/components/RecognizeDone.vue'
import RecognizeProgress from '@/sub-recognize/components/RecognizeProgress.vue'
import RecognizeUpload from '@/sub-recognize/components/RecognizeUpload.vue'
import { useReviewState, type ReviewRow } from '@/sub-recognize/useReviewState'
import { preciseDateLabel, todayString } from '@/utils/date'
import { centToYuanText, formatCent, yuanToCent } from '@/utils/money'
import Modal from '@/web/components/Modal.vue'
import PageHead from '@/web/components/PageHead.vue'
import WebIcon from '@/web/components/WebIcon.vue'
import WebReviewTable from '@/web/components/WebReviewTable.vue'
import { goWeb } from '@/web/nav'

type Step = 'upload' | 'recognizing' | 'review' | 'done'

interface ItemError {
  index: number
  message: string
}

interface PageInstanceLike {
  options?: Record<string, string | undefined>
  $page?: { options?: Record<string, string | undefined> }
}

const STEPS = ['上传', '识别', '核对', '完成']
const STEP_ORDER: Step[] = ['upload', 'recognizing', 'review', 'done']
const POLL_INTERVAL_MS = 1500
const POLL_LIMIT_SECONDS = 100
const HOME_URL = '/pages/web/index'
const HISTORY_URL = '/sub-web/recognize-history'

const auth = useAuthStore()
const ui = useWebUiStore()
const review = useReviewState()

const step = ref<Step>('upload')
const stepIndex = computed(() => STEP_ORDER.indexOf(step.value))

const books = ref<BookItem[]>([])
const bookId = ref(0)
const eventId = ref(0)
const eventName = ref('')
const direction = ref<'in' | 'out'>('in')
const date = ref(todayString())
const pageNo = ref('')
const pageTotalText = ref('')

const events = ref<EventItem[]>([])
const recentEvents = ref<EventItem[]>([])
const eventPicker = ref(false)
const eventKeyword = ref('')
const eventResults = ref<EventItem[]>([])
const eventSearching = ref(false)

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

let pollTimer: ReturnType<typeof setInterval> | null = null
let pollToken = 0
let pollBusy = false
let pollStarted = 0
let flowToken = 0
let eventTimer: ReturnType<typeof setTimeout> | null = null
let eventSearchSeq = 0

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

const pageMeta = computed(() => {
  if (step.value === 'upload') {
    return '一次一张：上传 → 识别 → 核对 → 写入；核对确认前不写入任何账目'
  }
  if (step.value === 'recognizing') {
    return '识别通常需要 20–60 秒，最多等待 100 秒'
  }
  if (step.value === 'review') {
    return readonly.value
      ? '只读查看该任务；记录明细请在礼簿中查看'
      : '标黄的行（同名、大小写不一致、字迹不清）必须逐条确认后才能写入'
  }
  return '写入完成，可继续识别下一张，或去礼簿查看'
})

/** 行是否需要人工处理；与 useReviewState.needsConfirm 同口径，并额外要求未勾选的黄色行显式处理。 */
function isUnresolved(row: ReviewRow): boolean {
  const low = review.isLowConfidence(row)
  if (!row.checked) {
    return row.matchStatus === 'multiple' || row.amountMismatch || low
  }
  if (!row.name.trim()) return true
  if (row.itemKind !== 'gift' && yuanToCent(row.amountText) <= 0) return true
  if (row.matchStatus === 'multiple') return true
  if (row.amountMismatch && !row.confirmed) return true
  if (low && !row.confirmed) return true
  return false
}

const unresolvedRows = computed(() => review.rows.value.filter(isUnresolved))
const unresolvedCount = computed(() => unresolvedRows.value.length)

const commitBlockReason = computed(() => {
  if (readonly.value) return '只读查看，不能写入'
  if (saving.value) return '正在写入…'
  if (!task.value) return '没有可写入的识别任务'
  if (!review.selectedRows.value.length) return '请至少勾选一条'
  if (review.needPersonRows.value.length) return `还有 ${review.needPersonRows.value.length} 条需要选择亲友`
  if (unresolvedCount.value) return `还有 ${unresolvedCount.value} 行待确认`
  if (!bookId.value) return '先选择礼簿'
  if (!eventId.value) return '先选择事项'
  return ''
})
const canCommit = computed(() => !commitBlockReason.value)

const writeLabel = computed(() => {
  if (saving.value) return '写入中…'
  if (unresolvedCount.value) return `还有 ${unresolvedCount.value} 行待确认`
  if (!review.selectedRows.value.length) return '未勾选行'
  return `写入 ${review.selectedRows.value.length} 条 · ${formatCent(review.selectedSumCent.value)}`
})

/** 读取页面查询参数；优先用 uni-app 页面 options，H5 刷新时退回 hash 查询串。 */
function readPageQuery(): Record<string, string> {
  const result: Record<string, string> = {}
  const pages = getCurrentPages()
  const current = pages[pages.length - 1] as unknown as PageInstanceLike | undefined
  const options = current?.options ?? current?.$page?.options ?? {}
  Object.entries(options).forEach(([key, value]) => {
    if (value !== undefined && value !== null) result[key] = String(value)
  })
  if (Object.keys(result).length || typeof window === 'undefined') return result
  const hash = window.location.hash
  const queryIndex = hash.indexOf('?')
  const search = queryIndex >= 0 ? hash.slice(queryIndex + 1) : window.location.search
  new URLSearchParams(search).forEach((value, key) => {
    result[key] = value
  })
  return result
}

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
  date.value = review.rows.value[0]?.date ?? date.value
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
    ui.toast('先选择这页属于哪本礼簿')
    return
  }
  if (!eventId.value) {
    ui.toast('先选择这页属于哪个事项')
    openEventPicker()
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

function revokePreview(): void {
  if (previewUrl.value.startsWith('blob:')) {
    URL.revokeObjectURL(previewUrl.value)
  }
  previewUrl.value = ''
}

function resetToUpload(): void {
  flowToken += 1
  stopPolling()
  revokePreview()
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
  if (bookId.value) {
    goWeb(`/sub-web/book?id=${bookId.value}&tab=cont`)
    return
  }
  goWeb('/sub-web/books')
}

function nextPageLabel(value: string): string {
  return value.replace(/(\d+)(?!.*\d)/, (match) => String(Number(match) + 1))
}

function continueNext(): void {
  const nextPage = pageNo.value ? nextPageLabel(pageNo.value) : ''
  resetToUpload()
  pageNo.value = nextPage
  pageTotalText.value = ''
  date.value = todayString()
}

function viewRecords(): void {
  if (bookId.value) {
    goWeb(`/sub-web/book?id=${bookId.value}`)
    return
  }
  goWeb('/sub-web/books')
}

async function loadBooks(): Promise<void> {
  try {
    books.value = await fetchBooks('normal')
  } catch (err) {
    error.value = err instanceof Error ? err.message : '礼簿加载失败'
  }
  if (!bookId.value) {
    bookId.value = books.value[0]?.id ?? 0
  }
  applyBookDefault()
}

async function loadEventOptions(): Promise<void> {
  try {
    events.value = (await fetchEvents({ pageSize: 100 })).items
  } catch {
    events.value = []
  }
}

async function loadRecentEvents(): Promise<void> {
  try {
    recentEvents.value = await fetchRecentEvents(8)
  } catch {
    recentEvents.value = []
  }
}

async function ensureEventOption(id: number, name: string): Promise<void> {
  if (!id) return
  if (events.value.some((item) => item.id === id)) {
    eventName.value = events.value.find((item) => item.id === id)?.name ?? name
    return
  }
  eventName.value = name
  try {
    const detail = await fetchEventDetail(id)
    if (!events.value.some((item) => item.id === id)) {
      events.value.unshift(detail)
    }
    eventName.value = detail.name
  } catch {
    // 事项详情取不到时保留名称，写入前仍可重新选择。
  }
}

function applyBookDefault(): void {
  if (eventId.value) return
  const book = books.value.find((item) => item.id === bookId.value)
  if (!book?.defaultEventId) return
  eventId.value = book.defaultEventId
  void ensureEventOption(book.defaultEventId, book.defaultEventName)
}

function setBookId(value: number): void {
  bookId.value = value
  applyBookDefault()
}

function onBookChange(event: Event): void {
  setBookId(Number((event.target as HTMLSelectElement).value))
}

function onEventChange(event: Event): void {
  const id = Number((event.target as HTMLSelectElement).value)
  eventId.value = id
  eventName.value = events.value.find((item) => item.id === id)?.name ?? ''
}

function onDateChange(event: Event): void {
  const value = (event.target as HTMLInputElement).value || todayString()
  date.value = value
  review.rows.value.forEach((row) => {
    row.date = value
  })
}

function onPageChange(event: Event): void {
  const value = (event.target as HTMLInputElement).value.trim()
  const previous = pageNo.value
  pageNo.value = value
  review.rows.value.forEach((row) => {
    if (!row.paperPage || row.paperPage === previous) {
      row.paperPage = value
    }
  })
}

function onPageTotalInput(event: Event): void {
  pageTotalText.value = (event.target as HTMLInputElement).value
}

function openEventPicker(): void {
  eventPicker.value = true
  eventKeyword.value = ''
  eventResults.value = recentEvents.value
  if (!eventResults.value.length) {
    void searchEvents('')
  }
}

function onEventKeywordInput(): void {
  if (eventTimer) clearTimeout(eventTimer)
  eventTimer = setTimeout(() => {
    void searchEvents(eventKeyword.value)
  }, 220)
}

async function searchEvents(keyword: string): Promise<void> {
  const seq = ++eventSearchSeq
  eventSearching.value = true
  try {
    const page = await fetchEvents({ keyword: keyword.trim() || undefined, pageSize: 30 })
    if (seq !== eventSearchSeq) return
    eventResults.value = page.items
  } catch {
    if (seq === eventSearchSeq) eventResults.value = []
  } finally {
    if (seq === eventSearchSeq) eventSearching.value = false
  }
}

function chooseEvent(item: EventItem): void {
  eventId.value = item.id
  eventName.value = item.name
  if (!events.value.some((event) => event.id === item.id)) {
    events.value.unshift(item)
  }
  eventPicker.value = false
}

function focusCell(row: ReviewRow, field: 'name' | 'amount' | 'note' = 'name'): void {
  review.activate(row)
  nextTick(() => {
    document.querySelector<HTMLInputElement>(`[data-cell="${row.key}:${field}"]`)?.focus()
  })
}

function locateFirstUnresolved(): void {
  const row = review.firstUnresolved.value ?? unresolvedRows.value[0]
  if (!row) return
  review.activate(row)
  nextTick(() => {
    document
      .querySelector(`[data-row-key="${row.key}"]`)
      ?.scrollIntoView({ block: 'center', behavior: 'smooth' })
    document.querySelector<HTMLInputElement>(`[data-cell="${row.key}:name"]`)?.focus()
  })
}

function addRow(): void {
  if (readonly.value) return
  const row = review.appendRow({ page: pageNo.value, direction: direction.value })
  focusCell(row)
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

async function commit(): Promise<void> {
  if (saving.value) return
  const current = task.value
  if (!current) return
  if (commitBlockReason.value) {
    ui.toast(commitBlockReason.value)
    locateFirstUnresolved()
    return
  }
  const selected = review.selectedRows.value
  const items = review.buildCommitItems({
    eventId: eventId.value,
    bookId: bookId.value,
    page: pageNo.value,
  })
  review.rows.value.forEach((row) => {
    row.error = ''
  })
  saving.value = true
  try {
    const result = await commitRecognizeTask(
      current.id,
      items,
      pageNo.value,
      pageTotalCent.value > 0 ? pageTotalCent.value : null,
      bookId.value,
    )
    doneResult.value = result
    readonly.value = true
    step.value = 'done'
    ui.markChanged()
    ui.toast(`已写入 ${result.count} 条记录`)
  } catch (err) {
    if (err instanceof ApiError) {
      const itemErrors = readItemErrors(err.custom)
      itemErrors.forEach((itemError) => {
        const row = selected[itemError.index]
        if (row) row.error = itemError.message
      })
      const first = itemErrors[0] ? selected[itemErrors[0].index] : undefined
      if (first) focusCell(first)
      ui.toast(itemErrors[0]?.message ?? err.message)
    } else {
      ui.toast(err instanceof Error ? err.message : '写入失败')
    }
  } finally {
    saving.value = false
  }
}

watch(
  [() => auth.ready, () => auth.isAdmin],
  ([ready, isAdmin]) => {
    if (ready && !isAdmin) {
      ui.toast('识别礼簿仅管理员可用')
      goWeb(HOME_URL)
    }
  },
  { immediate: true },
)

onMounted(async () => {
  const query = readPageQuery()
  const taskId = Number(query.taskId ?? 0)
  if (query.bookId) bookId.value = Number(query.bookId) || 0
  if (query.direction === 'out') direction.value = 'out'
  if (query.page) pageNo.value = query.page

  await Promise.all([loadBooks(), loadEventOptions(), loadRecentEvents()])
  if (query.eventId) {
    eventId.value = Number(query.eventId) || 0
    await ensureEventOption(eventId.value, eventName.value)
  } else {
    applyBookDefault()
  }
  if (taskId > 0) {
    await resumeTask(taskId, String(query.readonly ?? '') === '1')
  }
})

onUnmounted(() => {
  stopPolling()
  revokePreview()
  if (eventTimer) clearTimeout(eventTimer)
})
</script>

<template>
  <div v-if="auth.ready && !auth.isAdmin" class="web-alert web-alert--warn">
    识别礼簿仅管理员可用，正在返回工作台…
  </div>

  <template v-else>
    <PageHead title="识别礼簿" :meta="pageMeta">
      <template #tools>
        <div class="web-steps">
          <template v-for="(label, index) in STEPS" :key="label">
            <span v-if="index" class="web-steps__line" />
            <span :class="{ 'web-steps__done': index < stepIndex, 'web-steps__cur': index === stepIndex }">
              <i>{{ index < stepIndex ? '✓' : index + 1 }}</i>
              {{ label }}
            </span>
          </template>
        </div>
        <button class="web-btn" type="button" @click="goWeb(HISTORY_URL, 'push')">
          <WebIcon name="list" :size="15" />
          识别历史
        </button>
      </template>
    </PageHead>

    <div v-if="resumeError" class="web-alert web-alert--err">{{ resumeError }}</div>
    <div v-if="resumeLoading" class="web-empty">正在加载识别任务…</div>

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
        :wide="true"
        @update:book-id="setBookId"
        @update:direction="direction = $event"
        @update:page-no="pageNo = $event"
        @update:page-total-text="pageTotalText = $event"
        @pick-event="openEventPicker"
        @create-book="goWeb('/sub-web/books', 'push')"
        @file-change="onFileChange"
        @start="startRecognize"
      />
    </template>

    <template v-else-if="step === 'recognizing'">
      <div class="web-card rec-progress">
        <RecognizeProgress
          :image-url="imageUrl"
          :elapsed="elapsed"
          :error="error"
          @cancel="resetToUpload"
          @retry="retryRecognize"
          @manual="toManualEntry"
        />
      </div>
    </template>

    <template v-else-if="step === 'review'">
      <div v-if="readonly" class="web-alert web-alert--info">
        只读查看：该任务已写入，记录明细请在礼簿中查看。
      </div>

      <div class="web-rec">
        <div class="web-card web-photo">
          <div class="rec-viewer">
            <ImageViewer :src="imageUrl" :task-id="task?.id ?? 0" />
          </div>
          <div class="web-sm web-muted">滚轮缩放、拖动平移、双击适应；旋转只影响查看，不会重新识别。</div>
        </div>

        <div class="rec-right">
          <div class="web-card rec-ctx">
            <div class="web-ctx">
              <div class="web-field">
                <label for="rec-book">写入礼簿</label>
                <select
                  id="rec-book"
                  class="web-select rec-ctx__book"
                  :value="String(bookId)"
                  :disabled="readonly"
                  @change="onBookChange"
                >
                  <option v-for="book in books" :key="book.id" :value="String(book.id)">{{ book.name }}</option>
                </select>
              </div>

              <div class="web-field">
                <label for="rec-event">事项</label>
                <select
                  id="rec-event"
                  class="web-select rec-ctx__event"
                  :value="String(eventId)"
                  :disabled="readonly"
                  @change="onEventChange"
                >
                  <option value="0">选择事项</option>
                  <option v-for="event in events" :key="event.id" :value="String(event.id)">{{ event.name }}</option>
                </select>
                <button v-if="!readonly" class="web-btn web-btn--text web-btn--sm" type="button" @click="openEventPicker">
                  查找
                </button>
              </div>

              <div class="web-field">
                <label for="rec-date">日期</label>
                <input
                  id="rec-date"
                  class="web-input rec-ctx__date"
                  type="date"
                  :value="date"
                  :disabled="readonly"
                  @change="onDateChange"
                />
              </div>

              <div class="web-field">
                <label for="rec-page">纸本标记</label>
                <input
                  id="rec-page"
                  class="web-input rec-ctx__page"
                  type="text"
                  :value="pageNo"
                  :disabled="readonly"
                  placeholder="如 21-30"
                  @change="onPageChange"
                />
              </div>

              <div class="web-field">
                <label for="rec-total">本页合计</label>
                <input
                  id="rec-total"
                  class="web-input rec-ctx__total"
                  type="text"
                  inputmode="decimal"
                  :value="pageTotalText"
                  :disabled="readonly"
                  placeholder="纸本页脚合计"
                  @input="onPageTotalInput"
                />
              </div>
            </div>

            <div class="rec-ctx__check">
              <PageCheckBar
                :review="review"
                :page-total-cent="pageTotalCent"
                :page-consistent="pageConsistent"
                :readonly="readonly"
                @locate="locateFirstUnresolved"
              />
            </div>
          </div>

          <div class="web-card rec-review">
            <WebReviewTable :review="review" :readonly="readonly" />

            <div class="web-sumbar">
              <button v-if="!readonly" class="web-btn web-btn--text web-btn--sm" type="button" @click="addRow">
                <WebIcon name="plus" :size="14" />
                添加一行
              </button>
              <span class="web-sm web-muted">
                {{ review.rows.value.length }} 行 · 已选 {{ review.selectedRows.value.length }} 条 ·
                {{ formatCent(review.selectedSumCent.value) }}
              </span>
              <div class="rec-foot">
                <span v-if="!readonly && !canCommit" class="web-sm web-muted rec-foot__reason">
                  {{ commitBlockReason }}
                </span>
                <template v-if="!readonly">
                  <button class="web-btn" type="button" :disabled="saving" @click="restartRecognize">
                    重新识别
                  </button>
                  <button
                    class="web-btn web-btn--primary"
                    type="button"
                    :disabled="!canCommit"
                    :title="commitBlockReason"
                    @click="commit"
                  >
                    {{ writeLabel }}
                  </button>
                </template>
                <button v-else class="web-btn" type="button" @click="viewRecords">查看礼簿记录</button>
              </div>
            </div>
          </div>
        </div>
      </div>
    </template>

    <template v-else>
      <div class="web-card rec-done">
        <RecognizeDone :result="doneResult" @view="viewRecords" @next="continueNext" />
      </div>
    </template>

    <div v-if="eventPicker" class="web-mask" @click="eventPicker = false" />
    <Modal v-if="eventPicker" title="选择事项" @close="eventPicker = false">
      <div class="ev-search">
        <input
          v-model="eventKeyword"
          class="web-input"
          type="search"
          placeholder="输入事项名称查找"
          @input="onEventKeywordInput"
        />
      </div>
      <div class="ev-list">
        <button
          v-for="item in eventResults"
          :key="item.id"
          class="ev-item"
          type="button"
          @click="chooseEvent(item)"
        >
          <span class="ev-item__name">{{ item.name }}</span>
          <small class="ev-item__meta">
            {{ item.typeLabel }}
            <template v-if="item.eventDate"> · {{ preciseDateLabel(item.eventDate, item.datePrecision) }}</template>
          </small>
        </button>
        <div v-if="!eventResults.length" class="web-empty">
          {{ eventSearching ? '正在查找…' : '没有找到事项' }}
        </div>
      </div>
    </Modal>
  </template>
</template>

<style lang="scss" scoped>
.rec-progress {
  max-width: 560px;
  width: 100%;
  margin: 0 auto;
  padding: var(--sp-6);
}

.rec-done {
  max-width: 640px;
  margin: 0 auto;
}

.rec-ctx {
  position: sticky;
  top: var(--top-h);
  z-index: 10;
  overflow: hidden;
}

/* 桌面核对按文档保持左原图 45%、右表格 55%；窄屏仍回落单列。 */
.web-rec {
  grid-template-columns: minmax(0, 45fr) minmax(0, 55fr);
}

@media (max-width: 960px) {
  .web-rec {
    grid-template-columns: minmax(0, 1fr);
  }
}

.rec-ctx__book {
  min-width: 180px;
}

.rec-ctx__event {
  min-width: 200px;
}

.rec-ctx__date {
  width: 156px;
}

.rec-ctx__page {
  width: 120px;
}

.rec-ctx__total {
  width: 130px;
  font-family: $font-num;
}

.rec-ctx__check {
  padding: var(--sp-3) var(--sp-4);
}

.rec-viewer {
  height: min(62vh, 620px);
  min-height: 360px;
}

.rec-right {
  display: flex;
  flex-direction: column;
  gap: var(--sp-4);
  min-width: 0;
}

.rec-review {
  display: flex;
  flex-direction: column;
  min-width: 0;
}

.rec-review .web-sumbar {
  position: sticky;
  bottom: 0;
  z-index: 5;
  background: var(--c-surface);
}

.rec-review .web-sumbar {
  gap: var(--sp-3);
}

.rec-foot {
  display: flex;
  align-items: center;
  gap: var(--sp-2);
  margin-left: auto;
}

.rec-foot__reason {
  max-width: 240px;
  line-height: 16px;
  text-align: right;
}

.rec-foot .web-btn--primary {
  min-width: 180px;
}

.ev-search {
  padding-bottom: var(--sp-2);
}

.ev-search .web-input {
  width: 100%;
}

.ev-list {
  display: flex;
  flex-direction: column;
  max-height: 46vh;
  overflow-y: auto;
  border-radius: var(--r-s);
  background: var(--c-fill);
}

.ev-item {
  display: flex;
  align-items: baseline;
  gap: var(--sp-3);
  padding: 9px var(--sp-3);
  border: 0;
  border-bottom: 1px solid var(--c-divider);
  background: transparent;
  text-align: left;
}

.ev-item:last-child {
  border-bottom: 0;
}

.ev-item:hover {
  background: var(--c-surface);
}

.ev-item__name {
  flex: 1;
  min-width: 0;
  font-size: 14px;
}

.ev-item__meta {
  flex: none;
  color: var(--c-ink-3);
  font-size: 12px;
}
</style>
