<script setup lang="ts">
import { onBackPress, onLoad } from '@dcloudio/uni-app'
import { computed, nextTick, ref } from 'vue'

import { fetchBookDetail } from '@/api/book'
import { fetchEventDetail } from '@/api/event'
import { ApiError } from '@/api/http'
import { fetchPersons } from '@/api/person'
import { createRecord, deleteRecord } from '@/api/record'
import type { BookItem, EventItem, PersonBrief, RecordItem } from '@/api/types'
import AppNav from '@/components/AppNav.vue'
import EventPicker from '@/components/EventPicker.vue'
import EventSeal from '@/components/EventSeal.vue'
import Money from '@/components/Money.vue'
import { useDictStore } from '@/stores/dict'
import { todayString, weekdayOf } from '@/utils/date'
import { back, go, toast } from '@/utils/nav'
import { formatCent, yuanToCent } from '@/utils/money'

const dictStore = useDictStore()

const loading = ref(true)
const error = ref('')

const book = ref<BookItem | null>(null)
const event = ref<EventItem | null>(null)
const direction = ref<'in' | 'out'>('in')
const itemKind = ref('cash')
const date = ref(todayString())

const page = ref('')
const seq = ref(1)
const declaredText = ref('')

const name = ref('')
const amountText = ref('')
const paperNote = ref('')
const noteEnabled = ref(true)

const candidates = ref<PersonBrief[]>([])
const selectedPersonId = ref(0)
const searching = ref(false)
let searchTimer: ReturnType<typeof setTimeout> | null = null

const recorded = ref<RecordItem[]>([])
const hiId = ref(0)
const submitting = ref(false)
let hiTimer: ReturnType<typeof setTimeout> | null = null

const conflict = ref(false)
const conflictMessage = ref('')

const contextSheet = ref(false)
const eventPickerShow = ref(false)
const showLeave = ref(false)
let eventFromSheet = false
let initialBookId = 0
let initialEventId = 0

const focusField = ref<'name' | 'amount' | 'note' | ''>('')

const directionOptions = [
  { code: 'in' as const, label: '收礼' },
  { code: 'out' as const, label: '送礼' },
]

const exactMatches = computed(() => {
  const key = name.value.trim()
  if (!key) return []
  return candidates.value.filter((item) => item.name === key)
})

const giftMode = computed(() => {
  const text = amountText.value.trim()
  return Boolean(text) && !/^\d+(\.\d{0,2})?$/.test(text)
})

const parsedCent = computed(() => {
  const text = amountText.value.trim()
  if (!text || giftMode.value) return 0
  return yuanToCent(text)
})

const declaredCent = computed(() => yuanToCent(declaredText.value))

const pageRecordedCent = computed(() =>
  recorded.value
    .filter((record) => (record.paperPage || '') === page.value)
    .reduce((sum, record) => sum + record.totalCent, 0),
)

const pageCompare = computed(() => {
  if (declaredCent.value <= 0) return 'none'
  if (pageRecordedCent.value === declaredCent.value) return 'equal'
  if (pageRecordedCent.value > declaredCent.value) return 'over'
  return 'under'
})

const sessionCent = computed(() => recorded.value.reduce((sum, record) => sum + record.totalCent, 0))
const annotatedCount = computed(() => recorded.value.filter((record) => record.paperNote).length)
const dateLabel = computed(() => (date.value ? `${date.value} ${weekdayOf(date.value)}` : ''))
const differs = computed(() => pageCompare.value === 'over')

function focusOn(field: 'name' | 'amount' | 'note'): void {
  focusField.value = ''
  nextTick(() => {
    focusField.value = field
  })
}

function parsePageNumber(value: string): number {
  const digits = value.match(/\d+/g)
  if (!digits?.length) return 0
  return Number(digits[digits.length - 1])
}

function nextPage(): void {
  const current = parsePageNumber(page.value)
  if (/^\d+$/.test(page.value.trim())) {
    page.value = String(current + 1)
  } else if (page.value.trim()) {
    page.value = page.value.replace(/(\d+)(?!.*\d)/, (match) => String(Number(match) + 1))
  } else {
    page.value = '1'
  }
  seq.value = 1
  declaredText.value = ''
  focusOn('name')
}

