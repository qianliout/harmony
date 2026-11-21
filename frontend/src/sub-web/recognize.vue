<script setup lang="ts">
// 识别只在 H5 提供：小程序产物里不引入识别视图与接口。
import WebShell from '@/web/shell/WebShell.vue'
// #ifdef H5
import RecognizeView from '@/web/views/RecognizeView.vue'
// #endif
</script>

<template>
  <WebShell nav="recognize" title="识别礼簿" crumbs="识别礼簿">
    <!-- #ifdef H5 -->
    <RecognizeView />
    <!-- #endif -->
  </WebShell>
</template>
