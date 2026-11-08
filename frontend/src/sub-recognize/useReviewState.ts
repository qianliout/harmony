// 核对状态的唯一来源：窄屏列表与宽屏表格共用这里的行数组、勾选、匹配选择与回礼草稿，
// 窗口在 1024px 上下切换时只切换渲染组件，状态不重建。
import { computed, ref, type ComputedRef, type Ref } from 'vue'

import type {
  PaperDraft,
  RecognizeCommitItem,
  RecognizeItem,
  RecognizeTask,
} from '@/api/types'
import { todayString } from '@/utils/date'
import { centToYuanText, yuanToCent } from '@/utils/money'

import { inputValue } from './input'

/** 置信度低于该值的行需要人工确认；含 □ 的姓名一律算低置信度。 */
export const CONFIDENCE_THRESHOLD = 0.6

/** Tab / Shift+Tab 在行内跳转的字段顺序。 */
export const REVIEW_FIELDS = ['name', 'amount', 'note'] as const
export type ReviewField = (typeof REVIEW_FIELDS)[number]

export const ITEM_KIND_OPTIONS = ['现金', '火炮', '实物礼品']
export const ITEM_KIND_CODES = ['cash', 'huopao', 'gift']

export interface ReviewRepay {
  key: number
  enabled: boolean
  eventName: string
  occurredOn: string
  datePrecision: string
  cashText: string
  huopaoText: string
}

export interface ReviewRow {
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

export interface ReviewFallback {
  page: string
  direction: string
}

/** 核对页与两种布局组件之间的状态契约。 */
export interface ReviewState {
  rows: Ref<ReviewRow[]>
  activeKey: Ref<number>
  activeRow: ComputedRef<ReviewRow | null>
  selectedRows: ComputedRef<ReviewRow[]>
  selectedSumCent: ComputedRef<number>
  pendingRows: ComputedRef<ReviewRow[]>
  needPersonRows: ComputedRef<ReviewRow[]>
  firstUnresolved: ComputedRef<ReviewRow | null>
  isUnclearName(row: ReviewRow): boolean
  isLowConfidence(row: ReviewRow): boolean
  needsConfirm(row: ReviewRow): boolean
  loadTask(current: RecognizeTask, fallback: ReviewFallback): void
  reset(): void
  activate(row: ReviewRow): void
  moveActive(delta: number): ReviewRow | null
  appendRow(fallback: ReviewFallback): ReviewRow
  insertAfter(row: ReviewRow, fallback: ReviewFallback): ReviewRow
  insertAfterActive(fallback: ReviewFallback): ReviewRow
  removeRow(row: ReviewRow): void
  toggleRow(row: ReviewRow): void
  toggleAll(): void
  applyBatch(kind: 'direction' | 'itemKind', value: string): void
  chooseCandidate(row: ReviewRow, candidate: { id: number; name: string }): void
  confirmRow(row: ReviewRow): void
  useUpper(row: ReviewRow): void
  useLower(row: ReviewRow): void
  mismatchTitle(row: ReviewRow): string
  onNameInput(row: ReviewRow, event: Event): void
  onAmountInput(row: ReviewRow, event: Event): void
  onNoteInput(row: ReviewRow, event: Event): void
  restoreField(row: ReviewRow, field: ReviewField, value: string): void
  onDateChange(row: ReviewRow, event: Event): void
  onItemKindChange(row: ReviewRow, event: Event): void
  itemKindIndex(kind: string): number
  toggleRepays(row: ReviewRow): void
  onRepayEventInput(repay: ReviewRepay, event: Event): void
  onRepayDateChange(repay: ReviewRepay, event: Event): void
  onRepayCashInput(repay: ReviewRepay, event: Event): void
  onRepayHuopaoInput(repay: ReviewRepay, event: Event): void
  repayItems(repay: ReviewRepay): { itemKind: string; amountCent: number }[]
  repayTotalCent(repay: ReviewRepay): number
  repaySummary(repay: ReviewRepay): string
  buildCommitItems(ctx: { eventId: number; bookId: number; page: string }): RecognizeCommitItem[]
}

function draftToRepay(draft: PaperDraft, key: number): ReviewRepay {
  const items = draft.items ?? []
  return {
    key,
    enabled: true,
    eventName: draft.event?.name ?? '',
    occurredOn: draft.occurredOn || todayString(),
    datePrecision: draft.datePrecision || 'day',
    cashText: centToYuanText(items.find((item) => item.itemKind === 'cash')?.amountCent ?? 0),
    huopaoText: centToYuanText(items.find((item) => item.itemKind === 'huopao')?.amountCent ?? 0),
  }
}

export function useReviewState(): ReviewState {
  const rows = ref<ReviewRow[]>([])
  const activeKey = ref(0)
  let keySeq = 1

  const selectedRows = computed(() => rows.value.filter((row) => row.checked))
  const selectedSumCent = computed(() =>
    selectedRows.value.reduce((sum, row) => sum + yuanToCent(row.amountText), 0),
  )
  const needPersonRows = computed(() =>
    rows.value.filter((row) => row.checked && row.matchStatus === 'multiple' && !row.personId),
  )
  const activeRow = computed(() => rows.value.find((row) => row.key === activeKey.value) ?? null)

  function isUnclearName(row: ReviewRow): boolean {
    return row.name.includes('□')
  }

  function isLowConfidence(row: ReviewRow): boolean {
    return isUnclearName(row) || row.confidence < CONFIDENCE_THRESHOLD
  }

  /** 必须人工处理、否则不能写入的行。 */
  function needsConfirm(row: ReviewRow): boolean {
    if (!row.checked) return false
    if (!row.name.trim()) return true
    if (row.matchStatus === 'multiple' && !row.personId) return true
    if (isLowConfidence(row) && !row.confirmed) return true
    if (row.amountMismatch && !row.confirmed) return true
    return false
  }

  const pendingRows = computed(() => rows.value.filter(needsConfirm))
  const firstUnresolved = computed(
    () => pendingRows.value[0] ?? needPersonRows.value[0] ?? null,
  )

  function newRepay(draft: PaperDraft): ReviewRepay {
    return draftToRepay(draft, keySeq++)
  }

  function toRow(item: RecognizeItem, index: number, current: RecognizeTask, fallback: ReviewFallback): ReviewRow {
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
      direction: item.direction || fallback.direction,
      date: item.date && /^\d{4}-\d{2}-\d{2}$/.test(item.date) ? item.date : todayString(),
      note: item.note ?? '',
      paperPage: current.page || fallback.page,
      paperSeq: item.seq || index + 1,
      confidence: item.confidence ?? 0,
      confirmed: false,
      duplicate: Boolean(item.duplicateOf),
      error: '',
      repays,
      repaysEnabled: repays.length > 0,
    }
  }

