<script setup lang="ts">
// H5 桌面识别历史：分页读取 /recognize/tasks，卡片显示缩略图、时间、状态、条数与合计。
// 待核对任务回到核对步骤，已写入任务只读查看识别时的原图与识别项。
import { computed, onMounted, ref } from 'vue'

import { fetchRecognizeTasks } from '@/api/recognize'
import type { RecognizeTask } from '@/api/types'
import { pad } from '@/utils/date'
import { formatCent } from '@/utils/money'
import PageHead from '@/web/components/PageHead.vue'
import Pager from '@/web/components/Pager.vue'
import WebIcon from '@/web/components/WebIcon.vue'
import { goWeb } from '@/web/nav'

interface TaskStatusView {
  label: string
  cls: 'running' | 'review' | 'committed' | 'failed'
  action: string
}

const PAGE_SIZE = 12

const items = ref<RecognizeTask[]>([])
const total = ref(0)
const page = ref(1)
const loading = ref(false)
const error = ref('')
let loadSeq = 0

const pages = computed(() => Math.max(1, Math.ceil(total.value / PAGE_SIZE)))

function statusOf(task: RecognizeTask): TaskStatusView {
  if (task.status === 'failed') return { label: '失败', cls: 'failed', action: '查看原因' }
  if (task.status !== 'succeeded') return { label: '识别中', cls: 'running', action: '继续等待' }
  if (task.committed) return { label: '已写入', cls: 'committed', action: '只读查看' }
  return { label: '待核对', cls: 'review', action: '继续核对' }
}

function statusClass(task: RecognizeTask): string {
  const tone = statusOf(task).cls
  if (tone === 'review') return 'web-pill web-pill--pending'
  if (tone === 'committed') return 'web-pill web-pill--repaid'
  if (tone === 'failed') return 'web-pill web-pill--warn'
  return 'web-pill web-pill--neutral'
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
    goWeb(`/sub-web/recognize?taskId=${task.id}&readonly=1`, 'push')
    return
  }
  goWeb(`/sub-web/recognize?taskId=${task.id}`, 'push')
}

async function load(target = page.value): Promise<void> {
  const seq = ++loadSeq
  page.value = target
  loading.value = true
  error.value = ''
  try {
    const result = await fetchRecognizeTasks(target, PAGE_SIZE)
    if (seq !== loadSeq) return
    items.value = result.items
    total.value = result.total
  } catch (err) {
    if (seq !== loadSeq) return
    error.value = err instanceof Error ? err.message : '加载识别历史失败'
  } finally {
    if (seq === loadSeq) loading.value = false
  }
}

onMounted(() => {
  void load(1)
})
</script>

<template>
  <PageHead title="识别历史" meta="每张原图对应一次识别；待核对的可以继续，写入后只能只读回看">
    <template #tools>
      <button class="web-btn web-btn--primary" type="button" @click="goWeb('/sub-web/recognize', 'push')">
        <WebIcon name="scan" :size="16" />
        去识别
      </button>
    </template>
  </PageHead>

  <div v-if="error" class="web-alert web-alert--err">
    {{ error }}
    <button class="web-btn web-btn--text web-btn--sm hist-retry" type="button" @click="load()">重试</button>
  </div>

  <div v-if="loading && !items.length" class="web-card web-empty">正在加载识别记录…</div>

  <div v-else-if="!loading && !items.length" class="web-card web-empty">
    <div class="hist-empty__seal">簿</div>
    <div class="hist-empty__title">还没有识别记录</div>
    <div class="hist-empty__desc">上传一张礼簿照片，核对确认后写入账目</div>
    <button class="web-btn web-btn--primary hist-empty__btn" type="button" @click="goWeb('/sub-web/recognize', 'push')">
      去识别
    </button>
  </div>

  <template v-else>
    <div class="hist-grid">
      <button
        v-for="task in items"
        :key="task.id"
        class="hist-card"
        type="button"
        @click="openTask(task)"
      >
        <div class="hist-thumb">
          <img v-if="task.imageUrl" class="hist-thumb__img" :src="task.imageUrl" alt="" loading="lazy" />
          <span v-else class="hist-thumb__seal">簿</span>
        </div>

        <div class="hist-body">
          <div class="hist-top">
            <span :class="statusClass(task)">{{ statusOf(task).label }}</span>
            <span class="web-num hist-time">{{ formatTime(task.createdAt) }}</span>
          </div>
          <div class="hist-page">
            {{ task.page ? `第 ${task.page} 页` : '未标页码' }}
          </div>
          <div class="hist-foot">
            <span class="web-sm web-muted">{{ task.items?.length ?? 0 }} 条</span>
            <span class="web-amt web-amt--out">{{ formatCent(task.sumCent) }}</span>
          </div>
          <div class="hist-action">
            {{ statusOf(task).action }}
            <WebIcon name="right" :size="14" />
          </div>
        </div>
      </button>
    </div>

    <div class="hist-footbar">
      <span class="web-sm web-muted">
        共 {{ total }} 张<span v-if="loading"> · 正在加载…</span>
      </span>
      <Pager :page="page" :pages="pages" @change="load" />
    </div>

    <div class="hist-note">
      已写入的任务只能查看识别时的原图与识别项；暂不支持列出当时生成的记录明细，请在礼簿中查看。
    </div>
  </template>
</template>

<style lang="scss" scoped>
.hist-retry {
  margin-left: auto;
}

.hist-empty__seal {
  display: grid;
  place-items: center;
  width: 64px;
  height: 64px;
  margin: 0 auto 12px;
  border: 1px solid var(--c-none);
  border-radius: 16px;
  color: var(--c-none);
  font: 600 30px/1 $font-serif;
  opacity: 0.65;
}

.hist-empty__title {
  color: var(--c-ink-2);
  font-size: 17px;
  font-weight: 600;
}

.hist-empty__desc {
  margin-top: 6px;
  font-size: 13px;
}

.hist-empty__btn {
  margin-top: var(--sp-5);
}

.hist-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(250px, 1fr));
  gap: var(--sp-4);
}

.hist-card {
  display: flex;
  flex-direction: column;
  overflow: hidden;
  padding: 0;
  border: 0;
  border-radius: var(--r-m);
  background: var(--c-surface);
  box-shadow: var(--shadow-card);
  text-align: left;
  transition: box-shadow 0.15s, transform 0.15s;
}

.hist-card:hover {
  box-shadow: var(--shadow-float);
  transform: translateY(-1px);
}

.hist-thumb {
  display: grid;
  place-items: center;
  height: 156px;
  overflow: hidden;
  background: var(--c-fill);
}

.hist-thumb__img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.hist-thumb__seal {
  color: var(--c-none);
  font: 600 28px/1 $font-serif;
  opacity: 0.6;
}

.hist-body {
  display: flex;
  flex: 1;
  flex-direction: column;
  gap: 4px;
  padding: var(--sp-3) var(--sp-4) var(--sp-4);
}

.hist-top {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--sp-2);
}

.hist-time {
  color: var(--c-ink-3);
  font-size: 11px;
  white-space: nowrap;
}

.hist-page {
  color: var(--c-ink-3);
  font-size: 12px;
}

.hist-foot {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  margin-top: 2px;
}

.hist-action {
  display: flex;
  align-items: center;
  gap: 2px;
  margin-top: 2px;
  color: var(--c-primary);
  font-size: 12px;
}

.hist-footbar {
  display: flex;
  align-items: center;
  gap: var(--sp-3);
  padding: var(--sp-2) 0;
}

.hist-note {
  color: var(--c-ink-3);
  font-size: 11px;
  line-height: 18px;
}
</style>
