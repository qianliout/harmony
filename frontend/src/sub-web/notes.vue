<script setup lang="ts">
import WebShell from '@/web/shell/WebShell.vue'
import NotesView from '@/web/views/NotesView.vue'
</script>

<template>
  <WebShell nav="notes" title="整理纸本标注" crumbs="整理纸本标注">
    <NotesView />
  </WebShell>
</template>
