<script setup lang="ts">
// 完成步骤：写入结果汇总，以及查看记录 / 继续识别下一页。
import type { RecognizeCommitResult } from '@/api/types'
import { formatCent } from '@/utils/money'

defineProps<{
  result: RecognizeCommitResult | null
}>()

defineEmits<{
  view: []
  next: []
}>()
</script>

<template>
  <view class="done">
    <view class="done__seal">✓</view>
    <text class="done__title">已写入 {{ result?.count ?? 0 }} 条记录</text>
    <text class="done__sum">合计 {{ formatCent(result?.sumCent ?? 0) }}</text>
    <text class="done__meta">新建亲友 {{ result?.newPersons ?? 0 }} 人 · 补录还礼 {{ result?.repayCount ?? 0 }} 笔</text>
    <view class="done__actions">
      <view class="btn btn--secondary done__btn" @click="$emit('view')">查看这些记录</view>
      <view class="btn btn--primary done__btn" @click="$emit('next')">继续识别下一页</view>
    </view>
  </view>
</template>

<style lang="scss" scoped>
.done {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 56px 16px;
}

.done__seal {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 64px;
  height: 64px;
  border-radius: var(--r-full);
  background: var(--c-repaid-light);
  color: var(--c-repaid);
  font-size: 30px;
}

.done__title {
  margin-top: 16px;
  font-size: 20px;
  font-weight: 600;
}

.done__sum {
  margin-top: 6px;
  font-family: $font-num;
  font-size: 17px;
  color: var(--c-ink-2);
}

.done__meta {
  margin-top: 6px;
  font-size: 13px;
  color: var(--c-ink-3);
}

.done__actions {
  display: flex;
  gap: 10px;
  width: 100%;
  margin-top: 28px;
}

.done__btn {
  flex: 1;
}
</style>
