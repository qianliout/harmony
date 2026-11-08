<script setup lang="ts">
import { computed, ref, watch } from 'vue'

import { markRecordRepaid, parsePaperNote, repayRecords } from '@/api/record'
import type { PaperDraft, RecordItem, RepayItem } from '@/api/types'
import { eventTypes } from '@/styles/palette'
import { draftDateParts, pad } from '@/utils/date'
import { centToYuanText, formatCent, yuanToCent } from '@/utils/money'

interface EditableItem {
  key: number
  itemKind: string
  amountText: string
  giftName: string
}

interface EditableDraft {
  raw: string
  year: number
  month: number
  day: number
  eventName: string
  eventType: string
  items: EditableItem[]
}

const props = defineProps<{
  show: boolean
  record: RecordItem | null
}>()

const emit = defineEmits<{
  'update:show': [value: boolean]
  saved: [count: number, cent: number]
}>()

const visible = computed({
  get: () => props.show,
  set: (value: boolean) => emit('update:show', value),
})

const note = ref('')
const drafts = ref<EditableDraft[]>([])
const active = ref(0)
const onlyMark = ref(false)
const parsing = ref(false)
const saving = ref(false)
const error = ref('')
const markRepaid = ref(true)
let parseTimer: ReturnType<typeof setTimeout> | null = null
let keySeq = 1

const currentYear = new Date().getFullYear()
const yearOptions = Array.from({ length: 50 }, (_, index) => currentYear - index)
const yearLabels = yearOptions.map((year) => `${year} 年`)
const monthOptions = ['不填月', ...Array.from({ length: 12 }, (_, index) => `${index + 1} 月`)]
const dayOptions = ['不填日', ...Array.from({ length: 31 }, (_, index) => `${index + 1} 日`)]
const festivalOptions = [
  { label: '正月', month: 2 },
  { label: '二月二', month: 2 },
  { label: '五一', month: 5 },
  { label: '端午', month: 6 },
  { label: '中秋', month: 9 },
  { label: '国庆', month: 10 },
  { label: '腊月', month: 1 },
  { label: '过年', month: 2 },
]

const activeDraft = computed(() => drafts.value[active.value] ?? null)
const currentItems = computed(() => activeDraft.value?.items ?? [])

const activeTotal = computed(() =>
  currentItems.value.reduce((sum, item) => sum + yuanToCent(item.amountText), 0),
)
const allTotal = computed(() =>
  drafts.value.reduce(
    (sum, draft) => sum + draft.items.reduce((itemSum, item) => itemSum + yuanToCent(item.amountText), 0),
    0,
  ),
)

const yearIndex = computed(() => Math.max(0, yearOptions.indexOf(activeDraft.value?.year ?? currentYear)))
const monthIndex = computed(() => activeDraft.value?.month ?? 0)
const dayIndex = computed(() => activeDraft.value?.day ?? 0)

const eventTypeLabels = eventTypes.map((type) => type.label)
const eventTypeIndex = computed(() => {
  const index = eventTypes.findIndex((type) => type.code === activeDraft.value?.eventType)
  return index < 0 ? eventTypes.length - 1 : index
})

function newItem(itemKind = 'cash', amountCent = 0, giftName = ''): EditableItem {
  return {
    key: keySeq++,
    itemKind,
    amountText: amountCent ? centToYuanText(amountCent) : '',
    giftName,
  }
}

function blankDraft(): EditableDraft {
  return {
    raw: '',
    year: currentYear,
    month: 0,
    day: 0,
    eventName: '',
    eventType: 'other',
    items: [newItem()],
  }
}

function toEditable(draft: PaperDraft): EditableDraft {
  const parts = draftDateParts(draft)
  const items = (draft.items ?? []).map((item) =>
    newItem(item.itemKind || 'cash', item.amountCent, item.giftName || ''),
  )
  if (items.length === 0) {
    items.push(newItem())
  }
  return {
    raw: draft.raw ?? '',
    year: parts.year,
    month: parts.month,
    day: parts.day,
    eventName: draft.event?.name ?? '',
    eventType: draft.event?.type ?? 'other',
    items,
  }
}

function precisionOf(draft: EditableDraft): 'day' | 'month' | 'year' {
  if (draft.day > 0) return 'day'
  if (draft.month > 0) return 'month'
  return 'year'
}

function dateOf(draft: EditableDraft): string {
  return `${draft.year}-${pad(draft.month || 1)}-${pad(draft.day || 1)}`
}

function reset(): void {
  note.value = props.record?.paperNote ?? ''
  drafts.value = []
  active.value = 0
  onlyMark.value = false
  error.value = ''
  markRepaid.value = true
  parseNow()
}

