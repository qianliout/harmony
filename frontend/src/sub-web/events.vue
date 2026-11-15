<script setup lang="ts">
import WebShell from '@/web/shell/WebShell.vue'
import EventsView from '@/web/views/EventsView.vue'
</script>

<template>
  <WebShell nav="events" title="事项" crumbs="事项">
    <EventsView />
  </WebShell>
</template>
