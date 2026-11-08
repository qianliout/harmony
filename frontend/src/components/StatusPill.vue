<script setup lang="ts">
import { computed } from 'vue'

const props = withDefaults(
  defineProps<{
    status: 'pending' | 'repaid' | 'none' | string
    direction?: 'in' | 'out' | string
    isRepay?: boolean
    label?: string
    clickable?: boolean
  }>(),
  { direction: 'in', isRepay: false, label: '', clickable: false },
)

const text = computed(() => {
  if (props.isRepay) return '回礼'
  if (props.label) return props.label
  if (props.status === 'repaid') return props.direction === 'out' ? '对方已回' : '已回礼'
  if (props.status === 'none') return '不用回'
  return props.direction === 'out' ? '对方未回' : '未回礼'
})

const tone = computed(() => {
  if (props.isRepay) return 'repay'
  return props.status
})
</script>

<template>
  <view class="pill" :class="[`pill--${tone}`, { 'pill--clickable': clickable && !isRepay }]">
    <text v-if="!isRepay && status === 'repaid'" class="pill__mark">✓</text>
    <view v-else-if="!isRepay && status === 'pending'" class="pill__dot" />
    <text>{{ text }}</text>
  </view>
</template>

<style lang="scss" scoped>
.pill {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  height: 20px;
  padding: 0 8px;
  border-radius: var(--r-full);
  font-size: 12px;
  line-height: 1;
  font-weight: 500;
  flex: none;
}

.pill--pending {
  background: var(--c-pending-light);
  color: var(--c-pending);
}

.pill--repaid {
  background: var(--c-repaid-light);
  color: var(--c-repaid);
}

.pill--none {
  background: var(--c-none-light);
  color: var(--c-none);
}

.pill--repay {
  background: var(--c-out-light);
  color: var(--c-out);
}

.pill__dot {
  width: 6px;
  height: 6px;
  border-radius: var(--r-full);
  background: currentColor;
}

.pill__mark {
  font-size: 10px;
}
</style>