async function parseNow(): Promise<void> {
  if (!note.value.trim()) {
    onlyMark.value = false
    drafts.value = [blankDraft()]
    return
  }
  parsing.value = true
  error.value = ''
  try {
    const result = await parsePaperNote(note.value, props.record?.personId ?? 0)
    onlyMark.value = result.onlyMark
    drafts.value = result.repays.length ? result.repays.map(toEditable) : result.onlyMark ? [] : [blankDraft()]
    active.value = 0
  } catch (err) {
    error.value = err instanceof Error ? err.message : '解析失败，请手动填写'
    drafts.value = [blankDraft()]
  } finally {
    parsing.value = false
  }
}

function onNoteInput(event: Event): void {
  note.value = (event as unknown as { detail: { value: string } }).detail.value
  if (parseTimer) clearTimeout(parseTimer)
  parseTimer = setTimeout(() => parseNow(), 500)
}

function onYearChange(event: Event): void {
  const index = Number((event as unknown as { detail: { value: string } }).detail.value)
  if (activeDraft.value) activeDraft.value.year = yearOptions[index] ?? currentYear
}

function onMonthChange(event: Event): void {
  const index = Number((event as unknown as { detail: { value: string } }).detail.value)
  if (activeDraft.value) activeDraft.value.month = index
}

function onDayChange(event: Event): void {
  const index = Number((event as unknown as { detail: { value: string } }).detail.value)
  if (activeDraft.value) activeDraft.value.day = index
}

function applyFestival(month: number, isLayue: boolean): void {
  if (!activeDraft.value) return
  if (isLayue) {
    activeDraft.value.year += 1
  }
  activeDraft.value.month = month
  activeDraft.value.day = 0
}

function onEventNameInput(event: Event): void {
  if (activeDraft.value) activeDraft.value.eventName = (event as unknown as { detail: { value: string } }).detail.value
}

function onEventTypeChange(event: Event): void {
  const index = Number((event as unknown as { detail: { value: string } }).detail.value)
  if (activeDraft.value) activeDraft.value.eventType = eventTypes[index]?.code ?? 'other'
}

function onAmountInput(item: EditableItem, event: Event): void {
  item.amountText = (event as unknown as { detail: { value: string } }).detail.value
}

function addItem(): void {
  if (!activeDraft.value) return
  uni.showActionSheet({
    itemList: ['现金', '火炮'],
    success: (res) => {
      const kind = res.tapIndex === 1 ? 'huopao' : 'cash'
      activeDraft.value?.items.push(newItem(kind))
    },
  })
}

function removeItem(key: number): void {
  if (!activeDraft.value || activeDraft.value.items.length <= 1) return
  activeDraft.value.items = activeDraft.value.items.filter((item) => item.key !== key)
}

function addDraft(): void {
  drafts.value.push(blankDraft())
  active.value = drafts.value.length - 1
}

function buildItems(draft: EditableDraft): RepayItem[] {
  return draft.items
    .filter((item) => yuanToCent(item.amountText) > 0)
    .map((item) => ({
      bookId: 0,
      itemKind: item.itemKind,
      amountCent: yuanToCent(item.amountText),
      giftName: item.giftName,
      giftValueCent: 0,
    }))
}

async function saveAll(): Promise<void> {
  const record = props.record
  if (!record || saving.value) return
  error.value = ''
  if (onlyMark.value) {
    await markOnly()
    return
  }
  if (drafts.value.length === 0) {
    error.value = '请先填写还礼信息'
    return
  }
  saving.value = true
  let repayCount = 0
  let totalCent = 0
  try {
    for (const draft of drafts.value) {
      const items = buildItems(draft)
      if (items.length === 0) {
        error.value = '每次还礼至少填写一个金额'
        return
      }
      const occurredOn = dateOf(draft)
      const precision = precisionOf(draft)
      await repayRecords({
        originIds: [record.id],
        markRepaid: markRepaid.value,
        event: draft.eventName.trim()
          ? {
              id: 0,
              name: draft.eventName.trim(),
              type: draft.eventType,
              date: occurredOn,
              datePrecision: precision,
            }
          : null,
        occurredOn,
        datePrecision: precision,
        paperNote: note.value,
        items,
      })
      repayCount += 1
      totalCent += items.reduce((sum, item) => sum + item.amountCent, 0)
    }
    emit('saved', repayCount, totalCent)
    visible.value = false
  } catch (err) {
    error.value = err instanceof Error ? err.message : '保存失败'
  } finally {
    saving.value = false
  }
}

async function markOnly(): Promise<void> {
  const record = props.record
  if (!record || saving.value) return
  saving.value = true
  try {
    await markRecordRepaid(record.id, note.value)
    emit('saved', 0, 0)
    visible.value = false
  } catch (err) {
    error.value = err instanceof Error ? err.message : '标记失败'
  } finally {
    saving.value = false
  }
}

