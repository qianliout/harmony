<script setup lang="ts">
withDefaults(defineProps<{ title?: string; linkText?: string }>(), { title: '', linkText: '' })
defineEmits<{ link: [] }>()
</script>

<template>
  <view class="section">
    <view v-if="title" class="section__head">
      <text class="section__title">{{ title }}</text>
      <text v-if="linkText" class="section__link" @click="$emit('link')">{{ linkText }} ›</text>
    </view>
    <view class="section__body card">
      <slot />
    </view>
  </view>
</template>

<style lang="scss" scoped>
.section {
  margin-top: 20px;
}

.section__head {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  margin: 0 2px 10px;
}

.section__title {
  font-size: 17px;
  line-height: 24px;
  font-weight: 600;
  color: var(--c-ink-1);
}

.section__link {
  font-size: 13px;
  color: var(--c-ink-3);
}

.section__body {
  overflow: hidden;
}
</style>
