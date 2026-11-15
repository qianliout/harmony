<script setup lang="ts">
import { computed, onMounted, watch } from 'vue'

import { fetchRecords } from '@/api/record'
import { fetchRecentEvents } from '@/api/event'
import { fetchYearStats, fetchYears } from '@/api/stats'
import type { EventItem, PendingList, RecordItem, YearStats } from '@/api/types'
import { fetchPending } from '@/api/view'
import EventSeal from '@/components/EventSeal.vue'
import PersonAvatar from '@/components/PersonAvatar.vue'
import { useAuthStore } from '@/stores/auth'
import { useWebUiStore } from '@/stores/web'
import { useAsync } from '@/web/composables/useAsync'
import PageHead from '@/web/components/PageHead.vue'
import RecordTable from '@/web/components/RecordTable.vue'
import SectionHead from '@/web/components/SectionHead.vue'
import WebIcon from '@/web/components/WebIcon.vue'
import { goWeb } from '@/web/nav'
import { formatDate, formatFullDate, relativeSince, todayString, weekdayOf } from '@/utils/date'
import { formatCent } from '@/utils/money'

interface HomeData {
  year: number
  stats: YearStats | null
  recent: RecordItem[]
  pending: PendingList | null
  events: EventItem[]
}

const ui = useWebUiStore()
const auth = useAuthStore()

const today = todayString()
const todayLabel = `${formatFullDate(today)} · ${weekdayOf(today)}`

/** 年份默认当年；当年没有记录时退到最近一个有数据的年份。 */
async function loadHome(): Promise<HomeData> {
  const currentYear = new Date().getFullYear()
  const [years, recentPage, pending, events] = await Promise.all([
    fetchYears(),
    fetchRecords({ pageSize: 7 }),
    fetchPending('owed'),
    fetchRecentEvents(4),
  ])
  let stats = await fetchYearStats(currentYear)
  if (stats.recordCount === 0) {
    const latest = years.find((item) => item !== currentYear)
    if (latest) {
      stats = await fetchYearStats(latest)
    }
  }
  return {
    year: stats.year || currentYear,
    stats,
    recent: recentPage.items,
    pending,
    events,
  }
}

const home = useAsync<HomeData>(loadHome, {
  year: new Date().getFullYear(),
  stats: null,
  recent: [],
  pending: null,
  events: [],
})

const stats = computed(() => home.data.value.stats)
const yearLabel = computed(() => home.data.value.year)
const recentRecords = computed(() => home.data.value.recent)
const duePersons = computed(() => home.data.value.pending?.persons.slice(0, 5) ?? [])
const recentEvents = computed(() => home.data.value.events)
const dueCount = computed(() => home.data.value.pending?.totalCount ?? 0)
const dueCent = computed(() => home.data.value.pending?.totalCent ?? 0)
const isEmpty = computed(
  () =>
    !home.loading.value &&
    !home.error.value &&
    !recentRecords.value.length &&
    !duePersons.value.length &&
    !recentEvents.value.length,
)

function openRecord(record: RecordItem): void {
  ui.openRecord(record.id)
}

function openPerson(id: number): void {
  goWeb(`/sub-web/person?id=${id}`, 'push')
}

function openEvent(id: number): void {
  goWeb(`/sub-web/event?id=${id}`, 'push')
}

function eventAmount(event: EventItem): string {
  return formatCent(event.inCent || event.outCent)
}

onMounted(() => {
  void home.reload()
})

watch(
  () => ui.revision,
  () => {
    void home.reload()
  },
)
</script>

