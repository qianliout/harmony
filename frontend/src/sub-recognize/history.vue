<script setup lang="ts">
// 识别历史：GET /recognize/tasks 分页列表；宽屏 4 列卡片，窄屏单列。
// 待核对任务带 taskId 回到核对步骤；已写入任务带 readonly=1 只读查看原图与识别项。
import { onReachBottom, onShow } from '@dcloudio/uni-app'
import { computed } from 'vue'

import { fetchRecognizeTasks } from '@/api/recognize'
import type { RecognizeTask } from '@/api/types'
import AppNav from '@/components/AppNav.vue'
import EmptyState from '@/components/EmptyState.vue'
import Skeleton from '@/components/Skeleton.vue'
import { usePagedList } from '@/composables/usePagedList'
import { useWideLayout } from '@/composables/useWideLayout'
import { pad } from '@/utils/date'
import { formatCent } from '@/utils/money'
import { go } from '@/utils/nav'

interface TaskStatusView {
  label: string
  cls: 'running' | 'review' | 'committed' | 'failed'
  action: string
}

const { isWide } = useWideLayout()
const { items, loading, finished, error, load, loadMore } = usePagedList(fetchRecognizeTasks, 10)

const cards = computed(() =>
  items.value.map((task) => ({
    task,
    status: statusOf(task),
    count: task.items?.length ?? 0,
  })),
)

function statusOf(task: RecognizeTask): TaskStatusView {
  if (task.status === 'failed') return { label: '失败', cls: 'failed', action: '查看原因' }
  if (task.status !== 'succeeded') return { label: '识别中', cls: 'running', action: '继续等待' }
  if (task.committed) return { label: '已写入', cls: 'committed', action: '只读查看' }
  return { label: '待核对', cls: 'review', action: '继续核对' }
}

function formatTime(ms: number): string {
  if (!ms) return ''
  const date = new Date(ms)
  const sameYear = date.getFullYear() === new Date().getFullYear()
  const day = `${pad(date.getMonth() + 1)}-${pad(date.getDate())}`
  return `${sameYear ? '' : `${date.getFullYear()}-`}${day} ${pad(date.getHours())}:${pad(date.getMinutes())}`
}

function openTask(task: RecognizeTask): void {
  if (task.status === 'succeeded' && task.committed) {
    go(`/sub-recognize/index?taskId=${task.id}&readonly=1`)
    return
  }
  go(`/sub-recognize/index?taskId=${task.id}`)
}

onShow(() => {
  void load(true)
})

onReachBottom(() => {
  void loadMore()
})
</script>

<template>
  <view class="screen" :class="{ 'screen--wide': isWide }">
    <AppNav title="识别历史" />

    <view class="page" :class="{ 'page--wide': isWide }">
      <Skeleton v-if="loading && !items.length" />

      <view v-else-if="error && !items.length" class="error-state">
        <text class="error-state__text">{{ error }}</text>
        <view class="btn btn--secondary error-state__btn" @click="load(true)">重试</view>
      </view>

      <EmptyState
        v-else-if="!cards.length"
        title="还没有识别记录"
        description="上传一张礼簿照片，识别核对后写入账目"
        action-text="去识别"
        @action="go('/sub-recognize/index')"
      />

      <view v-else class="history-list" :class="{ 'history-list--grid': isWide }">
        <view v-for="card in cards" :key="card.task.id" class="hcard" @click="openTask(card.task)">
          <image v-if="card.task.imageUrl" class="hcard__thumb" :src="card.task.imageUrl" mode="aspectFill" lazy-load />
          <view v-else class="hcard__thumb hcard__thumb--empty"><text class="hcard__seal">簿</text></view>

          <view class="hcard__body">
            <view class="hcard__top">
              <text class="status" :class="`status--${card.status.cls}`">{{ card.status.label }}</text>
              <text class="hcard__time">{{ formatTime(card.task.createdAt) }}</text>
            </view>
            <text class="hcard__page">{{ card.task.page ? `第 ${card.task.page} 页` : '未标页码' }}</text>
            <view class="hcard__foot">
              <text class="hcard__count">{{ card.count }} 条</text>
              <text class="hcard__sum">{{ formatCent(card.task.sumCent) }}</text>
            </view>
            <text class="hcard__action">{{ card.status.action }} ›</text>
          </view>
        </view>
      </view>

      <text v-if="loading && items.length" class="loading-tip">正在加载…</text>
      <text v-else-if="finished && items.length" class="loading-tip">没有更多了</text>
      <text v-if="error && items.length" class="loading-tip">{{ error }}</text>

      <text v-if="items.length" class="history-note">
        已写入的任务只能查看识别时的原图与识别项；暂不支持列出当时生成的记录明细，请在礼簿中查看。
      </text>
    </view>
  </view>
</template>

<style lang="scss" scoped>
.page--wide {
  padding-left: 24px;
  padding-right: 24px;
}

.history-list {
  display: flex;
  flex-direction: column;
  gap: 10px;
  padding-top: 4px;
}

.history-list--grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 14px;
}

.hcard {
  display: flex;
  gap: 12px;
  padding: 10px;
  border-radius: var(--r-m);
  background: var(--c-surface);
  box-shadow: var(--shadow-card);
  cursor: pointer;
}

.history-list--grid .hcard {
  flex-direction: column;
  gap: 8px;
}

.hcard__thumb {
  width: 88px;
  height: 88px;
  border-radius: var(--r-s);
  background: var(--c-fill);
  flex: none;
}

.history-list--grid .hcard__thumb {
  width: 100%;
  height: 150px;
}

.hcard__thumb--empty {
  display: flex;
  align-items: center;
  justify-content: center;
}

.hcard__seal {
  font-family: $font-serif;
  font-size: 26px;
  color: var(--c-none);
  opacity: 0.6;
}

.hcard__body {
  display: flex;
  flex-direction: column;
  min-width: 0;
  flex: 1;
}

.hcard__top {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}

.status {
  flex: none;
  padding: 1px 7px;
  border-radius: var(--r-full);
  font-size: 11px;
  font-weight: 500;
}

.status--running {
  background: var(--c-pending-light);
  color: var(--c-pending);
}

.status--review {
  background: var(--c-primary-light);
  color: var(--c-primary);
}

.status--committed {
  background: var(--c-repaid-light);
  color: var(--c-repaid);
}

.status--failed {
  background: var(--c-danger-light);
  color: var(--c-danger);
}

.hcard__time {
  font-family: $font-num;
  font-size: 11px;
  color: var(--c-ink-3);
  white-space: nowrap;
}

.hcard__page {
  margin-top: 6px;
  font-size: 12px;
  color: var(--c-ink-3);
}

.hcard__foot {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  margin-top: 4px;
}

.hcard__count {
  font-size: 13px;
  color: var(--c-ink-2);
}

.hcard__sum {
  font-family: $font-num;
  font-size: 15px;
  font-weight: 600;
  color: var(--c-out);
}

.hcard__action {
  margin-top: 6px;
  font-size: 12px;
  color: var(--c-primary);
}

.error-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 12px;
  padding: 48px 16px;
}

.error-state__text {
  font-size: 13px;
  color: var(--c-danger);
}

.error-state__btn {
  padding: 0 24px;
}

.history-note {
  display: block;
  margin: 18px 2px 8px;
  font-size: 11px;
  line-height: 16px;
  color: var(--c-ink-3);
}
</style>
