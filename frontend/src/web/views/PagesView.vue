<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'

import { fetchBooks, fetchPaperPages } from '@/api/book'
import type { BookItem, PaperPage } from '@/api/types'
import EmptyState from '@/components/EmptyState.vue'
import Money from '@/components/Money.vue'
import PageHead from '@/web/components/PageHead.vue'
import SectionHead from '@/web/components/SectionHead.vue'
import StatCard from '@/web/components/StatCard.vue'
import { useAsync } from '@/web/composables/useAsync'
import { goWeb } from '@/web/nav'

interface PageRow {
  book: BookItem
  page: PaperPage
}

async function loadRows(): Promise<PageRow[]> {
  const books = await fetchBooks('normal')
  const grouped = await Promise.all(
    books.map(async (book) => {
      const pages = await fetchPaperPages(book.id)
      return pages.map((page) => ({ book, page }))
    }),
  )
  return grouped.flat()
}

const res = useAsync<PageRow[]>(loadRows, [])
const initialLoading = ref(true)
const rows = computed(() => res.data.value)
const bookCount = computed(() => new Set(rows.value.map((row) => row.book.id)).size)
const declaredCount = computed(() => rows.value.filter((row) => row.page.hasDeclared).length)
const mismatchCount = computed(
  () => rows.value.filter((row) => row.page.hasDeclared && !row.page.consistent).length,
)

function diffText(row: PageRow): string {
  if (!row.page.hasDeclared) return '未填'
  if (row.page.diffCent === 0) return '对上了'
  return `${row.page.diffCent > 0 ? '多' : '少'} ${(Math.abs(row.page.diffCent) / 100).toFixed(2)}`
}

function viewBook(row: PageRow): void {
  goWeb(`/sub-web/book?id=${row.book.id}&tab=records`, 'push')
}

onMounted(async () => {
  try {
    await res.reload()
  } finally {
    initialLoading.value = false
  }
})
</script>

<template>
  <div>
    <PageHead title="按页核对" :meta="`${bookCount} 本礼簿 · ${rows.length} 页`">
      <template #tools>
        <button class="web-btn web-btn--text" type="button" @click="res.reload()">刷新</button>
      </template>
    </PageHead>

    <div v-if="initialLoading" class="web-empty">
      <span class="web-muted">正在读取各本礼簿的纸本页…</span>
    </div>

    <div v-else-if="res.error.value && !rows.length" class="web-alert web-alert--err pages-error">
      <span>{{ res.error.value }}</span>
      <button class="web-btn web-btn--text web-btn--sm" type="button" @click="res.reload()">重试</button>
    </div>

    <div v-else-if="res.loading.value && !rows.length" class="web-empty">
      <span class="web-muted">正在读取各本礼簿的纸本页…</span>
    </div>

    <EmptyState
      v-else-if="!rows.length"
      title="还没有填纸本页码"
      description="连续录入或识别时填上纸本标记与本页合计，这里就能逐页对账"
    />

    <template v-else>
      <div class="web-stats pages-stats">
        <StatCard :value="String(bookCount)" label="礼簿数" />
        <StatCard :value="String(rows.length)" label="纸本页数" />
        <StatCard :value="String(declaredCount)" label="已填合计" />
        <StatCard :value="String(mismatchCount)" label="待查" :tone="mismatchCount ? 'pending' : ''" />
      </div>

      <section class="pages-section">
        <SectionHead title="跨礼簿核对" sub="纸本合计与录入合计对比；差额不为零说明漏录或金额录错" />
        <div class="web-card pages-card">
          <div class="web-table-wrap">
            <table class="web-table">
              <thead>
                <tr>
                  <th>礼簿</th>
                  <th>纸本标记</th>
                  <th class="web-table__r">笔数</th>
                  <th class="web-table__r">纸本合计</th>
                  <th class="web-table__r">录入合计</th>
                  <th class="web-table__r">差额</th>
                  <th class="web-table__r">操作</th>
                </tr>
              </thead>
              <tbody>
                <tr
                  v-for="row in rows"
                  :key="`${row.book.id}-${row.page.page}`"
                  class="web-table__row"
                  @click="viewBook(row)"
                >
                  <td>
                    <div class="pages-book">
                      <span class="pages-book__name">{{ row.book.name }}</span>
                      <small class="web-muted">{{ row.book.categoryLabel }}</small>
                    </div>
                  </td>
                  <td class="web-num">{{ row.page.page }}</td>
                  <td class="web-table__r web-num">{{ row.page.recordCount }}</td>
                  <td class="web-table__r">
                    <Money v-if="row.page.hasDeclared" :cent="row.page.declaredCent" direction="none" size="s" />
                    <span v-else class="web-muted">未填</span>
                  </td>
                  <td class="web-table__r">
                    <Money :cent="row.page.recordCent" direction="none" size="s" />
                  </td>
                  <td class="web-table__r">
                    <span v-if="!row.page.hasDeclared" class="web-muted">—</span>
                    <span v-else-if="row.page.diffCent === 0" class="pages-diff pages-diff--ok">对上了</span>
                    <span v-else class="pages-diff pages-diff--err">{{ diffText(row) }}</span>
                  </td>
                  <td class="web-table__r" @click.stop>
                    <button class="web-btn web-btn--text web-btn--sm" type="button" @click="viewBook(row)">
                      查看
                    </button>
                  </td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>
      </section>
    </template>
  </div>
</template>

<style lang="scss" scoped>
.pages-error {
  margin-top: var(--sp-4);
  align-items: center;
  justify-content: space-between;
}

.pages-stats {
  margin-top: var(--sp-4);
}

.pages-section {
  margin-top: var(--sp-6);
}

.pages-card {
  overflow: hidden;
}

.pages-book {
  display: flex;
  flex-direction: column;
  gap: 1px;
}

.pages-book__name {
  font-weight: 500;
  color: var(--c-ink-1);
}

.pages-book small {
  font-size: 12px;
}

.pages-diff--ok {
  color: var(--c-repaid);
  font-weight: 600;
}

.pages-diff--err {
  color: var(--c-danger);
}
</style>
