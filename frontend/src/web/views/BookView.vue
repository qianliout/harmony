<script setup lang="ts">
import { computed, nextTick, onMounted, ref, watch } from 'vue'

import { fetchBookDetail, fetchBooks, fetchPaperPages, setPageCheck } from '@/api/book'
import { fetchEvents } from '@/api/event'
import { ApiError } from '@/api/http'
import { createRecord, deleteRecord, fetchPendingNotes, fetchRecords } from '@/api/record'
import type { BookItem, EventItem, PaperPage, RecordItem, RecordSummary } from '@/api/types'
import PersonAvatar from '@/components/PersonAvatar.vue'
import StatusPill from '@/components/StatusPill.vue'
import { useDictStore } from '@/stores/dict'
import { useWebUiStore } from '@/stores/web'
import { centToChinese } from '@/utils/chinese-amount'
import { preciseDateLabel, todayString } from '@/utils/date'
import { centToYuanText, formatCent, yuanToCent } from '@/utils/money'
import PageHead from '@/web/components/PageHead.vue'
import Pager from '@/web/components/Pager.vue'
import RecordTable from '@/web/components/RecordTable.vue'
import WebIcon from '@/web/components/WebIcon.vue'
import WebTable from '@/web/components/WebTable.vue'
import { backWeb, goWeb } from '@/web/nav'

type TabKey = 'records' | 'entry' | 'pages' | 'notes'

interface WebPageQuery {
  options?: Record<string, string>
  $page?: { options?: Record<string, string> }
}

// H5 页面的 query 挂在当前页面实例的 options 上，直接刷新时再从地址栏兜底
function readQuery(): Record<string, string> {
  const stack = getCurrentPages()
  const current = stack[stack.length - 1] as unknown as WebPageQuery | undefined
  const options = current?.options ?? current?.$page?.options ?? {}
  if (options.id) return options
  if (typeof window === 'undefined') return options
  const raw = `${window.location.search}${window.location.hash}`
  const idMatched = raw.match(/[?&]id=(\d+)/)
  const tabMatched = raw.match(/[?&]tab=(records|entry|pages|notes)/)
  const query: Record<string, string> = {}
  if (idMatched) query.id = idMatched[1]
  if (tabMatched) query.tab = tabMatched[1]
  return Object.keys(query).length ? query : options
}

function normalizeTab(raw?: string): TabKey {
  return raw === 'entry' || raw === 'pages' || raw === 'notes' ? raw : 'records'
}

const PAGE_SIZE = 20

const ui = useWebUiStore()
const dictStore = useDictStore()

const bookId = ref(Number(readQuery().id ?? 0))
const tab = ref<TabKey>(normalizeTab(readQuery().tab))

const book = ref<BookItem | null>(null)
const bookLoading = ref(true)
const bookError = ref('')
const books = ref<BookItem[]>([])
const events = ref<EventItem[]>([])
const entryEventId = ref(0)

// 记录分页
const records = ref<RecordItem[]>([])
const recordsTotal = ref(0)
const recordsSummary = ref<RecordSummary>({ count: 0, inCent: 0, outCent: 0, pendingCent: 0 })
const recordsPage = ref(1)
const recordsLoading = ref(false)
const recordsError = ref('')

// 连续录入
const pageNo = ref('1')
const seqNo = ref(1)
const entryDate = ref(todayString())
const itemKind = ref('cash')
const name = ref('')
const amountText = ref('')
const paperNote = ref('')
const submitting = ref(false)
const conflict = ref('')
const lastRecord = ref<RecordItem | null>(null)
const declaredText = ref('')
const savingDeclared = ref(false)
let kindInitialized = false

// 按页核对 / 纸本标注
const pages = ref<PaperPage[]>([])
const notes = ref<RecordItem[]>([])

const isMerged = computed(() => book.value?.status === 'merged')
const defaultEventId = computed(() => entryEventId.value)
const pageCount = computed(() => Math.max(1, Math.ceil(recordsTotal.value / PAGE_SIZE)))
const currentPage = computed(() => pages.value.find((item) => item.page === pageNo.value.trim()) ?? null)
const recordedCent = computed(() => currentPage.value?.recordCent ?? 0)
const amountCent = computed(() => yuanToCent(amountText.value))
const amountUpper = computed(() => (amountCent.value > 0 ? centToChinese(amountCent.value) : ''))
const seqValue = computed(() => Number(seqNo.value) || 1)
const consistentPages = computed(() => pages.value.filter((item) => item.consistent).length)

