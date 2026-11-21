<script setup lang="ts">
// 识别礼簿：上传照片 → 后台识别 → 逐条核对 → 选礼簿和方向后写入。仅 H5 编译。
import { computed, onBeforeUnmount, onMounted, reactive, ref } from 'vue'

import { fetchBooks } from '@/api/book'
import { commitRecognizeTask, fetchRecognizeTask, recognizeImageURL, uploadRecognizeImage } from '@/api/recognize'
import type { BookItem, Direction, RecognizeCommitResult, RecognizeTask } from '@/api/types'
import { useWebUiStore } from '@/stores/web'
import { todayString } from '@/utils/date'
import { formatCent, yuanToCent } from '@/utils/money'
import DateField from '@/web/components/DateField.vue'
import PageHead from '@/web/components/PageHead.vue'
import WebIcon from '@/web/components/WebIcon.vue'
import WebReviewTable from '@/web/components/WebReviewTable.vue'
import { goWeb } from '@/web/nav'
import { rowProblem, toCommitItem, toReviewRows, type ReviewRow } from '@/web/recognize'

type Stage = 'upload' | 'recognizing' | 'review' | 'done'

const POLL_INTERVAL_MS = 1500

const ui = useWebUiStore()
const stage = ref<Stage>('upload')
const task = ref<RecognizeTask | null>(null)
const rows = ref<ReviewRow[]>([])
const books = ref<BookItem[]>([])
const error = ref('')
const dragOver = ref(false)
const committing = ref(false)
const result = ref<RecognizeCommitResult | null>(null)
let pollTimer: ReturnType<typeof setTimeout> | undefined

const settings = reactive({
  direction: 'in' as Direction,
  bookId: 0,
  matter: '',
  date: todayString(),
})

const checkedRows = computed(() => rows.value.filter((row) => row.checked))
const checkedSumCent = computed(() => checkedRows.value.reduce((sum, row) => sum + yuanToCent(row.amountText), 0))
const blocking = computed(() => checkedRows.value.find((row) => rowProblem(row)) ?? null)
const imageURL = computed(() => (task.value ? recognizeImageURL(task.value.id) : ''))

function stopPolling(): void {
  if (pollTimer) clearTimeout(pollTimer)
  pollTimer = undefined
}

function poll(id: number): void {
  pollTimer = setTimeout(async () => {
    try {
      const latest = await fetchRecognizeTask(id)
      task.value = latest
      if (latest.status === 'succeeded') {
        rows.value = toReviewRows(latest.items)
        stage.value = 'review'
        return
      }
      if (latest.status === 'failed') {
        error.value = latest.errorMessage || '识别失败，请重新上传'
        stage.value = 'upload'
        return
      }
      poll(id)
    } catch (err) {
      error.value = err instanceof Error ? err.message : '查询识别进度失败'
      stage.value = 'upload'
    }
  }, POLL_INTERVAL_MS)
}

async function upload(file: File | undefined): Promise<void> {
  if (!file) return
  error.value = ''
  stage.value = 'recognizing'
  try {
    const created = await uploadRecognizeImage(file)
    task.value = created
    poll(created.id)
  } catch (err) {
    error.value = err instanceof Error ? err.message : '上传失败'
    stage.value = 'upload'
  }
}

/** uni-app H5 会把模板里的 <input type="file"> 编译成文本输入组件，只能用原生 DOM 弹出文件选择。
 * 输入框必须挂到文档上，部分浏览器对游离节点不派发 change。 */
function pickFile(): void {
  const input = document.createElement('input')
  input.type = 'file'
  input.accept = 'image/jpeg,image/png'
  input.style.display = 'none'
  const cleanup = (): void => input.remove()
  input.addEventListener('change', () => {
    void upload(input.files?.[0])
    cleanup()
  })
  input.addEventListener('cancel', cleanup)
  document.body.appendChild(input)
  input.click()
}

function onDrop(event: DragEvent): void {
  dragOver.value = false
  void upload(event.dataTransfer?.files?.[0])
}

async function commit(): Promise<void> {
  if (!task.value) return
  error.value = ''
  if (!checkedRows.value.length) {
    error.value = '请至少勾选一条识别结果'
    return
  }
  if (blocking.value) {
    error.value = `第 ${blocking.value.seq} 行：${rowProblem(blocking.value)}`
    return
  }
  if (settings.direction === 'in' && !settings.bookId) {
    error.value = '收礼需要选择礼簿'
    return
  }
  committing.value = true
  try {
    result.value = await commitRecognizeTask({
      id: task.value.id,
      bookId: settings.direction === 'in' ? settings.bookId : 0,
      direction: settings.direction,
      matter: settings.direction === 'out' ? settings.matter.trim() : '',
      occurredOn: settings.date,
      items: checkedRows.value.map(toCommitItem),
    })
    ui.markChanged()
    stage.value = 'done'
  } catch (err) {
    error.value = err instanceof Error ? err.message : '写入失败'
  } finally {
    committing.value = false
  }
}

function restart(): void {
  stopPolling()
  task.value = null
  rows.value = []
  result.value = null
  error.value = ''
  stage.value = 'upload'
}