  function loadTask(current: RecognizeTask, fallback: ReviewFallback): void {
    rows.value = current.items.map((item, index) => toRow(item, index, current, fallback))
    activeKey.value = rows.value[0]?.key ?? 0
  }

  function reset(): void {
    rows.value = []
    activeKey.value = 0
  }

  function blankRow(fallback: ReviewFallback): ReviewRow {
    const last = rows.value[rows.value.length - 1]
    return {
      key: keySeq++,
      checked: true,
      seq: (last?.seq ?? rows.value.length) + 1,
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
      direction: fallback.direction,
      date: todayString(),
      note: '',
      paperPage: fallback.page,
      paperSeq: (last?.paperSeq ?? rows.value.length) + 1,
      confidence: 1,
      confirmed: true,
      duplicate: false,
      error: '',
      repays: [],
      repaysEnabled: false,
    }
  }

  function activate(row: ReviewRow): void {
    activeKey.value = row.key
  }

  function moveActive(delta: number): ReviewRow | null {
    const list = rows.value
    if (!list.length) return null
    const current = list.findIndex((row) => row.key === activeKey.value)
    const base = current < 0 ? (delta > 0 ? -1 : 0) : current
    const next = Math.min(Math.max(base + delta, 0), list.length - 1)
    activeKey.value = list[next].key
    return list[next]
  }

  function appendRow(fallback: ReviewFallback): ReviewRow {
    const row = blankRow(fallback)
    rows.value.push(row)
    activeKey.value = row.key
    return row
  }

  function insertAfter(row: ReviewRow, fallback: ReviewFallback): ReviewRow {
    const index = rows.value.findIndex((item) => item.key === row.key)
    if (index < 0) return appendRow(fallback)
    const blank = blankRow(fallback)
    blank.seq = row.seq + 1
    blank.paperSeq = row.paperSeq + 1
    rows.value.splice(index + 1, 0, blank)
    activeKey.value = blank.key
    return blank
  }

  function insertAfterActive(fallback: ReviewFallback): ReviewRow {
    const row = activeRow.value
    return row ? insertAfter(row, fallback) : appendRow(fallback)
  }

