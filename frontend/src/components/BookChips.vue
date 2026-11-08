<script setup lang="ts">
import type { BookItem } from '@/api/types'
import { paletteOf } from '@/styles/palette'

defineProps<{
  books: BookItem[]
  modelValue: number
}>()

defineEmits<{
  'update:modelValue': [value: number]
  create: []
}>()
</script>

<template>
  <scroll-view scroll-x class="chips">
    <view class="chips__inner">
      <view
        v-for="book in books"
        :key="book.id"
        class="book-chip"
        :class="{ 'book-chip--on': modelValue === book.id }"
        @click="$emit('update:modelValue', book.id)"
      >
        <view class="book-chip__dot" :style="{ background: paletteOf(book.categoryColor).main }" />
        <text class="book-chip__text">{{ book.name }}</text>
      </view>
      <view class="book-chip book-chip--create" @click="$emit('create')">
        <text>＋ 新建礼簿</text>
      </view>
    </view>
  </scroll-view>
</template>

<style lang="scss" scoped>
.chips {
  width: 100%;
  white-space: nowrap;
}

.chips__inner {
  display: flex;
  gap: 8px;
}

.book-chip {
  display: flex;
  align-items: center;
  gap: 6px;
  height: 32px;
  padding: 0 12px;
  border-radius: var(--r-s);
  background: var(--c-surface);
  box-shadow: inset 0 0 0 1px var(--c-line);
  font-size: 13px;
  color: var(--c-ink-2);
  flex: none;
}

.book-chip--on {
  background: var(--c-primary-light);
  color: var(--c-primary);
  box-shadow: inset 0 0 0 1px var(--c-primary);
  font-weight: 500;
}

.book-chip--create {
  color: var(--c-primary);
}

.book-chip__dot {
  width: 8px;
  height: 8px;
  border-radius: var(--r-full);
}
</style>
