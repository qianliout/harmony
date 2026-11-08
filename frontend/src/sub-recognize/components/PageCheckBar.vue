<script setup lang="ts">
// 本页核对条：纸本合计与本页已选合计对比，并给出待确认行的定位入口。
import { computed } from 'vue'

import { formatCent } from '@/utils/money'

import type { ReviewState } from '../useReviewState'

const props = withDefaults(
  defineProps<{
    review: ReviewState
    pageTotalCent: number
    pageConsistent: boolean
    readonly?: boolean
  }>(),
  { readonly: false },
)

defineEmits<{ locate: [] }>()

const diffCent = computed(() => props.review.selectedSumCent.value - props.pageTotalCent)
const pendingCount = computed(() => (props.readonly ? 0 : props.review.pendingRows.value.length))
const personCount = computed(() => (props.readonly ? 0 : props.review.needPersonRows.value.length))
</script>

<template>
  <view
    class="checkbar"
    :class="{
      'checkbar--ok': pageTotalCent > 0 && pageConsistent,
      'checkbar--warn': (pageTotalCent > 0 && !pageConsistent) || pendingCount > 0,
    }"
  >
    <view class="checkbar__main">
      <text v-if="pageTotalCent > 0" class="checkbar__line">
        纸本合计 {{ formatCent(pageTotalCent) }} · 已选 {{ formatCent(review.selectedSumCent.value) }}
        <text v-if="pageConsistent"> ✓</text>
        <text v-else> · {{ diffCent < 0 ? '少' : '多' }} {{ formatCent(Math.abs(diffCent)) }}</text>
      </text>
      <text v-else class="checkbar__line">未填本页合计 · 已选 {{ formatCent(review.selectedSumCent.value) }}</text>
      <text v-if="pageTotalCent > 0 && !pageConsistent" class="checkbar__reason">
        可能漏识别了一行，或有金额识别错误
      </text>
      <text v-if="pendingCount > 0" class="checkbar__reason">待确认的行不能写入，点右侧定位</text>
    </view>
    <text v-if="pendingCount > 0" class="checkbar__pending" @click="$emit('locate')">
      还有 {{ pendingCount }} 行待确认 ›
    </text>
    <text v-else-if="personCount > 0" class="checkbar__pending" @click="$emit('locate')">
      还有 {{ personCount }} 条需要选择亲友 ›
    </text>
  </view>
</template>

<style lang="scss" scoped>
.checkbar {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 9px 12px;
  border-radius: var(--r-s);
  background: var(--c-fill);
  font-size: 12px;
  color: var(--c-ink-2);
}

.checkbar--ok {
  background: var(--c-repaid-light);
  color: #2f5e35;
}

.checkbar--warn {
  background: var(--c-pending-light);
  color: #7a4e07;
}

.checkbar__main {
  flex: 1;
  min-width: 0;
}

.checkbar__line {
  display: block;
}

.checkbar__reason {
  display: block;
  margin-top: 2px;
  font-size: 11px;
  opacity: 0.86;
}

.checkbar__pending {
  flex: none;
  font-weight: 600;
  white-space: nowrap;
  cursor: pointer;
}
</style>
