<script setup lang="ts">
import PersonAvatar from '@/components/PersonAvatar.vue'

withDefaults(defineProps<{ id: number; name: string; sub?: string; size?: number }>(), {
  sub: '',
  size: 28,
})

const emit = defineEmits<{ open: [] }>()
</script>

<template>
  <button type="button" class="web-cell2" style="border: 0; background: none; padding: 0; text-align: left" @click="emit('open')">
    <PersonAvatar :id="id" :name="name" :size="size" />
    <span class="web-cell2__t">
      {{ name }}
      <small v-if="sub">{{ sub }}</small>
    </span>
  </button>
</template>