const meta = computed(() => {
  const item = book.value
  if (!item) return ''
  return [
    item.categoryLabel,
    item.defaultEventName ? `默认事项 ${item.defaultEventName}` : '未指定默认事项',
    `${item.recordCount} 笔记录`,
    `收 ${formatCent(item.inCent)}`,
    `送 ${formatCent(item.outCent)}`,
  ].join(' · ')
})

watch(
  entryEventId,
  (id) => {
    const selected = events.value.find((item) => item.id === id)
    if (selected?.eventDate) {
      entryDate.value = selected.eventDate
    }
  },
  { immediate: true },
)

const tabs = computed(() => [
  { value: 'records' as TabKey, label: '记录', count: recordsTotal.value },
  { value: 'entry' as TabKey, label: '连续录入', count: undefined },
  { value: 'pages' as TabKey, label: '按页核对', count: pages.value.length },
  { value: 'notes' as TabKey, label: '整理纸本标注', count: notes.value.length },
])

function setTab(next: TabKey): void {
  tab.value = next
}

function leadText(next: number): string {
  return next > 0 ? '多' : '少'
}

function diffLabel(page: PaperPage): string {
  if (!page.hasDeclared) return '—'
  if (page.diffCent === 0) return '一致'
  return `${leadText(page.diffCent)} ${formatCent(Math.abs(page.diffCent))}`
}

function diffClass(page: PaperPage): string {
  if (!page.hasDeclared) return 'web-muted'
  return page.diffCent === 0 ? 'page-diff--ok' : 'web-amt--pend'
}

async function loadBook(): Promise<void> {
  if (!bookId.value) {
    bookError.value = '缺少礼簿参数'
    return
  }
  if (!book.value) bookLoading.value = true
  bookError.value = ''
  try {
    book.value = await fetchBookDetail(bookId.value)
    if (!kindInitialized) {
      itemKind.value = book.value.category === 'huopao' ? 'huopao' : 'cash'
      kindInitialized = true
    }
    if (!entryEventId.value) {
      entryEventId.value = book.value.defaultEventId || events.value[0]?.id || 0
    }
  } catch (err) {
    const message = err instanceof Error ? err.message : '礼簿加载失败'
    if (book.value) {
      ui.toast(message)
    } else {
      bookError.value = message
    }
  } finally {
    bookLoading.value = false
  }
}

async function loadBookOptions(): Promise<void> {
  try {
    const [bookList, eventPage] = await Promise.all([
      fetchBooks('normal'),
      fetchEvents({ page: 1, pageSize: 100 }),
    ])
    books.value = bookList
    events.value = eventPage.items
  } catch {
    books.value = []
    events.value = []
  }
}

async function loadRecords(): Promise<void> {
  if (!bookId.value) return
  recordsLoading.value = true
  recordsError.value = ''
  try {
    const params = isMerged.value
      ? { originBookId: bookId.value, page: recordsPage.value, pageSize: PAGE_SIZE }
      : { bookId: bookId.value, page: recordsPage.value, pageSize: PAGE_SIZE }
    const result = await fetchRecords(params)
    records.value = result.items
    recordsTotal.value = result.total
    recordsSummary.value = result.summary
  } catch (err) {
    recordsError.value = err instanceof Error ? err.message : '记录加载失败'
    records.value = []
    recordsTotal.value = 0
  } finally {
    recordsLoading.value = false
  }
}

async function loadPages(syncDeclared = false): Promise<void> {
  if (!bookId.value) return
  try {
    pages.value = await fetchPaperPages(bookId.value)
  } catch {
    pages.value = []
  }
  if (syncDeclared) syncDeclaredFromPage()
}

async function loadNotes(): Promise<void> {
  if (!bookId.value) return
  try {
    notes.value = await fetchPendingNotes(bookId.value)
  } catch {
    notes.value = []
  }
}

