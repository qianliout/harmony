<script setup lang="ts">
// 新建 / 编辑礼簿：名称、种类、备注。种类预置三类，也可直接输入自定义值。
import { reactive, ref } from 'vue'

import { createBook, updateBook } from '@/api/book'
import type { BookItem } from '@/api/types'
import { bookCategories } from '@/styles/palette'
import Modal from '@/web/components/Modal.vue'

const props = defineProps<{ book: BookItem | null }>()
const emit = defineEmits<{ close: []; saved: [book: BookItem] }>()

const saving = ref(false)
const error = ref('')
const form = reactive({
  name: props.book?.name ?? '',
  category: props.book?.category ?? bookCategories[0].code,
  description: props.book?.description ?? '',
})

async function save(): Promise<void> {
  if (!form.name.trim()) {
    error.value = '请填写礼簿名称'
    return
  }
  saving.value = true
  error.value = ''
  const params = { name: form.name.trim(), category: form.category.trim(), description: form.description.trim() }
  try {
    const saved = props.book ? await updateBook(props.book.id, params) : await createBook(params)
    emit('saved', saved)
  } catch (err) {
    error.value = err instanceof Error ? err.message : '保存失败'
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <div class="web-mask" @click="emit('close')" />
  <Modal :title="book ? '编辑礼簿' : '新建礼簿'" @close="emit('close')">
    <div v-if="error" class="web-alert web-alert--err">{{ error }}</div>
    <div class="web-field-row">
      <div class="web-field">
        <label for="book-name">名称</label>
        <input id="book-name" v-model="form.name" class="web-input" maxlength="64" placeholder="如 2024 年婚宴礼簿" />
      </div>
      <div class="web-field">
        <label for="book-category">种类</label>
        <input id="book-category" v-model="form.category" class="web-input" list="book-category-options" maxlength="32" />
        <datalist id="book-category-options">
          <option v-for="item in bookCategories" :key="item.code" :value="item.code">{{ item.label }}</option>
        </datalist>
        <span class="web-field__hint">renqing 人情礼 · huopao 火炮 · other 其他，也可输入自定义种类</span>
      </div>
    </div>
    <div class="web-field">
      <label for="book-desc">备注</label>
      <textarea id="book-desc" v-model="form.description" class="web-textarea" rows="3" placeholder="存放位置、起始年份等" />
    </div>
    <template #footer>
      <button class="web-btn" type="button" @click="emit('close')">取消</button>
      <button class="web-btn web-btn--primary" type="button" :disabled="saving" @click="save">
        {{ saving ? '保存中…' : '保存' }}
      </button>
    </template>
  </Modal>
</template>
