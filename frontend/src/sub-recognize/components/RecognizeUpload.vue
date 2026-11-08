<script setup lang="ts">
// 上传步骤：选择/拖放/粘贴照片，压缩到长边 3000 的 JPEG（质量 0.85），
// 以及礼簿、事项、方向、页码、本页合计这些上下文。
import { computed, onMounted, onUnmounted, ref } from 'vue'

import type { BookItem } from '@/api/types'
import BookChips from '@/components/BookChips.vue'

import { inputValue } from '../input'

const props = defineProps<{
  books: BookItem[]
  bookId: number
  eventName: string
  direction: 'in' | 'out'
  pageNo: string
  pageTotalText: string
  uploading: boolean
  error: string
  wide: boolean
}>()

const emit = defineEmits<{
  'update:bookId': [value: number]
  'update:direction': [value: 'in' | 'out']
  'update:pageNo': [value: string]
  'update:pageTotalText': [value: string]
  'pick-event': []
  'create-book': []
  'file-change': [payload: { file: File | null; previewUrl: string }]
  start: [file: File]
}>()

const MAX_EDGE = 3000
const QUALITY = 0.85
const MAX_BYTES = 5 * 1024 * 1024
const ACCEPTED_TYPES = ['image/jpeg', 'image/png']

const source = ref<File | null>(null)
const processed = ref<File | null>(null)
const previewUrl = ref('')
const rotation = ref(0)
const busy = ref(false)
const localError = ref('')
const dragging = ref(false)
const desktop = ref(false)

const shownError = computed(() => localError.value || props.error)

const bookModel = computed({
  get: () => props.bookId,
  set: (value: number) => emit('update:bookId', value),
})

function validate(picked: File): string {
  if (!ACCEPTED_TYPES.includes(picked.type)) {
    return '只支持 jpg、png 图片'
  }
  if (picked.size > MAX_BYTES) {
    return '图片不能超过 5MB'
  }
  return ''
}

function revokePreview(): void {
  if (previewUrl.value.startsWith('blob:')) {
    URL.revokeObjectURL(previewUrl.value)
  }
}

async function render(picked: File, degrees: number): Promise<Blob> {
  const bitmap = await createImageBitmap(picked, { imageOrientation: 'from-image' })
  try {
    const ratio = Math.min(1, MAX_EDGE / Math.max(bitmap.width, bitmap.height))
    const width = Math.max(1, Math.round(bitmap.width * ratio))
    const height = Math.max(1, Math.round(bitmap.height * ratio))
    const swap = degrees % 180 !== 0
    const canvas = document.createElement('canvas')
    canvas.width = swap ? height : width
    canvas.height = swap ? width : height
    const ctx = canvas.getContext('2d')
    if (!ctx) {
      throw new Error('当前浏览器不支持图片处理')
    }
    ctx.fillStyle = '#ffffff'
    ctx.fillRect(0, 0, canvas.width, canvas.height)
    ctx.translate(canvas.width / 2, canvas.height / 2)
    ctx.rotate((degrees * Math.PI) / 180)
    ctx.drawImage(bitmap, -width / 2, -height / 2, width, height)
    const blob = await new Promise<Blob | null>((resolve) => {
      canvas.toBlob(resolve, 'image/jpeg', QUALITY)
    })
    if (!blob) {
      throw new Error('图片压缩失败，请换一张试试')
    }
    return blob
  } finally {
    bitmap.close()
  }
}

async function process(): Promise<void> {
  const current = source.value
  if (!current) return
  busy.value = true
  localError.value = ''
  try {
    const blob = await render(current, rotation.value)
    const baseName = current.name.replace(/\.[^.]+$/, '') || 'recognize'
    const next = new File([blob], `${baseName}.jpg`, { type: 'image/jpeg' })
    revokePreview()
    previewUrl.value = URL.createObjectURL(blob)
    processed.value = next
    emit('file-change', { file: next, previewUrl: previewUrl.value })
  } catch (err) {
    localError.value = err instanceof Error ? err.message : '图片处理失败'
  } finally {
    busy.value = false
  }
}

