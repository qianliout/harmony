<script setup lang="ts">
import WebShell from '@/web/shell/WebShell.vue'
import PeopleView from '@/web/views/PeopleView.vue'
</script>

<template>
  <WebShell nav="people" title="亲友" crumbs="亲友">
    <PeopleView />
  </WebShell>
</template>
