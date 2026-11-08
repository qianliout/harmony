<script setup lang="ts">
// 宽屏核对表：七列紧凑表格，行高 44。与窄屏共用 useReviewState。
import { ref } from 'vue'

import { useDictStore } from '@/stores/dict'
import { formatDate } from '@/utils/date'

import { ITEM_KIND_OPTIONS, type ReviewRow, type ReviewState } from '../useReviewState'

const props = defineProps<{
  review: ReviewState
  readonly: boolean
  page: string
  direction: string
}>()

const dictStore = useDictStore()
const expandedKey = ref(0)

function toggleCandidates(row: ReviewRow): void {
  expandedKey.value = expandedKey.value === row.key ? 0 : row.key
}

function candidateLabel(row: ReviewRow): string {
  const candidate = row.candidates.find((item) => item.id === row.personId)
  return candidate ? `${candidate.name} · ${candidate.relationLabel}` : '已匹配'
}

function appendRow(): void {
  props.review.appendRow({ page: props.page, direction: props.direction })
}

function pickCandidate(row: ReviewRow, candidate: { id: number; name: string }): void {
  props.review.chooseCandidate(row, candidate)
  expandedKey.value = 0
}
</script>

<template>
  <view class="table">
    <view class="table__head">
      <text class="th">序</text>
      <text class="th">姓名</text>
      <text class="th">大写</text>
      <text class="th">小写</text>
      <text class="th">名目</text>
      <text class="th">标注</text>
      <text class="th th--center">确认</text>
    </view>

    <view
      v-for="row in review.rows.value"
      :key="row.key"
      class="group"
      :class="{
        'group--off': !row.checked,
        'group--pending': !readonly && review.needsConfirm(row),
        'group--error': row.error,
        'group--active': review.activeKey.value === row.key,
      }"
      :data-row-key="row.key"
    >
      <view class="trow" @click="review.activate(row)">
        <text class="cell cell--seq">{{ row.seq }}</text>

        <view class="cell cell--name" :class="{ 'cell--low': review.isLowConfidence(row) }">
          <input
            class="cell-input"
            :data-cell="`${row.key}:name`"
            :disabled="readonly"
            :value="row.name"
            placeholder="姓名"
            @input="review.onNameInput(row, $event)"
          />
          <text
            v-if="row.matchStatus === 'multiple' && !row.personId"
            class="tag tag--pick"
            @click.stop="readonly ? undefined : toggleCandidates(row)"
          >
            {{ readonly ? '同名' : '选择' }}
          </text>
          <text v-else-if="row.matchStatus === 'single'" class="tag tag--ok">{{ candidateLabel(row) }}</text>
          <text v-else class="tag tag--new">新</text>
          <text v-if="row.duplicate" class="tag tag--dup">重复</text>
        </view>

        <view
          class="cell cell--upper"
          :class="{ 'cell--mismatch': row.amountMismatch }"
          :title="row.amountMismatch ? review.mismatchTitle(row) : undefined"
        >
          <text class="cell-text">{{ row.amountUpper || '—' }}</text>
        </view>

        <view
          class="cell cell--lower"
          :class="{ 'cell--mismatch': row.amountMismatch }"
          :title="row.amountMismatch ? review.mismatchTitle(row) : undefined"
        >
          <input
            class="cell-input cell-input--num"
            type="digit"
            :data-cell="`${row.key}:amount`"
            :disabled="readonly"
            :value="row.amountText"
            placeholder="金额"
            @input="review.onAmountInput(row, $event)"
          />
        </view>

        <view class="cell cell--kind">
          <picker
            mode="selector"
            :range="ITEM_KIND_OPTIONS"
            :value="review.itemKindIndex(row.itemKind)"
            :disabled="readonly"
            @change="review.onItemKindChange(row, $event)"
          >
            <text class="cell-text">{{ dictStore.itemKindLabel(row.itemKind) }}</text>
          </picker>
        </view>

        <view class="cell cell--note">
          <input
            class="cell-input"
            :data-cell="`${row.key}:note`"
            :disabled="readonly"
            :value="row.note"
            placeholder="纸本标注"
            @input="review.onNoteInput(row, $event)"
          />
        </view>

        <view class="cell cell--check">
          <view
            class="cb"
            :class="{ 'cb--on': row.checked }"
            @click.stop="readonly ? undefined : review.toggleRow(row)"
          >
            <text v-if="row.checked">✓</text>
          </view>
        </view>
      </view>

      <view v-if="expandedKey === row.key && row.matchStatus === 'multiple' && row.candidates.length" class="subrow subrow--pick">
        <text
          v-for="candidate in row.candidates"
          :key="candidate.id"
          class="chip chip--flat"
          :class="{ 'chip--flat-on': row.personId === candidate.id }"
          @click="pickCandidate(row, candidate)"
        >
          {{ candidate.name }} · {{ candidate.relationLabel }}
        </text>
      </view>

      <view v-if="row.amountMismatch" class="subrow subrow--mismatch">
        <text class="sub-hint">大写 {{ row.amountUpper }} · 小写 {{ row.amountLower ?? '-' }}，以大写为准</text>
        <template v-if="!readonly">
          <text class="sub-link" @click="review.useUpper(row)">按大写</text>
          <text class="sub-link" @click="review.useLower(row)">按小写</text>
        </template>
      </view>

      <view v-if="row.error" class="subrow subrow--error">
        <text class="sub-error">{{ row.error }}</text>
      </view>

      <view v-if="row.repays.length" class="subrow subrow--repay">
        <view class="repay-head">
          <text class="repay-title">标注里的还礼</text>
          <view
            class="switch"
            :class="{ 'switch--on': row.repaysEnabled }"
            @click="readonly ? undefined : review.toggleRepays(row)"
          >
            <view class="switch__dot" />
          </view>
          <text class="repay-hint">{{ row.repaysEnabled ? '一并补录' : '暂不补录' }}</text>
        </view>

        <template v-if="readonly">
          <text v-for="(repay, index) in row.repays" :key="repay.key" class="repay-summary">
            {{ index + 1 }}. {{ review.repaySummary(repay) }}
          </text>
        </template>
        <template v-else>
          <view v-for="(repay, index) in row.repays" :key="repay.key" class="repay-line">
            <text class="repay-no">{{ index + 1 }}</text>
            <input
              class="repay-event"
              :disabled="!row.repaysEnabled"
              :value="repay.eventName"
              placeholder="对方事项"
              @input="review.onRepayEventInput(repay, $event)"
            />
            <picker
              mode="date"
              :value="repay.occurredOn"
              :disabled="!row.repaysEnabled"
              @change="review.onRepayDateChange(repay, $event)"
            >
              <text class="repay-date">{{ formatDate(repay.occurredOn) }}</text>
            </picker>
            <input
              class="repay-amount"
              type="digit"
              :disabled="!row.repaysEnabled"
              :value="repay.cashText"
              placeholder="现金"
              @input="review.onRepayCashInput(repay, $event)"
            />
            <input
              class="repay-amount"
              type="digit"
              :disabled="!row.repaysEnabled"
              :value="repay.huopaoText"
              placeholder="火炮"
              @input="review.onRepayHuopaoInput(repay, $event)"
            />
          </view>
        </template>
      </view>
    </view>

    <view v-if="!readonly" class="addrow" @click="appendRow">＋ 添加一行（照片不全或漏识别时手动补录）</view>
  </view>
