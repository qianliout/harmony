<script setup lang="ts">
export interface SegmentOption {
  value: string
  label: string
  count?: number
}

withDefaults(defineProps<{ modelValue: string; options: SegmentOption[]; variant?: string }>(), {
  variant: '',
})

const emit = defineEmits<{ 'update:modelValue': [value: string] }>()
</script>

<template>
  <div class="web-seg" :class="variant ? `web-seg--${variant}` : ''">
    <button
      v-for="option in options"
      :key="option.value"
      type="button"
      :data-v="option.value"
      :class="{ 'web-seg__on': modelValue === option.value }"
      @click="emit('update:modelValue', option.value)"
    >
      {{ option.label }}
      <span v-if="option.count !== undefined" class="web-num web-muted"> {{ option.count }}</span>
    </button>
  </div>
</template>
