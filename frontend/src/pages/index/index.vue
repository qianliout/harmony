<script setup lang="ts">
import { onShow } from '@dcloudio/uni-app'
import { computed, ref } from 'vue'

import { fetchHealth } from '@/api/health'
import type { HealthStatus } from '@/api/types'
import { themeVars } from '@/styles/theme'
import { formatTimestamp } from '@/utils/format'

const health = ref<HealthStatus | null>(null)
const loading = ref(false)
const error = ref('')

const state = computed(() => {
  if (error.value) {
    return { label: '不可用', tone: 'error' }
  }
  if (!health.value) {
    return { label: '检查中', tone: 'pending' }
  }
  return { label: '正常', tone: 'ok' }
})

const metaRows = computed(() => [
  { label: '版本', value: health.value?.version ?? '-' },
  { label: '数据库', value: health.value?.database ?? '-' },
  { label: '服务时间', value: health.value ? formatTimestamp(health.value.time) : '-' },
])

async function loadHealth(): Promise<void> {
  loading.value = true
  error.value = ''
  try {
    health.value = await fetchHealth()
  } catch (err) {
    health.value = null
    error.value = err instanceof Error ? err.message : '健康检查失败'
  } finally {
    loading.value = false
  }
}

onShow(loadHealth)
</script>

<template>
  <wd-config-provider :theme-vars="themeVars">
    <view class="page">
      <view class="hero">
        <text class="hero__watermark">礼</text>
        <text class="hero__eyebrow">HARMONY</text>
        <text class="hero__title">人情往来账本</text>
        <text class="hero__caption">工作台 · 业务模块按需求文档逐步接入</text>
      </view>

      <view class="card">
        <view class="card__head">
          <text class="card__title">后端服务</text>
          <wd-button size="small" plain :loading="loading" @click="loadHealth">刷新</wd-button>
        </view>

        <view class="state">
          <view class="state__dot" :class="`state__dot--${state.tone}`" />
          <text class="state__label">{{ state.label }}</text>
          <text class="state__endpoint">/api/v1/health</text>
        </view>

        <text v-if="error" class="card__error">{{ error }}</text>

        <view v-for="row in metaRows" :key="row.label" class="meta">
          <text class="meta__key">{{ row.label }}</text>
          <text class="meta__value">{{ row.value }}</text>
        </view>
      </view>
    </view>
  </wd-config-provider>
</template>

<style lang="scss" scoped>
.page {
  padding: var(--sp-3) var(--gutter) var(--sp-8);
}

.hero {
  position: relative;
  overflow: hidden;
  display: flex;
  flex-direction: column;
  padding: var(--sp-5);
  border-radius: var(--r-l);
  background: linear-gradient(135deg, var(--c-primary) 0%, var(--c-primary-dark) 100%);
  box-shadow: 0 10px 24px rgba(169, 43, 34, 0.28);
}

.hero__watermark {
  position: absolute;
  right: -14px;
  bottom: -40px;
  font-family: $font-serif;
  font-size: 140px;
  font-weight: 700;
  line-height: 1;
  color: var(--c-surface);
  opacity: 0.08;
}

.hero__eyebrow {
  font-size: 11px;
  line-height: 16px;
  letter-spacing: 0.12em;
  color: var(--c-surface);
  opacity: 0.75;
}

.hero__title {
  margin-top: var(--sp-1);
  font-family: $font-serif;
  font-size: 24px;
  line-height: 32px;
  font-weight: 600;
  color: var(--c-surface);
}

.hero__caption {
  margin-top: var(--sp-1);
  font-size: 13px;
  line-height: 18px;
  color: var(--c-surface);
  opacity: 0.75;
}

.card {
  display: flex;
  flex-direction: column;
  margin-top: var(--card-gap);
  padding: var(--card-pad) var(--card-pad) var(--sp-1);
  border-radius: var(--r-m);
  background: var(--c-surface);
  box-shadow: var(--shadow-card);
}

.card__head {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.card__title {
  font-size: 17px;
  line-height: 24px;
  font-weight: 600;
}

.card__error {
  margin-top: var(--sp-2);
  font-size: 13px;
  line-height: 18px;
  color: var(--c-danger);
}

.state {
  display: flex;
  align-items: center;
  gap: var(--sp-2);
  margin-top: var(--sp-3);
}

.state__dot {
  width: 8px;
  height: 8px;
  border-radius: var(--r-full);
  background: var(--c-ink-4);
}

.state__dot--ok {
  background: var(--c-repaid);
}

.state__dot--error {
  background: var(--c-danger);
}

.state__label {
  font-size: 15px;
  font-weight: 500;
}

.state__endpoint {
  margin-left: auto;
  font-family: ui-monospace, Menlo, monospace;
  font-size: 12px;
  color: var(--c-ink-3);
}

.meta {
  display: flex;
  justify-content: space-between;
  padding: var(--sp-3) 0;
  border-top: 1px solid var(--c-divider);
}

.meta:first-of-type {
  margin-top: var(--sp-3);
}

.meta__key {
  font-size: 13px;
  color: var(--c-ink-3);
}

.meta__value {
  font-family: $font-num;
  font-size: 14px;
  font-variant-numeric: tabular-nums;
}
</style>
