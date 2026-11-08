<script setup lang="ts">
import { computed, ref, watch } from 'vue'

import { ApiError } from '@/api/http'
import { createPerson, fetchPersons, fetchRecentPersons } from '@/api/person'
import type { PersonBrief } from '@/api/types'
import { relations } from '@/styles/palette'

import PersonAvatar from './PersonAvatar.vue'

const props = defineProps<{ show: boolean }>()
const emit = defineEmits<{
  'update:show': [value: boolean]
  select: [person: PersonBrief]
}>()

const visible = computed({
  get: () => props.show,
  set: (value: boolean) => emit('update:show', value),
})

const keyword = ref('')
const results = ref<PersonBrief[]>([])
const recent = ref<PersonBrief[]>([])
const loading = ref(false)
const creating = ref(false)
const createName = ref('')
const createRelation = ref('other')
const createError = ref('')
const inlineConfirm = ref(false)
let searchTimer: ReturnType<typeof setTimeout> | null = null

const hasExact = computed(() => {
  const name = keyword.value.trim()
  if (!name) return true
  return results.value.some((item) => item.name === name)
})

function segments(name: string, query: string): { text: string; hit: boolean }[] {
  const q = query.trim()
  if (!q) return [{ text: name, hit: false }]
  const index = name.toLowerCase().indexOf(q.toLowerCase())
  if (index < 0) return [{ text: name, hit: false }]
  return [
    { text: name.slice(0, index), hit: false },
    { text: name.slice(index, index + q.length), hit: true },
    { text: name.slice(index + q.length), hit: false },
  ].filter((part) => part.text)
}

async function loadRecent(): Promise<void> {
  try {
    recent.value = await fetchRecentPersons(8)
  } catch {
    recent.value = []
  }
}

async function search(value: string): Promise<void> {
  keyword.value = value
  if (searchTimer) clearTimeout(searchTimer)
  if (!value.trim()) {
    results.value = []
    return
  }
  searchTimer = setTimeout(async () => {
    loading.value = true
    try {
      const page = await fetchPersons({ keyword: value.trim(), pageSize: 20 })
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

function pick(person: PersonBrief): void {
  emit('select', person)
  visible.value = false
}

function startCreate(): void {
  createName.value = keyword.value.trim()
  createRelation.value = 'other'
  createError.value = ''
  inlineConfirm.value = false
  creating.value = true
}

async function submitCreate(confirm = false): Promise<void> {
  if (!createName.value.trim()) return
  createError.value = ''
  try {
    const person = await createPerson({
      name: createName.value.trim(),
      relation: createRelation.value,
      confirm,
    })
    pick({
      id: person.id,
      name: person.name,
      relation: person.relation,
      relationLabel: person.relationLabel,
      phone: person.phone,
      remark: person.remark,
      createdAt: person.createdAt,
      updatedAt: person.updatedAt,
      inCent: 0,
      outCent: 0,
      pendingCount: 0,
      lastDate: '',
    })
  } catch (error) {
    if (error instanceof ApiError && error.code === 409 && (error.custom as Record<string, unknown> | null)?.reason === 'need_confirm') {
      inlineConfirm.value = true
      createError.value = error.message
      return
    }
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
    inlineConfirm.value = false
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
        <text class="picker__title">选择亲友</text>
        <text class="picker__cancel" @click="visible = false">取消</text>
      </view>
      <view class="picker__search">
        <text class="picker__icon">⌕</text>
        <input
          class="picker__input"
          :value="keyword"
          placeholder="姓名 / 拼音首字母"
          confirm-type="search"
          focus
          @input="onKeywordInput"
        />
      </view>

      <scroll-view scroll-y class="picker__body">
        <view v-if="!keyword && recent.length" class="picker__recent">
          <text class="picker__label">最近</text>
          <view class="picker__chips">
            <text v-for="person in recent" :key="person.id" class="chip chip--flat" @click="pick(person)">
              {{ person.name }}
            </text>
          </view>
        </view>

        <view v-if="!creating && keyword.trim() && !hasExact" class="picker__create" @click="startCreate">
          ＋ 新建「{{ keyword.trim() }}」
        </view>

        <view v-if="creating" class="picker__form">
          <input
            class="field"
            :value="createName"
            placeholder="姓名"
            @input="onNameInput"
          />
          <view class="picker__relations">
            <text
              v-for="relation in relations"
              :key="relation.code"
              class="chip"
              :class="{ 'chip--on': createRelation === relation.code }"
              @click="createRelation = relation.code"
            >
              {{ relation.label }}
            </text>
          </view>
          <view v-if="inlineConfirm" class="inline-confirm">
            <text class="inline-confirm__text">{{ createError }}</text>
            <view class="inline-confirm__actions">
              <text class="inline-confirm__cancel" @click="inlineConfirm = false">取消</text>
              <text class="inline-confirm__ok" @click="submitCreate(true)">仍然新建</text>
            </view>
          </view>
          <text v-else-if="createError" class="picker__error">{{ createError }}</text>
          <view class="picker__form-actions">
            <view class="btn btn--secondary picker__form-btn" @click="creating = false">取消</view>
            <view class="btn btn--primary picker__form-btn" @click="submitCreate(false)">确定</view>
          </view>
        </view>

        <view v-if="!creating">
          <view v-if="loading" class="picker__hint">正在查找…</view>
          <view v-for="person in results" :key="person.id" class="picker__row" @click="pick(person)">
            <PersonAvatar :id="person.id" :name="person.name" :size="32" />
            <view class="picker__row-main">
              <text class="picker__name">
                <text v-for="(part, index) in segments(person.name, keyword)" :key="index" :class="{ 'picker__hit': part.hit }">{{ part.text }}</text>
              </text>
              <text class="picker__meta">{{ person.relationLabel }}<text v-if="person.pendingCount"> · {{ person.pendingCount }} 笔待还</text></text>
            </view>
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

.picker__recent {
  padding: 6px 16px;
}

.picker__label {
  font-size: 12px;
  color: var(--c-ink-3);
}

.picker__chips {
  display: flex;
  gap: 8px;
  margin-top: 6px;
  overflow-x: auto;
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

.picker__relations {
  display: flex;
  gap: 8px;
  margin-top: 10px;
}

.picker__error {
  display: block;
  margin-top: 8px;
  font-size: 12px;
  color: var(--c-danger);
}

.picker__form-actions {
  display: flex;
  gap: 10px;
  margin-top: 14px;
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

.picker__hit {
  color: var(--c-primary);
}

.picker__meta {
  display: block;
  font-size: 12px;
  color: var(--c-ink-3);
}

.picker__pending {
  font-size: 12px;
  color: var(--c-pending);
}

.picker__hint {
  padding: 14px 16px;
  font-size: 13px;
  color: var(--c-ink-3);
}

.field {
  height: 44px;
  padding: 0 12px;
  border-radius: var(--r-s);
  background: var(--c-fill);
  font-size: 15px;
}

.inline-confirm {
  margin-top: 10px;
  padding: 10px 12px;
  border-radius: var(--r-s);
  background: var(--c-pending-light);
}

.inline-confirm__text {
  font-size: 13px;
  color: #7a4e07;
}

.inline-confirm__actions {
  display: flex;
  justify-content: flex-end;
  gap: 14px;
  margin-top: 8px;
  font-size: 13px;
  font-weight: 500;
}

.inline-confirm__cancel {
  color: var(--c-ink-3);
}

.inline-confirm__ok {
  color: var(--c-primary);
}
</style>
