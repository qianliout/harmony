<script setup lang="ts">
withDefaults(defineProps<{ title: string; meta?: string }>(), { meta: '' })
</script>

<template>
  <div class="web-page-head">
    <div>
      <h1>{{ title }}</h1>
      <div v-if="meta" class="web-page-head__meta">{{ meta }}</div>
      <slot name="meta" />
    </div>
    <div v-if="$slots.tools" class="web-page-head__tools">
      <slot name="tools" />
    </div>
  </div>
</template>
