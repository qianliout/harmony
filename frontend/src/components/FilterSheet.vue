<script setup lang="ts">
import { computed, ref, watch } from 'vue'

import { fetchBooks } from '@/api/book'
import { fetchEvents } from '@/api/event'
import { fetchPersonDetail } from '@/api/person'
import type { BookItem, EventItem, PersonBrief, RecordFilters } from '@/api/types'
import { useDictStore } from '@/stores/dict'
import { eventTypes } from '@/styles/palette'
import { todayString } from '@/utils/date'

import PersonPicker from './PersonPicker.vue'

const props = withDefaults(
  defineProps<{
    show: boolean
    value: RecordFilters
    count?: number
    /** full：完整筛选；search：只留事项类型与时间 */
    mode?: 'full' | 'search'
    hideBookCategory?: boolean
    hideBook?: boolean
    hidePerson?: boolean
  }>(),
  {
    count: 0,
    mode: 'full',
    hideBookCategory: false,
    hideBook: false,
    hidePerson: false,
  },
)

const emit = defineEmits<{
  'update:show': [value: boolean]
  apply: [filters: RecordFilters]
  reset: []
}>()

function emptyFilters(): RecordFilters {
  return {
    bookCategory: '',
    eventType: '',
    personId: 0,
    bookId: 0,
    eventId: 0,
    itemKind: '',
    direction: '',
    status: '',
    from: '',
    to: '',
  }
}

const visible = computed({
  get: () => props.show,
  set: (value: boolean) => emit('update:show', value),
})

const dictStore = useDictStore()
const local = ref<RecordFilters>(emptyFilters())
const books = ref<BookItem[]>([])
const events = ref<EventItem[]>([])
const timePreset = ref('')
const personName = ref('')
const personPickerShow = ref(false)
const expanded = ref(false)

const directions = [
  { code: 'in', label: '收' },
  { code: 'out', label: '送' },
]
const timePresets = [
  { code: 'this-year', label: '今年' },
  { code: 'last-year', label: '去年' },
  { code: 'last3', label: '近 3 年' },
]

const statusOptions = computed(() => {
  if (local.value.direction === 'out') {
    return [
      { code: 'pending', label: '对方未回' },
      { code: 'repaid', label: '对方已回' },
      { code: 'none', label: '不用回' },
    ]
  }
  return [
    { code: 'pending', label: '未回礼' },
    { code: 'repaid', label: '已回礼' },
    { code: 'none', label: '不用回' },
  ]
})

const showBookCategory = computed(
  () => props.mode === 'full' && !props.hideBookCategory && dictStore.categories.length > 0,
)
const showPerson = computed(() => props.mode === 'full' && !props.hidePerson)
const showMore = computed(() => props.mode === 'full')
const showBook = computed(() => props.mode === 'full' && !props.hideBook)
const showEvent = computed(() => props.mode === 'full')
const showItemKind = computed(() => props.mode === 'full')
const showDirection = computed(() => props.mode === 'full')
const showStatus = computed(() => props.mode === 'full')

const moreActive = computed(() =>
  Boolean(
    local.value.bookId ||
      local.value.eventId ||
      local.value.itemKind ||
      local.value.direction ||
      local.value.status,
  ),
)

function toggle(field: 'bookCategory' | 'eventType' | 'itemKind' | 'direction' | 'status', code: string): void {
  const next = local.value[field] === code ? '' : code
  local.value[field] = next
  if (field === 'direction' && !next) {
    local.value.status = ''
  }
}

function chooseBook(id: number): void {
  local.value.bookId = local.value.bookId === id ? 0 : id
}

function chooseEvent(id: number): void {
  local.value.eventId = local.value.eventId === id ? 0 : id
}

function openPersonPicker(): void {
  personPickerShow.value = true
}

function onPersonSelect(person: PersonBrief): void {
  local.value.personId = person.id
  personName.value = person.name
}

function clearPerson(): void {
  local.value.personId = 0
  personName.value = ''
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
  local.value = emptyFilters()
  timePreset.value = ''
  personName.value = ''
  expanded.value = false
  emit('reset')
}

function apply(): void {
  emit('apply', { ...local.value })
  visible.value = false
}

async function resolvePersonName(id: number): Promise<void> {
  if (!id) {
    personName.value = ''
    return
  }
  try {
    const detail = await fetchPersonDetail(id)
    personName.value = detail.name
  } catch {
    personName.value = ''
  }
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
    expanded.value = Boolean(
      props.value.bookId || props.value.eventId || props.value.itemKind || props.value.direction || props.value.status,
    )
    void resolvePersonName(props.value.personId)
    loadOptions()
  },
)
</script>

