<script setup lang="ts">
defineProps<{ direction: 'in' | 'out' | string }>()
</script>

<template>
  <text class="tag" :class="direction === 'in' ? 'tag--in' : 'tag--out'">
    {{ direction === 'in' ? '收' : '送' }}
  </text>
</template>

<style lang="scss" scoped>
.tag {
  display: inline-flex;
  align-items: center;
  height: 20px;
  padding: 0 6px;
  border-radius: var(--r-xs);
  font-size: 12px;
  line-height: 1;
  font-weight: 500;
  flex: none;
}

.tag--in {
  background: var(--c-in-light);
  color: var(--c-in);
}

.tag--out {
  background: var(--c-out-light);
  color: var(--c-out);
}
</style>