async function loadContext(bookId: number, eventId: number): Promise<void> {
  try {
    if (bookId) {
      book.value = await fetchBookDetail(bookId)
      itemKind.value = book.value.category === 'huopao' ? 'huopao' : 'cash'
    }
    const targetEventId = eventId || book.value?.defaultEventId || 0
    if (targetEventId) {
      event.value = await fetchEventDetail(targetEventId)
      date.value = event.value.eventDate || todayString()
    }
  } catch (err) {
    error.value = err instanceof Error ? err.message : '上下文加载失败'
  } finally {
    loading.value = false
  }
}

onLoad((raw) => {
  const query = (raw ?? {}) as Record<string, string>
  const bookId = Number(query.bookId ?? 0)
  const eventId = Number(query.eventId ?? 0)
  initialBookId = bookId
  initialEventId = eventId
  if (query.direction === 'in' || query.direction === 'out') direction.value = query.direction
  if (query.page) page.value = query.page
  if (query.seq) seq.value = Math.max(1, Number(query.seq) || 1)
  void Promise.all([dictStore.ensure(), loadContext(bookId, eventId)]).then(() => {
    focusOn('name')
    if (!event.value) {
      eventPickerShow.value = true
    }
  })
})

function retry(): void {
  loading.value = true
  error.value = ''
  void loadContext(initialBookId, initialEventId).then(() => {
    focusOn('name')
    if (!event.value) eventPickerShow.value = true
  })
}

function onNameInput(event: Event): void {
  const value = (event as unknown as { detail: { value: string } }).detail.value
  name.value = value
  selectedPersonId.value = 0
  if (searchTimer) clearTimeout(searchTimer)
  if (!value.trim()) {
    candidates.value = []
    return
  }
  searchTimer = setTimeout(async () => {
    const keywordAtRequest = value.trim()
    searching.value = true
    try {
      const result = await fetchPersons({ keyword: keywordAtRequest, pageSize: 20 })
      if (name.value.trim() === keywordAtRequest) {
        candidates.value = result.items
      }
    } catch {
      if (name.value.trim() === keywordAtRequest) {
        candidates.value = []
      }
    } finally {
      searching.value = false
    }
  }, 180)
}

function onAmountInput(event: Event): void {
  amountText.value = (event as unknown as { detail: { value: string } }).detail.value
}

function onNoteInput(event: Event): void {
  paperNote.value = (event as unknown as { detail: { value: string } }).detail.value
}

function pickCandidate(personItem: PersonBrief): void {
  name.value = personItem.name
  selectedPersonId.value = personItem.id
  focusOn('amount')
}

function currentPersonId(): number {
  const key = name.value.trim()
  if (!key) return 0
  const exact = exactMatches.value
  const selected = exact.find((item) => item.id === selectedPersonId.value)
  if (selected) return selected.id
  if (exact.length === 1) return exact[0].id
  return 0
}

function clearConflict(): void {
  conflict.value = false
  conflictMessage.value = ''
}

