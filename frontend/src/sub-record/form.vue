<script setup lang="ts">
import { onLoad, onShow } from '@dcloudio/uni-app'
import { computed, nextTick, ref, watch } from 'vue'

import { fetchBooks } from '@/api/book'
import { fetchEventDetail } from '@/api/event'
import { fetchPersonDetail } from '@/api/person'
import {
  createRecord,
  deleteRecord,
  fetchRecordDetail,
  repayRecords,
  updateRecord,
} from '@/api/record'
import type { BookItem, EventItem, RecordDetail } from '@/api/types'
import AmountPad from '@/components/AmountPad.vue'
import AppNav from '@/components/AppNav.vue'
import BookChips from '@/components/BookChips.vue'
import ConfirmSheet from '@/components/ConfirmSheet.vue'
import EventPicker from '@/components/EventPicker.vue'
import Money from '@/components/Money.vue'
import PersonAvatar from '@/components/PersonAvatar.vue'
import PersonPicker from '@/components/PersonPicker.vue'
import { submitWithConfirm } from '@/composables/useConfirmSubmit'
import { relations } from '@/styles/palette'
import { useContextStore } from '@/stores/context'
import { useDictStore } from '@/stores/dict'
import { preciseDateLabel, todayString, weekdayOf } from '@/utils/date'
import { back, confirmModal, go, toast } from '@/utils/nav'
import { centToYuanText, yuanToCent } from '@/utils/money'

type Mode = 'create' | 'edit' | 'repay'

interface SelectedPerson {
  id: number
  name: string
  relationLabel: string
}

interface SelectedEvent {
  id: number
  name: string
  type: string
  eventDate: string
  datePrecision: string
  host: string
}

const dictStore = useDictStore()
const contextStore = useContextStore()

const mode = ref<Mode>('create')
const recordId = ref(0)
const originIds = ref<number[]>([])
const origins = ref<RecordDetail[]>([])
const editRecord = ref<RecordDetail | null>(null)

const books = ref<BookItem[]>([])
const bookId = ref(0)
const person = ref<SelectedPerson | null>(null)
const event = ref<SelectedEvent | null>(null)
const direction = ref<'in' | 'out'>('in')
const amountText = ref('')
const giftName = ref('')
const giftValueText = ref('')
const showGift = ref(false)
const itemKind = ref('cash')
const itemKindTouched = ref(false)
const date = ref(todayString())
const status = ref<'pending' | 'repaid' | 'none'>('pending')
const remark = ref('')
const markRepaid = ref(true)

const loading = ref(true)
const submitting = ref(false)
const error = ref('')
const personPickerShow = ref(false)
const eventPickerShow = ref(false)
const kindSheet = ref(false)
const originExpanded = ref(false)
const eventFromBook = ref(false)
const dateTouched = ref(false)
let initializing = true
let lastQuery: Record<string, string> = {}

const statusOptions = [
  { code: 'pending', label: '待还' },
  { code: 'repaid', label: '已还' },
  { code: 'none', label: '无需还' },
] as const

const title = computed(() => {
  if (mode.value === 'edit') return '编辑记录'
  if (mode.value === 'repay') return person.value ? `还礼给 ${person.value.name}` : '还礼'
  return '记一笔'
})

const linked = computed(() => {
  const record = editRecord.value
  if (!record) return false
  return record.repayRecords.length > 0 || record.originRecords.length > 0
})

const directionLocked = computed(() => mode.value === 'repay' || (mode.value === 'edit' && linked.value))
const personLocked = computed(() => mode.value !== 'create')

