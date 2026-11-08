<script setup lang="ts">
import { computed } from 'vue'

import type { RecordItem } from '@/api/types'
import { preciseDateLabel } from '@/utils/date'

import DirTag from './DirTag.vue'
import Money from './Money.vue'
import PersonAvatar from './PersonAvatar.vue'
import StatusPill from './StatusPill.vue'

const props = withDefaults(
  defineProps<{
    record: RecordItem
    mode?: 'global' | 'book' | 'timeline'
    selectable?: boolean
    selected?: boolean
    disabled?: boolean
  }>(),
  { mode: 'global', selectable: false, selected: false, disabled: false },
)

const emit = defineEmits<{
  tap: [record: RecordItem]
  toggle: [record: RecordItem]
}>()

const dateLabel = computed(() => preciseDateLabel(props.record.occurredOn, props.record.datePrecision))
const bookLabel = computed(() => {
  if (!props.record.originBookName) {
    return props.record.bookName
  }
  return `${props.record.bookName}（原：${props.record.originBookName}）`
})

const meta = computed(() => {
  if (props.mode === 'book') {
    return `${props.record.itemKindLabel} · ${dateLabel.value}`
  }
  if (props.mode === 'timeline') {
    return `${props.record.itemKindLabel} · ${bookLabel.value}`
  }
  return `${bookLabel.value} · ${props.record.itemKindLabel} · ${dateLabel.value}`
})

const RELATION_LABELS: Record<string, string> = {
  relative: '亲戚',
  friend: '朋友',
  colleague: '同事',
  neighbor: '邻居',
  other: '其他',
}

const relationLabel = computed(() => RELATION_LABELS[props.record.personRelation] ?? '其他')
const isPending = computed(
  () => props.record.direction === 'in' && props.record.status === 'pending' && !props.record.isRepay,
)
const isRepay = computed(() => Boolean(props.record.isRepay))
const giftOnly = computed(() => props.record.amountCent === 0 && Boolean(props.record.giftName))
const hasGift = computed(() => props.record.giftValueCent > 0)

function onClick(): void {
  if (props.selectable) {
    if (!props.disabled) {
      emit('toggle', props.record)
    }
    return
  }
  emit('tap', props.record)
}
</script>

<template>
  <view class="row" :class="{ 'row--pending': isPending, 'row--disabled': selectable && disabled }" @click="onClick">
    <view v-if="isPending" class="row__bar" />
    <view v-if="selectable" class="row__check" :class="{ 'row__check--on': selected }">
      <text v-if="selected">✓</text>
    </view>
    <PersonAvatar v-if="mode !== 'timeline'" :id="record.personId" :name="record.personName" :size="40" />
    <view class="row__main">
      <view v-if="mode !== 'timeline'" class="row__line">
        <text class="row__name">{{ record.personName }}</text>
        <text class="row__relation">{{ relationLabel }}</text>
        <DirTag v-if="mode === 'global'" :direction="record.direction" />
      </view>
      <view v-else class="row__line">
        <DirTag :direction="record.direction" />
        <text class="row__meta">{{ meta }}</text>
      </view>
      <text v-if="mode !== 'timeline'" class="row__meta">{{ meta }}</text>
      <view v-if="record.paperNote" class="row__note">
        <text class="row__zhu">注</text>
        <text class="row__note-text">{{ record.paperNote }}</text>
      </view>
      <text v-if="record.paperPage" class="row__page">· 第 {{ record.paperPage }} 页</text>
    </view>
    <view class="row__right">
      <Money
        v-if="!giftOnly"
        :cent="record.totalCent"
        :direction="record.direction"
        size="m"
        :tone="isPending ? 'pending' : undefined"
      />
      <text v-else class="row__gift">{{ record.giftName }}</text>
      <text v-if="giftOnly && record.giftValueCent" class="row__gift-value">估 ¥{{ (record.giftValueCent / 100).toFixed(0) }}</text>
      <StatusPill
        :status="record.status"
        :direction="record.direction"
        :is-repay="isRepay"
        :label="record.statusLabel"
      />
      <text v-if="hasGift && !giftOnly" class="row__gift-hint">含礼品</text>
    </view>
  </view>
</template>

<style lang="scss" scoped>
.row {
  position: relative;
  display: flex;
  align-items: flex-start;
  gap: 12px;
  min-height: var(--row-h);
  padding: 12px 16px;
  background: var(--c-surface);
}

.row + .row {
  box-shadow: inset 0 1px 0 var(--c-divider);
}

.row--disabled {
  opacity: 0.45;
}

.row__bar {
  position: absolute;
  left: 0;
  top: 12px;
  bottom: 12px;
  width: 3px;
  border-radius: 2px;
  background: var(--c-pending);
}

.row__check {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 20px;
  height: 20px;
  margin-top: 10px;
  border: 1.5px solid var(--c-ink-4);
  border-radius: var(--r-xs);
  color: var(--c-surface);
  font-size: 12px;
  flex: none;
}

.row__check--on {
  background: var(--c-primary);
  border-color: var(--c-primary);
}

.row__main {
  flex: 1;
  min-width: 0;
}

.row__line {
  display: flex;
  align-items: center;
  gap: 6px;
}

.row__name {
  font-size: 15px;
  line-height: 22px;
  font-weight: 500;
  color: var(--c-ink-1);
}

.row__relation {
  height: 20px;
  padding: 0 6px;
  border-radius: var(--r-xs);
  background: var(--c-fill);
  color: var(--c-ink-2);
  font-size: 12px;
  line-height: 20px;
}

.row__meta {
  display: block;
  margin-top: 2px;
  font-size: 13px;
  line-height: 18px;
  color: var(--c-ink-3);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.row__note {
  display: flex;
  align-items: center;
  gap: 5px;
  margin-top: 4px;
}

.row__zhu {
  width: 16px;
  height: 16px;
  border-radius: 4px;
  background: var(--c-none-light);
  color: var(--c-none);
  font-size: 10px;
  font-weight: 600;
  line-height: 16px;
  text-align: center;
  flex: none;
}

.row__note-text {
  font-family: $font-serif;
  font-size: 13px;
  color: var(--c-ink-2);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.row__page {
  font-size: 12px;
  color: var(--c-ink-3);
}

.row__right {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  gap: 4px;
  flex: none;
}

.row__gift {
  max-width: 120px;
  font-size: 15px;
  font-weight: 500;
  color: var(--c-ink-1);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.row__gift-value,
.row__gift-hint {
  font-size: 12px;
  color: var(--c-ink-3);
}
</style>
