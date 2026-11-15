<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'

import { fetchRecords, type RecordPage } from '@/api/record'
import { fetchYears } from '@/api/stats'
import { exportRecords } from '@/api/transfer'
import type { RecordItem } from '@/api/types'
import { useDictStore } from '@/stores/dict'
import { useWebUiStore } from '@/stores/web'
import { eventTypes } from '@/styles/palette'
import { useAsync } from '@/web/composables/useAsync'
import { useRecordQuery } from '@/web/composables/useRecordQuery'
import PageHead from '@/web/components/PageHead.vue'
import Pager from '@/web/components/Pager.vue'
import RecordTable from '@/web/components/RecordTable.vue'
import Segmented from '@/web/components/Segmented.vue'
import WebIcon from '@/web/components/WebIcon.vue'
import { goWeb } from '@/web/nav'
import { formatCent } from '@/utils/money'

const ui = useWebUiStore()
const dictStore = useDictStore()
const query = useRecordQuery({ page: 1 })

const pageSize = 20
const keywordInput = ref(query.filters.personKeyword)
const years = ref<number[]>([])
let keywordTimer: ReturnType<typeof setTimeout> | undefined

function emptyPage(): RecordPage {
  return {
    items: [],
    total: 0,
    summary: { count: 0, inCent: 0, outCent: 0, pendingCent: 0 },
  }
}

const page = useAsync<RecordPage>(() => fetchRecords(query.params.value), emptyPage())

const items = computed(() => page.data.value.items)
const summary = computed(() => page.data.value.summary)
const pageCount = computed(() => Math.max(1, Math.ceil(page.data.value.total / pageSize)))

const directionOptions = [
  { value: '', label: '全部' },
  { value: 'in', label: '收' },
  { value: 'out', label: '送' },
]

const statusOptions = computed(() => {
  const labels =
    query.filters.direction === 'out'
      ? { pending: '对方未回', repaid: '对方已回', none: '不用回' }
      : { pending: '未回礼', repaid: '已回礼', none: '不用回' }
  return [
    { value: '', label: '回礼状态' },
    { value: 'pending', label: labels.pending },
    { value: 'repaid', label: labels.repaid },
    { value: 'none', label: labels.none },
  ]
})

/** 除分页外的筛选变化都回到第一页。 */
const resetKeys = ['bookCategory', 'eventType', 'year', 'direction', 'status', 'itemKind'] as const
for (const key of resetKeys) {
  watch(
    () => query.filters[key],
    () => {
      query.filters.page = 1
    },
  )
}

watch(
  () => query.params.value,
  () => {
    void page.reload()
  },
  { deep: true },
)

/** 亲友关键字 250ms 防抖，避免每敲一个字打一次请求。 */
watch(keywordInput, (value) => {
  if (keywordTimer) clearTimeout(keywordTimer)
  keywordTimer = setTimeout(() => {
    query.filters.personKeyword = value
    query.filters.page = 1
  }, 250)
})

function clearFilters(): void {
  if (keywordTimer) clearTimeout(keywordTimer)
  query.reset()
  keywordInput.value = ''
}

function changePage(next: number): void {
  query.changePage(next)
}

function openRecord(record: RecordItem): void {
  ui.openRecord(record.id)
}

function openPerson(id: number): void {
  goWeb(`/sub-web/person?id=${id}`, 'push')
}

function exportResult(): void {
  if (!items.value.length) {
    ui.toast('当前筛选没有记录')
    return
  }
  exportRecords({
    format: 'csv',
    bookCategory: query.filters.bookCategory || undefined,
    eventType: query.filters.eventType || undefined,
    itemKind: query.filters.itemKind || undefined,
    personKeyword: query.filters.personKeyword.trim() || undefined,
    direction: query.filters.direction || undefined,
    status: query.filters.status || undefined,
    from: query.params.value.from,
    to: query.params.value.to,
  })
  ui.toast('已开始导出当前筛选结果')
}

