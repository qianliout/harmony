<script setup lang="ts">
// 桌面识别核对表：与窄屏 ReviewTable 共用 useReviewState，只替换成原生 table 与横向滚动。
// 标黄行（同名待选、大小写不一致、字迹不清、缺姓名/金额）必须勾选确认后才会放行写入。
import {
  ITEM_KIND_OPTIONS,
  type ReviewRow,
  type ReviewState,
} from '@/sub-recognize/useReviewState'
import { formatCent, yuanToCent } from '@/utils/money'

import WebIcon from './WebIcon.vue'
import WebTable from './WebTable.vue'

const props = withDefaults(
  defineProps<{
    review: ReviewState
    readonly?: boolean
  }>(),
  { readonly: false },
)

interface FlagView {
  key: string
  label: string
  tone: 'warn' | 'info' | 'muted'
}

/** 把原生 input/select 事件包装成 useReviewState 期望的 detail.value 形状。 */
function asDetail(event: Event): Event {
  const target = event.target as HTMLInputElement | HTMLSelectElement | null
  return { detail: { value: target?.value ?? '' } } as unknown as Event
}

/** 这一行是否需要人工确认：同名、大小写不一致、低置信度、缺姓名或金额。 */
function isFlagged(row: ReviewRow): boolean {
  return (
    row.matchStatus === 'multiple' ||
    row.amountMismatch ||
    props.review.isLowConfidence(row) ||
    !row.name.trim() ||
    (row.itemKind !== 'gift' && yuanToCent(row.amountText) <= 0)
  )
}

