<script setup lang="ts">
withDefaults(
  defineProps<{
    title?: string
    description?: string
    actionText?: string
  }>(),
  { title: '还没有记录', description: '记下第一笔人情往来吧', actionText: '' },
)

defineEmits<{ action: [] }>()
</script>

<template>
  <view class="empty">
    <view class="empty__seal"><text>簿</text></view>
    <text class="empty__title">{{ title }}</text>
    <text v-if="description" class="empty__desc">{{ description }}</text>
    <view v-if="actionText" class="btn btn--primary empty__action" @click="$emit('action')">
      <text>{{ actionText }}</text>
    </view>
  </view>
</template>

<style lang="scss" scoped>
.empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 48px 24px;
}

.empty__seal {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 64px;
  height: 64px;
  border: 1px solid var(--c-none);
  border-radius: 16px;
  color: var(--c-none);
  font-family: $font-serif;
  font-size: 32px;
  opacity: 0.6;
  position: relative;
}

.empty__seal::after {
  content: '';
  position: absolute;
  inset: 2px;
  border: 1px solid currentColor;
  border-radius: inherit;
  opacity: 0.3;
}

.empty__title {
  margin-top: 16px;
  font-size: 17px;
  line-height: 24px;
  font-weight: 600;
  color: var(--c-ink-2);
}

.empty__desc {
  margin-top: 6px;
  font-size: 13px;
  color: var(--c-ink-3);
}

.empty__action {
  margin-top: 20px;
  padding: 0 28px;
}
</style>