const selectedBook = computed(() => books.value.find((book) => book.id === bookId.value) ?? null)
const cent = computed(() => yuanToCent(amountText.value))
const giftValueCent = computed(() => yuanToCent(giftValueText.value))
const totalCent = computed(() => cent.value + giftValueCent.value)
const valid = computed(() => cent.value > 0 || Boolean(giftName.value.trim()))
const dateLabel = computed(() => (date.value ? `${date.value} ${weekdayOf(date.value)}` : ''))
const bookDefaultHint = computed(() =>
  eventFromBook.value && selectedBook.value?.defaultEventName
    ? `${selectedBook.value.defaultEventName} · 礼簿默认`
    : '',
)

const originTotalCent = computed(() =>
  origins.value.reduce((sum, record) => sum + record.totalCent, 0),
)
const originEventName = computed(() => {
  const names = Array.from(new Set(origins.value.map((record) => record.eventName).filter(Boolean)))
  if (names.length === 1) return names[0]
  return names.length > 1 ? '多个事项' : ''
})

function relationLabelOf(code: string): string {
  return relations.find((item) => item.code === code)?.label ?? '其他'
}

function toSelectedEvent(item: EventItem): SelectedEvent {
  return {
    id: item.id,
    name: item.name,
    type: item.type,
    eventDate: item.eventDate,
    datePrecision: item.datePrecision,
    host: item.host,
  }
}

function fallbackEvent(detail: RecordDetail): SelectedEvent {
  return {
    id: detail.eventId,
    name: detail.eventName,
    type: detail.eventType || 'other',
    eventDate: detail.occurredOn,
    datePrecision: detail.datePrecision || 'day',
    host: 'other',
  }
}

function inferItemKind(): string {
  if (giftName.value.trim() && cent.value === 0) return 'gift'
  return selectedBook.value?.category === 'huopao' ? 'huopao' : 'cash'
}

function syncItemKind(): void {
  if (itemKindTouched.value) return
  itemKind.value = inferItemKind()
}

watch([bookId, giftName, amountText], syncItemKind)

function usedWithin30Days(updatedAt: number, eventDate: string): boolean {
  const now = Date.now()
  if (updatedAt && now - updatedAt <= 30 * 24 * 3600 * 1000) return true
  if (eventDate) {
    const ts = new Date(`${eventDate}T00:00:00`).getTime()
    if (Number.isFinite(ts) && Math.abs(now - ts) <= 30 * 24 * 3600 * 1000) return true
  }
  return false
}

async function loadBooks(): Promise<void> {
  try {
    books.value = await fetchBooks('normal')
  } catch (err) {
    books.value = []
    toast(err instanceof Error ? err.message : '礼簿加载失败')
  }
}

function pickBookId(...candidates: number[]): number {
  for (const id of candidates) {
    if (id && books.value.some((book) => book.id === id)) return id
  }
  return books.value[0]?.id ?? 0
}

async function loadEventById(id: number): Promise<EventItem | null> {
  if (!id) return null
  try {
    return await fetchEventDetail(id)
  } catch {
    return null
  }
}

async function loadPersonById(id: number): Promise<void> {
  if (!id) return
  try {
    const detail = await fetchPersonDetail(id)
    person.value = { id: detail.id, name: detail.name, relationLabel: detail.relationLabel }
  } catch {
    person.value = null
  }
}

async function applyBookDefault(): Promise<void> {
  if (mode.value !== 'create') return
  const book = selectedBook.value
  if (!book?.defaultEventId) return
  if (event.value && !eventFromBook.value) return
  if (event.value?.id === book.defaultEventId) {
    eventFromBook.value = true
    return
  }
  const item = await loadEventById(book.defaultEventId)
  if (!item) return
  event.value = toSelectedEvent(item)
  eventFromBook.value = true
  if (!dateTouched.value) date.value = item.eventDate || todayString()
}