onMounted(() => {
  void page.reload()
  void dictStore.ensure()
  fetchYears()
    .then((list) => {
      years.value = list
    })
    .catch(() => {
      years.value = []
    })
})

onBeforeUnmount(() => {
  if (keywordTimer) clearTimeout(keywordTimer)
})

watch(
  () => ui.revision,
  () => {
    void page.reload()
  },
)
</script>

<template>
  <PageHead title="全部记录" meta="跨全部礼簿；已合并的礼簿按目标礼簿显示">
    <template #tools>
      <button class="web-btn" type="button" @click="exportResult">
        <WebIcon name="download" :size="16" />
        导出当前结果
      </button>
      <button class="web-btn web-btn--primary" type="button" @click="ui.openEntry()">
        <WebIcon name="plus" :size="16" />
        记一笔
      </button>
    </template>
  </PageHead>

  <div class="web-card">
    <div class="web-filters">
      <select v-model="query.filters.bookCategory" class="web-select" aria-label="礼簿类别">
        <option value="">礼簿类别</option>
        <option v-for="item in dictStore.categories" :key="item.id" :value="item.code">
          {{ item.label }}
        </option>
      </select>

      <select v-model="query.filters.eventType" class="web-select" aria-label="事项类型">
        <option value="">事项类型</option>
        <option v-for="type in eventTypes" :key="type.code" :value="type.code">{{ type.label }}</option>
      </select>

      <input
        v-model="keywordInput"
        class="web-input"
        style="width: 160px"
        type="search"
        placeholder="亲友姓名或拼音"
        aria-label="亲友"
      />

      <select v-model="query.filters.year" class="web-select" aria-label="年份">
        <option value="">全部年份</option>
        <option v-for="year in years" :key="year" :value="String(year)">{{ year }} 年</option>
      </select>

      <span class="web-filters__gap" />

      <span class="web-filters__lbl">方向</span>
      <Segmented
        v-model="query.filters.direction"
        :options="directionOptions"
        variant="dir"
      />

      <select v-model="query.filters.status" class="web-select" aria-label="回礼状态">
        <option v-for="option in statusOptions" :key="option.value" :value="option.value">
          {{ option.label }}
        </option>
      </select>

      <select v-model="query.filters.itemKind" class="web-select" aria-label="名目">
        <option value="">名目</option>
        <option v-for="item in dictStore.itemKinds" :key="item.id" :value="item.code">
          {{ item.label }}
        </option>
      </select>

      <div class="web-filters__right">
        <button
          v-if="query.hasFilters.value"
          class="web-btn web-btn--text web-btn--sm"
          type="button"
          @click="clearFilters"
        >
          清除筛选
        </button>
      </div>
    </div>

    <div v-if="page.error.value" class="web-alert web-alert--err record-alert">
      <WebIcon name="alert" :size="16" />
      <span>{{ page.error.value }}</span>
      <button class="web-btn web-btn--sm" type="button" @click="page.reload()">重试</button>
    </div>

    <RecordTable
      v-else-if="items.length"
      :records="items"
      :columns="['date', 'person', 'event', 'book', 'cat', 'amount', 'status', 'page', 'anno']"
      @row="openRecord"
      @person="openPerson"
    />

    <div v-else class="web-empty">
      {{ page.loading.value ? '加载中…' : '没有符合条件的记录' }}
    </div>

    <div class="web-sumbar">
      <span>{{ summary.count }} 笔</span>
      <span>收 <b class="web-amt web-amt--in">{{ formatCent(summary.inCent) }}</b></span>
      <span>送 <b class="web-amt web-amt--out">{{ formatCent(summary.outCent) }}</b></span>
      <span>其中未回礼 <b class="web-amt web-amt--pend">{{ formatCent(summary.pendingCent) }}</b></span>
      <Pager :page="query.filters.page" :pages="pageCount" @change="changePage" />
    </div>
  </div>
</template>

<style lang="scss" scoped>
.record-alert {
  margin: var(--sp-4);
}

.record-alert .web-btn {
  margin-left: auto;
}
</style>
