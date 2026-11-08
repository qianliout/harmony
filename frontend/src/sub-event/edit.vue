<script setup lang="ts">
import { onLoad } from '@dcloudio/uni-app'
import { computed, ref } from 'vue'

import { createEvent, deleteEvent, fetchEventDetail, updateEvent } from '@/api/event'
import AppNav from '@/components/AppNav.vue'
import EmptyState from '@/components/EmptyState.vue'
import EventSeal from '@/components/EventSeal.vue'
import PersonPicker from '@/components/PersonPicker.vue'
import Skeleton from '@/components/Skeleton.vue'
import { eventTypes } from '@/styles/palette'
import { parseDateParts, todayString, weekdayOf } from '@/utils/date'
import { back, confirmModal, toast } from '@/utils/nav'

interface PickedPerson {
  id: number
  name: string
}

const id = ref(0)
const name = ref('')
const type = ref('other')
const eventDate = ref(todayString())
const datePrecision = ref('day')
const host = ref<'self' | 'other'>('other')
const location = ref('')
const remark = ref('')
const hostPersonId = ref(0)
const hostPersonName = ref('')
const personPickerShow = ref(false)
const loading = ref(true)
const showSkeleton = ref(false)
const error = ref('')
const saving = ref(false)

let skeletonTimer: ReturnType<typeof setTimeout> | null = null

const isEdit = computed(() => id.value > 0)
const title = computed(() => (isEdit.value ? '编辑事项' : '新建事项'))
const dateLabel = computed(() => {
  if (!eventDate.value) return '选择日期'
  const parts = parseDateParts(eventDate.value)
  if (!parts) return '选择日期'
  if (datePrecision.value === 'year') return `${parts.year}年 约`
  if (datePrecision.value === 'month') return `${parts.year}年${parts.month}月 约`
  return `${parts.year}年${parts.month}月${parts.day}日 ${weekdayOf(eventDate.value)}`
})

async function load(): Promise<void> {
  loading.value = true
  error.value = ''
  if (skeletonTimer) clearTimeout(skeletonTimer)
  skeletonTimer = setTimeout(() => {
    showSkeleton.value = true
  }, 300)
  try {
    if (!id.value) return
    const detail = await fetchEventDetail(id.value)
    name.value = detail.name
    type.value = detail.type
    eventDate.value = detail.eventDate || todayString()
    datePrecision.value = detail.datePrecision || 'day'
    host.value = detail.host === 'self' ? 'self' : 'other'
    location.value = detail.location
    remark.value = detail.remark
    hostPersonId.value = detail.hostPersonId
    hostPersonName.value = detail.hostPersonName
  } catch (err) {
    error.value = err instanceof Error ? err.message : '加载失败'
  } finally {
    if (skeletonTimer) clearTimeout(skeletonTimer)
    skeletonTimer = null
    showSkeleton.value = false
    loading.value = false
  }
}

function retry(): void {
  load()
}

function onNameInput(event: Event): void {
  name.value = (event as unknown as { detail: { value: string } }).detail.value
}

function onLocationInput(event: Event): void {
  location.value = (event as unknown as { detail: { value: string } }).detail.value
}

function onRemarkInput(event: Event): void {
  remark.value = (event as unknown as { detail: { value: string } }).detail.value
}

function onDateChange(event: Event): void {
  eventDate.value = (event as unknown as { detail: { value: string } }).detail.value
  datePrecision.value = 'day'
}

function chooseType(code: string): void {
  type.value = code
}

function chooseHost(value: 'self' | 'other'): void {
  host.value = value
}

function openPersonPicker(): void {
  personPickerShow.value = true
}

function onPersonSelect(person: PickedPerson): void {
  hostPersonId.value = person.id
  hostPersonName.value = person.name
  personPickerShow.value = false
}

function clearPerson(): void {
  hostPersonId.value = 0
  hostPersonName.value = ''
}

async function save(): Promise<void> {
  if (saving.value) return
  const trimmedName = name.value.trim()
  if (!trimmedName) {
    toast('请填写事项名称')
    return
  }
  saving.value = true
  try {
    const params = {
      name: trimmedName,
      type: type.value,
      eventDate: eventDate.value || undefined,
      datePrecision: datePrecision.value,
      location: location.value.trim(),
      host: host.value,
      hostPersonId: hostPersonId.value || undefined,
      remark: remark.value.trim(),
    }
    if (isEdit.value) {
      await updateEvent(id.value, params)
    } else {
      await createEvent(params)
    }
    toast('已保存', 'success')
    setTimeout(() => back(), 500)
  } catch (err) {
    toast(err instanceof Error ? err.message : '保存失败')
  } finally {
    saving.value = false
  }
}

async function remove(): Promise<void> {
  if (!isEdit.value) return
  const ok = await confirmModal('删除后不可恢复，这个事项下的记录也会受到影响。', '删除事项')
  if (!ok) return
  try {
    await deleteEvent(id.value)
    toast('已删除', 'success')
    setTimeout(() => back(), 500)
  } catch (err) {
    toast(err instanceof Error ? err.message : '删除失败')
  }
}

