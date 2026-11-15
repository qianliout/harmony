<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'

import { createBook, fetchBooks } from '@/api/book'
import { fetchEvents } from '@/api/event'
import type { BookItem, EventItem } from '@/api/types'
import { useDictStore } from '@/stores/dict'
import { useWebUiStore } from '@/stores/web'
import { paletteOf } from '@/styles/palette'
import { formatCent } from '@/utils/money'
import Modal from '@/web/components/Modal.vue'
import PageHead from '@/web/components/PageHead.vue'
import SectionHead from '@/web/components/SectionHead.vue'
import WebIcon from '@/web/components/WebIcon.vue'
import WebTable from '@/web/components/WebTable.vue'
import { useAsync } from '@/web/composables/useAsync'
import { goWeb } from '@/web/nav'

const ui = useWebUiStore()
const dictStore = useDictStore()

const events = ref<EventItem[]>([])
const formOpen = ref(false)
const saving = ref(false)
const form = reactive({
  name: '',
  category: '',
  defaultEventId: 0,
  description: '',
})

const listing = useAsync<{ normal: BookItem[]; merged: BookItem[] }>(
  async () => {
    const [normal, merged] = await Promise.all([fetchBooks('normal'), fetchBooks('merged')])
    return { normal, merged }
  },
  { normal: [], merged: [] },
)

const normalBooks = computed(() => listing.data.value.normal)
const mergedBooks = computed(() => listing.data.value.merged)
const loading = computed(() => listing.loading.value)
const error = computed(() => listing.error.value)
const meta = computed(() => {
  const parts = [`${normalBooks.value.length} 本常用礼簿`]
  if (mergedBooks.value.length) parts.push(`${mergedBooks.value.length} 本已合并`)
  return parts.join(' · ')
})

function openBook(item: BookItem): void {
  goWeb(`/sub-web/book?id=${item.id}`, 'push')
}

function categoryStyle(code: string): Record<string, string> {
  const color = paletteOf(dictStore.categoryColor(code))
  return { background: color.light, color: color.main }
}

async function loadEvents(): Promise<void> {
  try {
    const result = await fetchEvents({ pageSize: 100 })
    events.value = result.items
  } catch {
    events.value = []
  }
}

function openCreate(): void {
  form.name = ''
  form.category = dictStore.categories[0]?.code ?? ''
  form.defaultEventId = 0
  form.description = ''
  formOpen.value = true
}

async function save(): Promise<void> {
  if (saving.value) return
  const name = form.name.trim()
  if (!name) {
    ui.toast('请填写礼簿名称')
    return
  }
  if (!form.category) {
    ui.toast('请选择礼簿类别')
    return
  }
  saving.value = true
  try {
    await createBook({
      name,
      category: form.category,
      defaultEventId: form.defaultEventId || undefined,
      description: form.description.trim(),
    })
    ui.markChanged()
    ui.toast('礼簿已新建')
    formOpen.value = false
    await listing.reload()
  } catch (err) {
    ui.toast(err instanceof Error ? err.message : '新建失败')
  } finally {
    saving.value = false
  }
}

void dictStore.ensure()
void Promise.all([listing.reload(), loadEvents()])

watch(
  () => ui.revision,
  () => {
    void listing.reload()
    void loadEvents()
  },
)
</script>

