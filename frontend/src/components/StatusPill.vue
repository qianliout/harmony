<script setup lang="ts">
import { computed } from 'vue'

const props = withDefaults(
  defineProps<{
    status: 'pending' | 'repaid' | 'none' | string
    clickable?: boolean
  }>(),
  { clickable: false },
)

const label = computed(() => {
  if (props.status === 'repaid') return '已还'
  if (props.status === 'none') return '无需还'
  return '待还'
})
</script>

<template>
  <view class="pill" :class="[`pill--${status}`, { 'pill--clickable': clickable }]">
    <text v-if="status === 'repaid'" class="pill__mark">✓</text>
    <view v-else-if="status === 'pending'" class="pill__dot" />
    <text>{{ label }}</text>
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