/** 待确认：缺姓名/金额、同名、大小写不一致或低置信度，且尚未勾选确认。 */
function isPending(row: ReviewRow): boolean {
  const low = props.review.isLowConfidence(row)
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

function flagsOf(row: ReviewRow): FlagView[] {
  const out: FlagView[] = []
  if (row.duplicate) out.push({ key: 'dup', label: '重复', tone: 'muted' })
  if (row.matchStatus === 'multiple') {
    out.push({ key: 'same', label: `同名 ${row.candidates.length} 位`, tone: 'warn' })
  }
  if (row.amountMismatch) out.push({ key: 'mismatch', label: '大小写不一致', tone: 'warn' })
  if (props.review.isLowConfidence(row)) out.push({ key: 'low', label: '字迹不清', tone: 'warn' })
  if (!row.name.trim()) out.push({ key: 'name', label: '缺姓名', tone: 'warn' })
  if (row.itemKind !== 'gift' && yuanToCent(row.amountText) <= 0) {
    out.push({ key: 'amount', label: '缺金额', tone: 'warn' })
  }
  if (row.matchStatus === 'single') out.push({ key: 'match', label: candidateLabel(row), tone: 'info' })
  if (row.matchStatus === 'none' && row.name.trim()) out.push({ key: 'new', label: '新亲友', tone: 'info' })
  return out
}

function candidateLabel(row: ReviewRow): string {
  const candidate = row.candidates.find((item) => item.id === row.personId)
  return candidate ? `${candidate.name} · ${candidate.relationLabel}` : '已匹配'
}

function toggleRow(row: ReviewRow): void {
  if (props.readonly) return
  // 勾选即确认：黄色行只有勾上并确认后才不再阻塞写入。
  if (!row.checked) props.review.confirmRow(row)
  props.review.toggleRow(row)
}

function pickCandidate(row: ReviewRow, event: Event): void {
  const value = (event.target as HTMLSelectElement).value
  if (value === 'new') {
    row.personId = 0
    row.matchStatus = 'none'
    row.candidates = []
    props.review.confirmRow(row)
    return
  }
  const candidate = row.candidates.find((item) => item.id === Number(value))
  if (candidate) props.review.chooseCandidate(row, candidate)
}

function hasSubRow(row: ReviewRow): boolean {
  return (
    (row.matchStatus === 'multiple' && row.candidates.length > 0) ||
    row.amountMismatch ||
    row.repays.length > 0 ||
    Boolean(row.error)
  )
}

function repayTotalText(row: ReviewRow, index: number): string {
  const repay = row.repays[index]
  const total = repay ? props.review.repayTotalCent(repay) : 0
  return total > 0 ? formatCent(total) : ''
}
</script>

<template>
  <WebTable class="rv-table">
    <thead>
      <tr>
        <th class="rv-col-check">确认</th>
        <th class="rv-col-seq">序</th>
        <th>姓名</th>
        <th>大写</th>
        <th>小写</th>
        <th>名目</th>
        <th>纸本标注</th>
        <th>提示</th>
        <th class="rv-col-ops" />
      </tr>
    </thead>
    <tbody>
      <template v-for="row in review.rows.value" :key="row.key">
        <tr
          class="rv-row"
          :class="{
            'rv-row--off': !row.checked,
            'rv-row--pend': isPending(row),
            'rv-row--err': Boolean(row.error),
            'rv-row--active': review.activeKey.value === row.key,
          }"
          :data-row-key="row.key"
          @click="review.activate(row)"
        >
          <td class="rv-col-check" @click.stop>
            <input
              class="web-check"
              type="checkbox"
              :checked="row.checked"
              :disabled="readonly"
              aria-label="确认这一行"
              @change="toggleRow(row)"
            />
          </td>
          <td class="rv-col-seq web-num web-muted">{{ row.seq }}</td>

          <td class="rv-col-name">
            <input
              class="web-input rv-input"
              type="text"
              :data-cell="`${row.key}:name`"
              :value="row.name"
              :disabled="readonly"
              placeholder="姓名"
              @focus="review.activate(row)"
              @input="review.onNameInput(row, asDetail($event))"
            />
            <span v-if="row.matchStatus === 'none' && row.name.trim()" class="rv-tag rv-tag--new">新</span>
            <span v-else-if="row.matchStatus === 'single'" class="rv-tag">{{ candidateLabel(row) }}</span>
          </td>

          <td
            class="rv-col-upper"
            :class="{ 'rv-mismatch': row.amountMismatch }"
            :title="row.amountMismatch ? review.mismatchTitle(row) : undefined"
          >
            {{ row.amountUpper || '—' }}
          </td>

          <td class="rv-col-lower" :class="{ 'rv-mismatch': row.amountMismatch }">
            <input
              class="web-input rv-input rv-input--num"
              type="text"
              inputmode="decimal"
              :data-cell="`${row.key}:amount`"
              :value="row.amountText"
              :disabled="readonly"
              placeholder="金额"
              @input="review.onAmountInput(row, asDetail($event))"
            />
          </td>

          <td class="rv-col-kind">
            <select
              class="web-select rv-select"
              :value="String(review.itemKindIndex(row.itemKind))"
              :disabled="readonly"
              aria-label="名目"
              @change="review.onItemKindChange(row, asDetail($event))"
            >
              <option v-for="(label, index) in ITEM_KIND_OPTIONS" :key="label" :value="String(index)">
                {{ label }}
              </option>
            </select>
          </td>

          <td class="rv-col-note">
            <input
              class="web-input rv-input"
              type="text"
              :data-cell="`${row.key}:note`"
              :value="row.note"
              :disabled="readonly"
              placeholder="纸本标注"
              @input="review.onNoteInput(row, asDetail($event))"
            />
          </td>

          <td class="rv-col-flags">
            <div class="rv-flags">
              <span
                v-for="flag in flagsOf(row)"
                :key="flag.key"
                class="rv-flag"
                :class="`rv-flag--${flag.tone}`"
              >
                {{ flag.label }}
              </span>
              <span v-if="isPending(row)" class="rv-flag rv-flag--pend">待确认</span>
              <span v-else-if="isFlagged(row)" class="rv-flag rv-flag--ok">已确认</span>
            </div>
          </td>

          <td class="rv-col-ops" @click.stop>
            <button
              v-if="!readonly"
              class="web-btn web-btn--text web-btn--sm rv-del"
              type="button"
              aria-label="删除这一行"
              @click="review.removeRow(row)"
            >
              <WebIcon name="trash" :size="15" />
            </button>
          </td>
        </tr>

        <tr v-if="hasSubRow(row)" class="rv-subrow">
          <td :colspan="9">
            <div class="rv-sub">
              <div v-if="row.matchStatus === 'multiple' && row.candidates.length" class="rv-sub__block">
                <span class="rv-sub__label">同名亲友</span>
                <template v-if="readonly">
                  <span class="web-sm web-muted">尚未选择</span>
                </template>
                <select
                  v-else
                  class="web-select rv-select rv-select--wide"
                  value=""
                  @change="pickCandidate(row, $event)"
                >
                  <option value="" disabled>选择已有亲友…</option>
                  <option v-for="candidate in row.candidates" :key="candidate.id" :value="String(candidate.id)">
                    {{ candidate.name }} · {{ candidate.relationLabel }}
                  </option>
                  <option value="new">＋ 新建同名亲友</option>
                </select>
              </div>

              <div v-if="row.amountMismatch" class="rv-sub__block">
                <span class="rv-sub__label">金额</span>
                <span class="web-sm">
                  大写 {{ row.amountUpper }} · 小写 {{ row.amountLower ?? '—' }}
                </span>
                <template v-if="!readonly">
                  <button class="web-btn web-btn--text web-btn--sm" type="button" @click="review.useUpper(row)">
                    按大写
                  </button>
                  <button class="web-btn web-btn--text web-btn--sm" type="button" @click="review.useLower(row)">
                    按小写
                  </button>
                </template>
              </div>

              <div v-if="row.repays.length" class="rv-sub__block rv-sub__block--repay">
                <span class="rv-sub__label">回礼标注</span>
                <label v-if="!readonly" class="rv-switch">
                  <input
                    type="checkbox"
                    :checked="row.repaysEnabled"
                    @change="review.toggleRepays(row)"
                  />
                  一并补录
                </label>
                <template v-if="readonly">
                  <span v-for="(repay, index) in row.repays" :key="repay.key" class="web-sm web-muted">
                    {{ index + 1 }}. {{ review.repaySummary(repay) }}
                  </span>
                </template>
                <template v-else>
                  <div v-for="(repay, index) in row.repays" :key="repay.key" class="rv-repay">
                    <span class="rv-repay__no web-num web-muted">{{ index + 1 }}</span>
                    <input
                      class="web-input rv-input rv-repay__event"
                      type="text"
                      :value="repay.eventName"
                      :disabled="!row.repaysEnabled"
                      placeholder="对方事项"
                      @input="review.onRepayEventInput(repay, asDetail($event))"
                    />
                    <input
                      class="web-input rv-input rv-repay__date"
                      type="date"
                      :value="repay.occurredOn"
                      :disabled="!row.repaysEnabled"
                      aria-label="还礼日期"
                      @change="review.onRepayDateChange(repay, asDetail($event))"
                    />
                    <input
                      class="web-input rv-input rv-repay__amount"
                      type="text"
                      inputmode="decimal"
                      :value="repay.cashText"
                      :disabled="!row.repaysEnabled"
                      placeholder="现金"
                      @input="review.onRepayCashInput(repay, asDetail($event))"
                    />
                    <input
                      class="web-input rv-input rv-repay__amount"
                      type="text"
                      inputmode="decimal"
                      :value="repay.huopaoText"
                      :disabled="!row.repaysEnabled"
                      placeholder="火炮"
                      @input="review.onRepayHuopaoInput(repay, asDetail($event))"
                    />
                    <span class="web-sm web-muted">{{ repayTotalText(row, index) }}</span>
                  </div>
                </template>
              </div>

              <div v-if="row.error" class="rv-sub__block">
                <span class="rv-flag rv-flag--err">{{ row.error }}</span>
              </div>
            </div>
          </td>
        </tr>
      </template>
    </tbody>
  </WebTable>

