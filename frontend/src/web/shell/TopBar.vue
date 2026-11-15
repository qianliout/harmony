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
    <button class="web-search-trigger" type="button" aria-label="搜索" @click="ui.openSearch()">
      <WebIcon name="search" />
      <span>搜姓名、金额、年份、页码</span>
      <kbd class="web-kbd">⌘K</kbd>
    </button>
    <button class="web-btn web-btn--text" type="button" title="本页交互说明" @click="ui.openNotes()">
      <WebIcon name="info" />
      <span class="web-top__label">交互说明</span>
    </button>
    <button class="web-btn web-btn--primary" type="button" @click="ui.openEntry()">
      <WebIcon name="plus" />
      记一笔
    </button>
  </header>
</template>
