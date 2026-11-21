<script setup lang="ts">
// 礼簿列表：按名称、种类查询；新建、编辑、删除礼簿。
import { computed, onBeforeUnmount, ref, watch } from 'vue'

import { deleteBook, fetchBooks } from '@/api/book'
import type { BookItem, Page } from '@/api/types'
import { useWebUiStore } from '@/stores/web'
import { bookCategories, bookCategoryColor, paletteOf } from '@/styles/palette'
import { formatCent } from '@/utils/money'
import PageHead from '@/web/components/PageHead.vue'
import WebIcon from '@/web/components/WebIcon.vue'
import WebTable from '@/web/components/WebTable.vue'
import { useAsync } from '@/web/composables/useAsync'
import { goWeb } from '@/web/nav'
import BookDialog from '@/web/overlays/BookDialog.vue'

const ui = useWebUiStore()
const keywordInput = ref('')
const keyword = ref('')
const category = ref('')
const dialogOpen = ref(false)
const editing = ref<BookItem | null>(null)
let keywordTimer: ReturnType<typeof setTimeout> | undefined

const listing = useAsync<Page<BookItem>>(
  () => fetchBooks({ keyword: keyword.value || undefined, category: category.value || undefined }),
  { items: [], total: 0 },
)
const books = computed(() => listing.data.value.items)

function categoryStyle(code: string): Record<string, string> {
  const color = paletteOf(bookCategoryColor(code))
  return { background: color.light, color: color.main }
}

function openCreate(): void {
  editing.value = null
  dialogOpen.value = true
}

function openEdit(book: BookItem): void {
  editing.value = book
  dialogOpen.value = true
}

function onSaved(): void {
  ui.toast(editing.value ? '礼簿已保存' : '礼簿已新建')
  dialogOpen.value = false
  ui.markChanged()
}

async function remove(book: BookItem): Promise<void> {
  if (!window.confirm(`删除礼簿「${book.name}」？`)) return
  try {
    await deleteBook(book.id)
    ui.toast('礼簿已删除')
    ui.markChanged()
  } catch (err) {
    ui.toast(err instanceof Error ? err.message : '删除失败')
  }
}

watch(keywordInput, (value) => {
  if (keywordTimer) clearTimeout(keywordTimer)
  keywordTimer = setTimeout(() => {
    keyword.value = value.trim()
  }, 250)
})
watch([keyword, category], () => void listing.reload())
watch(
  () => ui.revision,
  () => void listing.reload(),
)
onBeforeUnmount(() => {
  if (keywordTimer) clearTimeout(keywordTimer)
})

void listing.reload()
</script>

<template>
  <PageHead title="礼簿" :meta="`${listing.data.value.total} 本`">
    <template #tools>
      <button class="web-btn web-btn--primary" type="button" @click="openCreate">
        <WebIcon name="plus" />
        新建礼簿
      </button>
    </template>
  </PageHead>

  <div class="web-card">
    <div class="web-filters">
      <input v-model="keywordInput" class="web-input books-keyword" type="search" placeholder="礼簿名称或备注" aria-label="搜索礼簿" />
      <select v-model="category" class="web-select" aria-label="种类">
        <option value="">全部种类</option>
        <option v-for="item in bookCategories" :key="item.code" :value="item.code">{{ item.label }}</option>
      </select>
    </div>

    <div v-if="listing.error.value" class="web-alert web-alert--err books-alert">
      <WebIcon name="alert" />
      <span>{{ listing.error.value }}</span>
      <button class="web-btn web-btn--sm" type="button" @click="listing.reload()">重试</button>
    </div>
    <WebTable v-else>
      <thead>
        <tr>
          <th>礼簿</th>
          <th>种类</th>
          <th class="web-table__r">收礼笔数</th>
          <th class="web-table__r">收礼合计</th>
          <th class="web-table__r">操作</th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="item in books" :key="item.id" class="web-table__row" @click="goWeb(`/sub-web/book?id=${item.id}`, 'push')">
          <td>
            <div class="web-cell2">
              <WebIcon name="book" />
              <span class="web-cell2__t">
                {{ item.name }}
                <small v-if="item.description">{{ item.description }}</small>
              </span>
            </div>
          </td>
          <td><span class="web-pill" :style="categoryStyle(item.category)">{{ item.categoryLabel }}</span></td>
          <td class="web-table__r web-num">{{ item.recordCount }}</td>
          <td class="web-table__r web-amt web-amt--in">{{ formatCent(item.inCent) }}</td>
          <td class="web-table__r" @click.stop>
            <button class="web-btn web-btn--text web-btn--sm" type="button" @click="openEdit(item)">编辑</button>
            <button class="web-btn web-btn--text web-btn--sm" type="button" @click="remove(item)">删除</button>
          </td>
        </tr>
      </tbody>
    </WebTable>
    <div v-if="!listing.error.value && !books.length" class="web-empty">
      {{ listing.loading.value ? '正在加载礼簿…' : '还没有礼簿，先新建一本。' }}
    </div>
  </div>

  <BookDialog v-if="dialogOpen" :book="editing" @close="dialogOpen = false" @saved="onSaved" />
</template>

<style lang="scss" scoped>
.books-keyword {
  width: 220px;
}

.books-alert {
  margin: var(--sp-4);
}

.books-alert .web-btn {
  margin-left: auto;
}
</style>