async function initCreate(query: Record<string, string>): Promise<void> {
  await Promise.all([dictStore.ensure(), contextStore.load(true), loadBooks()])

  const queryBookId = Number(query.bookId ?? 0)
  bookId.value = pickBookId(queryBookId, contextStore.settings.lastBookId)

  const queryEventId = Number(query.eventId ?? 0)
  if (queryEventId) {
    const chosen = await loadEventById(queryEventId)
    if (chosen) {
      event.value = toSelectedEvent(chosen)
      date.value = chosen.eventDate || todayString()
    }
  }
  // 礼簿默认事项优先于上次使用的事项。
  await applyBookDefault()
  if (!event.value) {
    const lastId = contextStore.settings.lastEventId
    if (lastId) {
      const candidate = await loadEventById(lastId)
      if (candidate && usedWithin30Days(candidate.updatedAt, candidate.eventDate)) {
        event.value = toSelectedEvent(candidate)
        date.value = candidate.eventDate || todayString()
      }
    }
  }

  const queryPersonId = Number(query.personId ?? 0)
  if (queryPersonId) {
    await loadPersonById(queryPersonId)
  }
  if (query.direction === 'in' || query.direction === 'out') {
    direction.value = query.direction
  } else if (event.value?.host === 'other') {
    direction.value = 'out'
  }

  syncItemKind()
  if (!event.value) date.value = todayString()
  initializing = false
  if (!person.value) {
    nextTick(() => {
      personPickerShow.value = true
    })
  }
}

async function initEdit(id: number): Promise<void> {
  await Promise.all([dictStore.ensure(), contextStore.load(), loadBooks()])
  const detail = await fetchRecordDetail(id)
  editRecord.value = detail
  recordId.value = detail.id
  person.value = {
    id: detail.personId,
    name: detail.personName,
    relationLabel: relationLabelOf(detail.personRelation),
  }
  bookId.value = detail.bookId
  direction.value = detail.direction
  amountText.value = detail.amountCent ? centToYuanText(detail.amountCent) : ''
  giftName.value = detail.giftName || ''
  giftValueText.value = detail.giftValueCent ? centToYuanText(detail.giftValueCent) : ''
  showGift.value = Boolean(detail.giftName || detail.giftValueCent)
  itemKind.value = detail.itemKind
  itemKindTouched.value = true
  date.value = detail.occurredOn
  status.value = detail.status
  remark.value = detail.remark || ''
  if (detail.eventId) {
    const item = await loadEventById(detail.eventId)
    event.value = item ? toSelectedEvent(item) : fallbackEvent(detail)
  }
  initializing = false
}

async function initRepay(ids: number[]): Promise<void> {
  await Promise.all([dictStore.ensure(), contextStore.load(), loadBooks()])
  const details = await Promise.all(ids.map((id) => fetchRecordDetail(id)))
  if (!details.length) {
    throw new Error('没有找到要还的记录')
  }
  const first = details[0]
  if (details.some((item) => item.personId !== first.personId)) {
    throw new Error('一次还礼只能对应同一个人')
  }
  origins.value = details
  originIds.value = details.map((item) => item.id)
  person.value = {
    id: first.personId,
    name: first.personName,
    relationLabel: relationLabelOf(first.personRelation),
  }
  direction.value = 'out'
  amountText.value = centToYuanText(details.reduce((sum, item) => sum + item.totalCent, 0))
  bookId.value = pickBookId(contextStore.settings.lastBookId, first.bookId)
  date.value = todayString()

  const lastId = contextStore.settings.lastEventId
  if (lastId) {
    const candidate = await loadEventById(lastId)
    if (candidate && usedWithin30Days(candidate.updatedAt, candidate.eventDate)) {
      event.value = toSelectedEvent(candidate)
    }
  }
  syncItemKind()
  initializing = false
}

