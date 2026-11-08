<script setup lang="ts">
import { computed, ref, watch } from 'vue'

import { fetchBooks } from '@/api/book'
import { fetchEvents } from '@/api/event'
import type { BookItem, EventItem, RecordFilters } from '@/api/types'
import { useDictStore } from '@/stores/dict'
import { todayString } from '@/utils/date'

const props = withDefaults(
  defineProps<{
    show: boolean
    value: RecordFilters
    count?: number
  }>(),
  { count: 0 },
)

const emit = defineEmits<{
  'update:show': [value: boolean]
  apply: [filters: RecordFilters]
  reset: []
}>()

const visible = computed({
  get: () => props.show,
  set: (value: boolean) => emit('update:show', value),
})

const dictStore = useDictStore()
const local = ref<RecordFilters>({ ...props.value })
const books = ref<BookItem[]>([])
const events = ref<EventItem[]>([])
const timePreset = ref('')

const directions = [
  { code: 'in', label: '收' },
  { code: 'out', label: '送' },
]
const statuses = [
  { code: 'pending', label: '待还' },
  { code: 'repaid', label: '已还' },
  { code: 'none', label: '无需还' },
]
const timePresets = [
  { code: 'this-year', label: '今年' },
  { code: 'last-year', label: '去年' },
  { code: 'last3', label: '近 3 年' },
]

function toggle(field: 'itemKind' | 'direction' | 'status', code: string): void {
  local.value[field] = local.value[field] === code ? '' : code
}

function chooseBook(id: number): void {
  local.value.bookId = local.value.bookId === id ? 0 : id
}

function chooseEvent(id: number): void {
  local.value.eventId = local.value.eventId === id ? 0 : id
}

function choosePreset(code: string): void {
  timePreset.value = timePreset.value === code ? '' : code
  const year = new Date().getFullYear()
  if (!timePreset.value) {
    local.value.from = ''
    local.value.to = ''
    return
  }
  if (timePreset.value === 'this-year') {
    local.value.from = `${year}-01-01`
    local.value.to = todayString()
    return
  }
  if (timePreset.value === 'last-year') {
    local.value.from = `${year - 1}-01-01`
    local.value.to = `${year - 1}-12-31`
    return
  }
  local.value.from = `${year - 2}-01-01`
  local.value.to = `${year}-12-31`
}

function onFromChange(event: Event): void {
  local.value.from = (event as unknown as { detail: { value: string } }).detail.value
  timePreset.value = ''
}

function onToChange(event: Event): void {
  local.value.to = (event as unknown as { detail: { value: string } }).detail.value
  timePreset.value = ''
}

function reset(): void {
  local.value = { bookId: 0, eventId: 0, itemKind: '', direction: '', status: '', from: '', to: '' }
  timePreset.value = ''
  emit('reset')
}

function apply(): void {
  emit('apply', { ...local.value })
  visible.value = false
}

async function loadOptions(): Promise<void> {
  dictStore.ensure()
  try {
    books.value = await fetchBooks('normal')
    const page = await fetchEvents({ pageSize: 100 })
    events.value = page.items
  } catch {
    books.value = []
    events.value = []
  }
}

watch(
  () => props.show,
  (show) => {
    if (!show) return
    local.value = { ...props.value }
    timePreset.value = ''
    loadOptions()
  },
)
</script>

<template>
  <wd-popup
    v-model="visible"
    position="bottom"
    :safe-area-inset-bottom="true"
    custom-style="max-height: 85vh; border-radius: 20px 20px 0 0;"
  >
    <view class="sheet">
      <view class="sheet__head">
        <text class="sheet__title">筛选</text>
        <text class="sheet__close" @click="visible = false">关闭</text>
      </view>
      <scroll-view scroll-y class="sheet__body">
        <view v-if="books.length" class="group">
          <text class="group__title">礼簿</text>
          <view class="chips">
            <text v-for="book in books" :key="book.id" class="chip" :class="{ 'chip--on': local.bookId === book.id }" @click="chooseBook(book.id)">
              {{ book.name }}
            </text>
          </view>
        </view>
        <view v-if="events.length" class="group">
          <text class="group__title">事项</text>
          <view class="chips">
            <text v-for="event in events" :key="event.id" class="chip" :class="{ 'chip--on': local.eventId === event.id }" @click="chooseEvent(event.id)">
              {{ event.name }}
            </text>
          </view>
        </view>
        <view class="group">
          <text class="group__title">名目</text>
          <view class="chips">
            <text
              v-for="kind in dictStore.itemKinds"
              :key="kind.code"
              class="chip"
              :class="{ 'chip--on': local.itemKind === kind.code }"
              @click="toggle('itemKind', kind.code)"
            >
              {{ kind.label }}
            </text>
          </view>
        </view>
        <view class="group">
          <text class="group__title">方向</text>
          <view class="chips">
            <text v-for="item in directions" :key="item.code" class="chip" :class="{ 'chip--on': local.direction === item.code }" @click="toggle('direction', item.code)">
              {{ item.label }}
            </text>
          </view>
        </view>
        <view class="group">
          <text class="group__title">状态</text>
          <view class="chips">
            <text v-for="item in statuses" :key="item.code" class="chip" :class="{ 'chip--on': local.status === item.code }" @click="toggle('status', item.code)">
              {{ item.label }}
            </text>
          </view>
        </view>
        <view class="group">
          <text class="group__title">时间</text>
          <view class="chips">
            <text v-for="item in timePresets" :key="item.code" class="chip" :class="{ 'chip--on': timePreset === item.code }" @click="choosePreset(item.code)">
              {{ item.label }}
            </text>
          </view>
          <view class="range">
            <picker mode="date" :value="local.from || todayString()" @change="onFromChange">
              <text class="range__value">{{ local.from || '开始日期' }}</text>
            </picker>
            <text class="range__sep">至</text>
            <picker mode="date" :value="local.to || todayString()" @change="onToChange">
              <text class="range__value">{{ local.to || '结束日期' }}</text>
            </picker>
          </view>
        </view>
      </scroll-view>
      <view class="sheet__actions">
        <view class="btn btn--secondary sheet__btn" @click="reset">重置</view>
        <view class="btn btn--primary sheet__btn" @click="apply">
          {{ count ? `查看 ${count} 条` : '查看结果' }}
        </view>
      </view>
    </view>
  </wd-popup>
</template>

<style lang="scss" scoped>
.sheet {
  display: flex;
  flex-direction: column;
  max-height: 85vh;
}

.sheet__head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 16px 16px 8px;
}

.sheet__title {
  font-size: 17px;
  font-weight: 600;
}

.sheet__close {
  font-size: 14px;
  color: var(--c-ink-3);
}

.sheet__body {
  flex: 1;
  min-height: 0;
  padding: 0 16px;
}

.group {
  margin-bottom: 18px;
}

.group__title {
  display: block;
  margin-bottom: 8px;
  font-size: 13px;
  color: var(--c-ink-3);
}

.chips {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.range {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-top: 10px;
  font-size: 13px;
  color: var(--c-ink-2);
}

.range__value {
  padding: 6px 10px;
  border-radius: var(--r-s);
  background: var(--c-fill);
}

.range__sep {
  color: var(--c-ink-3);
}

.sheet__actions {
  display: flex;
  gap: 10px;
  padding: 12px 16px calc(16px + env(safe-area-inset-bottom));
}

.sheet__btn {
  flex: 1;
}
</style>