function syncDeclaredFromPage(): void {
  const current = currentPage.value
  declaredText.value = current?.hasDeclared ? centToYuanText(current.declaredCent) : ''
}

watch(pageNo, () => {
  syncDeclaredFromPage()
})

async function loadAll(): Promise<void> {
  await loadBook()
  if (!book.value) return
  await Promise.all([loadRecords(), loadPages(true), loadNotes()])
}

function changePage(next: number): void {
  if (next < 1 || next > pageCount.value) return
  recordsPage.value = next
  void loadRecords()
}

function openRecord(record: RecordItem): void {
  ui.openRecord(record.id)
}

function openPerson(id: number): void {
  goWeb(`/sub-web/person?id=${id}`, 'push')
}

function openNote(record: RecordItem): void {
  if (isMerged.value) return
  ui.openPaperAnno(record.id)
}

function openEntryDrawer(): void {
  ui.openEntry({ bookId: bookId.value })
}

function switchBook(event: Event): void {
  const next = Number((event.target as HTMLSelectElement).value)
  if (!next || next === bookId.value) return
  goWeb(`/sub-web/book?id=${next}`, 'redirect')
}

function openTarget(): void {
  if (!book.value?.mergedIntoId) return
  goWeb(`/sub-web/book?id=${book.value.mergedIntoId}`, 'push')
}

function focusName(): void {
  nextTick(() => {
    const input = document.querySelector<HTMLInputElement>('.web-entry-row .uni-input-input')
    input?.focus()
  })
}

// 切到连续录入并带上纸本页码
function entryOnPage(page: string): void {
  pageNo.value = page
  tab.value = 'entry'
  syncDeclaredFromPage()
  focusName()
}

function clearConflict(): void {
  conflict.value = ''
}

async function submit(confirm = false): Promise<void> {
  if (submitting.value) return
  if (isMerged.value) {
    ui.toast('已合并的礼簿只读')
    return
  }
  if (!entryEventId.value) {
    ui.toast('请选择事项')
    return
  }
  const personName = name.value.trim()
  if (!personName) {
    ui.toast('请填写姓名')
    focusName()
    return
  }
  const cent = yuanToCent(amountText.value)
  if (cent <= 0) {
    ui.toast('请填写金额')
    return
  }
  submitting.value = true
  try {
    const record = await createRecord({
      personName,
      eventId: entryEventId.value,
      bookId: bookId.value,
      direction: 'in',
      itemKind: itemKind.value,
      amountCent: cent,
      giftName: '',
      giftValueCent: 0,
      occurredOn: entryDate.value || todayString(),
      datePrecision: 'day',
      status: 'pending',
      paperNote: paperNote.value.trim(),
      paperPage: pageNo.value.trim(),
      paperSeq: seqValue.value,
      confirm,
    })
    lastRecord.value = record
    name.value = ''
    amountText.value = ''
    paperNote.value = ''
    conflict.value = ''
    seqNo.value = seqValue.value + 1
    ui.markChanged()
    ui.toast(`已记下 ${personName}`)
    // 刷新本页已录合计
    await loadPages()
    focusName()
  } catch (err) {
    if (
      err instanceof ApiError &&
      err.code === 409 &&
      (err.custom as Record<string, unknown> | null)?.reason === 'need_confirm'
    ) {
      conflict.value = err.message || '这本礼簿上已有同名同金额的记录，可能重复。'
      return
    }
    ui.toast(err instanceof Error ? err.message : '保存失败')
  } finally {
    submitting.value = false
  }
}

async function undoLast(): Promise<void> {
  if (!lastRecord.value) return
  const target = lastRecord.value
  try {
    await deleteRecord(target.id)
    lastRecord.value = null
    ui.markChanged()
    ui.toast('已撤销最近一条')
    await loadPages()
  } catch (err) {
    ui.toast(err instanceof Error ? err.message : '撤销失败')
  }
}

async function saveDeclared(): Promise<void> {
  if (!pageNo.value.trim()) {
    ui.toast('请先填写纸本页码')
    return
  }
  savingDeclared.value = true
  try {
    await setPageCheck(bookId.value, pageNo.value.trim(), yuanToCent(declaredText.value))
    await loadPages()
    ui.markChanged()
    ui.toast('本页合计已保存')
  } catch (err) {
    ui.toast(err instanceof Error ? err.message : '保存失败')
  } finally {
    savingDeclared.value = false
  }
}