</template>

<style lang="scss" scoped>
// 1024 宽时右栏只有约 500px，七列必须收紧；更宽时由 fr 分配剩余空间。
$cols: 34px minmax(92px, 1.2fr) 76px 70px 72px minmax(108px, 1.5fr) 42px;

.table {
  --table-cols: #{$cols};
}

.table__head {
  display: grid;
  grid-template-columns: var(--table-cols);
  align-items: center;
  height: 36px;
  padding: 0 8px;
  background: var(--c-surface);
  border-bottom: 1px solid var(--c-line);
  font-size: 12px;
  color: var(--c-ink-3);
}

.th--center {
  text-align: center;
}

.group {
  border-bottom: 1px solid var(--c-divider);
  background: var(--c-surface);
}

.group--off {
  opacity: 0.55;
}

.group--pending {
  box-shadow: inset 3px 0 0 var(--c-pending);
}

.group--error {
  box-shadow: inset 3px 0 0 var(--c-danger);
}

.group--active {
  background: var(--c-primary-light);
}

.group--active.group--pending {
  background: var(--c-pending-light);
}

.trow {
  display: grid;
  grid-template-columns: var(--table-cols);
  align-items: center;
  min-height: 44px;
  padding: 0 8px;
  cursor: default;
}

.cell {
  display: flex;
  align-items: center;
  gap: 4px;
  min-width: 0;
  padding: 0 4px;
  font-size: 13px;
  color: var(--c-ink-1);
}

