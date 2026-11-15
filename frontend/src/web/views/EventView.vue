<script setup lang="ts">
import { computed, ref, watch } from 'vue'

import { fetchEventDetail } from '@/api/event'
import type { EventItem, EventPerson, EventPersonLine, EventSummary } from '@/api/types'
import { fetchEventSummary } from '@/api/view'
import DirTag from '@/components/DirTag.vue'
import PersonAvatar from '@/components/PersonAvatar.vue'
import { useWebUiStore } from '@/stores/web'
import { preciseDateLabel } from '@/utils/date'
import { formatCent } from '@/utils/money'
import PageHead from '@/web/components/PageHead.vue'
import Segmented from '@/web/components/Segmented.vue'
import StatCard from '@/web/components/StatCard.vue'
import WebIcon from '@/web/components/WebIcon.vue'
import WebTable from '@/web/components/WebTable.vue'
import { useAsync } from '@/web/composables/useAsync'
import { backWeb, goWeb } from '@/web/nav'

type TabKey = 'in' | 'repay' | 'pending'

interface WebPageQuery {
  options?: Record<string, string>
  $page?: { options?: Record<string, string> }
}

// H5 页面的 query 挂在当前页面实例的 options 上，直接刷新时再从地址栏兜底
function readQuery(): Record<string, string> {
  const stack = getCurrentPages()
  const current = stack[stack.length - 1] as unknown as WebPageQuery | undefined
  const options = current?.options ?? current?.$page?.options ?? {}
  if (options.id) return options
  if (typeof window === 'undefined') return options
  const matched = `${window.location.search}${window.location.hash}`.match(/[?&]id=(\d+)/)
  return matched ? { id: matched[1] } : options
}

const eventId = Number(readQuery().id ?? 0)

const ui = useWebUiStore()
const tab = ref<TabKey>('in')

const resource = useAsync<{ detail: EventItem | null; summary: EventSummary | null }>(
  async () => {
    if (!eventId) throw new Error('缺少事项参数')
    const [detail, summary] = await Promise.all([fetchEventDetail(eventId), fetchEventSummary(eventId)])
    return { detail, summary }
  },
  { detail: null, summary: null },
)

const detail = computed(() => resource.data.value.detail)
const summary = computed(() => resource.data.value.summary)
const error = computed(() => resource.error.value)
const figures = computed(
  () => summary.value?.summary ?? { inCent: 0, outCent: 0, personCount: 0, repaidCount: 0, pendingCount: 0, recordCount: 0 },
)
const persons = computed(() => summary.value?.persons ?? [])

const inPersons = computed(() => persons.value.filter((person) => person.inCent > 0))
const repayPersons = computed(() => persons.value.filter((person) => person.repayCount > 0))
const pendingPersons = computed(() => persons.value.filter((person) => person.pendingCount > 0))

const rows = computed<EventPerson[]>(() => {
  if (tab.value === 'repay') return repayPersons.value
  if (tab.value === 'pending') return pendingPersons.value
  return inPersons.value
})

const options = computed(() => [
  { value: 'in', label: '谁来了', count: inPersons.value.length },
  { value: 'repay', label: '我回了', count: repayPersons.value.length },
  { value: 'pending', label: '还有谁没回', count: pendingPersons.value.length },
])

const meta = computed(() => {
  const item = detail.value
  if (!item) return ''
  const date = preciseDateLabel(item.eventDate, item.datePrecision) || '日期未定'
  return [item.typeLabel, date, item.hostLabel, item.location].filter(Boolean).join(' · ')
})

const emptyText = computed(() => {
  if (tab.value === 'repay') return '这件事上还没有回过礼。'
  if (tab.value === 'pending') return '没有待回礼的人。'
  return '还没有收礼记录。'
})

// 明细按当前视图只取相关的礼簿 / 名目行
function linesOf(person: EventPerson): EventPersonLine[] {
  if (tab.value === 'repay') return person.details.filter((line) => line.isRepay || line.direction === 'out')
  return person.details.filter((line) => line.direction === 'in' && !line.isRepay)
}

function rowCent(person: EventPerson): number {
  if (tab.value === 'repay') return person.repayCent
  return person.inCent
}

function rowDirection(): 'in' | 'out' {
  return tab.value === 'repay' ? 'out' : 'in'
}

function setTab(value: string): void {
  tab.value = value as TabKey
}

function openPerson(id: number): void {
  goWeb(`/sub-web/person?id=${id}`, 'push')
}

