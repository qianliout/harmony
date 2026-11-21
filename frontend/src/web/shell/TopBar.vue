<script setup lang="ts">
import { useWebUiStore } from '@/stores/web'
import WebIcon from '@/web/components/WebIcon.vue'

defineProps<{ title: string; crumbs?: string }>()
const ui = useWebUiStore()
</script>

<template>
  <header class="web-top">
    <div class="web-crumbs">
      <template v-if="crumbs">
        <span class="web-crumbs__link">{{ crumbs }}</span>
        <WebIcon name="right" :size="14" />
      </template>
      <b class="web-crumbs__title">{{ title }}</b>
    </div>
    <button class="web-btn web-btn--primary" type="button" title="快捷键 N" @click="ui.openEntry()">
      <WebIcon name="plus" />
      记一笔
    </button>
  </header>
</template>