async function submit(confirm = false): Promise<void> {
  if (submitting.value) return
  const personName = name.value.trim()
  if (!personName) {
    toast('请填写姓名')
    focusOn('name')
    return
  }
  if (!book.value?.id) {
    toast('请先选择礼簿')
    return
  }
  if (!event.value?.id) {
    toast('请先在上下文里选择事项')
    contextSheet.value = true
    return
  }
  if (exactMatches.value.length > 1 && !exactMatches.value.some((item) => item.id === selectedPersonId.value)) {
    toast('有多个同名亲友，请点选其中一个')
    return
  }
  const giftName = giftMode.value ? amountText.value.trim() : ''
  const amountCent = giftMode.value ? 0 : parsedCent.value
  if (!amountCent && !giftName) {
    toast('请填写金额或礼品')
    focusOn('amount')
    return
  }
  submitting.value = true
  try {
    const record = await createRecord({
      personId: currentPersonId(),
      personName: currentPersonId() ? undefined : personName,
      eventId: event.value?.id ?? 0,
      bookId: book.value?.id ?? 0,
      direction: direction.value,
      itemKind: giftName ? 'gift' : itemKind.value,
      amountCent,
      giftName,
      giftValueCent: 0,
      occurredOn: date.value,
      datePrecision: 'day',
      status: 'pending',
      paperNote: noteEnabled.value ? paperNote.value.trim() : '',
      paperPage: page.value.trim(),
      paperSeq: seq.value,
      confirm,
    })
    recorded.value.unshift(record)
    hiId.value = record.id
    if (hiTimer) clearTimeout(hiTimer)
    hiTimer = setTimeout(() => {
      hiId.value = 0
    }, 500)
    name.value = ''
    amountText.value = ''
    paperNote.value = ''
    selectedPersonId.value = 0
    candidates.value = []
    clearConflict()
    seq.value += 1
    focusOn('name')
  } catch (err) {
    if (err instanceof ApiError && err.code === 409) {
      const reason = (err.custom as Record<string, unknown> | null)?.reason
      if (reason === 'need_confirm') {
        conflict.value = true
        conflictMessage.value = err.message
        return
      }
    }
    if (err instanceof ApiError && err.code === 400) {
      const custom = err.custom as {
        candidates?: { id: number; name: string; relation: string; relationLabel: string }[]
      } | null
      if (custom?.candidates?.length) {
        candidates.value = custom.candidates.map((item) => ({
          id: item.id,
          name: item.name,
          relation: item.relation,
          relationLabel: item.relationLabel,
          phone: '',
          remark: '',
          createdAt: 0,
          updatedAt: 0,
          inCent: 0,
          outCent: 0,
          pendingCount: 0,
          lastDate: '',
        }))
        toast('有多个同名亲友，请点选其中一个')
        return
      }
    }
    toast(err instanceof Error ? err.message : '保存失败')
  } finally {
    submitting.value = false
  }
}

function onAmountConfirm(): void {
  if (conflict.value) {
    void submit(true)
    return
  }
  if (noteEnabled.value) {
    focusOn('note')
    return
  }
  void submit(false)
}

function onNoteConfirm(): void {
  void submit(conflict.value)
}

async function undoLatest(): Promise<void> {
  const latest = recorded.value[0]
  if (!latest) return
  try {
    await deleteRecord(latest.id)
    recorded.value.shift()
    if ((latest.paperPage || '') === page.value) {
      seq.value = Math.max(1, seq.value - 1)
    }
    toast('已撤销')
  } catch (err) {
    toast(err instanceof Error ? err.message : '撤销失败')
  }
}

function requestLeave(): void {
  if (!recorded.value.length) {
    back()
    return
  }
  showLeave.value = true
}

onBackPress(() => {
  if (showLeave.value) {
    showLeave.value = false
    return true
  }
  if (recorded.value.length) {
    showLeave.value = true
    return true
  }
  return false
})

function goNotes(): void {
  showLeave.value = false
  go(`/sub-record/notes?bookId=${book.value?.id ?? 0}`)
}

function leaveAnyway(): void {
  showLeave.value = false
  back()
}

function onEventSelect(selected: EventItem): void {
  event.value = selected
  if (selected.eventDate) date.value = selected.eventDate
  const reopen = eventFromSheet
  eventFromSheet = false
  if (reopen) {
    setTimeout(() => {
      contextSheet.value = true
    }, 220)
  }
}

function openEventPicker(): void {
  eventFromSheet = true
  contextSheet.value = false
  nextTick(() => {
    eventPickerShow.value = true
  })
}

function pickKind(code: string): void {
  itemKind.value = code
}

function onDateChange(event: Event): void {
  date.value = (event as unknown as { detail: { value: string } }).detail.value
}

function toggleNote(): void {
  noteEnabled.value = !noteEnabled.value
  if (!noteEnabled.value) paperNote.value = ''
}

