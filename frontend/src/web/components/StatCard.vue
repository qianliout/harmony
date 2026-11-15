<script setup lang="ts">
withDefaults(defineProps<{ value: string; label: string; tone?: 'in' | 'out' | 'pending' | '' }>(), {
  tone: '',
})
</script>

<template>
  <div class="web-card web-stat">
    <div class="web-stat__v" :class="tone ? `web-amt--${tone}` : ''">{{ value }}</div>
    <div class="web-stat__k">{{ label }}</div>
  </div>
</template>
