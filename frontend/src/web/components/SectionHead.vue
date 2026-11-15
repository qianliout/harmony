<script setup lang="ts">
withDefaults(defineProps<{ title: string; sub?: string }>(), { sub: '' })
</script>

<template>
  <div class="web-sec-head">
    <h2>{{ title }}</h2>
    <span v-if="sub" class="web-sec-head__sub">{{ sub }}</span>
    <div v-if="$slots.tools" class="web-sec-head__tools">
      <slot name="tools" />
    </div>
  </div>
</template>
