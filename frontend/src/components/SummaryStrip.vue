<script setup lang="ts">
import { computed } from 'vue'

import Money from './Money.vue'

const props = withDefaults(
  defineProps<{
    inCent: number
    outCent: number
    netCent?: number
    pendingCount: number
    pendingCent?: number
    showPendingAmount?: boolean
  }>(),
  { netCent: undefined, pendingCent: 0, showPendingAmount: false },
)

const emit = defineEmits<{ pending: [] }>()

const net = computed(() => props.netCent ?? props.inCent - props.outCent)
const netHint = computed(() => {
  if (net.value > 0) return '收多送少'
  if (net.value < 0) return '送多收少'
  return ''
})
</script>

<template>
  <view class="strip card">
    <view class="strip__cell">
      <text class="strip__label">总收</text>
      <Money :cent="inCent" direction="in" size="l" />
    </view>
    <view class="strip__cell">
      <text class="strip__label">总送</text>
      <Money :cent="outCent" direction="out" size="l" />
    </view>
    <view class="strip__cell">
      <text class="strip__label">差额</text>
      <Money :cent="net" direction="none" size="l" sign />
      <text v-if="netHint" class="strip__hint">{{ netHint }}</text>
    </view>
    <view class="strip__cell" @click="emit('pending')">
      <text class="strip__label">待还</text>
      <text v-if="showPendingAmount" class="strip__value strip__value--pending">
        <Money :cent="pendingCent" size="l" tone="pending" />
      </text>
      <text v-else class="strip__value strip__value--pending">{{ pendingCount }} <text class="strip__unit">笔</text></text>
    </view>
  </view>
</template>

<style lang="scss" scoped>
.strip {
  display: flex;
  padding: 14px 0;
}

.strip__cell {
  flex: 1;
  min-width: 0;
  text-align: center;
  position: relative;
}

.strip__cell + .strip__cell::before {
  content: '';
  position: absolute;
  left: 0;
  top: 4px;
  bottom: 4px;
  width: 1px;
  background: var(--c-divider);
}

.strip__label {
  display: block;
  margin-bottom: 4px;
  font-size: 12px;
  color: var(--c-ink-3);
}

.strip__value {
  display: inline-flex;
  align-items: baseline;
  gap: 2px;
  font-family: $font-num;
  font-size: 22px;
  line-height: 28px;
  font-weight: 600;
}

.strip__value--pending {
  color: var(--c-pending);
}

.strip__unit {
  font-size: 12px;
  font-family: $font-sans;
}

.strip__hint {
  display: block;
  margin-top: 2px;
  font-size: 12px;
  color: var(--c-ink-3);
}
</style>
