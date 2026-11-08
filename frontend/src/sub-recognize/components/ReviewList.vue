<script setup lang="ts">
// 窄屏核对：卡片列表。状态全部来自 useReviewState，与宽屏表格共用。
import { ITEM_KIND_OPTIONS, type ReviewRow, type ReviewState } from '../useReviewState'
import { useDictStore } from '@/stores/dict'
import { formatDate } from '@/utils/date'

const props = defineProps<{
  review: ReviewState
  readonly: boolean
  page: string
  direction: string
}>()

defineEmits<{
  menu: [row: ReviewRow]
  batch: []
  'choose-book': []
}>()

const dictStore = useDictStore()

/** 末尾“添加一行”带页面上下文作为纸本页码/方向默认值。 */
function appendRow(): void {
  props.review.appendRow({ page: props.page, direction: props.direction })
}
</script>

<template>
  <view class="review-list">
    <view v-if="!readonly" class="toolbar">
      <text class="toolbar__all" @click="review.toggleAll()">☑ 全选</text>
      <text class="toolbar__btn" @click="$emit('batch')">批量</text>
      <text class="toolbar__btn" @click="$emit('choose-book')">礼簿</text>
      <text class="toolbar__count">已选 {{ review.selectedRows.value.length }} 条</text>
    </view>

    <view
      v-for="row in review.rows.value"
      :key="row.key"
      class="rrow"
      :class="{
        'rrow--off': !row.checked,
        'rrow--pending': !readonly && review.needsConfirm(row),
        'rrow--error': row.error,
        'rrow--active': review.activeKey.value === row.key,
      }"
      :data-row-key="row.key"
      @click="review.activate(row)"
    >
      <view class="rrow__top">
        <view
          class="cb"
          :class="{ 'cb--on': row.checked, 'cb--readonly': readonly }"
          @click.stop="readonly ? undefined : review.toggleRow(row)"
        >
          <text v-if="row.checked">✓</text>
        </view>
        <text class="rrow__seq">{{ row.seq }}</text>
        <input
          class="rrow__name"
          :class="{ 'rrow__name--low': review.isLowConfidence(row) }"
          :data-cell="`${row.key}:name`"
          :disabled="readonly"
          :value="row.name"
          placeholder="姓名"
          @input="review.onNameInput(row, $event)"
        />
        <text v-if="row.matchStatus === 'multiple'" class="match match--multi">多个同名</text>
        <text v-else-if="row.matchStatus === 'single'" class="match match--ok">已匹配</text>
        <text v-else class="match match--new">将新建</text>
        <text v-if="row.duplicate" class="badge">可能重复</text>
        <text v-if="review.isUnclearName(row)" class="badge badge--low">看不清</text>
        <text v-if="!readonly && review.needsConfirm(row)" class="badge badge--warn">待确认</text>
      </view>

      <view v-if="row.candidates.length && row.matchStatus === 'multiple'" class="candidates">
        <text
          v-for="candidate in row.candidates"
          :key="candidate.id"
          class="chip chip--flat"
          :class="{ 'chip--flat-on': row.personId === candidate.id }"
          @click.stop="readonly ? undefined : review.chooseCandidate(row, candidate)"
        >
          {{ candidate.name }} · {{ candidate.relationLabel }}
        </text>
      </view>

      <view class="rrow__fields">
        <input
          class="rrow__amount"
          :class="{ 'rrow__amount--low': row.amountMismatch }"
          type="digit"
          :data-cell="`${row.key}:amount`"
          :disabled="readonly"
          :value="row.amountText"
          placeholder="金额"
          @input="review.onAmountInput(row, $event)"
        />
        <picker
          mode="selector"
          :range="ITEM_KIND_OPTIONS"
          :value="review.itemKindIndex(row.itemKind)"
          :disabled="readonly"
          @change="review.onItemKindChange(row, $event)"
        >
          <text class="rrow__select">{{ dictStore.itemKindLabel(row.itemKind) }}</text>
        </picker>
        <picker mode="date" :value="row.date" :disabled="readonly" @change="review.onDateChange(row, $event)">
          <text class="rrow__select">{{ formatDate(row.date) }}</text>
        </picker>
      </view>

      <view v-if="row.amountMismatch" class="mismatch">
        <text class="mismatch__text">大写 {{ row.amountUpper }} · 小写 {{ row.amountLower ?? '-' }}</text>
        <template v-if="!readonly">
          <text class="mismatch__pick" @click.stop="review.useUpper(row)">按大写</text>
          <text class="mismatch__pick" @click.stop="review.useLower(row)">按小写</text>
        </template>
      </view>

      <input
        class="rrow__note"
        :data-cell="`${row.key}:note`"
        :disabled="readonly"
        :value="row.note"
        placeholder="纸本标注原文（选填）"
        @input="review.onNoteInput(row, $event)"
      />

      <view v-if="row.repays.length" class="repays">
        <view class="repays__head">
          <text class="repays__title">标注里的还礼</text>
          <view
            class="switch"
            :class="{ 'switch--on': row.repaysEnabled }"
            @click.stop="readonly ? undefined : review.toggleRepays(row)"
          >
            <view class="switch__dot" />
          </view>
          <text class="repays__hint">{{ row.repaysEnabled ? '一并补录' : '暂不补录' }}</text>
        </view>
        <template v-if="readonly">
          <text v-for="(repay, repayIndex) in row.repays" :key="repay.key" class="repay__summary">
            {{ repayIndex + 1 }}. {{ review.repaySummary(repay) }}
          </text>
        </template>
        <template v-else>
          <view v-for="(repay, repayIndex) in row.repays" :key="repay.key" class="repay">
            <text class="repay__no">{{ repayIndex + 1 }}</text>
            <input
              class="repay__event"
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
              <text class="repay__date">{{ formatDate(repay.occurredOn) }}</text>
            </picker>
            <input
              class="repay__amount"
              type="digit"
              :disabled="!row.repaysEnabled"
              :value="repay.cashText"
              placeholder="现金"
              @input="review.onRepayCashInput(repay, $event)"
            />
            <input
              class="repay__amount"
              type="digit"
              :disabled="!row.repaysEnabled"
              :value="repay.huopaoText"
              placeholder="火炮"
              @input="review.onRepayHuopaoInput(repay, $event)"
            />
          </view>
        </template>
      </view>

      <view v-if="!readonly" class="rrow__actions">
        <text v-if="review.needsConfirm(row)" class="rrow__confirm" @click.stop="review.confirmRow(row)">
          确认无误
        </text>
        <text class="rrow__menu" @click.stop="$emit('menu', row)">⋯</text>
      </view>
      <text v-if="row.error" class="rrow__error">{{ row.error }}</text>
    </view>

    <view v-if="!readonly" class="addrow" @click="appendRow">
      ＋ 添加一行（照片不全或漏识别时手动补录）
    </view>
  </view>