async function init(query: Record<string, string>): Promise<void> {
  lastQuery = query
  const requested = query.mode
  mode.value = requested === 'edit' || requested === 'repay' ? requested : 'create'
  loading.value = true
  error.value = ''
  try {
    if (mode.value === 'edit') {
      const id = Number(query.id ?? 0)
      if (!id) throw new Error('缺少记录 id')
      await initEdit(id)
    } else if (mode.value === 'repay') {
      const ids = String(query.originIds ?? '')
        .split(',')
        .map((value) => Number(value.trim()))
        .filter((value) => value > 0)
      if (!ids.length) throw new Error('缺少要还的记录')
      await initRepay(ids)
    } else {
      await initCreate(query)
    }
  } catch (err) {
    error.value = err instanceof Error ? err.message : '加载失败'
  } finally {
    loading.value = false
  }
}

function retry(): void {
  void init(lastQuery)
}

onLoad((query) => {
  void init((query ?? {}) as Record<string, string>)
})

onShow(() => {
  if (initializing) return
  if (mode.value !== 'create') return
  void loadBooks().then(() => {
    if (!books.value.some((book) => book.id === bookId.value)) {
      bookId.value = pickBookId(bookId.value)
    }
  })
})

function onPersonSelect(selected: SelectedPerson): void {
  person.value = { id: selected.id, name: selected.name, relationLabel: selected.relationLabel }
}

function openPerson(): void {
  if (personLocked.value) {
    if (mode.value === 'edit') {
      toast(linked.value ? '已关联还礼，先解除关联再修改' : '编辑时不能更换亲友')
    }
    return
  }
  personPickerShow.value = true
}

function onEventSelect(selected: EventItem): void {
  event.value = toSelectedEvent(selected)
  eventFromBook.value = false
  if (!dateTouched.value) date.value = selected.eventDate || todayString()
}

function onDateChange(event: Event): void {
  date.value = (event as unknown as { detail: { value: string } }).detail.value
  dateTouched.value = true
}

function toggleDirection(value: 'in' | 'out'): void {
  if (directionLocked.value) {
    if (mode.value === 'edit' && linked.value) toast('已关联还礼，先解除关联再修改')
    return
  }
  direction.value = value
}

function pickKind(code: string): void {
  itemKind.value = code
  itemKindTouched.value = true
  kindSheet.value = false
}

function onBookChange(id: number): void {
  const wasFromBook = eventFromBook.value
  bookId.value = id
  if (mode.value === 'create') {
    eventFromBook.value = false
    if (wasFromBook) event.value = null
    void applyBookDefault()
  }
}

function createBook(): void {
  go('/sub-book/edit')
}

function addGift(): void {
  showGift.value = !showGift.value
  if (!showGift.value) {
    giftName.value = ''
    giftValueText.value = ''
  }
}

function resetForNext(): void {
  person.value = null
  amountText.value = ''
  giftName.value = ''
  giftValueText.value = ''
  showGift.value = false
  remark.value = ''
  status.value = 'pending'
  itemKindTouched.value = false
  syncItemKind()
  nextTick(() => {
    personPickerShow.value = true
  })
}

function buildCreateParams(confirm: boolean): Parameters<typeof createRecord>[0] {
  return {
    personId: person.value?.id ?? 0,
    personName: person.value ? undefined : '',
    eventId: event.value?.id ?? 0,
    bookId: bookId.value,
    direction: direction.value,
    itemKind: itemKind.value,
    amountCent: cent.value,
    giftName: giftName.value.trim(),
    giftValueCent: giftValueCent.value,
    occurredOn: date.value,
    datePrecision: 'day',
    status: status.value,
    remark: remark.value.trim(),
    confirm,
  }
}

function buildUpdateParams(confirm: boolean): Parameters<typeof updateRecord>[1] {
  const record = editRecord.value
  return {
    personId: person.value?.id ?? record?.personId ?? 0,
    eventId: event.value?.id ?? record?.eventId ?? 0,
    bookId: bookId.value,
    direction: direction.value,
    itemKind: itemKind.value,
    amountCent: cent.value,
    giftName: giftName.value.trim(),
    giftValueCent: giftValueCent.value,
    occurredOn: date.value,
    datePrecision: 'day',
    status: status.value,
    remark: remark.value.trim(),
    paperNote: record?.paperNote ?? '',
    paperPage: record?.paperPage ?? '',
    paperSeq: record?.paperSeq ?? 0,
    confirm,
  }
}

