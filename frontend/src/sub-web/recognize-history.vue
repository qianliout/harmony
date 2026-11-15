<script setup lang="ts">
import WebShell from '@/web/shell/WebShell.vue'
import RecognizeHistoryView from '@/web/views/RecognizeHistoryView.vue'
</script>

<template>
  <WebShell nav="recognize" title="识别历史" crumbs="识别礼簿">
    <RecognizeHistoryView />
  </WebShell>
</template>
