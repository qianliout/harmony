<script setup lang="ts">
withDefaults(defineProps<{ type?: 'list' | 'timeline' }>(), { type: 'list' })
</script>

<template>
  <view class="skeleton">
    <view v-for="item in type === 'timeline' ? 2 : 4" :key="item" class="skeleton__card">
      <view class="skeleton__bar" :class="{ 'skeleton__bar--head': type === 'timeline' }" />
      <view class="skeleton__bar skeleton__bar--short" />
    </view>
  </view>
</template>

<style lang="scss" scoped>
.skeleton {
  padding: 12px var(--gutter);
}

.skeleton__card {
  padding: 14px 16px;
  margin-bottom: 12px;
  border-radius: var(--r-m);
  background: var(--c-surface);
  box-shadow: var(--shadow-card);
}

.skeleton__bar {
  height: 14px;
  width: 100%;
  border-radius: 6px;
  background: var(--c-fill);
  animation: shimmer 1.2s ease-in-out infinite;
}

.skeleton__bar + .skeleton__bar {
  margin-top: 10px;
}

.skeleton__bar--short {
  width: 55%;
}

.skeleton__bar--head {
  width: 45%;
  height: 18px;
}

@keyframes shimmer {
  0% {
    opacity: 0.6;
  }
  50% {
    opacity: 1;
  }
  100% {
    opacity: 0.6;
  }
}
</style>
