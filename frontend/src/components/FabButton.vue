<script setup lang="ts">
withDefaults(defineProps<{ text?: string }>(), { text: '+' })
defineEmits<{ click: [] }>()
</script>

<template>
  <view class="fab" @click="$emit('click')">
    <text class="fab__text">{{ text }}</text>
  </view>
</template>

<style lang="scss" scoped>
.fab {
  position: fixed;
  right: calc(50% - 215px + 16px);
  bottom: calc(80px + env(safe-area-inset-bottom));
  z-index: 20;
  display: flex;
  align-items: center;
  justify-content: center;
  width: 56px;
  height: 56px;
  border-radius: var(--r-full);
  background: var(--c-primary);
  box-shadow: var(--shadow-float);
}

@media (max-width: 520px) {
  .fab {
    right: 16px;
  }
}

.fab__text {
  font-size: 28px;
  line-height: 1;
  color: var(--c-surface);
  font-weight: 300;
}
</style>
