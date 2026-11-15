<script setup lang="ts">
import WebShell from '@/web/shell/WebShell.vue'
import BooksView from '@/web/views/BooksView.vue'
</script>

<template>
  <WebShell nav="books" title="礼簿" crumbs="礼簿">
    <BooksView />
  </WebShell>
</template>
