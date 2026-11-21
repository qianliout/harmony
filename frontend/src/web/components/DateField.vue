<script setup lang="ts">
// 日期选择：uni-app H5 会把 <input type="date"> 编译成文本框，改用 uni 的 picker（mode=date）。
const props = withDefaults(defineProps<{ modelValue: string; placeholder?: string; clearable?: boolean }>(), {
  placeholder: '选择日期',
  clearable: false,
})
const emit = defineEmits<{ 'update:modelValue': [value: string] }>()

function onChange(event: { detail: { value: string } }): void {
  emit('update:modelValue', event.detail.value)
}
</script>

<template>
  <picker mode="date" :value="props.modelValue" class="date-field" @change="onChange">
    <div class="web-input date-field__box">
      <span :class="{ 'web-muted': !props.modelValue }">{{ props.modelValue || props.placeholder }}</span>
      <span
        v-if="props.clearable && props.modelValue"
        class="date-field__clear web-muted"
        role="button"
        aria-label="清除日期"
        @click.stop="emit('update:modelValue', '')"
      >×</span>
    </div>
  </picker>
</template>

<style lang="scss" scoped>
.date-field__box {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--sp-2);
  min-width: 128px;
  cursor: pointer;
}

.date-field__clear {
  padding: 0 var(--sp-1);
}
</style>
