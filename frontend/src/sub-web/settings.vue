<script setup lang="ts">
import WebShell from '@/web/shell/WebShell.vue'
import SettingsView from '@/web/views/SettingsView.vue'
</script>

<template>
  <WebShell nav="settings" title="设置" crumbs="设置">
    <SettingsView />
  </WebShell>
</template>