async function saveRepay(): Promise<void> {
  await repayRecords({
    originIds: originIds.value,
    markRepaid: markRepaid.value,
    event: event.value
      ? {
          id: event.value.id,
          name: event.value.name,
          type: event.value.type,
          date: event.value.eventDate,
          datePrecision: event.value.datePrecision || 'day',
        }
      : null,
    occurredOn: date.value,
    datePrecision: 'day',
    items: [
      {
        bookId: bookId.value,
        itemKind: itemKind.value,
        amountCent: cent.value,
        giftName: giftName.value.trim(),
        giftValueCent: giftValueCent.value,
      },
    ],
    remark: remark.value.trim(),
  })
  await contextStore.rememberContext(bookId.value, event.value?.id ?? 0)
  toast('已记录还礼', 'success')
  back()
}

async function save(again = false): Promise<void> {
  if (submitting.value) return
  if (!person.value) {
    toast('请选择亲友')
    personPickerShow.value = true
    return
  }
  if (!bookId.value) {
    toast('请选择礼簿')
    return
  }
  if (mode.value !== 'repay' && !event.value && !selectedBook.value?.defaultEventId) {
    toast('请选择事项')
    eventPickerShow.value = true
    return
  }
  if (!valid.value) {
    toast('金额和礼品至少填一项')
    return
  }
  if (!giftName.value.trim() && giftValueCent.value > 0) {
    toast('填写礼品估价前请先填礼品名称')
    return
  }
  submitting.value = true
  try {
    if (mode.value === 'repay') {
      await saveRepay()
      return
    }
    if (mode.value === 'edit') {
      const saved = await submitWithConfirm((confirm) => updateRecord(recordId.value, buildUpdateParams(confirm)))
      if (!saved) return
      await contextStore.rememberContext(bookId.value, event.value?.id ?? 0)
      toast('已保存', 'success')
      back()
      return
    }
    const saved = await submitWithConfirm((confirm) => createRecord(buildCreateParams(confirm)))
    if (!saved) return
    await contextStore.rememberContext(bookId.value, event.value?.id ?? 0)
    toast('已保存', 'success')
    if (again) {
      resetForNext()
      return
    }
    back()
  } catch (err) {
    toast(err instanceof Error ? err.message : '保存失败')
  } finally {
    submitting.value = false
  }
}

async function removeRecord(): Promise<void> {
  if (!recordId.value) return
  const ok = await confirmModal('删除后无法恢复，关联的还礼记录不受影响。', '删除这笔记录')
  if (!ok) return
  try {
    await deleteRecord(recordId.value)
    toast('已删除', 'success')
    back()
  } catch (err) {
    toast(err instanceof Error ? err.message : '删除失败')
  }
}

function goDetail(record: RecordDetail): void {
  go(`/sub-record/detail?id=${record.id}`)
}
</script>

