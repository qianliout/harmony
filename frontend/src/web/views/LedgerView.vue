<script setup lang="ts">
// 总账：所有礼簿合并后的全部收送记录，可按礼簿、方向、日期、关键词筛选，顶部显示总收、总送、笔数。
import { onBeforeUnmount, onMounted, ref, watch } from 'vue'

import { fetchBooks } from '@/api/book'
import type { BookItem, RecordItem } from '@/api/types'
import { useWebUiStore } from '@/stores/web'
import { formatCent } from '@/utils/money'
import DateField from '@/web/components/DateField.vue'
import PageHead from '@/web/components/PageHead.vue'
import Pager from '@/web/components/Pager.vue'
import RecordTable from '@/web/components/RecordTable.vue'
import Segmented from '@/web/components/Segmented.vue'
import StatCard from '@/web/components/StatCard.vue'
import WebIcon from '@/web/components/WebIcon.vue'
import { useRecordList } from '@/web/composables/useRecordList'
import { goWeb } from '@/web/nav'

const ui = useWebUiStore()
const list = useRecordList()
const books = ref<BookItem[]>([])
const keywordInput = ref('')
let keywordTimer: ReturnType<typeof setTimeout> | undefined

const directionOptions = [
  { value: '', label: '全部' },
  { value: 'in', label: '收' },
  { value: 'out', label: '送' },
]

/** 关键字 250ms 防抖，避免每敲一个字打一次请求。 */
watch(keywordInput, (value) => {
  if (keywordTimer) clearTimeout(keywordTimer)
  keywordTimer = setTimeout(() => {
    list.filter.keyword = value
  }, 250)
})

function clearFilters(): void {
  if (keywordTimer) clearTimeout(keywordTimer)
  keywordInput.value = ''
  list.reset()
}

function openPerson(id: number): void {
  goWeb(`/sub-web/person?id=${id}`, 'push')
}

function editRecord(record: RecordItem): void {
  ui.openEntry({ recordId: record.id })
}

onMounted(() => {
  void list.reload()
  fetchBooks()
    .then((page) => {
      books.value = page.items
    })
    .catch(() => {
      books.value = []
    })
})

onBeforeUnmount(() => {
  if (keywordTimer) clearTimeout(keywordTimer)
})
</script>

<template>
  <PageHead title="总账" meta="所有礼簿合并，收送都在这里">
    <template #tools>
      <button class="web-btn" type="button" @click="goWeb('/sub-web/recognize')">
        <WebIcon name="scan" :size="16" />
        识别礼簿
      </button>
      <button class="web-btn web-btn--primary" type="button" @click="ui.openEntry()">
        <WebIcon name="plus" :size="16" />
        记一笔
      </button>
    </template>
  </PageHead>

  <div class="web-stats">
    <StatCard :value="formatCent(list.summary.value.inCent)" label="总收" tone="in" />
    <StatCard :value="formatCent(list.summary.value.outCent)" label="总送" tone="out" />
    <StatCard :value="`${list.summary.value.count} 笔`" label="笔数" />
  </div>

  <div class="web-card">
    <div class="web-filters">
      <select v-model.number="list.filter.bookId" class="web-select" aria-label="礼簿">
        <option :value="0">全部礼簿</option>
        <option v-for="book in books" :key="book.id" :value="book.id">{{ book.name }}</option>
      </select>
      <input
        v-model="keywordInput"
        class="web-input ledger-keyword"
        type="search"
        placeholder="亲友姓名、拼音、备注或事情"
        aria-label="关键词"
      />
      <DateField v-model="list.filter.from" placeholder="开始日期" clearable />
      <span class="web-muted">至</span>
      <DateField v-model="list.filter.to" placeholder="结束日期" clearable />
      <span class="web-filters__gap" />
      <span class="web-filters__lbl">方向</span>
      <Segmented v-model="list.filter.direction" :options="directionOptions" variant="dir" />
      <div class="web-filters__right">
        <button v-if="list.hasFilters.value" class="web-btn web-btn--text web-btn--sm" type="button" @click="clearFilters">
          清除筛选
        </button>
      </div>
    </div>

    <div v-if="list.error.value" class="web-alert web-alert--err ledger-alert">
      <WebIcon name="alert" :size="16" />
      <span>{{ list.error.value }}</span>
      <button class="web-btn web-btn--sm" type="button" @click="list.reload()">重试</button>
    </div>
    <RecordTable
      v-else-if="list.items.value.length"
      :records="list.items.value"
      :columns="['date', 'direction', 'person', 'place', 'amount', 'note', 'ops']"
      @row="ui.openRecord($event.id)"
      @person="openPerson"
      @edit="editRecord"
      @remove="list.remove"
    />
    <div v-else class="web-empty">{{ list.loading.value ? '加载中…' : '没有符合条件的记录' }}</div>

    <div class="web-sumbar">
      <span>{{ list.total.value }} 笔</span>
      <Pager :page="list.page.value" :pages="list.pageCount.value" @change="list.page.value = $event" />
    </div>
  </div>
</template>

<style lang="scss" scoped>
.ledger-keyword {
  width: 220px;
}

.ledger-alert {
  margin: var(--sp-4);
}

.ledger-alert .web-btn {
  margin-left: auto;
}
</style>
