<script setup lang="ts">
import WebShell from '@/web/shell/WebShell.vue'
import BookView from '@/web/views/BookView.vue'
</script>

<template>
  <WebShell nav="books" title="礼簿详情" crumbs="礼簿">
    <BookView />
  </WebShell>
</template>
