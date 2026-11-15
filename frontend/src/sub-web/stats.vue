<script setup lang="ts">
import WebShell from '@/web/shell/WebShell.vue'
import StatsView from '@/web/views/StatsView.vue'
</script>

<template>
  <WebShell nav="stats" title="统计" crumbs="统计">
    <StatsView />
  </WebShell>
</template>
