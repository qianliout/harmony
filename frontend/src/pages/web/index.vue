<script setup lang="ts">
// H5 首页即总账：所有礼簿合并后的全部收送记录。
import WebShell from '@/web/shell/WebShell.vue'
import LedgerView from '@/web/views/LedgerView.vue'
</script>

<template>
  <WebShell nav="ledger" title="总账">
    <LedgerView />
  </WebShell>
</template>