  function removeRow(row: ReviewRow): void {
    const index = rows.value.findIndex((item) => item.key === row.key)
    if (index < 0) return
    rows.value.splice(index, 1)
    if (activeKey.value === row.key) {
      activeKey.value = rows.value[Math.min(index, rows.value.length - 1)]?.key ?? 0
    }
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

  function applyBatch(kind: 'direction' | 'itemKind', value: string): void {
    selectedRows.value.forEach((row) => {
      if (kind === 'direction') row.direction = value
      else row.itemKind = value
    })
  }

  function chooseCandidate(row: ReviewRow, candidate: { id: number; name: string }): void {
    row.personId = candidate.id
    row.matchStatus = 'single'
    row.confirmed = true
  }

  function confirmRow(row: ReviewRow): void {
    row.confirmed = true
  }

  /** 按大写为准：金额保持不变，只确认不再提示。 */
  function useUpper(row: ReviewRow): void {
    row.amountMismatch = false
    row.confirmed = true
  }

  function useLower(row: ReviewRow): void {
    if (row.amountLower === null) return
    row.amountText = centToYuanText(Math.round(row.amountLower * 100))
    row.amountMismatch = false
    row.confirmed = true
  }

  function mismatchTitle(row: ReviewRow): string {
    return `以大写为准：${row.amountUpper} = ${row.amountLower ?? row.amountText}`
  }

  function onNameInput(row: ReviewRow, event: Event): void {
    const value = inputValue(event)
    if (value !== row.name) {
      row.confirmed = true
    }
    row.name = value
    row.personId = 0
    row.matchStatus = 'none'
    row.candidates = []
  }

  function onAmountInput(row: ReviewRow, event: Event): void {
    row.amountText = inputValue(event)
  }

  function onNoteInput(row: ReviewRow, event: Event): void {
    row.note = inputValue(event)
  }

  /** Esc 取消编辑：把字段恢复到获得焦点时的值。 */
  function restoreField(row: ReviewRow, field: ReviewField, value: string): void {
    if (field === 'name') {
      row.name = value
      row.personId = 0
      row.matchStatus = 'none'
      row.candidates = []
      return
    }
    if (field === 'amount') {
      row.amountText = value
      return
    }
    row.note = value
  }

  function onDateChange(row: ReviewRow, event: Event): void {
    row.date = inputValue(event)
  }

  function onItemKindChange(row: ReviewRow, event: Event): void {
    const index = Number(inputValue(event))
    row.itemKind = ITEM_KIND_CODES[index] ?? 'cash'
  }

  function itemKindIndex(kind: string): number {
    const index = ITEM_KIND_CODES.indexOf(kind)
    return index < 0 ? 0 : index
  }

  function toggleRepays(row: ReviewRow): void {
    row.repaysEnabled = !row.repaysEnabled
  }

  function onRepayEventInput(repay: ReviewRepay, event: Event): void {
    repay.eventName = inputValue(event)
  }

  function onRepayDateChange(repay: ReviewRepay, event: Event): void {
    repay.occurredOn = inputValue(event)
  }

  function onRepayCashInput(repay: ReviewRepay, event: Event): void {
    repay.cashText = inputValue(event)
  }

  function onRepayHuopaoInput(repay: ReviewRepay, event: Event): void {
    repay.huopaoText = inputValue(event)
  }

  function repayItems(repay: ReviewRepay): { itemKind: string; amountCent: number }[] {
    return [
      { itemKind: 'cash', amountCent: yuanToCent(repay.cashText) },
      { itemKind: 'huopao', amountCent: yuanToCent(repay.huopaoText) },
    ].filter((item) => item.amountCent > 0)
  }

  function repayTotalCent(repay: ReviewRepay): number {
    return repayItems(repay).reduce((sum, item) => sum + item.amountCent, 0)
  }

  /** 子行摘要：2011.10 · 孙女 · 现金 100 + 火炮 20 */
  function repaySummary(repay: ReviewRepay): string {
    const parts = [repay.occurredOn.slice(0, 7).replace('-', '.'), repay.eventName.trim()]
    const items: string[] = []
    if (yuanToCent(repay.cashText) > 0) items.push(`现金 ${repay.cashText}`)
    if (yuanToCent(repay.huopaoText) > 0) items.push(`火炮 ${repay.huopaoText}`)
    if (items.length) parts.push(items.join(' + '))
    return parts.filter(Boolean).join(' · ')
  }

  function buildCommitItems(ctx: { eventId: number; bookId: number; page: string }): RecognizeCommitItem[] {
    return selectedRows.value.map((row) => ({
      personId: row.personId || undefined,
      personName: row.personId ? undefined : row.name.trim(),
      eventId: ctx.eventId,
      bookId: ctx.bookId,
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
      paperPage: row.paperPage || ctx.page,
      paperSeq: row.paperSeq,
      repays: row.repaysEnabled
        ? row.repays
            .filter((repay) => repay.enabled && repayItems(repay).length > 0)
            .map((repay) => ({
              event: repay.eventName.trim()
                ? {
                    id: 0,
                    name: repay.eventName.trim(),
                    type: 'other',
                    date: repay.occurredOn,
                    datePrecision: repay.datePrecision,
                  }
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
  }

  return {
    rows,
    activeKey,
    activeRow,
    selectedRows,
    selectedSumCent,
    pendingRows,
    needPersonRows,
    firstUnresolved,
    isUnclearName,
    isLowConfidence,
    needsConfirm,
    loadTask,
    reset,
    activate,
    moveActive,
    appendRow,
    insertAfter,
    insertAfterActive,
    removeRow,
    toggleRow,
    toggleAll,
    applyBatch,
    chooseCandidate,
    confirmRow,
    useUpper,
    useLower,
    mismatchTitle,
    onNameInput,
    onAmountInput,
    onNoteInput,
    restoreField,
    onDateChange,
    onItemKindChange,
    itemKindIndex,
    toggleRepays,
    onRepayEventInput,
    onRepayDateChange,
    onRepayCashInput,
    onRepayHuopaoInput,
    repayItems,
    repayTotalCent,
    repaySummary,
    buildCommitItems,
  }
}
