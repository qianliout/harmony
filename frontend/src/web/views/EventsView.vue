<script setup lang="ts">
import { computed, onUnmounted, reactive, ref, watch } from 'vue'

import { createEvent, deleteEvent, fetchEvents, updateEvent } from '@/api/event'
import type { EventItem, Page } from '@/api/types'
import EventSeal from '@/components/EventSeal.vue'
import { eventTypes } from '@/styles/palette'
import { useWebUiStore } from '@/stores/web'
import { preciseDateLabel, todayString } from '@/utils/date'
import { formatCent } from '@/utils/money'
import Modal from '@/web/components/Modal.vue'
import PageHead from '@/web/components/PageHead.vue'
import Pager from '@/web/components/Pager.vue'
import WebIcon from '@/web/components/WebIcon.vue'
import WebTable from '@/web/components/WebTable.vue'
import { useAsync } from '@/web/composables/useAsync'
import { goWeb } from '@/web/nav'

const ui = useWebUiStore()

const PAGE_SIZE = 50
const precisionOptions = [
  { value: 'day', label: '按日' },
  { value: 'month', label: '按月' },
  { value: 'year', label: '按年' },
]

const keyword = ref('')
const typeFilter = ref('')
const page = ref(1)

const listing = useAsync<Page<EventItem>>(
  () =>
    fetchEvents({
      keyword: keyword.value.trim() || undefined,
      type: typeFilter.value || undefined,
      page: page.value,
      pageSize: PAGE_SIZE,
    }),
  { items: [], total: 0 },
)

const events = computed(() => listing.data.value.items)
const total = computed(() => listing.data.value.total)
const loading = computed(() => listing.loading.value)
const error = computed(() => listing.error.value)
const pageCount = computed(() => Math.max(1, Math.ceil(total.value / PAGE_SIZE)))

// 新建 / 编辑共用同一个弹窗
const formOpen = ref(false)
const editingId = ref(0)
const editingHostPersonId = ref(0)
const saving = ref(false)
const form = reactive({
  name: '',
  type: 'other',
  date: todayString(),
  precision: 'day',
  location: '',
  host: 'other',
  remark: '',
})

let keywordTimer: ReturnType<typeof setTimeout> | null = null

watch(keyword, () => {
  if (keywordTimer) clearTimeout(keywordTimer)
  keywordTimer = setTimeout(() => {
    page.value = 1
    void listing.reload()
  }, 240)
})

watch(typeFilter, () => {
  page.value = 1
  void listing.reload()
})

watch(
  () => ui.revision,
  () => {
    void listing.reload()
  },
)

onUnmounted(() => {
  if (keywordTimer) clearTimeout(keywordTimer)
})

void listing.reload()

function openEvent(item: EventItem): void {
  goWeb(`/sub-web/event?id=${item.id}`, 'push')
}

function changePage(next: number): void {
  if (next < 1 || next > pageCount.value) return
  page.value = next
  void listing.reload()
}

function resetForm(): void {
  form.name = ''
  form.type = 'other'
  form.date = todayString()
  form.precision = 'day'
  form.location = ''
  form.host = 'other'
  form.remark = ''
}

function openCreate(): void {
  editingId.value = 0
  editingHostPersonId.value = 0
  resetForm()
  formOpen.value = true
}

function openEdit(item: EventItem): void {
  editingId.value = item.id
  editingHostPersonId.value = item.hostPersonId
  form.name = item.name
  form.type = item.type || 'other'
  form.date = item.eventDate || todayString()
  form.precision = item.datePrecision || 'day'
  form.location = item.location
  form.host = item.host === 'self' ? 'self' : 'other'
  form.remark = item.remark
  formOpen.value = true
}

// 月 / 年精度按该月、该年首日保存，展示时统一加“约”
function normalizedDate(): string {
  if (!form.date) return ''
  const parts = form.date.split('-')
  if (form.precision === 'year') return `${parts[0]}-01-01`
  if (form.precision === 'month') return `${parts[0]}-${parts[1]}-01`
  return form.date
}

async function save(): Promise<void> {
  if (saving.value) return
  const name = form.name.trim()
  if (!name) {
    ui.toast('请填写事项名称')
    return
  }
  saving.value = true
  try {
    const params = {
      name,
      type: form.type,
      eventDate: normalizedDate() || undefined,
      datePrecision: form.precision,
      location: form.location.trim(),
      host: form.host,
      hostPersonId: editingHostPersonId.value || undefined,
      remark: form.remark.trim(),
    }
    if (editingId.value) {
      await updateEvent(editingId.value, params)
    } else {
      await createEvent(params)
    }
    ui.markChanged()
    ui.toast(editingId.value ? '事项已更新' : '事项已新建')
    formOpen.value = false
    await listing.reload()
  } catch (err) {
    ui.toast(err instanceof Error ? err.message : '保存失败')
  } finally {
    saving.value = false
  }
}

async function remove(item: EventItem): Promise<void> {
  if (!window.confirm(`删除事项「${item.name}」？删除后不可恢复。`)) return
  try {
    await deleteEvent(item.id)
    ui.markChanged()
    ui.toast('事项已删除')
    await listing.reload()
  } catch (err) {
    ui.toast(err instanceof Error ? err.message : '删除失败')
  }
}
</script>

