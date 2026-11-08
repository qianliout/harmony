<script setup lang="ts">
import { computed } from 'vue'

const props = withDefaults(
  defineProps<{
    quick: string
    count?: number
    options?: { value: string; label: string }[]
  }>(),
  {
    count: 0,
    options: () => [
      { value: 'all', label: '全部' },
      { value: 'pending', label: '未回礼' },
      { value: 'in', label: '收' },
      { value: 'out', label: '送' },
    ],
  },
)

defineEmits<{
  'update:quick': [value: string]
  open: []
}>()

const items = computed(() => props.options)
</script>

<template>
  <view class="bar">
    <view class="bar__chips">
      <text
        v-for="option in items"
        :key="option.value"
        class="chip chip--flat"
        :class="{ 'chip--flat-on': quick === option.value }"
        @click="$emit('update:quick', option.value)"
      >
        {{ option.label }}
      </text>
    </view>
    <text class="bar__filter" @click="$emit('open')">
      筛选<text v-if="count" class="bar__badge">{{ count }}</text>
    </text>
  </view>
</template>

<style lang="scss" scoped>
.bar {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 0;
}

.bar__chips {
  display: flex;
  gap: 8px;
  flex: 1;
  overflow-x: auto;
}

.bar__chips .chip {
  flex: none;
}

.bar__filter {
  flex: none;
  font-size: 13px;
  color: var(--c-ink-2);
}

.bar__badge {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 16px;
  height: 16px;
  margin-left: 3px;
  padding: 0 4px;
  border-radius: var(--r-full);
  background: var(--c-primary);
  color: var(--c-surface);
  font-size: 10px;
}
</style>