async function selectFile(picked: File): Promise<void> {
  const problem = validate(picked)
  if (problem) {
    localError.value = problem
    return
  }
  source.value = picked
  rotation.value = 0
  await process()
}

function reset(): void {
  revokePreview()
  source.value = null
  processed.value = null
  previewUrl.value = ''
  rotation.value = 0
  localError.value = ''
  dragging.value = false
  emit('file-change', { file: null, previewUrl: '' })
}

async function rotate(): Promise<void> {
  if (!source.value || busy.value) return
  rotation.value = (rotation.value + 90) % 360
  await process()
}

function openFileDialog(): void {
  // uni-app 会把模板里的 input 编译成 uni-input，file 类型只能动态创建原生元素。
  const input = document.createElement('input')
  input.type = 'file'
  input.accept = 'image/*'
  input.setAttribute('capture', 'environment')
  input.onchange = () => {
    const picked = input.files?.[0]
    if (picked) {
      void selectFile(picked)
    }
  }
  input.click()
}

function onDragOver(event: DragEvent): void {
  event.preventDefault()
  dragging.value = true
}

function onDragLeave(): void {
  dragging.value = false
}

function onDrop(event: DragEvent): void {
  event.preventDefault()
  dragging.value = false
  const dropped = event.dataTransfer?.files?.[0]
  if (dropped) {
    void selectFile(dropped)
  }
}

function onPaste(event: ClipboardEvent): void {
  const items = event.clipboardData?.items
  if (!items) return
  for (let index = 0; index < items.length; index += 1) {
    const item = items[index]
    if (item.kind === 'file' && item.type.startsWith('image/')) {
      const picked = item.getAsFile()
      if (picked) {
        event.preventDefault()
        void selectFile(picked)
        return
      }
    }
  }
}

function onPageNoInput(event: Event): void {
  emit('update:pageNo', inputValue(event))
}

function onPageTotalInput(event: Event): void {
  emit('update:pageTotalText', inputValue(event))
}

function start(): void {
  if (!processed.value) {
    localError.value = '先选择一张礼簿照片'
    return
  }
  emit('start', processed.value)
}

onMounted(() => {
  desktop.value = window.matchMedia('(pointer: fine)').matches
  window.addEventListener('paste', onPaste)
})

onUnmounted(() => {
  window.removeEventListener('paste', onPaste)
})

defineExpose({
  currentFile: () => processed.value,
  reset,
})
</script>