<template>
  <view class="screen form-screen">
    <AppNav :title="title" />

    <scroll-view scroll-y class="form-screen__body">
      <view v-if="loading" class="loading-tip">正在加载…</view>

      <view v-else-if="error" class="error-state">
        <text class="error-state__text">{{ error }}</text>
        <view class="btn btn--secondary error-state__btn" @click="retry">重试</view>
      </view>

      <template v-else>
        <view v-if="mode !== 'repay'" class="page">
          <view class="segmented direction">
            <view
              class="segmented__item direction__item"
              :class="{ 'segmented__item--on': direction === 'in', 'direction__item--locked': directionLocked }"
              @click="toggleDirection('in')"
            >
              收礼
            </view>
            <view
              class="segmented__item direction__item"
              :class="{ 'segmented__item--on': direction === 'out', 'direction__item--locked': directionLocked }"
              @click="toggleDirection('out')"
            >
              送礼
            </view>
          </view>
        </view>

        <view v-if="mode === 'repay' && origins.length" class="page">
          <view class="origin card" @click="originExpanded = !originExpanded">
            <view class="origin__line">
              <text class="origin__summary">还 {{ origins.length }} 笔 · {{ originEventName }} · </text>
              <Money :cent="originTotalCent" direction="in" size="s" />
              <text class="origin__arrow">{{ originExpanded ? '⌄' : '›' }}</text>
            </view>
            <view v-if="originExpanded" class="origin__list">
              <view v-for="record in origins" :key="record.id" class="origin__item" @click.stop="goDetail(record)">
                <text class="origin__date">{{ preciseDateLabel(record.occurredOn, record.datePrecision) }}</text>
                <text class="origin__event">{{ record.eventName }}</text>
                <Money :cent="record.totalCent" direction="in" size="s" />
                <text class="origin__go">›</text>
              </view>
            </view>
          </view>
        </view>

        <view class="page">
          <view class="amount">
            <Money :cent="totalCent" :direction="direction" size="display" />
            <view class="gift-toggle" @click="addGift">
              <text>{{ showGift ? '收起礼品' : '礼品 ⊕ 添加礼品' }}</text>
            </view>
            <view v-if="showGift" class="gift">
              <view class="gift__row">
                <text class="gift__label">礼品</text>
                <input v-model="giftName" class="gift__input" placeholder="礼品名称，如被面" />
              </view>
              <view class="gift__row">
                <text class="gift__label">估价</text>
                <input v-model="giftValueText" class="gift__input gift__input--num" type="digit" placeholder="元，可不填" />
              </view>
            </view>
          </view>
        </view>

        <view class="page">
          <view class="form-card">
            <view class="field" @click="openPerson">
              <text class="field__label">亲友</text>
              <view class="field__value person">
                <PersonAvatar v-if="person" :id="person.id" :name="person.name" :size="24" />
                <text v-if="person" class="person__name">{{ person.name }}</text>
                <text v-if="person" class="person__relation">{{ person.relationLabel }}</text>
                <text v-else class="person__placeholder">选择亲友</text>
                <text v-if="personLocked" class="locked-mark">锁定</text>
              </view>
              <text class="field__arrow">›</text>
            </view>

            <view class="field" @click="eventPickerShow = true">
              <text class="field__label">事项</text>
              <view class="field__value">
                <text v-if="event" class="event-name">{{ event.name }}</text>
                <text v-else class="placeholder">{{ mode === 'repay' ? '他家的哪个事项？' : '选择事项' }}</text>
                <text v-if="eventFromBook" class="default-hint">礼簿默认</text>
              </view>
              <text class="field__arrow">›</text>
            </view>

            <view class="field field--block">
              <text class="field__label">礼簿</text>
              <view class="field__value field__value--chips">
                <BookChips
                  :model-value="bookId"
                  :books="books"
                  @update:model-value="onBookChange"
                  @create="createBook"
                />
                <text v-if="bookDefaultHint" class="default-hint default-hint--line">{{ bookDefaultHint }}</text>
              </view>
            </view>

            <view class="field" @click="kindSheet = true">
              <text class="field__label">名目</text>
              <text class="field__value">{{ dictStore.itemKindLabel(itemKind) }}</text>
              <text class="field__arrow">›</text>
            </view>

            <picker mode="date" :value="date" @change="onDateChange">
              <view class="field">
                <text class="field__label">日期</text>
                <text class="field__value">{{ dateLabel }}</text>
                <text class="field__arrow">›</text>
              </view>
            </picker>

            <view v-if="mode !== 'repay'" class="field field--block">
              <text class="field__label">状态</text>
              <view class="field__value statuses">
                <text
                  v-for="option in statusOptions"
                  :key="option.code"
                  class="status-pill"
                  :class="[`status-pill--${option.code}`, { 'status-pill--on': status === option.code }]"
                  @click="status = option.code"
                >
                  {{ option.label }}
                </text>
              </view>
            </view>

            <view v-else class="field field--block">
              <text class="field__label">状态</text>
              <view class="field__value">
                <view class="switch-row" @click="markRepaid = !markRepaid">
                  <view class="switch" :class="{ 'switch--on': markRepaid }"><view class="switch__dot" /></view>
                  <text class="switch__label">同时标记为已还</text>
                </view>
                <text v-if="!markRepaid" class="repay-hint">部分还礼时可关闭，原记录保持待还</text>
              </view>
            </view>

            <view class="field field--block">
              <text class="field__label">备注</text>
              <input v-model="remark" class="field__input remark" placeholder="选填" />
            </view>
          </view>

          <text v-if="mode === 'edit' && linked" class="linked-hint">这笔记录已有还礼关联，方向与亲友不可修改。</text>

          <view v-if="mode === 'edit'" class="row-actions">
            <text class="row-actions__danger" @click="removeRecord">删除这笔记录</text>
          </view>
        </view>
      </template>
    </scroll-view>

    <view v-if="!loading && !error" class="form-footer">
      <view class="form-footer__links">
        <text v-if="mode === 'create'" class="form-footer__link" @click="save(true)">保存并再记一笔</text>
        <text v-else-if="mode === 'repay'" class="form-footer__hint">保存后，被还记录{{ markRepaid ? '会' : '不会' }}标为已还</text>
      </view>
      <AmountPad
        v-model="amountText"
        :quick-amounts="contextStore.settings.quickAmounts"
        :disabled="!valid"
        @save="save(false)"
      />
    </view>

    <PersonPicker v-model:show="personPickerShow" @select="onPersonSelect" />
    <EventPicker v-model:show="eventPickerShow" @select="onEventSelect" />

    <wd-popup v-model="kindSheet" position="bottom" :safe-area-inset-bottom="true" custom-style="border-radius: 20px 20px 0 0;">
      <view class="kind-sheet">
        <text class="kind-sheet__title">选择名目</text>
        <view class="kind-sheet__chips">
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
    </wd-popup>

    <ConfirmSheet />
  </view>