watch(
  () => [props.show, props.record?.id] as const,
  ([show]) => {
    if (show) reset()
  },
)
</script>

<template>
  <wd-popup
    v-model="visible"
    position="bottom"
    :safe-area-inset-bottom="true"
    custom-style="max-height: 90vh; border-radius: 20px 20px 0 0;"
  >
    <view class="sheet">
      <view class="sheet__head">
        <view>
          <text class="sheet__title">补录还礼 · {{ record?.personName }}</text>
          <text v-if="record" class="sheet__sub">原记录 {{ record.direction === 'in' ? '收' : '送' }} {{ formatCent(record.totalCent) }}<text v-if="record.paperPage"> · 第 {{ record.paperPage }} 页</text></text>
        </view>
        <text class="sheet__close" @click="visible = false">关闭</text>
      </view>

      <scroll-view scroll-y class="sheet__body">
        <text class="label">纸本标注</text>
        <textarea
          class="note"
          :value="note"
          auto-height
          placeholder="照抄纸本上的手写还礼标注"
          @input="onNoteInput"
        />
        <text v-if="parsing" class="hint">正在解析标注…</text>

        <view v-if="onlyMark" class="only-mark">
          <text class="only-mark__title">这条标注只有已还记号</text>
          <text class="only-mark__desc">保存后只把原记录标为已还，不生成送礼记录。</text>
        </view>

        <template v-else>
          <view v-if="drafts.length > 1" class="tabs">
            <text
              v-for="(draft, index) in drafts"
              :key="draft.year + '-' + index"
              class="tab"
              :class="{ 'tab--on': active === index }"
              @click="active = index"
            >
              {{ index + 1 }}. {{ draft.year }}.{{ draft.month || 1 }}{{ draft.day ? '.' + draft.day : '' }}
            </text>
            <text class="tab tab--add" @click="addDraft">＋</text>
          </view>
          <text v-if="drafts.length > 1" class="hint">标注里有 {{ drafts.length }} 次还礼，逐次核对</text>

          <view v-if="activeDraft" class="form">
            <view class="field-row">
              <text class="label">时间</text>
              <view class="pickers">
                <picker mode="selector" :range="yearLabels" :value="yearIndex" @change="onYearChange">
                  <text class="picker-value">{{ activeDraft.year }} 年</text>
                </picker>
                <picker mode="selector" :range="monthOptions" :value="monthIndex" @change="onMonthChange">
                  <text class="picker-value">{{ activeDraft.month ? activeDraft.month + ' 月' : '月' }}</text>
                </picker>
                <picker mode="selector" :range="dayOptions" :value="dayIndex" @change="onDayChange">
                  <text class="picker-value">{{ activeDraft.day ? activeDraft.day + ' 日' : '日' }}</text>
                </picker>
              </view>
            </view>
            <view class="festivals">
              <text
                v-for="festival in festivalOptions"
                :key="festival.label"
                class="chip chip--flat"
                @click="applyFestival(festival.month, festival.label === '腊月')"
              >
                {{ festival.label }}
              </text>
            </view>

            <view class="field-row">
              <text class="label">对方事项</text>
              <input
                class="input"
                :value="activeDraft.eventName"
                placeholder="不填则归入未注明事项"
                @input="onEventNameInput"
              />
              <picker mode="selector" :range="eventTypeLabels" :value="eventTypeIndex" @change="onEventTypeChange">
                <text class="picker-value">{{ eventTypeLabels[eventTypeIndex] }}</text>
              </picker>
            </view>

            <view class="items">
              <view v-for="item in currentItems" :key="item.key" class="item">
                <text class="item__kind">{{ item.itemKind === 'huopao' ? '火炮' : '现金' }}</text>
                <input
                  class="item__amount"
                  type="digit"
                  :value="item.amountText"
                  placeholder="金额（元）"
                  @input="onAmountInput(item, $event)"
                />
                <text v-if="currentItems.length > 1" class="item__remove" @click="removeItem(item.key)">×</text>
              </view>
              <text class="add-item" @click="addItem">＋ 添加名目</text>
            </view>

            <view class="total">
              <text class="total__label">本次合计</text>
              <text class="total__value">{{ formatCent(activeTotal) }}</text>
            </view>

            <view class="switch-row" @click="markRepaid = !markRepaid">
              <view class="switch" :class="{ 'switch--on': markRepaid }"><view class="switch__dot" /></view>
              <text class="switch__label">同时把原记录标为已还</text>
            </view>
            <text v-if="!markRepaid" class="hint">部分还礼时可关闭，原记录保持待还</text>
          </view>
        </template>

        <text v-if="error" class="error">{{ error }}</text>
        <text v-if="drafts.length > 1" class="all-total">共 {{ drafts.length }} 次 · {{ formatCent(allTotal) }}</text>
      </scroll-view>

      <view class="sheet__actions">
        <view class="btn btn--secondary sheet__btn" @click="markOnly">只标记已还</view>
        <view v-if="!onlyMark" class="btn btn--primary sheet__btn" @click="saveAll">
          {{ saving ? '保存中…' : `保存${drafts.length > 1 ? ` ${drafts.length} 次` : ''}还礼` }}
        </view>
      </view>
    </view>
  </wd-popup>
