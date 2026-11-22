<script setup lang="ts">
// 礼簿详情：礼簿信息与这本礼簿里的全部收礼记录。
import { onMounted, ref, watch } from 'vue'

import { deleteBook, fetchBookDetail } from '@/api/book'
import type { BookItem, RecordItem } from '@/api/types'
import { useWebUiStore } from '@/stores/web'
import { formatFullDate } from '@/utils/date'
import { formatCent } from '@/utils/money'
import PageHead from '@/web/components/PageHead.vue'
import Pager from '@/web/components/Pager.vue'
import RecordTable from '@/web/components/RecordTable.vue'
import StatCard from '@/web/components/StatCard.vue'
import WebIcon from '@/web/components/WebIcon.vue'
import { useAsync } from '@/web/composables/useAsync'
import { useRecordList } from '@/web/composables/useRecordList'
import { backWeb, goWeb, readPageId } from '@/web/nav'
import BookDialog from '@/web/overlays/BookDialog.vue'

const ui = useWebUiStore()
const bookId = ref(0)
const dialogOpen = ref(false)

const detail = useAsync<BookItem | null>(() => fetchBookDetail(bookId.value), null)
const list = useRecordList(() => ({ bookId: bookId.value, direction: 'in' }))

function onSaved(): void {
  dialogOpen.value = false
  ui.toast('礼簿已保存')
  ui.markChanged()
}

async function remove(): Promise<void> {
  const book = detail.data.value
  if (!book || !window.confirm(`删除礼簿「${book.name}」？`)) return
  try {
    await deleteBook(book.id)
    ui.toast('礼簿已删除')
    backWeb()
  } catch (err) {
    ui.toast(err instanceof Error ? err.message : '删除失败')
  }
}

function editRecord(record: RecordItem): void {
  ui.openEntry({ recordId: record.id })
}

watch(
  () => ui.revision,
  () => void detail.reload(),
)

onMounted(() => {
  bookId.value = readPageId()
  // 记录列表随 bookId 变化自动加载。
  void detail.reload()
})
</script>

<template>
  <div v-if="detail.error.value" class="web-alert web-alert--err">
    <WebIcon name="alert" />
    <span>{{ detail.error.value }}</span>
  </div>
  <template v-else-if="detail.data.value">
    <PageHead :title="detail.data.value.name" :meta="detail.data.value.categoryLabel">
      <template #tools>
        <button class="web-btn" type="button" @click="dialogOpen = true">
          <WebIcon name="edit" />
          编辑
        </button>
        <button class="web-btn web-btn--danger" type="button" @click="remove">
          <WebIcon name="trash" />
          删除
        </button>
        <button class="web-btn web-btn--primary" type="button" @click="ui.openEntry({ bookId })">
          <WebIcon name="plus" />
          记一笔
        </button>
      </template>
    </PageHead>

    <div class="web-card book-info">
      <dl class="web-dl">
        <dt>日期</dt>
        <dd>{{ formatFullDate(detail.data.value.occurredOn) || '未填写' }}</dd>
        <dt>备注</dt>
        <dd>{{ detail.data.value.description || '—' }}</dd>
      </dl>
    </div>

    <div class="web-stats">
      <StatCard :value="formatCent(list.summary.value.inCent)" label="收礼合计" tone="in" />
      <StatCard :value="`${list.summary.value.count} 笔`" label="收礼笔数" />
    </div>

    <div class="web-card">
      <RecordTable
        v-if="list.items.value.length"
        :records="list.items.value"
        :columns="['date', 'person', 'amount', 'note', 'ops']"
        @row="ui.openRecord($event.id)"
        @person="goWeb(`/sub-web/person?id=${$event}`, 'push')"
        @edit="editRecord"
        @remove="list.remove"
      />
      <div v-else class="web-empty">{{ list.loading.value ? '加载中…' : '这本礼簿还没有记录' }}</div>
      <div class="web-sumbar">
        <span>{{ list.total.value }} 笔</span>
        <Pager :page="list.page.value" :pages="list.pageCount.value" @change="list.page.value = $event" />
      </div>
    </div>
  </template>
  <div v-else class="web-empty">加载中…</div>

  <BookDialog v-if="dialogOpen" :book="detail.data.value" @close="dialogOpen = false" @saved="onSaved" />
</template>

<style lang="scss" scoped>
.book-info {
  padding: var(--sp-4) var(--sp-5);
}
</style>