<template>
  <PageHead title="事项" :meta="`共 ${total} 个事项 · 按日期倒序`">
    <template #tools>
      <button class="web-btn web-btn--primary" type="button" @click="openCreate">
        <WebIcon name="plus" />
        新建事项
      </button>
    </template>
  </PageHead>

  <div class="web-card">
    <div class="web-filters">
      <span class="web-filters__lbl">筛选</span>
      <input v-model="keyword" class="web-input" style="width: 200px" placeholder="事项名称" />
      <select v-model="typeFilter" class="web-select">
        <option value="">全部类型</option>
        <option v-for="item in eventTypes" :key="item.code" :value="item.code">{{ item.label }}</option>
      </select>
      <div class="web-filters__right web-muted web-sm">每页 {{ PAGE_SIZE }} 条</div>
    </div>

    <div v-if="error" class="web-alert web-alert--err" style="margin: 12px 16px">
      <WebIcon name="alert" />
      <span>{{ error }}</span>
      <button class="web-btn web-btn--sm" type="button" @click="listing.reload()">重试</button>
    </div>

    <div v-else-if="loading && !events.length" class="web-empty">正在加载事项…</div>

    <template v-else>
      <WebTable>
        <thead>
          <tr>
            <th>事项</th>
            <th>日期</th>
            <th>主办方</th>
            <th class="web-table__r">人数</th>
            <th class="web-table__r">收</th>
            <th class="web-table__r">送</th>
            <th>未回礼</th>
            <th />
          </tr>
        </thead>
        <tbody>
          <tr v-for="item in events" :key="item.id" class="web-table__row" @click="openEvent(item)">
            <td>
              <div class="web-cell2">
                <EventSeal :type="item.type" size="s" />
                <span class="web-cell2__t">
                  {{ item.name }}
                  <small>{{ item.typeLabel }}{{ item.placeholder ? ' · 待补全' : '' }}</small>
                </span>
              </div>
            </td>
            <td class="web-sm">
              {{ preciseDateLabel(item.eventDate, item.datePrecision) || '日期未定' }}
            </td>
            <td class="web-sm">
              {{ item.hostLabel }}
              <small v-if="item.hostPersonName" class="web-muted"> · {{ item.hostPersonName }}</small>
            </td>
            <td class="web-table__r web-num">{{ item.personCount }}</td>
            <td class="web-table__r web-amt web-amt--in">{{ formatCent(item.inCent) }}</td>
            <td class="web-table__r web-amt web-amt--out">{{ formatCent(item.outCent) }}</td>
            <td>
              <span v-if="item.pendingCount" class="web-pill web-pill--pending">{{ item.pendingCount }} 笔</span>
              <span v-else class="web-muted">—</span>
            </td>
            <td class="web-table__r" @click.stop>
              <button class="web-btn web-btn--text web-btn--sm" type="button" title="编辑" @click="openEdit(item)">
                <WebIcon name="edit" />
              </button>
              <button
                class="web-btn web-btn--text web-btn--sm web-btn--danger"
                type="button"
                title="删除"
                @click="remove(item)"
              >
                <WebIcon name="trash" />
              </button>
            </td>
          </tr>
        </tbody>
      </WebTable>

      <div v-if="!events.length" class="web-empty">
        {{ keyword || typeFilter ? '没有符合筛选条件的事项' : '还没有事项，先新建一个。' }}
      </div>

      <div class="web-sumbar">
        <span>第 {{ page }} / {{ pageCount }} 页</span>
        <Pager :page="page" :pages="pageCount" @change="changePage" />
      </div>
    </template>
  </div>

  <Modal v-if="formOpen" :title="editingId ? '编辑事项' : '新建事项'" @close="formOpen = false">
    <div class="web-field-row">
      <div class="web-field">
        <label>名称</label>
        <input v-model="form.name" class="web-input" placeholder="如 小明升学宴" maxlength="40" />
      </div>
      <div class="web-field">
        <label>类型</label>
        <select v-model="form.type" class="web-select">
          <option v-for="item in eventTypes" :key="item.code" :value="item.code">{{ item.label }}</option>
        </select>
      </div>
    </div>

    <div class="web-field-row">
      <div class="web-field">
        <label>日期</label>
        <input v-model="form.date" class="web-input" type="date" />
      </div>
      <div class="web-field">
        <label>日期精度</label>
        <select v-model="form.precision" class="web-select">
          <option v-for="item in precisionOptions" :key="item.value" :value="item.value">{{ item.label }}</option>
        </select>
        <span class="web-field__hint">按月 / 按年保存为该月、该年首日，列表显示“约”。</span>
      </div>
    </div>

    <div class="web-field-row">
      <div class="web-field">
        <label>地点</label>
        <input v-model="form.location" class="web-input" placeholder="选填，如县城酒店" maxlength="60" />
      </div>
      <div class="web-field">
        <label>主办方</label>
        <select v-model="form.host" class="web-select">
          <option value="other">他家办</option>
          <option value="self">自家办</option>
        </select>
      </div>
    </div>

    <div class="web-field">
      <label>备注</label>
      <textarea v-model="form.remark" class="web-textarea" rows="3" placeholder="选填" />
    </div>

    <template #footer>
      <button class="web-btn" type="button" @click="formOpen = false">取消</button>
      <button class="web-btn web-btn--primary" type="button" :disabled="saving" @click="save">
        {{ saving ? '保存中…' : '保存' }}
      </button>
    </template>
  </Modal>
</template>
