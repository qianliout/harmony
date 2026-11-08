<script setup lang="ts">
withDefaults(
  defineProps<{
    title: string
    value?: string
    arrow?: boolean
    danger?: boolean
  }>(),
  { value: '', arrow: true, danger: false },
)

defineEmits<{ click: [] }>()
</script>

<template>
  <view class="cell" @click="$emit('click')">
    <text class="cell__title" :class="{ 'cell__title--danger': danger }">{{ title }}</text>
    <text v-if="value" class="cell__value">{{ value }}</text>
    <text v-if="arrow" class="cell__arrow">›</text>
  </view>
</template>

<style lang="scss" scoped>
.cell {
  display: flex;
  align-items: center;
  gap: 8px;
  min-height: 52px;
  padding: 0 16px;
  background: var(--c-surface);
}

.cell + .cell {
  box-shadow: inset 0 1px 0 var(--c-divider);
}

.cell__title {
  flex: 1;
  font-size: 15px;
  color: var(--c-ink-1);
}

.cell__title--danger {
  color: var(--c-danger);
}

.cell__value {
  font-size: 14px;
  color: var(--c-ink-3);
}

.cell__arrow {
  font-size: 18px;
  color: var(--c-ink-4);
}
</style>
