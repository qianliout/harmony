<script setup lang="ts">
import { computed } from 'vue'

import type { RecordItem, TimelineGroup } from '@/api/types'
import { formatGroupTitle, preciseDateLabel } from '@/utils/date'

import DirTag from './DirTag.vue'
import EventSeal from './EventSeal.vue'
import Money from './Money.vue'
import StatusPill from './StatusPill.vue'

const props = withDefaults(
  defineProps<{
    group: TimelineGroup
    selectable?: boolean
    selectedIds?: number[]
  }>(),
  { selectable: false, selectedIds: () => [] },
)

const emit = defineEmits<{
  toggle: [record: RecordItem]
  selectAll: []
  record: [record: RecordItem]
}>()

const dateLabel = computed(() => {
  if (props.group.event.eventDate) {
    return formatGroupTitle(props.group.event.eventDate)
  }
  const first = props.group.records[0]
  return first ? formatGroupTitle(first.occurredOn) : ''
})

const hostLabel = computed(() => (props.group.event.host === 'self' ? '自家办' : '他家办'))
const hasPending = computed(() => props.group.records.some((record) => record.status === 'pending'))
const selectableRecords = computed(() =>
  props.group.records.filter((record) => record.direction === 'in' && record.status === 'pending'),
)
const formula = computed(() => {
  if (props.group.records.length <= 1 || props.group.parts.length === 0) {
    return ''
  }
  const parts = props.group.parts.map((part) => `${part.label} ¥${(part.cent / 100).toLocaleString('en-US')}`)
  const total = props.group.parts.reduce((sum, part) => sum + part.cent, 0)
  return `${parts.join(' + ')} = ¥${(total / 100).toLocaleString('en-US')}`
})

function isDisabled(record: RecordItem): boolean {
  return record.direction !== 'in' || record.status !== 'pending'
}

function onRow(record: RecordItem): void {
  if (props.selectable) {
    if (!isDisabled(record)) {
      emit('toggle', record)
    }
    return
  }
  emit('record', record)
}

function dateOf(record: RecordItem): string {
  return preciseDateLabel(record.occurredOn, record.datePrecision)
}
</script>

<template>
  <view class="group card">
    <view class="group__head">
      <EventSeal :type="group.event.type" size="m" />
      <view class="group__title-wrap">
        <view class="group__title">
          <text class="group__name">{{ group.event.name }}</text>
          <text class="group__host">{{ hostLabel }}</text>
          <view v-if="hasPending" class="group__dot" />
        </view>
        <text class="group__date">{{ dateLabel }}</text>
      </view>
      <view class="group__totals">
        <Money v-if="group.inCent" :cent="group.inCent" direction="in" size="m" />
        <Money v-if="group.outCent" :cent="group.outCent" direction="out" size="m" />
      </view>
    </view>

    <view v-if="selectable && selectableRecords.length" class="group__select" @click="emit('selectAll')">
      全选本组
    </view>

    <view
      v-for="record in group.records"
      :key="record.id"
      class="line"
      :class="{ 'line--pending': record.status === 'pending', 'line--disabled': selectable && isDisabled(record) }"
      @click="onRow(record)"
    >
      <view v-if="record.status === 'pending'" class="line__bar" />
      <view v-if="selectable" class="line__check" :class="{ 'line__check--on': selectedIds.includes(record.id) }">
        <text v-if="selectedIds.includes(record.id)">✓</text>
      </view>
      <DirTag :direction="record.direction" />
      <text class="line__book">{{ record.itemKindLabel }} · {{ record.bookName }}</text>
      <Money :cent="record.totalCent" :direction="record.direction" size="s" />
      <StatusPill v-if="!selectable" :status="record.status" />
      <text class="line__date">{{ dateOf(record) }}</text>
    </view>

    <view v-if="formula" class="formula">{{ formula }}</view>
  </view>
</template>

<style lang="scss" scoped>
.group {
  margin-bottom: 12px;
  overflow: hidden;
}

.group__head {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 14px 16px 12px;
}

.group__title-wrap {
  flex: 1;
  min-width: 0;
}

.group__title {
  display: flex;
  align-items: center;
  gap: 6px;
}

.group__name {
  font-size: 16px;
  line-height: 22px;
  font-weight: 600;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.group__host {
  flex: none;
  font-size: 12px;
  color: var(--c-ink-3);
}

.group__dot {
  width: 6px;
  height: 6px;
  border-radius: var(--r-full);
  background: var(--c-pending);
}

.group__date {
  display: block;
  margin-top: 2px;
  font-size: 13px;
  color: var(--c-ink-3);
}

.group__totals {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  gap: 2px;
  flex: none;
}

.group__select {
  padding: 0 16px 8px;
  font-size: 12px;
  color: var(--c-primary);
  text-align: right;
}

.line {
  position: relative;
  display: flex;
  align-items: center;
  gap: 8px;
  min-height: 44px;
  padding: 0 16px;
  box-shadow: inset 0 1px 0 var(--c-divider);
}

.line--disabled {
  opacity: 0.45;
}

.line__bar {
  position: absolute;
  left: 0;
  top: 10px;
  bottom: 10px;
  width: 3px;
  border-radius: 2px;
  background: var(--c-pending);
}

.line__check {
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

.line__check--on {
  background: var(--c-primary);
  border-color: var(--c-primary);
}

.line__book {
  flex: 1;
  min-width: 0;
  font-size: 14px;
  color: var(--c-ink-2);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.line__date {
  flex: none;
  font-size: 11px;
  color: var(--c-ink-3);
}

.formula {
  padding: 10px 16px 12px;
  background: #fcfaf7;
  box-shadow: inset 0 1px 0 var(--c-divider);
  font-family: $font-num;
  font-size: 13px;
  color: var(--c-ink-2);
}
</style>