function kindLabel(): string {
  return dictStore.itemKindLabel(itemKind.value)
}
</script>

<template>
  <view class="screen cont">
    <AppNav title="连续录入" right="完成" @right="requestLeave" />

    <scroll-view scroll-y class="cont__body">
      <view v-if="loading" class="loading-tip">正在加载…</view>
      <view v-else-if="error" class="error-state">
        <text class="error-state__text">{{ error }}</text>
        <view class="btn btn--secondary error-state__btn" @click="retry">重试</view>
      </view>

      <template v-else>
        <view class="page">
          <view class="context card">
            <view class="context__main" @click="contextSheet = true">
              <view class="context__line">
                <EventSeal v-if="event" :type="event.type" size="s" />
                <text class="context__event">{{ event?.name ?? '未选事项' }}</text>
                <text class="context__dir">{{ direction === 'in' ? '收礼' : '送礼' }}</text>
              </view>
              <text class="context__sub">
                {{ book?.name ?? '未选礼簿' }} · {{ kindLabel() }} · {{ dateLabel }}
              </text>
            </view>
            <text class="context__edit" @click="contextSheet = true">修改</text>
          </view>
        </view>

        <view class="page">
          <view class="pagebar">
            <text class="pagebar__k">第</text>
            <input v-model="page" class="pagebar__page" placeholder="页码" />
            <text class="pagebar__k">页 · 第 {{ seq }} 行</text>
            <text class="pagebar__next" @click="nextPage">下一页 ›</text>
          </view>
          <view class="declare">
            <text class="declare__k">本页已录</text>
            <text
              class="declare__compare"
              :class="{
                'declare__compare--equal': pageCompare === 'equal',
                'declare__compare--over': pageCompare === 'over',
              }"
            >
              {{ formatCent(pageRecordedCent) }}<text v-if="pageCompare === 'equal'"> ✓</text>
            </text>
            <text class="declare__sep">/ 纸本</text>
            <input v-model="declaredText" class="declare__input" type="digit" placeholder="页脚合计" />
          </view>
          <text v-if="pageCompare === 'equal'" class="declare__hint declare__hint--ok">本页已录与纸本一致</text>
          <text v-else-if="pageCompare === 'over'" class="declare__hint declare__hint--over">
            已录超过纸本 {{ formatCent(pageRecordedCent - declaredCent) }}
          </text>
        </view>

        <view class="page">
          <view v-if="conflict" class="conflict">
            <text class="conflict__text">{{ conflictMessage }}</text>
            <view class="conflict__actions">
              <text class="conflict__cancel" @click="clearConflict">取消</text>
              <text class="conflict__ok" @click="submit(true)">仍然记录</text>
            </view>
          </view>

          <view class="inputs">
            <view class="inputs__top">
              <input
                class="entry entry--name"
                :value="name"
                :focus="focusField === 'name'"
                placeholder="姓名"
                confirm-type="next"
                @input="onNameInput"
                @confirm="focusOn('amount')"
              />
              <input
                class="entry entry--amount"
                :value="amountText"
                :focus="focusField === 'amount'"
                :placeholder="'金额或礼品'"
                confirm-type="done"
                @input="onAmountInput"
                @confirm="onAmountConfirm"
              />
            </view>

            <view v-if="candidates.length" class="candidates">
              <text
                v-for="personItem in candidates"
                :key="personItem.id"
                class="chip chip--flat candidate"
                :class="{ 'candidate--on': selectedPersonId === personItem.id }"
                @click="pickCandidate(personItem)"
              >
                {{ personItem.name }}<text class="candidate__relation"> {{ personItem.relationLabel }}</text>
              </text>
            </view>
            <text v-if="searching" class="caption">正在查找同名亲友…</text>

            <view v-if="noteEnabled" class="note">
              <input
                class="note__input"
                :value="paperNote"
                :focus="focusField === 'note'"
                placeholder="纸本标注（选填），照抄手写内容"
                confirm-type="done"
                @input="onNoteInput"
                @confirm="onNoteConfirm"
              />
            </view>
            <view class="note-toggle-row">
              <text class="note-toggle" @click="toggleNote">
                {{ noteEnabled ? '收起标注' : '＋ 标注' }}
              </text>
              <text v-if="giftMode" class="gift-hint">按礼品记录：{{ amountText }}</text>
              <text v-else-if="differs" class="gift-hint gift-hint--over">页码合计超出，注意核对</text>
            </view>

            <view class="btn btn--primary submit" :class="{ 'btn--disabled': submitting }" @click="submit(false)">
              {{ submitting ? '记录中…' : '记 下' }}
            </view>
          </view>
        </view>

        <view class="page">
          <view class="session">
            <text class="session__text">本次已录 {{ recorded.length }} 笔 · 合计 </text>
            <Money :cent="sessionCent" direction="none" size="s" />
          </view>

          <view v-if="recorded.length" class="records">
            <view v-for="(record, index) in recorded" :key="record.id" class="record" :class="{ 'record--hi': hiId === record.id }">
              <view class="record__main">
                <text class="record__name">{{ record.personName }}</text>
                <text v-if="record.giftName && !record.amountCent" class="record__gift">{{ record.giftName }}</text>
                <Money v-else :cent="record.totalCent" :direction="record.direction" size="s" />
                <text v-if="record.paperNote" class="record__note">{{ record.paperNote }}</text>
                <text v-if="record.paperPage" class="record__page">第 {{ record.paperPage }} 页</text>
              </view>
              <text v-if="index === 0" class="record__undo" @click="undoLatest">撤销</text>
            </view>
          </view>
          <view v-else class="empty-tip">还没有录入，从姓名开始吧</view>
        </view>
      </template>
    </scroll-view>

    <wd-popup v-model="contextSheet" position="bottom" :safe-area-inset-bottom="true" custom-style="border-radius: 20px 20px 0 0;">
      <view class="sheet">
        <text class="sheet__title">修改本次上下文</text>
        <view class="sheet__row" @click="openEventPicker">
          <text class="sheet__k">事项</text>
          <text class="sheet__v">{{ event?.name ?? '选择事项' }}</text>
          <text class="sheet__arrow">›</text>
        </view>
        <view class="sheet__row">
          <text class="sheet__k">方向</text>
          <view class="sheet__segment">
            <text
              v-for="option in directionOptions"
              :key="option.code"
              class="sheet__segment-item"
              :class="{ 'sheet__segment-item--on': direction === option.code }"
              @click="direction = option.code"
            >
              {{ option.label }}
            </text>
          </view>
        </view>
        <view class="sheet__row">
          <text class="sheet__k">名目</text>
          <view class="sheet__chips">
            <text
              v-for="kind in dictStore.itemKinds"
              :key="kind.code"
              class="chip"
              :class="{ 'chip--on': itemKind === kind.code }"
              @click="pickKind(kind.code)"
            >
              {{ kind.label }}
            </text>
          </view>
        </view>
        <picker mode="date" :value="date" @change="onDateChange">
          <view class="sheet__row">
            <text class="sheet__k">日期</text>
            <text class="sheet__v">{{ dateLabel }}</text>
            <text class="sheet__arrow">›</text>
          </view>
        </picker>
        <view class="sheet__actions">
          <view class="btn btn--primary sheet__btn" @click="contextSheet = false">完成</view>
        </view>
      </view>
    </wd-popup>

    <EventPicker v-model:show="eventPickerShow" @select="onEventSelect" />

    <view v-if="showLeave" class="leave">
      <view class="leave__card">
        <text class="leave__title">
          {{ annotatedCount ? `有 ${annotatedCount} 条标注待整理` : `已录 ${recorded.length} 笔，均已保存` }}
        </text>
        <text class="leave__desc">
          {{ annotatedCount ? '标注已保存在记录上，可以稍后统一整理成还礼。' : '离开页面不会丢失，之后可以在全部记录中找到。' }}
        </text>
        <view class="leave__actions">
          <view class="btn btn--secondary leave__btn" @click="showLeave = false">继续录入</view>
          <view v-if="annotatedCount" class="btn btn--primary leave__btn" @click="goNotes">去整理</view>
          <view v-else class="btn btn--primary leave__btn" @click="leaveAnyway">离开</view>
        </view>
        <text v-if="annotatedCount" class="leave__link" @click="leaveAnyway">直接离开</text>
      </view>
    </view>
  </view>