<template>
  <PageHead title="工作台" :meta="todayLabel">
    <template #tools>
      <button class="web-btn web-btn--primary" type="button" @click="ui.openEntry()">
        <WebIcon name="plus" :size="16" />
        记一笔
      </button>
    </template>
  </PageHead>

  <div v-if="home.error.value" class="web-alert web-alert--err">
    <WebIcon name="alert" :size="16" />
    <span>{{ home.error.value }}</span>
    <button class="web-btn web-btn--sm" type="button" @click="home.reload()">重试</button>
  </div>

  <!-- 朱砂主卡：只显示当年收、送、净差额与全部未回礼 -->
  <section class="web-hero">
    <div>
      <div class="web-hero__greet">
        礼尚往来
        <small>{{ yearLabel }} 年 · {{ home.loading.value ? '加载中…' : '人情往来一览' }}</small>
      </div>
      <div class="web-hero__figs">
        <div>
          <div class="web-hero__k">{{ yearLabel }} 年收礼</div>
          <div class="web-hero__v">{{ stats ? formatCent(stats.inCent) : '—' }}</div>
        </div>
        <div>
          <div class="web-hero__k">{{ yearLabel }} 年送礼</div>
          <div class="web-hero__v">{{ stats ? formatCent(stats.outCent) : '—' }}</div>
        </div>
        <div>
          <div class="web-hero__k">净差额</div>
          <div class="web-hero__v web-hero__v--s">
            {{ stats ? formatCent(stats.netCent, { sign: true }) : '—' }}
          </div>
        </div>
      </div>
      <button class="web-hero__owe home-owe" type="button" @click="goWeb('/sub-web/repay')">
        <WebIcon name="return" :size="15" />
        全部未回礼 {{ dueCount }} 笔 · {{ formatCent(dueCent) }}
        <WebIcon name="right" :size="14" />
      </button>
    </div>
    <div class="web-hero__acts">
      <button class="web-btn web-btn--primary" type="button" @click="ui.openEntry()">
        <WebIcon name="plus" :size="16" />
        记一笔
      </button>
      <button class="web-btn" type="button" @click="goWeb('/sub-web/repay')">
        <WebIcon name="return" :size="16" />
        回礼清单
      </button>
      <button v-if="auth.isAdmin" class="web-btn" type="button" @click="goWeb('/sub-web/recognize')">
        <WebIcon name="scan" :size="16" />
        识别礼簿
      </button>
      <button class="web-btn" type="button" @click="goWeb('/sub-web/settings')">
        <WebIcon name="download" :size="16" />
        导入导出
      </button>
    </div>
  </section>

  <div v-if="!home.loading.value && isEmpty" class="web-card">
    <div class="web-empty">
      还没有人情往来记录，点「记一笔」开始第一笔。
    </div>
  </div>

  <div v-else class="web-grid-2">
    <section class="web-section">
      <SectionHead title="最近记录" sub="按发生日期">
        <template #tools>
          <button class="web-btn web-btn--text web-btn--sm" type="button" @click="goWeb('/sub-web/records')">
            全部记录
            <WebIcon name="right" :size="14" />
          </button>
        </template>
      </SectionHead>
      <div class="web-card">
        <RecordTable
          v-if="recentRecords.length"
          :records="recentRecords"
          :columns="['date', 'person', 'event', 'book', 'amount', 'status']"
          @row="openRecord"
          @person="openPerson"
        />
        <div v-else class="web-empty">{{ home.loading.value ? '加载中…' : '还没有记录' }}</div>
      </div>
    </section>

    <div class="home-side">
      <section class="web-section">
        <SectionHead title="最久没回的" :sub="`共 ${dueCount} 笔`">
          <template #tools>
            <button class="web-btn web-btn--text web-btn--sm" type="button" @click="goWeb('/sub-web/repay')">
              回礼清单
              <WebIcon name="right" :size="14" />
            </button>
          </template>
        </SectionHead>
        <div class="web-card web-list">
          <button
            v-for="item in duePersons"
            :key="item.person.id"
            class="web-list__row home-row"
            type="button"
            @click="openPerson(item.person.id)"
          >
            <PersonAvatar :id="item.person.id" :name="item.person.name" :size="28" />
            <span class="web-list__grow">
              <span class="home-row__name">{{ item.person.name }}</span>
              <small>{{ item.count }} 笔 · 最早 {{ relativeSince(item.earliest) || '—' }}</small>
            </span>
            <span class="web-amt web-amt--pend">{{ formatCent(item.totalCent) }}</span>
          </button>
          <div v-if="!duePersons.length" class="web-empty">
            {{ home.loading.value ? '加载中…' : '没有待回礼的往来' }}
          </div>
        </div>
      </section>

      <section class="web-section">
        <SectionHead title="近期事项">
          <template #tools>
            <button class="web-btn web-btn--text web-btn--sm" type="button" @click="goWeb('/sub-web/events')">
              全部事项
              <WebIcon name="right" :size="14" />
            </button>
          </template>
        </SectionHead>
        <div class="web-card web-list">
          <button
            v-for="event in recentEvents"
            :key="event.id"
            class="web-list__row home-row"
            type="button"
            @click="openEvent(event.id)"
          >
            <EventSeal :type="event.type" size="s" />
            <span class="web-list__grow">
              <span class="home-row__name">{{ event.name }}</span>
              <small>{{ formatDate(event.eventDate) || '未注明' }} · {{ event.hostLabel || event.host }}</small>
            </span>
            <span class="web-amt" :class="event.inCent ? 'web-amt--in' : 'web-amt--out'">
              {{ eventAmount(event) }}
            </span>
          </button>
          <div v-if="!recentEvents.length" class="web-empty">
            {{ home.loading.value ? '加载中…' : '还没有事项' }}
          </div>
        </div>
      </section>
    </div>
  </div>
</template>

<style lang="scss" scoped>
.home-side {
  display: flex;
  flex-direction: column;
  gap: var(--sp-7);
  min-width: 0;
}

.home-owe {
  border: 0;
  color: inherit;
  font: inherit;
  cursor: pointer;
}

.home-row {
  width: 100%;
  border: 0;
  border-bottom: 1px solid var(--c-divider);
  background: transparent;
  color: inherit;
  font: inherit;
  text-align: left;
  cursor: pointer;
}

.home-row:last-child {
  border-bottom: 0;
}

.home-row:hover {
  background: var(--c-hover);
}

.home-row__name {
  display: block;
  overflow: hidden;
  color: var(--c-ink-1);
  font-size: 14px;
  line-height: 20px;
  text-overflow: ellipsis;
  white-space: nowrap;
}
</style>
