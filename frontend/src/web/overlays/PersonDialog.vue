<script setup lang="ts">
// 新建 / 编辑亲友：姓名 + 备注唯一；同名的不同人必须靠备注区分。
import { reactive, ref } from 'vue'

import { ApiError } from '@/api/http'
import { createPerson, updatePerson } from '@/api/person'
import type { PersonItem } from '@/api/types'
import Modal from '@/web/components/Modal.vue'

const props = defineProps<{ person: PersonItem | null }>()
const emit = defineEmits<{ close: []; saved: [person: PersonItem] }>()

const saving = ref(false)
const error = ref('')
const duplicate = ref(false)
const form = reactive({
  name: props.person?.name ?? '',
  remark: props.person?.remark ?? '',
})

async function save(): Promise<void> {
  if (!form.name.trim()) {
    error.value = '请填写姓名'
    return
  }
  saving.value = true
  error.value = ''
  duplicate.value = false
  const params = { name: form.name.trim(), remark: form.remark.trim() }
  try {
    const saved = props.person ? await updatePerson(props.person.id, params) : await createPerson(params)
    emit('saved', saved)
  } catch (err) {
    duplicate.value = err instanceof ApiError && err.custom?.reason === 'duplicate_person'
    error.value = err instanceof Error ? err.message : '保存失败'
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <div class="web-mask" @click="emit('close')" />
  <Modal :title="person ? '编辑亲友' : '新建亲友'" @close="emit('close')">
    <div v-if="error" class="web-alert" :class="duplicate ? 'web-alert--warn' : 'web-alert--err'">{{ error }}</div>
    <div class="web-field-row">
      <div class="web-field">
        <label for="person-name">姓名</label>
        <input id="person-name" v-model="form.name" class="web-input" maxlength="32" placeholder="怎么称呼" />
      </div>
      <div class="web-field">
        <label for="person-remark">备注</label>
        <input id="person-remark" v-model="form.remark" class="web-input" maxlength="32" placeholder="选填，如 村东、表叔" />
      </div>
    </div>
    <span class="web-field__hint">同名的不同人用备注区分，列表里显示为「张三 · 村东」。</span>
    <template #footer>
      <button class="web-btn" type="button" @click="emit('close')">取消</button>
      <button class="web-btn web-btn--primary" type="button" :disabled="saving" @click="save">
        {{ saving ? '保存中…' : '保存' }}
      </button>
    </template>
  </Modal>
</template>