</template>

<style lang="scss" scoped>
.rv-table :deep(.web-table) {
  min-width: 960px;
}

.rv-table :deep(th),
.rv-table :deep(td) {
  padding: 7px 10px;
}

.rv-col-check,
.rv-col-seq {
  text-align: center;
}

.rv-col-ops {
  text-align: right;
}

.rv-row {
  cursor: default;
}

.rv-row--off {
  opacity: 0.62;
}

.rv-row--pend {
  background: var(--c-pending-light);
}

.rv-row--err {
  box-shadow: inset 3px 0 0 var(--c-danger);
}

.rv-row--active {
  background: var(--c-primary-light);
}

.rv-row--active.rv-row--pend {
  background: var(--c-pending-light);
}

.rv-input {
  width: 100%;
  height: 30px;
  padding: 0 8px;
  font-size: 13px;
}

.rv-input--num {
  font-family: $font-num;
  text-align: right;
}

.rv-select {
  height: 30px;
  padding: 0 24px 0 8px;
  font-size: 13px;
  background-position: calc(100% - 13px) 13px, calc(100% - 8px) 13px;
}

.rv-select--wide {
  min-width: 220px;
}

.rv-col-upper {
  font-family: $font-serif;
  font-size: 13px;
  white-space: nowrap;
}

