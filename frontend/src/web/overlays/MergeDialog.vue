<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'

import { fetchBooks, mergeBooks, previewMerge } from '@/api/book'
import type { BookItem, MergePreview } from '@/api/types'
import { useWebUiStore } from '@/stores/web'
import WebIcon from '@/web/components/WebIcon.vue'

const props = defineProps<{ sourceId?: number }>()
const ui = useWebUiStore()

const books = ref<BookItem[]>([])
const sourceIds = ref<number[]>(props.sourceId ? [props.sourceId] : [])
const targetId = ref(0)
const preview = ref<MergePreview | null>(null)
const loading = ref(true)
const saving = ref(false)
const error = ref('')

const sameAsTarget = computed(() => sourceIds.value.includes(targetId.value))
const canMerge = computed(
  () => sourceIds.value.length > 0 && targetId.value > 0 && !sameAsTarget.value,
)

watch(
  [sourceIds, targetId],
  async () => {
    preview.value = null
    if (!canMerge.value) return
    try {
      preview.value = await previewMerge(sourceIds.value, targetId.value)
    } catch (err) {
      error.value = err instanceof Error ? err.message : '预览失败'
    }
  },
  { deep: true },
)

function toggle(id: number): void {
  sourceIds.value = sourceIds.value.includes(id)
    ? sourceIds.value.filter((item) => item !== id)
    : [...sourceIds.value, id]
}

async function confirm(): Promise<void> {
  if (!canMerge.value) return
  saving.value = true
  error.value = ''
  try {
    await mergeBooks(sourceIds.value, targetId.value)
    ui.markChanged()
    ui.toast('礼簿已合并')
    ui.close()
  } catch (err) {
    error.value = err instanceof Error ? err.message : '合并失败'
  } finally {
    saving.value = false
  }
}

onMounted(async () => {
  try {
    books.value = await fetchBooks('normal')
    targetId.value = books.value.find((item) => !sourceIds.value.includes(item.id))?.id ?? 0
  } catch (err) {
    error.value = err instanceof Error ? err.message : '加载失败'
  } finally {
    loading.value = false
  }
})
</script>

<template>
  <div class="web-dialog" role="alertdialog" aria-modal="true">
    <div class="web-drawer__head">
      <h3>合并礼簿</h3>
      <button class="web-btn web-btn--text" type="button" aria-label="关闭" @click="ui.close()">
        <WebIcon name="x" />
      </button>
    </div>
    <div class="web-drawer__body">
      <div v-if="loading" class="web-empty">加载中…</div>
      <template v-else>
        <div v-if="error" class="web-alert web-alert--err">{{ error }}</div>
        <div class="web-field">
          <label>把这些礼簿</label>
          <div style="display: flex; flex-direction: column; gap: 6px">
            <label v-for="book in books" :key="book.id" class="web-cell2">
              <input
                class="web-check"
                type="checkbox"
                :checked="sourceIds.includes(book.id)"
                @change="toggle(book.id)"
              />
              {{ book.name }}
              <span class="web-pill web-pill--neutral">{{ book.categoryLabel }}</span>
            </label>
          </div>
        </div>
        <div class="web-field">
          <label for="web-merge-target">合并到</label>
          <select id="web-merge-target" v-model.number="targetId" class="web-select">
            <option v-for="book in books" :key="book.id" :value="book.id">
              {{ book.name }}（{{ book.categoryLabel }}）
            </option>
          </select>
        </div>
        <div v-if="sameAsTarget" class="web-alert web-alert--err">目标礼簿不能同时是源礼簿</div>
        <div v-else-if="preview" class="web-alert" :class="preview.crossCategory ? 'web-alert--warn' : 'web-alert--info'">
          <span>
            {{ preview.recordCount }} 条记录将归到《{{ preview.target.name }}》，源礼簿标为“已合并”并保留原礼簿信息。
            <template v-if="preview.crossCategory">
              <br /><b>跨类别合并：</b>记录的名目不变，统计继续按名目拆分。
            </template>
          </span>
        </div>
      </template>
    </div>
    <div class="web-drawer__foot">
      <button class="web-btn" type="button" @click="ui.close()">取消</button>
      <button class="web-btn web-btn--primary" type="button" :disabled="!canMerge || saving" @click="confirm">
        确认合并
      </button>
    </div>
  </div>
</template>