</template>

<style lang="scss" scoped>
.form-screen {
  display: flex;
  flex-direction: column;
  height: 100vh;
  overflow: hidden;
  padding-bottom: 0;
}

.form-screen__body {
  flex: 1;
  min-height: 0;
}

.direction {
  margin-top: 4px;
}

.direction__item {
  height: 44px;
}

.direction__item--locked {
  opacity: 0.55;
}

.origin {
  margin-top: 12px;
  padding: 12px 14px;
}

.origin__line {
  display: flex;
  align-items: center;
  gap: 8px;
}

.origin__summary {
  flex: 1;
  font-size: 14px;
  font-weight: 500;
  color: var(--c-ink-1);
}

.origin__arrow {
  color: var(--c-ink-3);
}

.origin__list {
  margin-top: 10px;
  border-top: 1px solid var(--c-divider);
}

.origin__item {
  display: flex;
  align-items: center;
  gap: 8px;
  height: 38px;
  border-bottom: 1px solid var(--c-divider);
}

.origin__date {
  font-size: 12px;
  color: var(--c-ink-3);
}

.origin__event {
  flex: 1;
  font-size: 13px;
  color: var(--c-ink-2);
  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
}

.origin__go {
  color: var(--c-ink-4);
}

.amount {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 20px 0 8px;
}

.gift-toggle {
  margin-top: 6px;
  padding: 6px 12px;
  border-radius: var(--r-full);
  background: var(--c-fill);
  font-size: 13px;
  color: var(--c-ink-2);
}

.gift {
  width: 100%;
  margin-top: 12px;
  padding: 4px 12px;
  border-radius: var(--r-m);
  background: var(--c-surface);
  box-shadow: var(--shadow-card);
}