<template>
  <view class="filter-root">
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
        <view v-if="showBookCategory" class="group">
          <text class="group__title">礼簿类别</text>
          <view class="chips">
            <text
              v-for="category in dictStore.categories"
              :key="category.code"
              class="chip"
              :class="{ 'chip--on': local.bookCategory === category.code }"
              @click="toggle('bookCategory', category.code)"
            >
              {{ category.label }}
            </text>
          </view>
        </view>

        <view class="group">
          <text class="group__title">事项类型</text>
          <view class="chips">
            <text
              v-for="type in eventTypes"
              :key="type.code"
              class="chip"
              :class="{ 'chip--on': local.eventType === type.code }"
              @click="toggle('eventType', type.code)"
            >
              {{ type.label }}
            </text>
          </view>
        </view>

        <view v-if="showPerson" class="group">
          <text class="group__title">亲友</text>
          <view class="chips">
            <text class="chip chip--entry" @click="openPersonPicker">
              {{ personName || '选择亲友' }} ›
            </text>
            <text v-if="local.personId" class="chip chip--clear" @click="clearPerson">清除</text>
          </view>
        </view>

        <view class="group">
          <text class="group__title">时间</text>
          <view class="chips">
            <text
              v-for="item in timePresets"
              :key="item.code"
              class="chip"
              :class="{ 'chip--on': timePreset === item.code }"
              @click="choosePreset(item.code)"
            >
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

        <template v-if="showMore">
          <view class="more-head" @click="expanded = !expanded">
            <text class="more-head__title">更多条件</text>
            <text v-if="moreActive" class="more-head__dot" />
            <text class="more-head__caret">{{ expanded ? '⌃' : '⌄' }}</text>
          </view>

          <template v-if="expanded">
            <view v-if="showBook && books.length" class="group">
              <text class="group__title">礼簿</text>
              <view class="chips">
                <text
                  v-for="book in books"
                  :key="book.id"
                  class="chip"
                  :class="{ 'chip--on': local.bookId === book.id }"
                  @click="chooseBook(book.id)"
                >
                  {{ book.name }}
                </text>
              </view>
            </view>

            <view v-if="showEvent && events.length" class="group">
              <text class="group__title">事项</text>
              <view class="chips">
                <text
                  v-for="event in events"
                  :key="event.id"
                  class="chip"
                  :class="{ 'chip--on': local.eventId === event.id }"
                  @click="chooseEvent(event.id)"
                >
                  {{ event.name }}
                </text>
              </view>
            </view>

            <view v-if="showItemKind" class="group">
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

            <view v-if="showDirection" class="group">
              <text class="group__title">方向</text>
              <view class="chips">
                <text
                  v-for="item in directions"
                  :key="item.code"
                  class="chip"
                  :class="{ 'chip--on': local.direction === item.code }"
                  @click="toggle('direction', item.code)"
                >
                  {{ item.label }}
                </text>
              </view>
            </view>

            <view v-if="showStatus" class="group">
              <text class="group__title">回礼状态</text>
              <view class="chips">
                <text
                  v-for="item in statusOptions"
                  :key="item.code"
                  class="chip"
                  :class="{ 'chip--on': local.status === item.code }"
                  @click="toggle('status', item.code)"
                >
                  {{ item.label }}
                </text>
              </view>
            </view>
          </template>
        </template>
      </scroll-view>
      <view class="sheet__actions">
        <view class="btn btn--secondary sheet__btn" @click="reset">重置</view>
        <view class="btn btn--primary sheet__btn" @click="apply">
          {{ count ? `查看 ${count} 条` : '查看结果' }}
        </view>
      </view>
      </view>
    </wd-popup>

    <PersonPicker v-model:show="personPickerShow" @select="onPersonSelect" />
  </view>
</template>

<style lang="scss" scoped>
.filter-root {
  display: contents;
}

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

.chip--entry {
  color: var(--c-primary);
}

.chip--clear {
  color: var(--c-ink-3);
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

.more-head {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 12px 0 10px;
  margin-bottom: 12px;
  box-shadow: inset 0 1px 0 var(--c-divider);
}

.more-head__title {
  font-size: 14px;
  font-weight: 500;
  color: var(--c-ink-2);
}

.more-head__dot {
  width: 6px;
  height: 6px;
  border-radius: var(--r-full);
  background: var(--c-primary);
}

.more-head__caret {
  margin-left: auto;
  font-size: 13px;
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
