<script setup lang="ts">
import { onShow } from '@dcloudio/uni-app'
import { computed, ref } from 'vue'

import { fetchPersons } from '@/api/person'
import type { PersonBrief } from '@/api/types'
import AppNav from '@/components/AppNav.vue'
import EmptyState from '@/components/EmptyState.vue'
import FabButton from '@/components/FabButton.vue'
import Money from '@/components/Money.vue'
import PersonAvatar from '@/components/PersonAvatar.vue'
import Skeleton from '@/components/Skeleton.vue'
import { relations } from '@/styles/palette'
import { themeVars } from '@/styles/theme'
import { formatDate } from '@/utils/date'
import { go, toast } from '@/utils/nav'

type SortKey = 'recent' | 'name' | 'pending'

const keyword = ref('')
const relation = ref('')
const sort = ref<SortKey>('recent')
const sortSheet = ref(false)

const persons = ref<PersonBrief[]>([])
const loading = ref(false)
const loaded = ref(false)
const error = ref('')
const skeleton = ref(false)

let timer: ReturnType<typeof setTimeout> | null = null
let skeletonTimer: ReturnType<typeof setTimeout> | null = null
let seq = 0

const relationOptions = [{ code: '', label: '全部' }, ...relations]
const sortOptions: Array<{ code: SortKey; label: string }> = [
  { code: 'recent', label: '最近往来' },
  { code: 'name', label: '姓名' },
  { code: 'pending', label: '待还优先' },
]
const sortLabel = computed(() => sortOptions.find((item) => item.code === sort.value)?.label ?? '最近往来')
const hasFilter = computed(() => Boolean(keyword.value.trim() || relation.value))

const sorted = computed(() => {
  const list = [...persons.value]
  if (sort.value === 'name') {
    return list.sort(
      (a, b) => a.name.localeCompare(b.name, 'zh-Hans-CN', { sensitivity: 'base' }) || a.id - b.id,
    )
  }
  if (sort.value === 'pending') {
    return list.sort(
      (a, b) =>
        b.pendingCount - a.pendingCount ||
        (b.lastDate || '').localeCompare(a.lastDate || '') ||
        a.name.localeCompare(b.name, 'zh-Hans-CN', { sensitivity: 'base' }),
    )
  }
  return list.sort(
    (a, b) => (b.lastDate || '').localeCompare(a.lastDate || '') || b.updatedAt - a.updatedAt,
  )
})

function inputValue(event: Event): string {
  return (event as unknown as { detail: { value: string } }).detail.value
}

function startSkeleton(): void {
  if (loaded.value) return
  if (skeletonTimer) clearTimeout(skeletonTimer)
  skeletonTimer = setTimeout(() => {
    skeleton.value = true
  }, 300)
}

function stopSkeleton(): void {
  if (skeletonTimer) {
    clearTimeout(skeletonTimer)
    skeletonTimer = null
  }
  skeleton.value = false
}

async function fetchAll(): Promise<PersonBrief[]> {
  const all: PersonBrief[] = []
  let page = 1
  let total = 0
  for (let round = 0; round < 20; round += 1) {
    const result = await fetchPersons({
      keyword: keyword.value.trim(),
      relation: relation.value,
      page,
      pageSize: 100,
    })
    total = result.total
    all.push(...result.items)
    if (!result.items.length || all.length >= total) break
    page += 1
  }
  return all
}

async function load(): Promise<void> {
  const current = ++seq
  loading.value = true
  error.value = ''
  startSkeleton()
  try {
    const items = await fetchAll()
    if (current !== seq) return
    persons.value = items
  } catch (err) {
    if (current !== seq) return
    persons.value = []
    error.value = err instanceof Error ? err.message : '加载失败'
    toast(error.value)
  } finally {
    if (current === seq) {
      loading.value = false
      loaded.value = true
      stopSkeleton()
    }
  }
}

function scheduleLoad(): void {
  if (timer) clearTimeout(timer)
  timer = setTimeout(() => {
    load()
  }, 200)
}

function onKeywordInput(event: Event): void {
  keyword.value = inputValue(event)
  scheduleLoad()
}

function clearKeyword(): void {
  keyword.value = ''
  load()
}

function chooseRelation(code: string): void {
  if (relation.value === code) return
  relation.value = code
  load()
}

function chooseSort(code: SortKey): void {
  sort.value = code
  sortSheet.value = false
}

function clearFilters(): void {
  keyword.value = ''
  relation.value = ''
  sort.value = 'recent'
  load()
}

function openPerson(person: PersonBrief): void {
  go(`/sub-person/detail?id=${person.id}`)
}

function createPerson(): void {
  go('/sub-person/edit')
}

onShow(load)
</script>

