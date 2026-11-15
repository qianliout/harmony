<script setup lang="ts">
import WebIcon from './WebIcon.vue'

defineProps<{ title: string }>()
const emit = defineEmits<{ close: [] }>()
</script>

<template>
  <div class="web-dialog" role="dialog" aria-modal="true">
    <div class="web-drawer__head">
      <h3>{{ title }}</h3>
      <button class="web-btn web-btn--text" type="button" aria-label="关闭" @click="emit('close')">
        <WebIcon name="x" />
      </button>
    </div>
    <div class="web-drawer__body">
      <slot />
    </div>
    <div v-if="$slots.footer" class="web-drawer__foot">
      <slot name="footer" />
    </div>
  </div>
</template>