.cell--seq {
  font-family: $font-num;
  font-size: 12px;
  color: var(--c-ink-3);
  justify-content: center;
}

.cell--name {
  border-radius: 4px;
}

.cell--low {
  background: var(--c-pending-light);
  box-shadow: inset 0 0 0 1px var(--c-pending);
}

.cell--upper,
.cell--lower,
.cell--kind {
  color: var(--c-ink-2);
}

.cell--mismatch {
  background: var(--c-pending-light);
  box-shadow: inset 0 0 0 1px var(--c-pending);
  border-radius: 4px;
}

.cell--check {
  justify-content: center;
}

.cell-input {
  flex: 1;
  min-width: 0;
  height: 30px;
  padding: 0 6px;
  border-radius: 6px;
  background: var(--c-fill);
  font-size: 13px;
  color: var(--c-ink-1);
}

.cell-input--num {
  font-family: $font-num;
  text-align: right;
}

.cell-text {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.tag {
  flex: none;
  font-size: 10px;
  white-space: nowrap;
}

.tag--ok {
  color: var(--c-repaid);
}

.tag--new {
  color: #3a63a0;
}

.tag--pick {
  padding: 1px 4px;
  border-radius: 4px;
  background: var(--c-pending);
  color: var(--c-surface);
  cursor: pointer;
}

.tag--dup {
  padding: 1px 4px;
  border-radius: 4px;
  background: var(--c-fill);
  color: var(--c-ink-2);
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
  cursor: pointer;
}

.cb--on {
  background: var(--c-primary);
  border-color: var(--c-primary);
}

.subrow {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
  padding: 5px 12px 7px 48px;
  background: #fcfaf7;
  font-size: 12px;
  color: var(--c-ink-2);
}

.subrow--pick {
  gap: 6px;
}

.sub-hint {
  color: var(--c-pending);
}

.sub-link {
  color: var(--c-primary);
  font-weight: 500;
  cursor: pointer;
}

.sub-error {
  color: var(--c-danger);
}

.repay-head {
  display: flex;
  align-items: center;
  gap: 8px;
  width: 100%;
}

.repay-title {
  font-size: 12px;
  font-weight: 500;
  color: var(--c-ink-2);
}

.repay-hint {
  font-size: 11px;
  color: var(--c-ink-3);
}

.switch {
  position: relative;
  width: 30px;
  height: 18px;
  border-radius: var(--r-full);
  background: var(--c-ink-4);
  flex: none;
  cursor: pointer;
}

.switch--on {
  background: var(--c-repaid);
}

.switch__dot {
  position: absolute;
  top: 2px;
  left: 2px;
  width: 14px;
  height: 14px;
  border-radius: var(--r-full);
  background: var(--c-surface);
  transition: transform 0.2s;
}

.switch--on .switch__dot {
  transform: translateX(12px);
}

.repay-line {
  display: flex;
  align-items: center;
  gap: 6px;
  width: 100%;
}

.repay-no {
  width: 14px;
  font-family: $font-num;
  font-size: 11px;
  color: var(--c-ink-3);
}

.repay-event {
  flex: 1;
  min-width: 90px;
  height: 26px;
  padding: 0 6px;
  border-radius: 6px;
  background: var(--c-surface);
  font-size: 12px;
}

.repay-date {
  font-size: 11px;
  color: var(--c-ink-2);
  white-space: nowrap;
}

.repay-amount {
  width: 64px;
  height: 26px;
  padding: 0 6px;
  border-radius: 6px;
  background: var(--c-surface);
  font-family: $font-num;
  font-size: 12px;
  text-align: right;
}

.repay-summary {
  width: 100%;
  font-size: 12px;
}

.addrow {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 38px;
  margin: 8px 0;
  border: 1.5px dashed var(--c-line);
  border-radius: var(--r-s);
  font-size: 13px;
  color: var(--c-ink-2);
  cursor: pointer;
}
</style>
