<script setup lang="ts">
// H5 桌面工作台入口；小程序首页仍是 pages/index/index，两套页面各自独立。
import WebShell from '@/web/shell/WebShell.vue'
import HomeView from '@/web/views/HomeView.vue'
</script>

<template>
  <WebShell nav="home" title="工作台">
    <HomeView />
  </WebShell>
</template>