</template>

<style lang="scss" scoped>
.cont {
  display: flex;
  flex-direction: column;
  height: 100vh;
  overflow: hidden;
  padding-bottom: 0;
}

.cont__body {
  flex: 1;
  min-height: 0;
}

.context {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-top: 12px;
  padding: 12px 14px;
  background: var(--c-fill);
  box-shadow: none;
}

.context__main {
  flex: 1;
  min-width: 0;
}

.context__line {
  display: flex;
  align-items: center;
  gap: 6px;
}

.context__event {
  font-size: 15px;
  font-weight: 500;
  color: var(--c-ink-1);
}

.context__dir {
  padding: 1px 6px;
  border-radius: var(--r-xs);
  background: var(--c-surface);
  font-size: 12px;
  color: var(--c-ink-2);
}

.context__sub {
  display: block;
  margin-top: 4px;
  font-size: 12px;
  color: var(--c-ink-3);
}

.context__edit {
  font-size: 13px;
  color: var(--c-primary);
}

.pagebar {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-top: 12px;
  font-size: 13px;
  color: var(--c-ink-2);
}

.pagebar__k {
  flex: none;
}

.pagebar__page {
  width: 64px;
  height: 30px;
  padding: 0 8px;
  border-radius: var(--r-xs);
  background: var(--c-surface);
  box-shadow: inset 0 0 0 1px var(--c-line);
  font-size: 13px;
  text-align: center;
}