</template>

<style lang="scss" scoped>
.toolbar {
  display: flex;
  align-items: center;
  gap: 14px;
  padding: 10px 0;
  font-size: 13px;
  color: var(--c-ink-2);
}

.toolbar__all {
  color: var(--c-ink-1);
  font-weight: 500;
}

.toolbar__btn {
  color: var(--c-primary);
}

.toolbar__count {
  margin-left: auto;
  font-size: 12px;
  color: var(--c-ink-3);
}

.rrow {
  margin-bottom: 10px;
  padding: 12px;
  border-radius: var(--r-m);
  background: var(--c-surface);
  box-shadow: var(--shadow-card);
  border: 1px solid transparent;
}

.rrow--off {
  opacity: 0.55;
}

.rrow--pending {
  border-color: var(--c-pending);
}

.rrow--error {
  border-color: var(--c-danger);
}

.rrow--active {
  box-shadow: inset 0 0 0 2px var(--c-primary), var(--shadow-card);
}

.rrow__top {
  display: flex;
  align-items: center;
  gap: 8px;
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
}

.cb--on {
  background: var(--c-primary);
  border-color: var(--c-primary);
}

.cb--readonly {
  opacity: 0.7;
}

.rrow__seq {
  width: 22px;
  font-family: $font-num;
  font-size: 12px;
  color: var(--c-ink-3);
  flex: none;
}

