<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'

import { fetchYearStats, fetchYears } from '@/api/stats'
import type { BookCategoryStat, EventTypeStat, MonthStat, PersonStat, YearStats } from '@/api/types'
import EmptyState from '@/components/EmptyState.vue'
import EventSeal from '@/components/EventSeal.vue'
import Money from '@/components/Money.vue'
import PageHead from '@/web/components/PageHead.vue'
import PersonCell from '@/web/components/PersonCell.vue'
import SectionHead from '@/web/components/SectionHead.vue'
import Segmented from '@/web/components/Segmented.vue'
import StatCard from '@/web/components/StatCard.vue'
import { useAsync } from '@/web/composables/useAsync'
import { useWebUiStore } from '@/stores/web'
import { preciseDateLabel } from '@/utils/date'
import { formatCent } from '@/utils/money'

const ui = useWebUiStore()

const emptyStats = (): YearStats => ({
  year: 0,
  inCent: 0,
  outCent: 0,
  netCent: 0,
  recordCount: 0,
  personCount: 0,
  monthly: [],
  byItem: [],
  byBookCategory: [],
  byEventType: [],
  topPersons: [],
  byBook: [],
})

const selectedYear = ref('all')
const initialLoading = ref(true)
let watchingYear = false

const years = useAsync(() => fetchYears(), [] as number[])
const stats = useAsync(
  () => fetchYearStats(selectedYear.value === 'all' ? 'all' : Number(selectedYear.value)),
  emptyStats(),
)

const yearOptions = computed(() => [
  { value: 'all', label: '全部' },
  ...[...years.data.value]
    .sort((a, b) => b - a)
    .map((year) => ({ value: String(year), label: `${year} 年` })),
])

const yearLabel = computed(() => (selectedYear.value === 'all' ? '全部年份' : `${selectedYear.value} 年`))

const monthly = computed<MonthStat[]>(() => {
  const map = new Map<number, MonthStat>()
  for (const item of stats.data.value.monthly) map.set(item.month, item)
  return Array.from({ length: 12 }, (_, index) => {
    const month = index + 1
    return map.get(month) ?? { month, inCent: 0, outCent: 0, recordCount: 0 }
  })
})

const monthMax = computed(() => {
  let max = 1
  for (const item of monthly.value) max = Math.max(max, item.inCent, item.outCent)
  return max
})

const categories = computed<BookCategoryStat[]>(() =>
  [...stats.data.value.byBookCategory].sort((a, b) => b.cent - a.cent),
)

const eventTypes = computed<EventTypeStat[]>(() =>
  [...stats.data.value.byEventType].sort((a, b) => b.cent - a.cent),
)

const personRows = computed<PersonStat[]>(() => stats.data.value.topPersons)

const categoryMax = computed(() => {
  let max = 1
  for (const item of categories.value) max = Math.max(max, item.inCent, item.outCent)
  return max
})

const eventMax = computed(() => {
  let max = 1
  for (const item of eventTypes.value) max = Math.max(max, item.inCent, item.outCent)
  return max
})

const CHART_W = 720
const SLOT = 60
const BASE = 176
const PLOT_H = 140

function barHeight(cent: number, max: number): number {
  if (cent <= 0) return 0
  return Math.max((cent / max) * PLOT_H, 2)
}

function monthBar(cent: number): number {
  return barHeight(cent, monthMax.value)
}

function categoryWidth(cent: number): string {
  if (cent <= 0) return '0%'
  return `${Math.max((cent / categoryMax.value) * 100, 2)}%`
}

function typeWidth(cent: number): string {
  if (cent <= 0) return '0%'
  return `${Math.max((cent / eventMax.value) * 100, 2)}%`
}

function dateLabel(value: string): string {
  return preciseDateLabel(value, 'day') || '—'
}

function netDirection(cent: number): 'in' | 'out' {
  return cent < 0 ? 'out' : 'in'
}

async function init(): Promise<void> {
  initialLoading.value = true
  try {
    await years.reload()
    if (years.data.value.length) {
      const current = String(new Date().getFullYear())
      const exists = years.data.value.some((year) => String(year) === current)
      selectedYear.value = exists ? current : String(Math.max(...years.data.value))
    } else {
      selectedYear.value = 'all'
    }
    watchingYear = true
    await stats.reload()
  } finally {
    initialLoading.value = false
  }
}

watch(selectedYear, () => {
  if (watchingYear) void stats.reload()
})