.rv-mismatch {
  color: var(--c-pending);
  font-weight: 600;
  box-shadow: inset 0 0 0 1px var(--c-pending);
  border-radius: var(--r-xs);
}

.rv-tag {
  display: inline-block;
  margin-top: 3px;
  color: var(--c-ink-3);
  font-size: 11px;
  line-height: 16px;
}

.rv-tag--new {
  color: var(--c-out);
}

.rv-flags {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
}

.rv-flag {
  display: inline-flex;
  align-items: center;
  height: 20px;
  padding: 0 7px;
  border-radius: var(--r-full);
  font-size: 11px;
  line-height: 1;
  white-space: nowrap;
}

.rv-flag--warn {
  background: var(--c-pending-light);
  color: var(--c-pending);
}

.rv-flag--info {
  background: var(--c-fill);
  color: var(--c-ink-2);
}

.rv-flag--muted {
  background: var(--c-none-light);
  color: var(--c-none);
}

.rv-flag--pend {
  background: var(--c-pending);
  color: var(--c-on-primary);
}

.rv-flag--ok {
  background: var(--c-repaid-light);
  color: var(--c-repaid);
}

.rv-flag--err {
  background: var(--c-danger-light);
  color: var(--c-danger);
  height: auto;
  padding: 3px 8px;
  white-space: normal;
}

.rv-del {
  padding: 0 6px;
  color: var(--c-ink-3);
}

.rv-del:hover {
  color: var(--c-danger);
}

.rv-subrow :deep(td) {
  padding: 0 10px 8px;
  background: var(--c-hover);
}

.rv-sub {
  display: flex;
  flex-direction: column;
  gap: 6px;
  padding: 8px 10px;
  border-radius: var(--r-s);
  background: var(--c-surface);
}

.rv-sub__block {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
}

.rv-sub__label {
  color: var(--c-ink-3);
  font-size: 12px;
  font-weight: 500;
}

.rv-sub__block--repay {
  align-items: flex-start;
}

.rv-switch {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  color: var(--c-ink-2);
  font-size: 12px;
}

.rv-repay {
  display: grid;
  grid-template-columns: 18px minmax(140px, 1.2fr) 130px 84px 84px 64px;
  align-items: center;
  gap: 6px;
  flex: 1;
  min-width: 0;
}

.rv-repay__no {
  font-size: 12px;
  text-align: center;
}

</style>
