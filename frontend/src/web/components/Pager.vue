<script setup lang="ts">
import WebIcon from './WebIcon.vue'

defineProps<{ page: number; pages: number }>()
const emit = defineEmits<{ change: [page: number] }>()
</script>

<template>
  <div class="web-pager">
    <button
      class="web-btn web-btn--text web-btn--sm"
      type="button"
      :disabled="page <= 1"
      aria-label="上一页"
      @click="emit('change', page - 1)"
    >
      <WebIcon name="left" />
    </button>
    <span class="web-num">{{ page }} / {{ pages }}</span>
    <button
      class="web-btn web-btn--text web-btn--sm"
      type="button"
      :disabled="page >= pages"
      aria-label="下一页"
      @click="emit('change', page + 1)"
    >
      <WebIcon name="right" />
    </button>
  </div>
</template>
