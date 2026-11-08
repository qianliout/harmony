<script setup lang="ts">
import { onLoad } from '@dcloudio/uni-app'
import { computed, ref } from 'vue'

import { createBook, deleteBook, fetchBookDetail, updateBook } from '@/api/book'
import AppNav from '@/components/AppNav.vue'
import EmptyState from '@/components/EmptyState.vue'
import EventPicker from '@/components/EventPicker.vue'
import Skeleton from '@/components/Skeleton.vue'
import { useDictStore } from '@/stores/dict'
import { paletteOf } from '@/styles/palette'
import { back, confirmModal, toast } from '@/utils/nav'

interface PickedEvent {
  id: number
  name: string
}

const dictStore = useDictStore()

const id = ref(0)
const name = ref('')
const category = ref('')
const defaultEventId = ref(0)
const defaultEventName = ref('')
const description = ref('')
const pickerShow = ref(false)
const loading = ref(true)
const showSkeleton = ref(false)
const error = ref('')
const saving = ref(false)

let skeletonTimer: ReturnType<typeof setTimeout> | null = null

const isEdit = computed(() => id.value > 0)
const title = computed(() => (isEdit.value ? '编辑礼簿' : '新建礼簿'))

function categoryColor(code: string): string {
  return paletteOf(dictStore.categoryColor(code)).main
}

async function load(): Promise<void> {
  loading.value = true
  error.value = ''
  if (skeletonTimer) clearTimeout(skeletonTimer)
  skeletonTimer = setTimeout(() => {
    showSkeleton.value = true
  }, 300)
  try {
    await dictStore.ensure()
    if (!category.value) {
      category.value = dictStore.categories[0]?.code ?? 'renqing'
    }
    if (id.value) {
      const book = await fetchBookDetail(id.value)
      name.value = book.name
      category.value = book.category
      defaultEventId.value = book.defaultEventId
      defaultEventName.value = book.defaultEventName
      description.value = book.description
    }
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

function onDescriptionInput(event: Event): void {
  description.value = (event as unknown as { detail: { value: string } }).detail.value
}

function chooseCategory(code: string): void {
  category.value = code
}

function openEventPicker(): void {
  pickerShow.value = true
}

function onEventSelect(event: PickedEvent): void {
  defaultEventId.value = event.id
  defaultEventName.value = event.name
  pickerShow.value = false
}

function clearDefaultEvent(): void {
  defaultEventId.value = 0
  defaultEventName.value = ''
}

async function save(): Promise<void> {
  if (saving.value) return
  const trimmedName = name.value.trim()
  if (!trimmedName) {
    toast('请填写礼簿名称')
    return
  }
  if (!category.value) {
    toast('请选择礼簿类别')
    return
  }
  saving.value = true
  try {
    const params = {
      name: trimmedName,
      category: category.value,
      defaultEventId: defaultEventId.value,
      description: description.value.trim(),
    }
    if (isEdit.value) {
      await updateBook(id.value, params)
    } else {
      await createBook(params)
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
  const ok = await confirmModal('删除后不可恢复，礼簿里的记录也会一起删除。', '删除礼簿')
  if (!ok) return
  try {
    await deleteBook(id.value)
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
      <EmptyState title="没能打开礼簿" :description="error" action-text="重试" @action="retry" />
    </view>

    <view v-else-if="!loading" class="page">
      <view class="form-card">
        <view class="field">
          <text class="field__label">名称</text>
          <input
            class="field__input"
            :value="name"
            placeholder="如 2024 人情礼簿"
            maxlength="30"
            @input="onNameInput"
          />
        </view>
      </view>

      <view class="section">
        <text class="section__title">类别</text>
        <view class="categories">
          <view
            v-for="item in dictStore.categories"
            :key="item.code"
            class="category-chip"
            :class="{ 'category-chip--on': category === item.code }"
            @click="chooseCategory(item.code)"
          >
            <view class="category-chip__dot" :style="{ background: categoryColor(item.code) }" />
            <text>{{ item.label }}</text>
          </view>
        </view>
      </view>

      <view class="section">
        <text class="section__title">默认事项</text>
        <view class="form-card">
          <view class="field" @click="openEventPicker">
            <text class="field__label">事项</text>
            <text class="field__value" :class="{ muted: !defaultEventName }">
              {{ defaultEventName || '不指定，记一笔时再选' }}
            </text>
            <text v-if="defaultEventId" class="field__clear" @click.stop="clearDefaultEvent">清除</text>
            <text class="field__arrow">›</text>
          </view>
        </view>
        <text class="section__hint">记一笔时优先带出这个事项，可以随时更换。</text>
      </view>

      <view class="section">
        <text class="section__title">描述</text>
        <view class="form-card">
          <textarea
            class="description"
            :value="description"
            placeholder="这本礼簿用来记什么，选填"
            maxlength="120"
            @input="onDescriptionInput"
          />
        </view>
      </view>

      <view class="btn btn--primary save" :class="{ 'btn--disabled': saving }" @click="save">
        {{ saving ? '保存中…' : '保存' }}
      </view>

      <text v-if="isEdit" class="delete-link" @click="remove">删除礼簿</text>
    </view>

    <EventPicker v-model:show="pickerShow" @select="onEventSelect" />
  </view>
</template>

<style lang="scss" scoped>
.form-card {
  margin-top: 8px;
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

.section__hint {
  display: block;
  margin: 6px 2px 0;
  font-size: 12px;
  line-height: 17px;
  color: var(--c-ink-3);
}

.field__input {
  text-align: left;
}

.categories {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.category-chip {
  display: flex;
  align-items: center;
  gap: 6px;
  height: 34px;
  padding: 0 12px;
  border-radius: var(--r-s);
  background: var(--c-surface);
  color: var(--c-ink-2);
  font-size: 14px;
  box-shadow: inset 0 0 0 1px var(--c-line);
}

.category-chip--on {
  background: var(--c-primary-light);
  color: var(--c-primary);
  box-shadow: inset 0 0 0 1px var(--c-primary);
  font-weight: 500;
}

.category-chip__dot {
  width: 8px;
  height: 8px;
  border-radius: var(--r-full);
}

.field__clear {
  margin-left: 8px;
  font-size: 13px;
  color: var(--c-ink-3);
}

.description {
  width: 100%;
  height: 96px;
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