onMounted(() => {
  fetchBooks()
    .then((page) => {
      books.value = page.items
      settings.bookId = page.items[0]?.id ?? 0
    })
    .catch(() => {
      books.value = []
    })
})

onBeforeUnmount(stopPolling)
</script>

<template>
  <PageHead title="识别礼簿" meta="上传一页礼簿照片，识别出姓名和金额，核对后写入">
    <template #tools>
      <div class="web-steps">
        <span :class="stage === 'upload' ? 'web-steps__cur' : 'web-steps__done'"><i>1</i>上传</span>
        <span class="web-steps__line" />
        <span :class="{ 'web-steps__cur': stage === 'recognizing', 'web-steps__done': stage === 'review' || stage === 'done' }"><i>2</i>识别</span>
        <span class="web-steps__line" />
        <span :class="{ 'web-steps__cur': stage === 'review', 'web-steps__done': stage === 'done' }"><i>3</i>核对</span>
        <span class="web-steps__line" />
        <span :class="{ 'web-steps__cur': stage === 'done' }"><i>4</i>写入</span>
      </div>
    </template>
  </PageHead>

  <div v-if="error" class="web-alert web-alert--err">
    <WebIcon name="alert" />
    <span>{{ error }}</span>
  </div>

  <div
    v-if="stage === 'upload'"
    class="web-drop rec-drop"
    :class="{ 'web-drop--over': dragOver }"
    role="button"
    tabindex="0"
    @click="pickFile"
    @keydown.enter="pickFile"
    @dragover.prevent="dragOver = true"
    @dragleave="dragOver = false"
    @drop.prevent="onDrop"
  >
    <WebIcon name="upload" :size="28" />
    <b>点击选择或拖入礼簿照片</b>
    <span class="web-muted web-sm">支持 jpg、png，不超过 5MB；一次一页</span>
  </div>

  <div v-else-if="stage === 'recognizing'" class="web-card web-empty">正在识别，通常需要十几秒…</div>

  <div v-else-if="stage === 'review'" class="web-rec">
    <div class="web-card web-photo">
      <div class="web-photo__stage">
        <img class="rec-image" :src="imageURL" alt="礼簿原图" />
      </div>
      <span class="web-muted web-sm">识别 {{ task?.items.length ?? 0 }} 行，合计 {{ formatCent(task?.sumCent ?? 0) }}</span>
    </div>

    <div class="rec-review">
      <div class="web-card rec-settings">
        <div class="web-field">
          <label>方向</label>
          <div class="web-seg web-seg--dir">
            <button type="button" data-v="in" :class="{ 'web-seg__on': settings.direction === 'in' }" @click="settings.direction = 'in'">收</button>
            <button type="button" data-v="out" :class="{ 'web-seg__on': settings.direction === 'out' }" @click="settings.direction = 'out'">送</button>
          </div>
        </div>
        <div v-if="settings.direction === 'in'" class="web-field">
          <label for="rec-book">写入礼簿</label>
          <select id="rec-book" v-model.number="settings.bookId" class="web-select">
            <option v-if="!books.length" :value="0">还没有礼簿，先去新建</option>
            <option v-for="book in books" :key="book.id" :value="book.id">{{ book.name }}</option>
          </select>
        </div>
        <div v-else class="web-field">
          <label for="rec-matter">事情</label>
          <input id="rec-matter" v-model="settings.matter" class="web-input" maxlength="128" placeholder="如：张三升学酒" />
        </div>
        <div class="web-field">
          <label>日期</label>
          <DateField v-model="settings.date" />
        </div>
      </div>

      <div class="web-card">
        <WebReviewTable :rows="rows" />
        <div class="web-sumbar">
          <span>已勾选 {{ checkedRows.length }} / {{ rows.length }} 行</span>
          <span>合计 <b class="web-amt web-amt--in">{{ formatCent(checkedSumCent) }}</b></span>
          <span class="web-filters__gap" />
          <button class="web-btn" type="button" @click="restart">重新上传</button>
          <button class="web-btn web-btn--primary" type="button" :disabled="committing" @click="commit">
            {{ committing ? '写入中…' : '写入记录' }}
          </button>
        </div>
      </div>
    </div>
  </div>

  <div v-else class="web-card rec-done">
    <WebIcon name="check" :size="28" />
    <b>已写入 {{ result?.count ?? 0 }} 条记录，合计 {{ formatCent(result?.sumCent ?? 0) }}</b>
    <span v-if="result?.newPersons" class="web-muted">其中新建亲友 {{ result.newPersons }} 位</span>
    <div class="rec-done__acts">
      <button class="web-btn" type="button" @click="restart">再识别一页</button>
      <button class="web-btn web-btn--primary" type="button" @click="goWeb('/pages/web/index')">去总账查看</button>
    </div>
  </div>
</template>

<style lang="scss" scoped>
.rec-drop {
  cursor: pointer;
}

.rec-image {
  max-width: 100%;
  max-height: 70vh;
}

.rec-review {
  display: flex;
  flex-direction: column;
  gap: var(--sp-4);
}

.rec-settings {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: var(--sp-4);
  padding: var(--sp-4);
}

.rec-done {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: var(--sp-3);
  padding: var(--sp-8) var(--sp-6);
}

.rec-done__acts {
  display: flex;
  gap: var(--sp-3);
}
</style>
