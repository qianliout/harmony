<script setup lang="ts">
import { onLoad } from '@dcloudio/uni-app'
import { ref } from 'vue'

import { fetchMergeLogs } from '@/api/book'
import type { MergeLog } from '@/api/types'
import AppNav from '@/components/AppNav.vue'
import EmptyState from '@/components/EmptyState.vue'
import Skeleton from '@/components/Skeleton.vue'
import { useDictStore } from '@/stores/dict'
import { paletteOf } from '@/styles/palette'
import { go } from '@/utils/nav'

const dictStore = useDictStore()

const logs = ref<MergeLog[]>([])
const loading = ref(true)
const showSkeleton = ref(false)
const error = ref('')

let skeletonTimer: ReturnType<typeof setTimeout> | null = null

function categoryLabel(code: string): string {
  return dictStore.categoryLabel(code) || code
}

function categoryColors(code: string): { main: string; light: string } {
  return paletteOf(dictStore.categoryColor(code))
}

function formatTime(ms: number): string {
  if (!ms) return ''
  const date = new Date(ms)
  const pad = (value: number) => String(value).padStart(2, '0')
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())} ${pad(date.getHours())}:${pad(date.getMinutes())}`
}

async function load(): Promise<void> {
  loading.value = true
  error.value = ''
  if (skeletonTimer) clearTimeout(skeletonTimer)
  skeletonTimer = setTimeout(() => {
    showSkeleton.value = true
  }, 300)
  try {
    await dictStore.ensure().catch(() => undefined)
    logs.value = await fetchMergeLogs()
  } catch (err) {
    error.value = err instanceof Error ? err.message : '加载失败'
  } finally {
    if (skeletonTimer) clearTimeout(skeletonTimer)
    skeletonTimer = null
    showSkeleton.value = false
    loading.value = false
  }
}

function retry(): void {
  load()
}

function openSource(id: number): void {
  go(`/sub-book/detail?id=${id}`)
}

onLoad(() => {
  load()
})
</script>

<template>
  <view class="screen">
    <AppNav title="已合并的礼簿" />

    <view class="page">
      <Skeleton v-if="showSkeleton" />

      <view v-else-if="error" class="state">
        <EmptyState title="没能加载合并记录" :description="error" action-text="重试" @action="retry" />
      </view>

      <view v-else-if="!loading && !logs.length" class="state">
        <EmptyState title="还没有合并过礼簿" description="合并后的礼簿会留在这里，随时可以回看" />
      </view>

      <template v-else-if="!loading">
        <view v-for="log in logs" :key="log.id" class="group card">
          <view class="group__head">
            <view class="group__head-main">
              <text class="group__target">并入「{{ log.targetName }}」</text>
              <text class="group__time">{{ formatTime(log.createdAt) }}</text>
            </view>
            <text v-if="log.crossCategory" class="cross-badge">跨类别</text>
          </view>
          <view
            v-for="source in log.sources"
            :key="`${log.id}-${source.id}`"
            class="source"
            @click="openSource(source.id)"
          >
            <view class="source__spine" :style="{ background: categoryColors(source.category).main }" />
            <view class="source__main">
              <view class="source__line">
                <text class="source__name">{{ source.name }}</text>
                <text
                  class="source__category"
                  :style="{
                    background: categoryColors(source.category).light,
                    color: categoryColors(source.category).main,
                  }"
                >
                  {{ categoryLabel(source.category) }}
                </text>
              </view>
              <text class="source__meta">原 {{ source.recordCount }} 笔 · 只读</text>
            </view>
            <text class="source__arrow">›</text>
          </view>
        </view>
      </template>
    </view>
  </view>
</template>

<style lang="scss" scoped>
.state {
  padding-top: 24px;
}

.group {
  margin-top: 12px;
  overflow: hidden;
}

.group:first-child {
  margin-top: 8px;
}

.group__head {
  display: flex;
  align-items: flex-start;
  gap: 8px;
  padding: 13px 16px;
  background: var(--c-fill);
}

.group__head-main {
  flex: 1;
  min-width: 0;
}

.group__target {
  display: block;
  font-size: 14px;
  line-height: 20px;
  font-weight: 600;
  color: var(--c-ink-1);
}

.group__time {
  display: block;
  margin-top: 2px;
  font-size: 12px;
  color: var(--c-ink-3);
}

.cross-badge {
  flex: none;
  height: 20px;
  padding: 0 8px;
  border-radius: var(--r-full);
  background: var(--c-pending-light);
  color: var(--c-pending);
  font-size: 11px;
  line-height: 20px;
}

.source {
  position: relative;
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 12px 16px;
  background: var(--c-surface);
}

.source + .source {
  box-shadow: inset 0 1px 0 var(--c-divider);
}

.source__spine {
  width: 5px;
  height: 36px;
  border-radius: 3px;
  flex: none;
}

.source__main {
  flex: 1;
  min-width: 0;
}

.source__line {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
}

.source__name {
  font-size: 15px;
  line-height: 22px;
  font-weight: 600;
  color: var(--c-ink-1);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.source__category {
  flex: none;
  height: 20px;
  padding: 0 6px;
  border-radius: var(--r-xs);
  font-size: 12px;
  line-height: 20px;
}

.source__meta {
  display: block;
  margin-top: 2px;
  font-size: 12px;
  color: var(--c-ink-3);
}

.source__arrow {
  flex: none;
  font-size: 18px;
  color: var(--c-ink-4);
}
</style>