</template>

<style lang="scss" scoped>
.sheet {
  display: flex;
  flex-direction: column;
  max-height: 90vh;
}

.sheet__head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  padding: 16px 16px 8px;
}

.sheet__title {
  display: block;
  font-size: 17px;
  font-weight: 600;
}

.sheet__sub {
  display: block;
  margin-top: 2px;
  font-size: 12px;
  color: var(--c-ink-3);
}

.sheet__close {
  font-size: 14px;
  color: var(--c-ink-3);
}

.sheet__body {
  flex: 1;
  min-height: 0;
  padding: 0 16px;
}

.label {
  display: block;
  margin: 10px 0 6px;
  font-size: 13px;
  color: var(--c-ink-3);
}

.note {
  width: 100%;
  min-height: 52px;
  padding: 10px 12px;
  border-radius: var(--r-s);
  background: var(--c-fill);
  font-family: $font-serif;
  font-size: 15px;
  line-height: 22px;
  color: var(--c-ink-1);
  box-sizing: border-box;
}

.hint {
  display: block;
  margin-top: 6px;
  font-size: 12px;
  color: var(--c-ink-3);
}

.only-mark {
  margin-top: 14px;
  padding: 12px;
  border-radius: var(--r-s);
  background: var(--c-repaid-light);
}

.only-mark__title {
  font-size: 14px;
  font-weight: 500;
  color: var(--c-repaid);
}

.only-mark__desc {
  display: block;
  margin-top: 4px;
  font-size: 12px;
  color: var(--c-ink-2);
}

.tabs {
  display: flex;
  gap: 6px;
  margin-top: 12px;
  overflow-x: auto;
}

.tab {
  flex: none;
  padding: 6px 10px;
  border-radius: var(--r-s);
  background: var(--c-fill);
  font-size: 12px;
  color: var(--c-ink-2);
}

.tab--on {
  background: var(--c-out-light);
  color: var(--c-out);
  box-shadow: inset 0 0 0 1px var(--c-out);
  font-weight: 600;
}

.tab--add {
  color: var(--c-primary);
}

.form {
  margin-top: 6px;
}

.field-row {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-top: 12px;
}

.field-row .label {
  width: 60px;
  flex: none;
  margin: 0;
}

.pickers {
  display: flex;
  gap: 6px;
}

.picker-value {
  padding: 7px 10px;
  border-radius: var(--r-s);
  background: var(--c-fill);
  font-size: 13px;
  color: var(--c-ink-1);
}

.festivals {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  margin-top: 8px;
  padding-left: 60px;
}

.input {
  flex: 1;
  height: 38px;
  padding: 0 10px;
  border-radius: var(--r-s);
  background: var(--c-fill);
  font-size: 14px;
}

.items {
  margin-top: 12px;
}

.item {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 8px;
}

.item__kind {
  width: 40px;
  font-size: 14px;
  color: var(--c-ink-2);
}

.item__amount {
  flex: 1;
  height: 40px;
  padding: 0 12px;
  border-radius: var(--r-s);
  background: var(--c-fill);
  font-family: $font-num;
  font-size: 16px;
  text-align: right;
}

.item__remove {
  font-size: 18px;
  color: var(--c-ink-3);
  padding: 0 4px;
}

.add-item {
  font-size: 13px;
  color: var(--c-primary);
}

.total {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-top: 14px;
  padding-top: 10px;
  border-top: 1px solid var(--c-divider);
}

.total__label {
  font-size: 13px;
  color: var(--c-ink-3);
}

.total__value {
  font-family: $font-num;
  font-size: 20px;
  font-weight: 600;
  color: var(--c-out);
}

.switch-row {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-top: 14px;
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
  font-size: 13px;
  color: var(--c-ink-2);
}

.error {
  display: block;
  margin-top: 10px;
  font-size: 13px;
  color: var(--c-danger);
}

.all-total {
  display: block;
  margin: 10px 0 4px;
  font-family: $font-num;
  font-size: 13px;
  color: var(--c-ink-2);
  text-align: right;
}

.sheet__actions {
  display: flex;
  gap: 10px;
  padding: 12px 16px calc(16px + env(safe-area-inset-bottom));
}

.sheet__btn {
  flex: 1;
}
</style>
