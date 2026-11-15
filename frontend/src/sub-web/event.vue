<script setup lang="ts">
import WebShell from '@/web/shell/WebShell.vue'
import EventView from '@/web/views/EventView.vue'
</script>

<template>
  <WebShell nav="events" title="事项详情" crumbs="事项">
    <EventView />
  </WebShell>
</template>