// 记录/回礼在抽屉里写库后同步刷新统计。
watch(
  () => ui.revision,
  () => {
    if (!initialLoading.value) void stats.reload()
  },
)

onMounted(() => {
  void init()
})
</script>

<template>
  <div>
    <PageHead title="统计" :meta="yearLabel">
      <template #tools>
        <button class="web-btn web-btn--text" type="button" @click="init">刷新</button>
      </template>
    </PageHead>

    <div class="stats-filter">
      <Segmented v-model="selectedYear" :options="yearOptions" />
    </div>

    <div v-if="initialLoading" class="web-empty">
      <span class="web-muted">正在统计…</span>
    </div>

    <div v-else-if="years.error.value && !years.data.value.length" class="web-alert web-alert--err stats-error">
      <span>{{ years.error.value }}</span>
      <button class="web-btn web-btn--text web-btn--sm" type="button" @click="init">重试</button>
    </div>

    <div v-else-if="stats.error.value && !stats.data.value.recordCount" class="web-alert web-alert--err stats-error">
      <span>{{ stats.error.value }}</span>
      <button class="web-btn web-btn--text web-btn--sm" type="button" @click="stats.reload()">重试</button>
    </div>

    <div v-else-if="stats.loading.value && !stats.data.value.recordCount" class="web-empty">
      <span class="web-muted">正在统计…</span>
    </div>

    <EmptyState
      v-else-if="stats.data.value.recordCount === 0"
      :title="selectedYear === 'all' ? '还没有统计' : '这一年还没有记录'"
      description="换个年份看看，或者先记一笔"
      action-text="去记一笔"
      @action="ui.openEntry()"
    />

    <template v-else>
      <div v-if="stats.error.value" class="web-alert web-alert--err stats-error">
        <span>{{ stats.error.value }}</span>
        <button class="web-btn web-btn--text web-btn--sm" type="button" @click="stats.reload()">
          重试
        </button>
      </div>

      <div class="web-stats stats-cards">
        <StatCard :value="formatCent(stats.data.value.inCent, { compact: true })" label="总收入" tone="in" />
        <StatCard :value="formatCent(stats.data.value.outCent, { compact: true })" label="总送出" tone="out" />
        <StatCard :value="formatCent(stats.data.value.netCent, { compact: true, sign: true })" label="净往来" />
        <StatCard :value="String(stats.data.value.recordCount)" label="记录笔数" />
        <StatCard :value="String(stats.data.value.personCount)" label="亲友人数" />
      </div>

      <section v-if="selectedYear !== 'all'" class="stats-section">
        <SectionHead title="月度往来" :sub="`${selectedYear} 年 · 双柱为收/送金额`">
          <template #tools>
            <div class="web-legend">
              <span><i class="legend-in" />收</span>
              <span><i class="legend-out" />送</span>
            </div>
          </template>
        </SectionHead>
        <div class="web-card web-chart">
          <svg :viewBox="`0 0 ${CHART_W} 220`" role="img" aria-label="月度收送金额柱状图">
            <line
              v-for="step in [0, 0.5, 1]"
              :key="step"
              x1="0"
              :y1="BASE - PLOT_H * step"
              :x2="CHART_W"
              :y2="BASE - PLOT_H * step"
              class="chart-grid"
            />
            <g v-for="(item, index) in monthly" :key="item.month">
              <rect
                v-if="item.inCent > 0"
                :x="index * SLOT + 14"
                :y="BASE - monthBar(item.inCent)"
                width="14"
                :height="monthBar(item.inCent)"
                rx="3"
                class="chart-bar chart-bar--in"
              >
                <title>{{ item.month }} 月收 {{ formatCent(item.inCent) }}</title>
              </rect>
              <rect
                v-if="item.outCent > 0"
                :x="index * SLOT + 32"
                :y="BASE - monthBar(item.outCent)"
                width="14"
                :height="monthBar(item.outCent)"
                rx="3"
                class="chart-bar chart-bar--out"
              >
                <title>{{ item.month }} 月送 {{ formatCent(item.outCent) }}</title>
              </rect>
              <text :x="index * SLOT + 30" y="200" text-anchor="middle" class="chart-label">
                {{ item.month }}月
              </text>
            </g>
          </svg>
        </div>
      </section>

      <section v-else class="web-alert web-alert--info stats-note">
        全部年份不显示按月走势，切换单一年份可查看。
      </section>

      <div class="web-grid-2e stats-grid">
        <section>
          <SectionHead title="按礼簿类别" />
          <div class="web-card">
            <div v-if="!categories.length" class="web-empty"><span class="web-muted">还没有礼簿数据</span></div>
            <div v-else class="web-bars">
              <div v-for="item in categories" :key="item.category" class="web-bar-row">
                <span>{{ item.label }}</span>
                <div class="web-bar-row__track">
                  <i class="web-bar-row__in" :style="{ width: categoryWidth(item.inCent) }" />
                  <i class="web-bar-row__out" :style="{ width: categoryWidth(item.outCent) }" />
                </div>
                <div class="web-bar-row__vals">
                  <div>收 <Money :cent="item.inCent" direction="in" size="s" /></div>
                  <div>送 <Money :cent="item.outCent" direction="out" size="s" /></div>
                </div>
              </div>
            </div>
          </div>
        </section>

        <section>
          <SectionHead title="按事项类型" />
          <div class="web-card">
            <div v-if="!eventTypes.length" class="web-empty"><span class="web-muted">还没有事项数据</span></div>
            <div v-else class="web-bars">
              <div v-for="item in eventTypes" :key="item.type" class="web-bar-row">
                <span class="type-label">
                  <EventSeal :type="item.type" size="s" />
                  {{ item.typeLabel }}
                </span>
                <div class="web-bar-row__track">
                  <i class="web-bar-row__in" :style="{ width: typeWidth(item.inCent) }" />
                  <i class="web-bar-row__out" :style="{ width: typeWidth(item.outCent) }" />
                </div>
                <div class="web-bar-row__vals">
                  <div>收 <Money :cent="item.inCent" direction="in" size="s" /></div>
                  <div>送 <Money :cent="item.outCent" direction="out" size="s" /></div>
                </div>
              </div>
            </div>
          </div>
        </section>
      </div>

      <section class="stats-section">
        <SectionHead title="往来最多的亲友" :sub="`共 ${personRows.length} 人`" />
        <div v-if="!personRows.length" class="web-card web-empty">
          <span class="web-muted">这一年还没有亲友往来</span>
        </div>
        <div v-else class="web-card stats-table">
          <table class="web-table">
            <thead>
              <tr>
                <th>亲友</th>
                <th class="web-table__r">收</th>
                <th class="web-table__r">送</th>
                <th class="web-table__r">净额</th>
                <th class="web-table__r">笔数</th>
                <th>最近往来</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="item in personRows" :key="item.person.id">
                <td>
                  <PersonCell
                    :id="item.person.id"
                    :name="item.person.name"
                    :sub="item.person.relationLabel"
                  />
                </td>
                <td class="web-table__r"><Money :cent="item.inCent" direction="in" size="s" /></td>
                <td class="web-table__r"><Money :cent="item.outCent" direction="out" size="s" /></td>
                <td class="web-table__r">
                  <Money :cent="item.netCent" :direction="netDirection(item.netCent)" sign size="s" />
                </td>
                <td class="web-table__r web-num">{{ item.recordCount }}</td>
                <td class="web-sm">{{ dateLabel(item.lastDate) }}</td>
              </tr>
            </tbody>
          </table>
        </div>
      </section>
    </template>
  </div>
</template>

<style lang="scss" scoped>
.stats-filter {
  margin: var(--sp-4) 0;
  overflow-x: auto;
  padding-bottom: 2px;
}

.stats-cards {
  margin-top: var(--sp-5);
}

.stats-error {
  margin-top: var(--sp-4);
  align-items: center;
  justify-content: space-between;
}

.stats-section {
  margin-top: var(--sp-6);
}

.stats-grid {
  margin-top: var(--sp-6);
}

.stats-grid section {
  min-width: 0;
}

.stats-note {
  margin-top: var(--sp-6);
}

.chart-grid {
  stroke: var(--c-line);
  stroke-dasharray: 3 4;
}

.chart-bar--in {
  fill: var(--c-in);
}

.chart-bar--out {
  fill: var(--c-out);
}

.chart-label {
  fill: var(--c-ink-3);
  font-size: 11px;
  font-family: $font-num;
}

.legend-in {
  background: var(--c-in);
}

.legend-out {
  background: var(--c-out);
}

.type-label {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  min-width: 0;
  font-size: 13px;
  white-space: nowrap;
  overflow: hidden;
}

.stats-table {
  overflow: hidden;
}
</style>