<template>
  <PageHead title="礼簿" :meta="meta">
    <template #tools>
      <button class="web-btn" type="button" @click="ui.openMerge()">
        <WebIcon name="merge" />
        合并礼簿
      </button>
      <button class="web-btn web-btn--primary" type="button" @click="openCreate">
        <WebIcon name="plus" />
        新建礼簿
      </button>
    </template>
  </PageHead>

  <div v-if="error" class="web-alert web-alert--err">
    <WebIcon name="alert" />
    <span>{{ error }}</span>
    <button class="web-btn web-btn--sm" type="button" @click="listing.reload()">重试</button>
  </div>

  <div v-else-if="loading && !normalBooks.length" class="web-empty">正在加载礼簿…</div>

  <template v-else>
    <section class="web-section">
      <SectionHead title="常用礼簿" :sub="`${normalBooks.length} 本，点一行打开`" />

      <div class="web-card">
        <WebTable>
          <thead>
            <tr>
              <th>礼簿</th>
              <th>类别</th>
              <th>默认事项</th>
              <th class="web-table__r">记录</th>
              <th class="web-table__r">收</th>
              <th class="web-table__r">送</th>
              <th>并入目标</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="item in normalBooks" :key="item.id" class="web-table__row" @click="openBook(item)">
              <td>
                <div class="web-cell2">
                  <WebIcon name="book" />
                  <span class="web-cell2__t">
                    {{ item.name }}
                    <small v-if="item.description">{{ item.description }}</small>
                  </span>
                </div>
              </td>
              <td>
                <span class="web-pill" :style="categoryStyle(item.category)">{{ item.categoryLabel }}</span>
              </td>
              <td class="web-sm">{{ item.defaultEventName || '未指定' }}</td>
              <td class="web-table__r web-num">{{ item.recordCount }}</td>
              <td class="web-table__r web-amt web-amt--in">{{ formatCent(item.inCent) }}</td>
              <td class="web-table__r web-amt web-amt--out">{{ formatCent(item.outCent) }}</td>
              <td class="web-muted">—</td>
            </tr>
          </tbody>
        </WebTable>
        <div v-if="!normalBooks.length" class="web-empty">还没有礼簿，先新建一本。</div>
      </div>
    </section>

    <section v-if="mergedBooks.length" class="web-section">
      <SectionHead title="已合并" :sub="`${mergedBooks.length} 本，只读查看`" />
      <div class="web-card">
        <WebTable>
          <thead>
            <tr>
              <th>礼簿</th>
              <th>类别</th>
              <th>默认事项</th>
              <th class="web-table__r">记录</th>
              <th class="web-table__r">收</th>
              <th class="web-table__r">送</th>
              <th>并入目标</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="item in mergedBooks" :key="item.id" class="web-table__row" @click="openBook(item)">
              <td>
                <div class="web-cell2">
                  <WebIcon name="book" />
                  <span class="web-cell2__t">
                    {{ item.name }}
                    <small v-if="item.description">{{ item.description }}</small>
                  </span>
                </div>
              </td>
              <td>
                <span class="web-pill" :style="categoryStyle(item.category)">{{ item.categoryLabel }}</span>
              </td>
              <td class="web-sm">{{ item.defaultEventName || '未指定' }}</td>
              <td class="web-table__r web-num">{{ item.recordCount }}</td>
              <td class="web-table__r web-amt web-amt--in">{{ formatCent(item.inCent) }}</td>
              <td class="web-table__r web-amt web-amt--out">{{ formatCent(item.outCent) }}</td>
              <td class="web-sm">{{ item.mergedIntoName || '—' }}</td>
            </tr>
          </tbody>
        </WebTable>
      </div>
    </section>
  </template>

  <Modal v-if="formOpen" title="新建礼簿" @close="formOpen = false">
    <div class="web-field-row">
      <div class="web-field">
        <label>名称</label>
        <input v-model="form.name" class="web-input" placeholder="如 2024 年婚宴礼簿" maxlength="40" />
      </div>
      <div class="web-field">
        <label>类别</label>
        <select v-model="form.category" class="web-select">
          <option v-if="!dictStore.categories.length" value="">加载中…</option>
          <option v-for="item in dictStore.categories" :key="item.id" :value="item.code">{{ item.label }}</option>
        </select>
      </div>
    </div>

    <div class="web-field">
      <label>默认事项</label>
      <select v-model.number="form.defaultEventId" class="web-select">
        <option :value="0">不指定</option>
        <option v-for="item in events" :key="item.id" :value="item.id">{{ item.name }}</option>
      </select>
      <span class="web-field__hint">连续录入时默认记在这个事项上，可稍后再改。</span>
    </div>

    <div class="web-field">
      <label>描述</label>
      <textarea v-model="form.description" class="web-textarea" rows="2" placeholder="选填，如存放位置或纸本页数" />
    </div>

    <template #footer>
      <button class="web-btn" type="button" @click="formOpen = false">取消</button>
      <button class="web-btn web-btn--primary" type="button" :disabled="saving" @click="save">
        {{ saving ? '保存中…' : '保存' }}
      </button>
    </template>
  </Modal>
</template>