<template>
  <view class="upload" :class="{ 'upload--wide': wide }">
    <view
      class="drop"
      :class="{ 'drop--filled': previewUrl, 'drop--drag': dragging }"
      @dragover="onDragOver"
      @dragleave="onDragLeave"
      @drop="onDrop"
      @click="openFileDialog"
    >
      <image v-if="previewUrl" class="drop__image" :src="previewUrl" mode="aspectFit" />
      <template v-else>
        <text class="drop__seal">簿</text>
        <text class="drop__title">{{ busy ? '正在处理图片…' : '拍照或选择礼簿照片' }}</text>
        <text class="drop__hint">支持 jpg / png，不超过 5MB</text>
        <text v-if="desktop" class="drop__hint drop__hint--desktop">也可以拖放文件，或按 Ctrl/⌘ + V 粘贴</text>
      </template>
    </view>

    <view v-if="previewUrl" class="drop-actions">
      <text class="drop-actions__btn" @click="rotate">{{ busy ? '处理中…' : '旋转' }}</text>
      <text class="drop-actions__btn" @click="openFileDialog">重选</text>
      <text class="drop-actions__btn" @click="reset">移除</text>
    </view>

    <view class="card context">
      <view class="field">
        <text class="field__label">礼簿</text>
        <view class="field__value">
          <BookChips v-model="bookModel" :books="books" @create="$emit('create-book')" />
        </view>
      </view>
      <view class="field" @click="$emit('pick-event')">
        <text class="field__label">事项</text>
        <text class="field__value">{{ eventName || '选填，礼簿默认事项可用' }}</text>
        <text class="field__arrow">›</text>
      </view>
      <view class="field">
        <text class="field__label">方向</text>
        <view class="segment">
          <text
            class="segment__item"
            :class="{ 'segment__item--on': direction === 'in' }"
            @click="$emit('update:direction', 'in')"
          >
            收礼
          </text>
          <text
            class="segment__item"
            :class="{ 'segment__item--on': direction === 'out' }"
            @click="$emit('update:direction', 'out')"
          >
            送礼
          </text>
        </view>
      </view>
      <view class="field">
        <text class="field__label">页码</text>
        <input class="field__input" :value="pageNo" placeholder="如 1-10" @input="onPageNoInput" />
      </view>
      <view class="field">
        <text class="field__label">本页合计</text>
        <input
          class="field__input"
          type="digit"
          :value="pageTotalText"
          placeholder="纸本页脚写的合计"
          @input="onPageTotalInput"
        />
      </view>
    </view>

    <text v-if="shownError" class="error">{{ shownError }}</text>
    <view class="btn btn--primary start" :class="{ 'btn--disabled': uploading || busy }" @click="start">
      {{ uploading ? '上传中…' : '开始识别' }}
    </view>
    <text class="hint">识别结果只是草稿，核对确认后才会写入账目。</text>
  </view>
</template>

<style lang="scss" scoped>
.drop {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  height: 220px;
  border: 1.5px dashed var(--c-line);
  border-radius: var(--r-m);
  background: var(--c-surface);
  overflow: hidden;
}

.upload--wide .drop {
  height: 280px;
}

.drop--drag {
  border-color: var(--c-primary);
  background: var(--c-primary-light);
}

.drop--filled {
  border-style: solid;
  padding: 0;
  cursor: default;
}

.drop__image {
  width: 100%;
  height: 100%;
}

.drop__seal {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 56px;
  height: 56px;
  border: 1px solid var(--c-none);
  border-radius: 14px;
  color: var(--c-none);
  font-family: $font-serif;
  font-size: 26px;
  opacity: 0.6;
}

.drop__title {
  margin-top: 12px;
  font-size: 15px;
  font-weight: 500;
}

.drop__hint {
  margin-top: 4px;
  font-size: 12px;
  color: var(--c-ink-3);
  text-align: center;
}

.drop__hint--desktop {
  margin-top: 8px;
  padding: 4px 10px;
  border-radius: var(--r-full);
  background: var(--c-fill);
  color: var(--c-ink-2);
}

.drop-actions {
  display: flex;
  justify-content: flex-end;
  gap: 16px;
  padding: 8px 4px 0;
}

.drop-actions__btn {
  font-size: 13px;
  color: var(--c-ink-2);
}

.context {
  margin-top: 12px;
  overflow: hidden;
}

.context .field {
  padding: 0 12px;
}

.segment {
  display: flex;
  gap: 4px;
  padding: 2px;
  border-radius: var(--r-s);
  background: var(--c-fill);
}

.segment__item {
  padding: 6px 14px;
  border-radius: 8px;
  font-size: 13px;
  color: var(--c-ink-2);
}

.segment__item--on {
  background: var(--c-surface);
  color: var(--c-ink-1);
  font-weight: 500;
}

.start {
  margin-top: 16px;
}

.hint {
  display: block;
  margin-top: 10px;
  font-size: 12px;
  color: var(--c-ink-3);
  text-align: center;
}

.error {
  display: block;
  margin-top: 10px;
  font-size: 13px;
  color: var(--c-danger);
}
</style>
