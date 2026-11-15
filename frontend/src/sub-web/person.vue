<script setup lang="ts">
import WebShell from '@/web/shell/WebShell.vue'
import PersonView from '@/web/views/PersonView.vue'
</script>

<template>
  <WebShell nav="people" title="亲友详情" crumbs="亲友">
    <PersonView />
  </WebShell>
</template>