.rrow__name {
  flex: 1;
  min-width: 0;
  height: 32px;
  padding: 0 8px;
  border-radius: 8px;
  background: var(--c-fill);
  font-size: 14px;
}

.rrow__name--low {
  background: var(--c-pending-light);
  box-shadow: inset 0 0 0 1px var(--c-pending);
}

.match {
  flex: none;
  font-size: 11px;
  white-space: nowrap;
}

.match--ok {
  color: var(--c-repaid);
}

.match--new {
  color: #3a63a0;
}

.match--multi {
  color: var(--c-pending);
}

.badge {
  flex: none;
  padding: 1px 5px;
  border-radius: 5px;
  background: var(--c-fill);
  font-size: 10px;
  color: var(--c-ink-2);
}

.badge--warn {
  background: var(--c-pending);
  color: var(--c-surface);
}

.badge--low {
  background: var(--c-pending-light);
  color: var(--c-pending);
}

.candidates {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  margin-top: 8px;
  padding-left: 48px;
}

.rrow__fields {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-top: 8px;
  padding-left: 48px;
}

.rrow__amount {
  width: 84px;
  height: 32px;
  padding: 0 8px;
  border-radius: 8px;
  background: var(--c-fill);
  font-family: $font-num;
  font-size: 15px;
  text-align: right;
}

.rrow__amount--low {
  box-shadow: inset 0 0 0 1px var(--c-pending);
}

.rrow__select {
  display: inline-flex;
  align-items: center;
  height: 32px;
  padding: 0 8px;
  border-radius: 8px;
  background: var(--c-fill);
  font-size: 12px;
  color: var(--c-ink-2);
  white-space: nowrap;
}

.mismatch {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-top: 6px;
  padding-left: 48px;
  font-size: 11px;
}

.mismatch__text {
  color: var(--c-pending);
}

.mismatch__pick {
  color: var(--c-primary);
  font-weight: 500;
}

.rrow__note {
  height: 32px;
  margin-top: 8px;
  padding: 0 8px;
  border-radius: 8px;
  background: var(--c-fill);
  font-family: $font-serif;
  font-size: 13px;
}

.repays {
  margin-top: 10px;
  padding: 8px 10px;
  border-radius: var(--r-s);
  background: #fcfaf7;
}

.repays__head {
  display: flex;
  align-items: center;
  gap: 8px;
}

.repays__title {
  font-size: 12px;
  font-weight: 500;
  color: var(--c-ink-2);
}

.repays__hint {
  font-size: 11px;
  color: var(--c-ink-3);
}

.switch {
  position: relative;
  width: 34px;
  height: 20px;
  border-radius: var(--r-full);
  background: var(--c-ink-4);
  flex: none;
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
  transform: translateX(14px);
}

.repay {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-top: 6px;
}

.repay__no {
  width: 16px;
  font-family: $font-num;
  font-size: 11px;
  color: var(--c-ink-3);
}

.repay__event {
  flex: 1;
  min-width: 0;
  height: 28px;
  padding: 0 6px;
  border-radius: 6px;
  background: var(--c-surface);
  font-size: 12px;
}

.repay__date {
  font-size: 11px;
  color: var(--c-ink-2);
  white-space: nowrap;
}

.repay__amount {
  width: 58px;
  height: 28px;
  padding: 0 6px;
  border-radius: 6px;
  background: var(--c-surface);
  font-family: $font-num;
  font-size: 12px;
  text-align: right;
}

.repay__summary {
  display: block;
  margin-top: 6px;
  font-size: 12px;
  color: var(--c-ink-2);
}

.rrow__actions {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 14px;
  margin-top: 8px;
}

.rrow__confirm {
  font-size: 12px;
  color: var(--c-pending);
  font-weight: 500;
}

.rrow__menu {
  font-size: 18px;
  color: var(--c-ink-3);
  padding: 0 4px;
}

.rrow__error {
  display: block;
  margin-top: 4px;
  font-size: 12px;
  color: var(--c-danger);
}

.addrow {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 38px;
  margin-top: 8px;
  border: 1.5px dashed var(--c-line);
  border-radius: var(--r-s);
  font-size: 13px;
  color: var(--c-ink-2);
}
</style>