<template>
  <wd-config-provider :theme-vars="themeVars">
    <view class="screen screen--tab">
      <AppNav :back="false" title="" />

      <view class="page">
        <view class="toolbar">
          <view class="search-box">
            <text class="search-box__icon">⌕</text>
            <input
              class="search-box__input"
              :value="keyword"
              placeholder="搜索姓名、拼音首字母"
              confirm-type="search"
              @input="onKeywordInput"
            />
            <text v-if="keyword" class="search-box__clear" @click="clearKeyword">×</text>
          </view>

          <view class="relations">
            <view class="relations__row">
              <text
                v-for="option in relationOptions"
                :key="option.code"
                class="chip"
                :class="{ 'chip--on': relation === option.code }"
                @click="chooseRelation(option.code)"
              >
                {{ option.label }}
              </text>
            </view>
          </view>

          <view class="sortline">
            <text class="sortline__text" @click="sortSheet = true">排序：{{ sortLabel }} <text class="sortline__caret">▾</text></text>
            <text v-if="!loading && loaded" class="sortline__count">共 {{ persons.length }} 人</text>
          </view>
        </view>

        <Skeleton v-if="skeleton && !persons.length" type="list" />

        <template v-else>
          <view v-if="error && !persons.length" class="load-error">
            <text class="load-error__text">{{ error }}</text>
            <view class="btn btn--secondary btn--small" @click="load">重新加载</view>
          </view>

          <EmptyState
            v-else-if="loaded && !persons.length && hasFilter"
            title="没有找到亲友"
            description="换个关键词，或试试拼音首字母"
            action-text="清除筛选"
            @action="clearFilters"
          />
          <EmptyState
            v-else-if="loaded && !persons.length"
            title="还没有亲友"
            description="记账时会自动添加"
            action-text="记一笔"
            @action="go('/sub-record/form')"
          />

          <view v-else-if="persons.length" class="card list">
            <view v-for="person in sorted" :key="person.id" class="person" @click="openPerson(person)">
              <PersonAvatar :id="person.id" :name="person.name" :size="40" />
              <view class="person__main">
                <view class="person__line">
                  <text class="person__name">{{ person.name }}</text>
                  <text class="person__relation">{{ person.relationLabel }}</text>
                </view>
                <view class="person__meta">
                  <text v-if="person.lastDate">最近 {{ formatDate(person.lastDate) }}</text>
                  <text v-else>还没有往来</text>
                  <text v-if="person.pendingCount" class="person__pending"> · {{ person.pendingCount }} 笔待还</text>
                </view>
              </view>
              <view class="person__money">
                <view class="person__money-line">
                  <text class="person__money-label">收</text>
                  <Money :cent="person.inCent" direction="in" size="s" />
                </view>
                <view class="person__money-line">
                  <text class="person__money-label">送</text>
                  <Money :cent="person.outCent" direction="out" size="s" />
                </view>
              </view>
            </view>
          </view>

          <view v-if="loading && persons.length" class="loading-tip">正在查找…</view>
        </template>
      </view>

      <FabButton @click="createPerson" />

      <wd-popup
        v-model="sortSheet"
        position="bottom"
        :safe-area-inset-bottom="true"
        custom-style="border-radius: 20px 20px 0 0;"
      >
        <view class="sort">
          <view class="sort__head">
            <text class="sort__title">排序</text>
            <text class="sort__close" @click="sortSheet = false">关闭</text>
          </view>
          <view
            v-for="option in sortOptions"
            :key="option.code"
            class="sort__option"
            @click="chooseSort(option.code)"
          >
            <text class="sort__label">{{ option.label }}</text>
            <text v-if="sort === option.code" class="sort__check">✓</text>
          </view>
        </view>
      </wd-popup>
    </view>
  </wd-config-provider>
</template>

<style lang="scss" scoped>
.toolbar {
  position: sticky;
  top: 0;
  z-index: 5;
  margin: 0 calc(-1 * var(--gutter));
  padding: 4px var(--gutter) 6px;
  background: var(--c-bg);
}

.search-box__clear {
  padding: 0 2px;
  font-size: 18px;
  line-height: 1;
  color: var(--c-ink-3);
}

.relations {
  width: 100%;
  margin-top: 10px;
  white-space: nowrap;
}

.relations__row {
  display: flex;
  gap: 8px;
}

.sortline {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-top: 10px;
  padding: 0 2px;
}

.sortline__text {
  font-size: 13px;
  color: var(--c-ink-2);
}

.sortline__caret {
  font-size: 11px;
  color: var(--c-ink-3);
}

.sortline__count {
  font-size: 12px;
  color: var(--c-ink-3);
}

.load-error {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 12px;
  padding: 40px 16px;
}

.load-error__text {
  font-size: 13px;
  color: var(--c-ink-3);
}

.list {
  margin-top: 12px;
  overflow: hidden;
}

.person {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px 16px;
  background: var(--c-surface);
}

.person + .person {
  box-shadow: inset 0 1px 0 var(--c-divider);
}

.person__main {
  flex: 1;
  min-width: 0;
}

.person__line {
  display: flex;
  align-items: center;
  gap: 6px;
}

.person__name {
  font-size: 15px;
  line-height: 22px;
  font-weight: 500;
  color: var(--c-ink-1);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.person__relation {
  flex: none;
  height: 20px;
  padding: 0 6px;
  border-radius: var(--r-xs);
  background: var(--c-fill);
  color: var(--c-ink-2);
  font-size: 12px;
  line-height: 20px;
}

.person__meta {
  margin-top: 1px;
  font-size: 12px;
  line-height: 18px;
  color: var(--c-ink-3);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.person__pending {
  color: var(--c-pending);
}

.person__money {
  flex: none;
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  gap: 1px;
}

.person__money-line {
  display: flex;
  align-items: baseline;
  gap: 2px;
}

.person__money-label {
  font-size: 11px;
  color: var(--c-ink-3);
}

.sort {
  padding: 16px 16px calc(16px + env(safe-area-inset-bottom));
}

.sort__head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 8px;
}

.sort__title {
  font-size: 17px;
  font-weight: 600;
}

.sort__close {
  font-size: 14px;
  color: var(--c-ink-3);
}

.sort__option {
  display: flex;
  align-items: center;
  justify-content: space-between;
  min-height: 52px;
  box-shadow: inset 0 1px 0 var(--c-divider);
}

.sort__label {
  font-size: 15px;
  color: var(--c-ink-1);
}

.sort__check {
  color: var(--c-primary);
  font-size: 16px;
}
</style>