.gift__row {
  display: flex;
  align-items: center;
  gap: 10px;
  min-height: 44px;
}

.gift__row + .gift__row {
  box-shadow: inset 0 1px 0 var(--c-divider);
}

.gift__label {
  width: 40px;
  font-size: 13px;
  color: var(--c-ink-3);
}

.gift__input {
  flex: 1;
  font-size: 14px;
}

.gift__input--num {
  text-align: right;
  font-family: $font-num;
}

.person {
  display: flex;
  align-items: center;
  gap: 8px;
}

.person__name {
  font-size: 15px;
  font-weight: 500;
}

.person__relation {
  padding: 1px 6px;
  border-radius: var(--r-xs);
  background: var(--c-fill);
  font-size: 12px;
  color: var(--c-ink-2);
}

.person__placeholder {
  color: var(--c-ink-3);
}

.locked-mark {
  padding: 1px 5px;
  border-radius: var(--r-xs);
  background: var(--c-none-light);
  font-size: 11px;
  color: var(--c-none);
}

.event-name {
  font-size: 15px;
  color: var(--c-ink-1);
}

.placeholder {
  font-size: 15px;
  color: var(--c-ink-3);
}

.default-hint {
  margin-left: 6px;
  font-size: 11px;
  color: var(--c-pending);
}

.default-hint--line {
  display: block;
  margin: 6px 0 0;
}

.field--block {
  align-items: flex-start;
  padding-top: 12px;
  padding-bottom: 12px;
}

.field__value--chips {
  overflow: hidden;
}

.statuses {
  display: flex;
  gap: 8px;
}

.status-pill {
  height: 28px;
  padding: 0 12px;
  border-radius: var(--r-full);
  background: var(--c-fill);
  font-size: 13px;
  line-height: 28px;
  color: var(--c-ink-2);
}

.status-pill--pending.status-pill--on {
  background: var(--c-pending-light);
  color: var(--c-pending);
  box-shadow: inset 0 0 0 1px var(--c-pending);
}

.status-pill--repaid.status-pill--on {
  background: var(--c-repaid-light);
  color: var(--c-repaid);
  box-shadow: inset 0 0 0 1px var(--c-repaid);
}

.status-pill--none.status-pill--on {
  background: var(--c-none-light);
  color: var(--c-none);
  box-shadow: inset 0 0 0 1px var(--c-none);
}

.switch-row {
  display: flex;
  align-items: center;
  gap: 8px;
}

.switch {
  position: relative;
  width: 36px;
  height: 20px;
  border-radius: var(--r-full);
  background: var(--c-ink-4);
  transition: background 0.2s;
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
  transform: translateX(16px);
}

.switch__label {
  font-size: 14px;
  color: var(--c-ink-1);
}

.repay-hint {
  display: block;
  margin-top: 6px;
  font-size: 12px;
  color: var(--c-ink-3);
}

.remark {
  text-align: left;
}

.linked-hint {
  display: block;
  margin-top: 10px;
  padding: 0 4px;
  font-size: 12px;
  color: var(--c-pending);
}

.row-actions {
  padding: 18px 0 12px;
  text-align: center;
}

.row-actions__danger {
  font-size: 14px;
  color: var(--c-danger);
}

.form-footer {
  flex: none;
  background: var(--c-bg);
}

.form-footer__links {
  display: flex;
  justify-content: center;
  padding: 4px 16px 6px;
}

.form-footer__link {
  font-size: 13px;
  color: var(--c-primary);
}

.form-footer__hint {
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

.kind-sheet {
  padding: 18px 16px calc(20px + env(safe-area-inset-bottom));
}

.kind-sheet__title {
  display: block;
  font-size: 16px;
  font-weight: 600;
}

.kind-sheet__chips {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-top: 14px;
}
</style>
