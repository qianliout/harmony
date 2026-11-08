<script setup lang="ts">
// 识别中与失败态：轮询上限 100 秒（千问整页输出通常 30–80 秒），超时由页面传入错误文案。
import { computed } from 'vue'

const props = defineProps<{
  imageUrl: string
  elapsed: number
  error: string
}>()

defineEmits<{
  cancel: []
  retry: []
  manual: []
}>()

const POLL_LIMIT_SECONDS = 100

const title = computed(() => {
  if (props.elapsed < 5) return '正在识别…'
  if (props.elapsed < 20) return '字迹较多，再等一下…'
  if (props.elapsed < 60) return '快好了…'
  return '模型还在读这一页，请再等一会儿…'
})
</script>

<template>
  <view class="progress">
    <view class="scan">
      <image v-if="imageUrl" class="scan__image" :src="imageUrl" mode="aspectFit" />
      <view v-else class="scan__empty">
        <text class="scan__empty-text">照片已提交</text>
      </view>
      <view v-if="!error" class="scan__line" />
    </view>

    <view v-if="!error" class="recognizing">
      <text class="recognizing__title">{{ title }}</text>
      <text class="recognizing__hint">已等待 {{ elapsed }} 秒 · 最多 {{ POLL_LIMIT_SECONDS }} 秒</text>
      <view class="progress__bar">
        <view class="progress__fill" :style="{ width: `${Math.min(100, (elapsed / POLL_LIMIT_SECONDS) * 100)}%` }" />
      </view>
      <view class="btn btn--secondary" @click="$emit('cancel')">取消</view>
    </view>

    <view v-else class="failed">
      <text class="failed__title">识别没有完成</text>
      <text class="failed__message">{{ error }}</text>
      <view class="failed__actions">
        <view class="btn btn--secondary failed__btn" @click="$emit('manual')">改为手动录入</view>
        <view class="btn btn--primary failed__btn" @click="$emit('retry')">重试</view>
      </view>
    </view>
  </view>
</template>

<style lang="scss" scoped>
.scan {
  position: relative;
  height: 260px;
  border-radius: var(--r-m);
  background: var(--c-ink-1);
  overflow: hidden;
}

.scan__image {
  width: 100%;
  height: 100%;
}

.scan__empty {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 100%;
}

.scan__empty-text {
  font-size: 13px;
  color: rgba(255, 255, 255, 0.7);
}

.scan__line {
  position: absolute;
  left: 0;
  right: 0;
  top: 0;
  height: 2px;
  background: var(--c-primary);
  box-shadow: 0 0 12px var(--c-primary);
  animation: scan 1.6s linear infinite;
}

@keyframes scan {
  0% {
    top: 0;
  }
  100% {
    top: 100%;
  }
}

.recognizing,
.failed {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 20px 0;
}

.recognizing__title,
.failed__title {
  font-size: 17px;
  font-weight: 600;
}

.recognizing__hint,
.failed__message {
  margin: 6px 0 16px;
  font-size: 13px;
  color: var(--c-ink-3);
  text-align: center;
}

.progress__bar {
  width: 100%;
  height: 4px;
  margin-bottom: 18px;
  border-radius: var(--r-full);
  background: var(--c-fill);
  overflow: hidden;
}

.progress__fill {
  height: 100%;
  border-radius: inherit;
  background: var(--c-primary);
  transition: width 0.4s linear;
}

.failed__actions {
  display: flex;
  gap: 10px;
  width: 100%;
}

.failed__btn {
  flex: 1;
}
</style>
