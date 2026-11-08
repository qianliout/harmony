<script setup lang="ts">
import { computed, ref, watch } from 'vue'

import { createEvent, fetchEvents, fetchRecentEvents } from '@/api/event'
import type { EventItem } from '@/api/types'
import { eventTypes } from '@/styles/palette'
import { todayString, formatDate } from '@/utils/date'

import EventSeal from './EventSeal.vue'

const props = defineProps<{ show: boolean }>()
const emit = defineEmits<{
  'update:show': [value: boolean]
  select: [event: EventItem]
}>()

const visible = computed({
  get: () => props.show,
  set: (value: boolean) => emit('update:show', value),
})

const keyword = ref('')
const results = ref<EventItem[]>([])
const recent = ref<EventItem[]>([])
const loading = ref(false)
const creating = ref(false)
const createName = ref('')
const createType = ref('other')
const createDate = ref(todayString())
const createHost = ref<'self' | 'other'>('other')
const createError = ref('')
let timer: ReturnType<typeof setTimeout> | null = null

const hasExact = computed(() => results.value.some((item) => item.name === keyword.value.trim()))

async function loadRecent(): Promise<void> {
  try {
    recent.value = await fetchRecentEvents(6)
  } catch {
    recent.value = []
  }
}

async function search(value: string): Promise<void> {
  keyword.value = value
  if (timer) clearTimeout(timer)
  if (!value.trim()) {
    results.value = []
    return
  }
  timer = setTimeout(async () => {
    loading.value = true
    try {
      const page = await fetchEvents({ keyword: value.trim(), pageSize: 20 })
      results.value = page.items
    } catch {
      results.value = []
    } finally {
      loading.value = false
    }
  }, 200)
}

function onKeywordInput(event: Event): void {
  search((event as unknown as { detail: { value: string } }).detail.value)
}

function onNameInput(event: Event): void {
  createName.value = (event as unknown as { detail: { value: string } }).detail.value
}

function onDateChange(event: Event): void {
  createDate.value = (event as unknown as { detail: { value: string } }).detail.value
}

function pick(event: EventItem): void {
  emit('select', event)
  visible.value = false
}

function startCreate(): void {
  createName.value = keyword.value.trim()
  createType.value = 'other'
  createDate.value = todayString()
  createHost.value = 'other'
  createError.value = ''
  creating.value = true
}

async function submitCreate(): Promise<void> {
  if (!createName.value.trim()) return
  try {
    const event = await createEvent({
      name: createName.value.trim(),
      type: createType.value,
      eventDate: createDate.value,
      datePrecision: 'day',
      host: createHost.value,
    })
    pick(event)
  } catch (error) {
    createError.value = error instanceof Error ? error.message : '新建失败'
  }
}

watch(
  () => props.show,
  (show) => {
    if (!show) return
    keyword.value = ''
    results.value = []
    creating.value = false
    createError.value = ''
    loadRecent()
  },
)
</script>

<template>
  <wd-popup
    v-model="visible"
    position="bottom"
    :safe-area-inset-bottom="true"
    custom-style="height: 85vh; border-radius: 20px 20px 0 0; overflow: hidden;"
  >
    <view class="picker">
      <view class="picker__head">
        <text class="picker__title">选择事项</text>
        <text class="picker__cancel" @click="visible = false">取消</text>
      </view>
      <view class="picker__search">
        <text class="picker__icon">⌕</text>
        <input
          class="picker__input"
          :value="keyword"
          placeholder="事项名称"
          focus
          @input="onKeywordInput"
        />
      </view>

      <scroll-view scroll-y class="picker__body">
        <view v-if="!keyword && recent.length">
          <text class="picker__label">最近事项</text>
          <view v-for="event in recent" :key="event.id" class="picker__row" @click="pick(event)">
            <EventSeal :type="event.type" size="s" />
            <view class="picker__row-main">
              <text class="picker__name">{{ event.name }}</text>
              <text class="picker__meta">{{ event.typeLabel }} · {{ formatDate(event.eventDate, event.datePrecision) }}</text>
            </view>
            <text class="picker__arrow">›</text>
          </view>
        </view>

        <view v-if="!creating && keyword.trim() && !hasExact" class="picker__create" @click="startCreate">
          ＋ 新建「{{ keyword.trim() }}」
        </view>

        <view v-if="creating" class="picker__form">
          <input
            class="field"
            :value="createName"
            placeholder="事项名称"
            @input="onNameInput"
          />
          <view class="picker__types">
            <view
              v-for="type in eventTypes"
              :key="type.code"
              class="picker__type"
              :class="{ 'picker__type--on': createType === type.code }"
              @click="createType = type.code"
            >
              <EventSeal :type="type.code" size="s" />
              <text class="picker__type-label">{{ type.label }}</text>
            </view>
          </view>
          <view class="picker__line">
            <text class="picker__k">日期</text>
            <picker mode="date" :value="createDate" @change="onDateChange">
              <text class="picker__v">{{ createDate }}</text>
            </picker>
          </view>
          <view class="picker__line">
            <text class="picker__k">主办</text>
            <view class="segment segment--small">
              <text class="segment__item" :class="{ 'segment__item--on': createHost === 'other' }" @click="createHost = 'other'">别人家办</text>
              <text class="segment__item" :class="{ 'segment__item--on': createHost === 'self' }" @click="createHost = 'self'">自家办</text>
            </view>
          </view>
          <text v-if="createError" class="picker__error">{{ createError }}</text>
          <view class="picker__form-actions">
            <view class="btn btn--secondary picker__form-btn" @click="creating = false">取消</view>
            <view class="btn btn--primary picker__form-btn" @click="submitCreate">确定</view>
          </view>
        </view>

        <view v-if="!creating">
          <view v-if="loading" class="picker__hint">正在查找…</view>
          <view v-for="event in results" :key="event.id" class="picker__row" @click="pick(event)">
            <EventSeal :type="event.type" size="s" />
            <view class="picker__row-main">
              <text class="picker__name">{{ event.name }}</text>
              <text class="picker__meta">{{ event.typeLabel }} · {{ formatDate(event.eventDate, event.datePrecision) }} · {{ event.hostLabel }}</text>
            </view>
            <text class="picker__arrow">›</text>
          </view>
          <view v-if="!loading && keyword && !results.length && !creating" class="picker__hint">没有找到，可直接新建</view>
        </view>
      </scroll-view>
    </view>
  </wd-popup>