onLoad((query) => {
  id.value = Number(query?.id ?? 0)
  load()
})
</script>

<template>
  <view class="screen">
    <AppNav :title="title" />

    <Skeleton v-if="showSkeleton" />

    <view v-else-if="error" class="page">
      <EmptyState title="没能打开事项" :description="error" action-text="重试" @action="retry" />
    </view>

    <view v-else-if="!loading" class="page">
      <view class="form-card">
        <view class="field">
          <text class="field__label">名称</text>
          <input
            class="field__input"
            :value="name"
            placeholder="如 小明升学宴"
            maxlength="40"
            @input="onNameInput"
          />
        </view>
      </view>

      <view class="section">
        <text class="section__title">类型</text>
        <view class="type-grid">
          <view
            v-for="item in eventTypes"
            :key="item.code"
            class="type-tile"
            :class="{ 'type-tile--on': type === item.code }"
            @click="chooseType(item.code)"
          >
            <EventSeal :type="item.code" size="m" />
            <text class="type-tile__label">{{ item.label }}</text>
          </view>
        </view>
      </view>

      <view class="form-card date-card">
        <picker mode="date" :value="eventDate" @change="onDateChange">
          <view class="field">
            <text class="field__label">日期</text>
            <text class="field__value">{{ dateLabel }}</text>
            <text class="field__arrow">›</text>
          </view>
        </picker>
        <view class="field">
          <text class="field__label">主办方</text>
          <view class="segmented host-segmented">
            <view
              class="segmented__item"
              :class="{ 'segmented__item--on': host === 'self' }"
              @click="chooseHost('self')"
            >
              自家办
            </view>
            <view
              class="segmented__item"
              :class="{ 'segmented__item--on': host === 'other' }"
              @click="chooseHost('other')"
            >
              别人家办
            </view>
          </view>
        </view>
        <view class="field">
          <text class="field__label">地点</text>
          <input
            class="field__input"
            :value="location"
            placeholder="选填，如县城酒店"
            maxlength="60"
            @input="onLocationInput"
          />
        </view>
        <view class="field" @click="openPersonPicker">
          <text class="field__label">主办人</text>
          <text class="field__value" :class="{ muted: !hostPersonName }">
            {{ hostPersonName || '选填' }}
          </text>
          <text v-if="hostPersonId" class="field__clear" @click.stop="clearPerson">清除</text>
          <text class="field__arrow">›</text>
        </view>
      </view>

      <view class="section">
        <text class="section__title">备注</text>
        <view class="form-card">
          <textarea
            class="remark"
            :value="remark"
            placeholder="选填，记下需要留意的事"
            maxlength="200"
            @input="onRemarkInput"
          />
        </view>
      </view>

      <view class="btn btn--primary save" :class="{ 'btn--disabled': saving }" @click="save">
        {{ saving ? '保存中…' : '保存' }}
      </view>

      <text v-if="isEdit" class="delete-link" @click="remove">删除事项</text>
    </view>

    <PersonPicker v-model:show="personPickerShow" @select="onPersonSelect" />
  </view>
</template>

<style lang="scss" scoped>
.form-card {
  margin-top: 8px;
}

.field__input {
  text-align: left;
}

.section {
  margin-top: 20px;
}

.section__title {
  display: block;
  margin: 0 2px 8px;
  font-size: 13px;
  line-height: 18px;
  color: var(--c-ink-3);
}

.type-grid {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.type-tile {
  display: flex;
  flex-direction: column;
  align-items: center;
  width: calc((100% - 24px) / 4);
  box-sizing: border-box;
  gap: 6px;
  padding: 12px 4px 10px;
  border-radius: var(--r-m);
  background: var(--c-surface);
  box-shadow: inset 0 0 0 1px var(--c-line);
  transition: transform 0.24s cubic-bezier(0.2, 0.8, 0.2, 1);
}

.type-tile--on {
  background: var(--c-primary-light);
  box-shadow: inset 0 0 0 1px var(--c-primary);
  transform: scale(1.04);
}

.type-tile__label {
  font-size: 12px;
  color: var(--c-ink-2);
}

.type-tile--on .type-tile__label {
  color: var(--c-primary);
  font-weight: 600;
}

.date-card {
  margin-top: 20px;
}

.host-segmented {
  flex: 1;
  height: 36px;
}

.host-segmented .segmented__item {
  height: 30px;
  font-size: 13px;
}

.field__clear {
  margin-left: 8px;
  font-size: 13px;
  color: var(--c-ink-3);
}

.remark {
  width: 100%;
  height: 88px;
  padding: 12px 16px;
  box-sizing: border-box;
  font-size: 15px;
  line-height: 22px;
  color: var(--c-ink-1);
}

.save {
  margin-top: 28px;
}

.delete-link {
  display: block;
  padding: 20px 0 8px;
  text-align: center;
  font-size: 14px;
  color: var(--c-danger);
}
</style>