.pagebar__next {
  margin-left: auto;
  font-size: 13px;
  color: var(--c-primary);
}

.declare {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-top: 8px;
  font-size: 13px;
  color: var(--c-ink-3);
}

.declare__input {
  width: 104px;
  height: 30px;
  padding: 0 8px;
  border-radius: var(--r-xs);
  background: var(--c-surface);
  box-shadow: inset 0 0 0 1px var(--c-line);
  font-family: $font-num;
  font-size: 13px;
  text-align: right;
}

.declare__compare {
  font-family: $font-num;
  color: var(--c-ink-2);
}

.declare__compare--equal {
  color: var(--c-repaid);
}

.declare__compare--over {
  color: var(--c-pending);
}

.declare__hint {
  display: block;
  margin-top: 6px;
  font-size: 12px;
}

.declare__hint--ok {
  color: var(--c-repaid);
}

.declare__hint--over {
  color: var(--c-pending);
}

.conflict {
  margin-top: 12px;
  padding: 10px 12px;
  border-radius: var(--r-s);
  background: var(--c-pending-light);
}

.conflict__text {
  font-size: 13px;
  line-height: 19px;
  color: var(--c-pending);
}

.conflict__actions {
  display: flex;
  justify-content: flex-end;
  gap: 14px;
  margin-top: 8px;
  font-size: 13px;
  font-weight: 500;
}

.conflict__cancel {
  color: var(--c-ink-3);
}

.conflict__ok {
  color: var(--c-primary);
}

.inputs {
  margin-top: 12px;
}

.inputs__top {
  display: flex;
  gap: 8px;
}

.entry {
  height: 52px;
  padding: 0 12px;
  border-radius: var(--r-m);
  background: var(--c-surface);
  box-shadow: var(--shadow-card);
  font-size: 18px;
  color: var(--c-ink-1);
}

.entry--name {
  flex: 1;
  min-width: 0;
}

.entry--amount {
  flex: 1;
  min-width: 0;
  font-family: $font-num;
  text-align: right;
}

.candidates {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-top: 10px;
}

.candidate {
  font-size: 13px;
}

.candidate--on {
  background: var(--c-primary-light);
  color: var(--c-primary);
  box-shadow: inset 0 0 0 1px var(--c-primary);
}

