<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'

import PageContainer from '@/core/components/PageContainer.vue'
import { formatTimestamp } from '@/core/utils/format'
import { fetchHealth } from '@/modules/dashboard/api/dashboard'
import type { HealthStatus } from '@/modules/dashboard/types'

defineOptions({ name: 'DashboardView' })

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

onMounted(loadHealth)
</script>

<template>
  <PageContainer title="工作台" caption="人情往来账本 · 本机 Web 端">
    <section class="service">
      <div class="service__head">
        <h3 class="service__title">后端服务</h3>
        <el-button text size="small" :loading="loading" @click="loadHealth">刷新</el-button>
      </div>

      <div class="service__state">
        <span class="service__dot" :class="`service__dot--${state.tone}`" />
        <span class="service__label">{{ state.label }}</span>
        <code class="service__endpoint">/api/v1/health</code>
      </div>

      <p v-if="error" class="service__error">{{ error }}</p>

      <dl class="service__meta">
        <div v-for="row in metaRows" :key="row.label" class="service__row">
          <dt class="service__key">{{ row.label }}</dt>
          <dd class="service__value">{{ row.value }}</dd>
        </div>
      </dl>
    </section>
  </PageContainer>
</template>

<style scoped>
.service {
  padding: 18px 20px 8px;
  border: 1px solid var(--el-border-color-light);
  border-radius: 8px;
}

.service__head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.service__title {
  margin: 0;
  font-size: 16px;
  line-height: 24px;
  font-weight: 600;
  color: var(--el-text-color-primary);
}

.service__state {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-top: 14px;
}

.service__dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: var(--el-text-color-placeholder);
}

.service__dot--ok {
  background: #00b42a;
}

.service__dot--error {
  background: #f53f3f;
}

.service__dot--pending {
  background: #c9cdd4;
}

.service__label {
  font-size: 15px;
  line-height: 24px;
  font-weight: 500;
  color: var(--el-text-color-primary);
}

.service__endpoint {
  margin-left: auto;
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  font-size: 12px;
  line-height: 20px;
  color: var(--el-text-color-secondary);
}

.service__error {
  margin: 10px 0 0;
  font-size: 13px;
  line-height: 20px;
  color: #c9242b;
}

.service__meta {
  margin: 16px 0 0;
  padding: 0;
  border-top: 1px solid var(--el-border-color-light);
}

.service__row {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: 16px;
  padding: 10px 0;
  border-bottom: 1px solid var(--el-border-color-light);
}

.service__row:last-child {
  border-bottom: none;
}

.service__key {
  font-size: 13px;
  line-height: 20px;
  color: var(--el-text-color-secondary);
}

.service__value {
  margin: 0;
  font-size: 14px;
  line-height: 20px;
  color: var(--el-text-color-primary);
  font-variant-numeric: tabular-nums;
}
</style>