onMounted(async () => {
  void dictStore.ensure()
  await loadBookOptions()
  await loadAll()
})

watch(
  () => ui.revision,
  () => {
    if (submitting.value || savingDeclared.value) return
    void (async () => {
      await loadBook()
      if (!book.value) return
      await Promise.all([loadRecords(), loadPages(), loadNotes()])
    })()
  },
)
</script>

<template>
  <PageHead :title="book?.name || '礼簿详情'" :meta="meta">
    <template #tools>
      <select
        v-if="!isMerged && books.length > 1"
        class="web-select"
        :value="String(bookId)"
        aria-label="切换礼簿"
        @change="switchBook"
      >
        <option v-for="item in books" :key="item.id" :value="String(item.id)">{{ item.name }}</option>
      </select>
      <button v-if="!isMerged && book" class="web-btn web-btn--primary" type="button" @click="openEntryDrawer">
        <WebIcon name="plus" />
        记一笔
      </button>
      <button class="web-btn" type="button" @click="backWeb()">
        <WebIcon name="left" />
        返回礼簿
      </button>
    </template>
  </PageHead>

  <div v-if="!bookId" class="web-alert web-alert--err">
    <WebIcon name="alert" />
    <span>缺少礼簿参数。</span>
  </div>

  <div v-else-if="bookError" class="web-alert web-alert--err">
    <WebIcon name="alert" />
    <span>{{ bookError }}</span>
    <button class="web-btn web-btn--sm" type="button" @click="loadAll()">重试</button>
  </div>

  <div v-else-if="bookLoading && !book" class="web-empty">正在打开礼簿…</div>

  <template v-else-if="book">
    <div v-if="isMerged" class="web-alert web-alert--warn">
      <WebIcon name="info" />
      <span style="flex: 1">
        这本礼簿已合并到「{{ book.mergedIntoName || '目标礼簿' }}」，记录与纸本核对均转入目标礼簿，这里只读查看。
      </span>
      <button v-if="book.mergedIntoId" class="web-btn web-btn--sm" type="button" @click="openTarget">
        查看目标礼簿
      </button>
    </div>

    <div class="web-tabs">
      <button
        v-for="item in tabs"
        :key="item.value"
        type="button"
        :class="{ 'web-tabs__on': tab === item.value }"
        @click="setTab(item.value)"
      >
        {{ item.label }}
        <span v-if="item.count !== undefined" class="web-tabs__n">{{ item.count }}</span>
      </button>
    </div>

    <!-- 记录 -->
    <div v-if="tab === 'records'" class="web-card">
      <div v-if="recordsError" class="web-alert web-alert--err" style="margin: 12px 16px">
        <WebIcon name="alert" />
        <span>{{ recordsError }}</span>
        <button class="web-btn web-btn--sm" type="button" @click="loadRecords()">重试</button>
      </div>
      <RecordTable
        :records="records"
        :columns="['date', 'person', 'event', 'cat', 'amount', 'status']"
        :clickable="!isMerged"
        @row="openRecord"
        @person="openPerson"
      />
      <div v-if="recordsLoading && !records.length" class="web-empty">正在加载记录…</div>
      <div v-else-if="!records.length" class="web-empty">这本礼簿还没有记录。</div>
      <div class="web-sumbar">
        <span>共 {{ recordsTotal }} 笔</span>
        <span>收 <b class="web-amt web-amt--in">{{ formatCent(recordsSummary.inCent) }}</b></span>
        <span>送 <b class="web-amt web-amt--out">{{ formatCent(recordsSummary.outCent) }}</b></span>
        <Pager :page="recordsPage" :pages="pageCount" @change="changePage" />
      </div>
    </div>

    <!-- 连续录入 -->
    <template v-else-if="tab === 'entry'">
      <div v-if="isMerged" class="web-alert web-alert--info">
        <WebIcon name="info" />
        <span>已合并礼簿只读，不能继续录入。</span>
      </div>
      <template v-else>
        <div class="web-card">
          <div class="web-ctx">
            <div class="web-field">
              <label>事项</label>
              <select v-model.number="entryEventId" class="web-select" style="min-width: 160px">
                <option v-for="item in events" :key="item.id" :value="item.id">{{ item.name }}</option>
              </select>
            </div>
            <div class="web-field">
              <label>日期</label>
              <input v-model="entryDate" class="web-input" type="date" />
            </div>
            <div class="web-field">
              <label>纸本标记</label>
              <input v-model="pageNo" class="web-input entry-page" placeholder="页" />
              <input v-model.number="seqNo" class="web-input entry-seq" type="number" min="1" aria-label="行号" />
            </div>
            <div class="web-field">
              <label>本页合计</label>
              <b class="web-amt web-amt--in">{{ formatCent(recordedCent) }}</b>
              <span class="web-muted web-sm">
                {{ currentPage?.hasDeclared ? `/ 纸本 ${formatCent(currentPage.declaredCent)}` : '纸本未填' }}
              </span>
            </div>
          </div>

          <div class="web-entry-row" :class="{ 'web-entry-row--conflict': conflict }">
            <input
              v-model="name"
              class="web-input"
              placeholder="姓名"
              @keydown.enter.prevent="submit()"
            />
            <div class="entry-amount">
              <input
                v-model="amountText"
                class="web-input"
                inputmode="decimal"
                placeholder="金额"
                @keydown.enter.prevent="submit()"
              />
              <span v-if="conflict" class="web-pill web-pill--warn">可能重复</span>
              <span v-else-if="amountUpper" class="web-muted entry-upper">{{ amountUpper }}</span>
            </div>
            <select v-model="itemKind" class="web-select">
              <option v-if="!dictStore.itemKinds.length" value="cash">现金</option>
              <option v-if="!dictStore.itemKinds.length" value="huopao">火炮</option>
              <option v-for="item in dictStore.itemKinds" :key="item.id" :value="item.code">{{ item.label }}</option>
            </select>
            <input
              v-model="paperNote"
              class="web-input"
              placeholder="纸本标注（选填）"
              @keydown.enter.prevent="submit()"
            />
            <button class="web-btn web-btn--primary" type="button" :disabled="submitting" @click="submit()">
              {{ submitting ? '记下中…' : '记下' }}
            </button>
          </div>

          <div v-if="conflict" class="web-alert web-alert--warn" style="margin: 12px 16px">
            <WebIcon name="alert" />
            <span style="flex: 1">{{ conflict }}</span>
            <button class="web-btn web-btn--sm web-btn--primary" type="button" :disabled="submitting" @click="submit(true)">
              仍然录入
            </button>
            <button class="web-btn web-btn--sm" type="button" @click="clearConflict">取消</button>
          </div>

          <div v-if="!defaultEventId" class="web-alert web-alert--warn" style="margin: 12px 16px">
            <WebIcon name="alert" />
            <span>这本礼簿还没有默认事项，请先在礼簿里指定默认事项，连续录入才能记账。</span>
          </div>

          <div class="web-sumbar">
            <span>本页已录 <b class="web-amt web-amt--in">{{ formatCent(recordedCent) }}</b></span>
            <span>{{ currentPage ? `${currentPage.recordCount} 笔` : '尚无记录' }}</span>
            <span v-if="lastRecord">
              最近一条：{{ lastRecord.personName }} {{ formatCent(lastRecord.totalCent) }}
              <button class="web-btn web-btn--text web-btn--sm web-btn--danger" type="button" @click="undoLast">
                撤销最近一条
              </button>
            </span>
            <div class="web-filters__right">
              <span class="web-filters__lbl">纸本合计</span>
              <input
                v-model="declaredText"
                class="web-input entry-declared"
                inputmode="decimal"
                placeholder="本页合计"
              />
              <button
                class="web-btn web-btn--sm"
                type="button"
                :disabled="savingDeclared || !pageNo.trim()"
                @click="saveDeclared"
              >
                {{ savingDeclared ? '保存中…' : '保存合计' }}
              </button>
            </div>
          </div>
        </div>
      </template>
    </template>

    <!-- 按页核对 -->
    <div v-else-if="tab === 'pages'" class="web-card">
      <div class="web-filters">
        <span class="web-filters__lbl">共 {{ pages.length }} 页</span>
        <span class="web-muted web-sm">{{ consistentPages }} 页与纸本一致</span>
      </div>
      <WebTable>
        <thead>
          <tr>
            <th>页码</th>
            <th class="web-table__r">纸本合计</th>
            <th class="web-table__r">已录合计</th>
            <th class="web-table__r">笔数</th>
            <th class="web-table__r">差额</th>
            <th>状态</th>
            <th v-if="!isMerged" />
          </tr>
        </thead>
        <tbody>
          <tr v-for="item in pages" :key="item.page">
            <td>{{ item.page }}</td>
            <td class="web-table__r">
              {{ item.hasDeclared ? formatCent(item.declaredCent) : '未填' }}
            </td>
            <td class="web-table__r web-amt web-amt--in">{{ formatCent(item.recordCent) }}</td>
            <td class="web-table__r web-num">{{ item.recordCount }}</td>
            <td class="web-table__r web-amt" :class="diffClass(item)">{{ diffLabel(item) }}</td>
            <td>
              <span class="web-pill" :class="item.consistent ? 'web-pill--repaid' : 'web-pill--pending'">
                {{ item.consistent ? '一致' : item.hasDeclared ? '有差额' : '未核对' }}
              </span>
            </td>
            <td v-if="!isMerged" class="web-table__r">
              <button class="web-btn web-btn--text web-btn--sm" type="button" @click="entryOnPage(item.page)">
                录入这一页
              </button>
            </td>
          </tr>
        </tbody>
      </WebTable>
      <div v-if="!pages.length" class="web-empty">还没有分页记录，先在连续录入里填写纸本页码。</div>
    </div>

    <!-- 整理纸本标注 -->
    <div v-else class="web-card">
      <div class="web-filters">
        <span class="web-filters__lbl">共 {{ notes.length }} 条待整理</span>
        <span class="web-muted web-sm">点一行补录成回礼</span>
      </div>
      <WebTable>
        <thead>
          <tr>
            <th>亲友</th>
            <th>纸本标注</th>
            <th>日期</th>
            <th class="web-table__r">金额</th>
            <th>状态</th>
            <th v-if="!isMerged" />
          </tr>
        </thead>
        <tbody>
          <tr
            v-for="item in notes"
            :key="item.id"
            :class="{ 'web-table__row': !isMerged }"
            @click="openNote(item)"
          >
            <td>
              <div class="web-cell2">
                <PersonAvatar :id="item.personId" :name="item.personName" :size="26" />
                <span class="web-cell2__t">{{ item.personName }}</span>
              </div>
            </td>
            <td class="note-text">{{ item.paperNote }}</td>
            <td class="web-sm">{{ preciseDateLabel(item.occurredOn, item.datePrecision) || '未注明' }}</td>
            <td class="web-table__r web-amt web-amt--in">{{ formatCent(item.totalCent) }}</td>
            <td><StatusPill :status="item.status" :direction="item.direction" :is-repay="item.isRepay" /></td>
            <td v-if="!isMerged" class="web-table__r">
              <button class="web-btn web-btn--text web-btn--sm" type="button" @click.stop="openNote(item)">补录</button>
            </td>
          </tr>
        </tbody>
      </WebTable>
      <div v-if="!notes.length" class="web-empty">没有待整理的纸本标注。</div>
    </div>
  </template>
</template>

<style lang="scss" scoped>
.entry-page {
  width: 76px;
}

.entry-seq {
  width: 64px;
}

.entry-amount {
  display: flex;
  flex-direction: column;
  gap: 4px;
  min-width: 0;
}

.entry-amount .web-pill {
  align-self: flex-start;
}

.entry-upper {
  font-family: $font-serif;
  font-size: 12px;
  line-height: 16px;
}

.entry-declared {
  width: 120px;
}

.web-entry-row--conflict {
  box-shadow: inset 3px 0 0 var(--c-danger);
}

.note-text {
  max-width: 320px;
  white-space: normal;
  line-height: 20px;
}

.page-diff--ok {
  color: var(--c-repaid);
}
</style>