.candidate__relation {
  font-size: 11px;
  color: var(--c-ink-3);
}

.note {
  margin-top: 10px;
}

.note__input {
  height: 44px;
  padding: 0 12px;
  border-radius: var(--r-s);
  background: var(--c-surface);
  box-shadow: inset 0 0 0 1px var(--c-line);
  font-family: $font-serif;
  font-size: 14px;
}

.note-toggle-row {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-top: 6px;
}

.note-toggle {
  font-size: 12px;
  color: var(--c-ink-3);
}

.gift-hint {
  font-size: 12px;
  color: var(--c-out);
}

.gift-hint--over {
  color: var(--c-pending);
}

.submit {
  margin-top: 12px;
}

.session {
  margin-top: 18px;
  padding: 8px 2px;
}

.session__text {
  font-size: 13px;
  color: var(--c-ink-2);
}

.records {
  border-radius: var(--r-m);
  background: var(--c-surface);
  box-shadow: var(--shadow-card);
  overflow: hidden;
}

.record {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px 14px;
  transition: background 0.5s ease;
}

.record + .record {
  box-shadow: inset 0 1px 0 var(--c-divider);
}

.record--hi {
  background: var(--c-primary-light);
}

.record__main {
  flex: 1;
  min-width: 0;
  display: flex;
  align-items: baseline;
  gap: 8px;
}

.record__name {
  font-size: 14px;
  font-weight: 500;
  color: var(--c-ink-1);
}

.record__gift {
  font-size: 14px;
  color: var(--c-ink-2);
}

.record__note {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
  font-family: $font-serif;
  font-size: 12px;
  color: var(--c-ink-3);
}

.record__page {
  font-size: 11px;
  color: var(--c-ink-4);
}

.record__undo {
  flex: none;
  font-size: 12px;
  color: var(--c-primary);
}

.sheet {
  padding: 18px 16px calc(20px + env(safe-area-inset-bottom));
}

.sheet__title {
  display: block;
  font-size: 16px;
  font-weight: 600;
  margin-bottom: 8px;
}

.sheet__row {
  display: flex;
  align-items: center;
  gap: 10px;
  min-height: 46px;
  box-shadow: inset 0 -1px 0 var(--c-divider);
}

.sheet__k {
  width: 44px;
  flex: none;
  font-size: 13px;
  color: var(--c-ink-3);
}

.sheet__v {
  flex: 1;
  font-size: 14px;
  color: var(--c-ink-1);
}

.sheet__arrow {
  color: var(--c-ink-4);
}

.sheet__segment {
  display: flex;
  gap: 6px;
}

.sheet__segment-item {
  padding: 6px 12px;
  border-radius: var(--r-s);
  background: var(--c-fill);
  font-size: 13px;
  color: var(--c-ink-2);
}

.sheet__segment-item--on {
  background: var(--c-primary-light);
  color: var(--c-primary);
  box-shadow: inset 0 0 0 1px var(--c-primary);
}

.sheet__chips {
  display: flex;
  gap: 8px;
  overflow-x: auto;
}

.sheet__chips--wrap {
  flex-wrap: wrap;
  margin-top: 12px;
}

.sheet__actions {
  margin-top: 16px;
}

.sheet__btn {
  width: 100%;
}

.leave {
  position: fixed;
  inset: 0;
  z-index: 60;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 32px;
  background: var(--c-mask);
}

.leave__card {
  width: 100%;
  max-width: 340px;
  padding: 22px 18px 16px;
  border-radius: var(--r-l);
  background: var(--c-surface);
}

.leave__title {
  display: block;
  font-size: 16px;
  font-weight: 600;
}

.leave__desc {
  display: block;
  margin-top: 8px;
  font-size: 13px;
  line-height: 19px;
  color: var(--c-ink-2);
}

.leave__actions {
  display: flex;
  gap: 10px;
  margin-top: 18px;
}

.leave__btn {
  flex: 1;
}

.leave__link {
  display: block;
  margin-top: 12px;
  text-align: center;
  font-size: 12px;
  color: var(--c-ink-3);
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