if (eventId) void resource.reload()

watch(
  () => ui.revision,
  () => {
    if (eventId) void resource.reload()
  },
)
</script>

<template>
  <PageHead :title="detail?.name || '事项详情'" :meta="meta">
    <template #meta>
      <div v-if="detail?.remark" class="web-muted web-sm">{{ detail.remark }}</div>
    </template>
    <template #tools>
      <button class="web-btn" type="button" @click="backWeb()">
        <WebIcon name="left" />
        返回事项
      </button>
      <button
        v-if="eventId"
        class="web-btn web-btn--primary"
        type="button"
        @click="ui.openEntry({ eventId })"
      >
        <WebIcon name="plus" />
        在此事项记一笔
      </button>
    </template>
  </PageHead>

  <div v-if="!eventId" class="web-alert web-alert--err">
    <WebIcon name="alert" />
    <span>缺少事项参数。</span>
  </div>

  <div v-else-if="error" class="web-card">
    <div class="web-alert web-alert--err" style="margin: 16px">
      <WebIcon name="alert" />
      <span>{{ error }}</span>
      <button class="web-btn web-btn--sm" type="button" @click="resource.reload()">重试</button>
    </div>
  </div>

  <div v-else-if="!detail" class="web-empty">正在打开事项…</div>

  <template v-else>
    <div class="web-stats">
      <StatCard :value="formatCent(figures.inCent)" label="收礼合计" tone="in" />
      <StatCard :value="formatCent(figures.outCent)" label="送礼合计" tone="out" />
      <StatCard :value="`${figures.personCount} 人`" label="到礼人数" />
      <StatCard :value="`${figures.pendingCount} 笔`" label="未回礼" tone="pending" />
    </div>

    <div class="web-card">
      <div class="web-filters">
        <Segmented :model-value="tab" :options="options" @update:model-value="setTab" />
        <div class="web-filters__right web-muted web-sm">
          共 {{ figures.recordCount }} 笔记录 · 已回 {{ figures.repaidCount }} 笔
        </div>
      </div>

      <WebTable>
        <thead>
          <tr>
            <th>亲友</th>
            <th>礼簿 / 名目 / 金额</th>
            <th class="web-table__r">合计</th>
            <th>状态</th>
          </tr>
        </thead>
        <tbody>
          <tr
            v-for="person in rows"
            :key="person.person.id"
            class="web-table__row"
            @click="openPerson(person.person.id)"
          >
            <td :class="{ 'web-table__pend': tab === 'pending' }">
              <div class="web-cell2">
                <PersonAvatar :id="person.person.id" :name="person.person.name" :size="28" />
                <span class="web-cell2__t">
                  {{ person.person.name }}
                  <small>{{ person.person.relationLabel }}</small>
                </span>
              </div>
            </td>
            <td>
              <div class="ev-lines">
                <span
                  v-for="(line, index) in linesOf(person)"
                  :key="`${person.person.id}-${index}`"
                  class="ev-line"
                >
                  <DirTag :direction="line.direction" />
                  <span class="web-sm">{{ line.bookName || '未注明礼簿' }}</span>
                  <span class="web-sm web-muted">{{ line.itemKindLabel }}</span>
                  <span class="web-amt" :class="`web-amt--${line.direction}`">{{ formatCent(line.cent) }}</span>
                </span>
                <span v-if="!linesOf(person).length" class="web-muted web-sm">暂无明细</span>
              </div>
            </td>
            <td class="web-table__r">
              <span class="web-amt" :class="`web-amt--${rowDirection()}`">{{ formatCent(rowCent(person)) }}</span>
            </td>
            <td>
              <span v-if="tab === 'pending'" class="web-pill web-pill--pending">{{ person.pendingCount }} 笔未回</span>
              <span v-else-if="tab === 'repay'" class="web-pill web-pill--repaid">已回 {{ person.repayCount }} 次</span>
              <span v-else class="web-pill web-pill--neutral">收礼 {{ formatCent(person.inCent) }}</span>
            </td>
          </tr>
        </tbody>
      </WebTable>

      <div v-if="!rows.length" class="web-empty">{{ emptyText }}</div>
    </div>
  </template>
</template>

<style lang="scss" scoped>
.ev-lines {
  display: flex;
  flex-direction: column;
  gap: var(--sp-1);
  white-space: normal;
}

.ev-line {
  display: flex;
  align-items: center;
  gap: var(--sp-2);
  line-height: 22px;
}
</style>