</template>

<style lang="scss" scoped>
.picker {
  display: flex;
  flex-direction: column;
  height: 100%;
}

.picker__head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 16px 16px 8px;
}

.picker__title {
  font-size: 17px;
  font-weight: 600;
}

.picker__cancel {
  font-size: 14px;
  color: var(--c-ink-3);
}

.picker__search {
  display: flex;
  align-items: center;
  gap: 8px;
  height: 38px;
  margin: 4px 16px 8px;
  padding: 0 12px;
  border-radius: var(--r-s);
  background: var(--c-fill);
}

.picker__icon {
  color: var(--c-ink-3);
}

.picker__input {
  flex: 1;
  font-size: 14px;
}

.picker__body {
  flex: 1;
  min-height: 0;
  padding-bottom: 20px;
}

.picker__label {
  display: block;
  margin: 8px 16px 4px;
  font-size: 12px;
  color: var(--c-ink-3);
}

.picker__create {
  padding: 10px 16px;
  color: var(--c-primary);
  font-size: 14px;
  border-bottom: 1px solid var(--c-divider);
}

.picker__form {
  padding: 8px 16px 16px;
}

.picker__types {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
  margin-top: 12px;
}

.picker__type {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 4px;
  padding: 6px;
  border-radius: var(--r-s);
}

.picker__type--on {
  background: var(--c-primary-light);
}

.picker__type-label {
  font-size: 11px;
  color: var(--c-ink-2);
}

.picker__line {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-top: 12px;
  font-size: 14px;
}

.picker__k {
  width: 40px;
  color: var(--c-ink-3);
}

.picker__v {
  color: var(--c-ink-1);
}

.picker__form-actions {
  display: flex;
  gap: 10px;
  margin-top: 16px;
}

.picker__form-btn {
  flex: 1;
}

.picker__row {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px 16px;
}

.picker__row-main {
  flex: 1;
  min-width: 0;
}

.picker__name {
  font-size: 15px;
  font-weight: 500;
}

.picker__meta {
  display: block;
  font-size: 12px;
  color: var(--c-ink-3);
}

.picker__arrow {
  color: var(--c-ink-4);
}

.picker__hint {
  padding: 14px 16px;
  font-size: 13px;
  color: var(--c-ink-3);
}

.picker__error {
  display: block;
  margin-top: 8px;
  font-size: 12px;
  color: var(--c-danger);
}

.field {
  height: 44px;
  padding: 0 12px;
  border-radius: var(--r-s);
  background: var(--c-fill);
  font-size: 15px;
}

.segment--small {
  display: flex;
  padding: 2px;
  border-radius: var(--r-s);
  background: var(--c-fill);
}

.segment__item {
  padding: 6px 12px;
  border-radius: 8px;
  font-size: 13px;
  color: var(--c-ink-2);
}

.segment__item--on {
  background: var(--c-surface);
  color: var(--c-ink-1);
  font-weight: 500;
  box-shadow: 0 1px 4px rgba(74, 50, 30, 0.1);
}
</style>
