<script setup lang="ts">
import WebShell from '@/web/shell/WebShell.vue'
import RecordsView from '@/web/views/RecordsView.vue'
</script>

<template>
  <WebShell nav="records" title="记录" crumbs="全部记录">
    <RecordsView />
  </WebShell>
</template>
