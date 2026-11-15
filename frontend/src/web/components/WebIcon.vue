<script setup lang="ts">
import { computed } from 'vue'

import { iconMarkup } from '@/web/icons'

const props = withDefaults(defineProps<{ name: string; size?: number }>(), { size: 18 })

const markup = computed(() => iconMarkup(props.name))
</script>

<template>
  <svg
    class="web-icon"
    :width="size"
    :height="size"
    viewBox="0 0 24 24"
    fill="none"
    stroke="currentColor"
    stroke-width="1.6"
    stroke-linecap="round"
    stroke-linejoin="round"
    aria-hidden="true"
    v-html="markup"
  />
</template>

<style scoped>
.web-icon {
  display: block;
  flex: none;
}
</style>
