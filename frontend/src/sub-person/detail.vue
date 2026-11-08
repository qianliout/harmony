<script setup lang="ts">
import { onBackPress, onLoad, onShow } from '@dcloudio/uni-app'
import { computed, ref } from 'vue'

import { fetchPersonDetail } from '@/api/person'
import type { PersonDetail, RecordFilters, RecordItem, TimelineGroup } from '@/api/types'
import { fetchPersonTimeline, type TimelineParams } from '@/api/view'
import AppNav from '@/components/AppNav.vue'
import EmptyState from '@/components/EmptyState.vue'
import EventGroup from '@/components/EventGroup.vue'
import FilterBar from '@/components/FilterBar.vue'
import FilterSheet from '@/components/FilterSheet.vue'
import Money from '@/components/Money.vue'
import PersonAvatar from '@/components/PersonAvatar.vue'
import Skeleton from '@/components/Skeleton.vue'
import SummaryStrip from '@/components/SummaryStrip.vue'
import { themeVars } from '@/styles/theme'
import { back, go, toast } from '@/utils/nav'

function emptyFilters(): RecordFilters {
  return { bookId: 0, eventId: 0, itemKind: '', direction: '', status: '', from: '', to: '' }
}

const personId = ref(0)
const person = ref<PersonDetail | null>(null)
const personLoading = ref(false)
const personError = ref('')

const groups = ref<TimelineGroup[]>([])
const timelineLoading = ref(false)
const timelineLoaded = ref(false)
const timelineError = ref('')
const skeleton = ref(false)

const quick = ref('all')
const filterSheet = ref(false)
const filters = ref<RecordFilters>(emptyFilters())

const selectMode = ref(false)
const selectedIds = ref<number[]>([])
let previousQuick = 'all'
let previousFilters: RecordFilters = emptyFilters()

let personSeq = 0
let timelineSeq = 0
let skeletonTimer: ReturnType<typeof setTimeout> | null = null

const pendingCount = computed(() => person.value?.summary.pendingCount ?? 0)
const phoneLabel = computed(() => {
  const phone = person.value?.phone ?? ''
  if (phone.length === 11) {
    return `${phone.slice(0, 3)}****${phone.slice(7)}`
  }
  return phone
})

const activeFilterCount = computed(() => {
  const values = [
    filters.value.bookId,
    filters.value.eventId,
    filters.value.itemKind,
    filters.value.direction,
    filters.value.status,
  ]
  let count = values.filter(Boolean).length
  if (filters.value.from || filters.value.to) count += 1
  return count
})

const selectedCent = computed(() => {
  if (!selectedIds.value.length) return 0
  let total = 0
  groups.value.forEach((group) => {
    group.records.forEach((record) => {
      if (selectedIds.value.includes(record.id)) {
        total += record.totalCent
      }
    })
  })
  return total
})

const hasAnyFilter = computed(() => activeFilterCount.value > 0 || quick.value !== 'all')
const emptyTitle = computed(() => {
  if (quick.value === 'pending' && !activeFilterCount.value) return '人情都还清了'
  if (hasAnyFilter.value) return '没有符合条件的记录'
  return '还没有记录'
})
const emptyDescription = computed(() => {
  if (quick.value === 'pending' && !activeFilterCount.value) return '有新的收礼时会出现在这里'
  if (hasAnyFilter.value) return ''
  return `给 ${person.value?.name ?? '这位亲友'} 记下第一笔人情往来吧`
})
const emptyActionText = computed(() => {
  if (quick.value === 'pending' && !activeFilterCount.value) return ''
  return hasAnyFilter.value ? '清除筛选' : '记一笔'
})

function onEmptyAction(): void {
  if (hasAnyFilter.value) {
    clearAllFilters()
    return
  }
  addRecord()
}

function startSkeleton(): void {
  if (timelineLoaded.value) return
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

function buildParams(): TimelineParams {
  const params: TimelineParams = { personId: personId.value }
  if (filters.value.bookId) params.bookId = filters.value.bookId
  if (filters.value.eventId) params.eventId = filters.value.eventId
  if (filters.value.itemKind) params.itemKind = filters.value.itemKind
  if (filters.value.direction) params.direction = filters.value.direction
  if (filters.value.status) params.status = filters.value.status
  if (filters.value.from) params.from = filters.value.from
  if (filters.value.to) params.to = filters.value.to
  if (quick.value === 'pending') {
    params.status = 'pending'
  } else if (quick.value === 'in' || quick.value === 'out') {
    params.direction = quick.value
  }
  return params
}

async function loadPerson(): Promise<void> {
  if (!personId.value) return
  const current = ++personSeq
  personLoading.value = true
  personError.value = ''
  try {
    const detail = await fetchPersonDetail(personId.value)
    if (current !== personSeq) return
    person.value = detail
  } catch (err) {
    if (current !== personSeq) return
    person.value = null
    personError.value = err instanceof Error ? err.message : '加载失败'
    toast(personError.value)
  } finally {
    if (current === personSeq) {
      personLoading.value = false
    }
  }
}

async function loadTimeline(): Promise<void> {
  if (!personId.value) return
  const current = ++timelineSeq
  timelineLoading.value = true
  timelineError.value = ''
  startSkeleton()
  try {
    const data = await fetchPersonTimeline(buildParams())
    if (current !== timelineSeq) return
    groups.value = data.groups ?? []
    if (selectedIds.value.length) {
      const existing = new Set<number>()
      groups.value.forEach((group) => {
        group.records.forEach((record) => existing.add(record.id))
      })
      selectedIds.value = selectedIds.value.filter((id) => existing.has(id))
    }
  } catch (err) {
    if (current !== timelineSeq) return
    groups.value = []
    timelineError.value = err instanceof Error ? err.message : '加载失败'
    toast(timelineError.value)
  } finally {
    if (current === timelineSeq) {
      timelineLoading.value = false
      timelineLoaded.value = true
      stopSkeleton()
    }
  }
}

function loadAll(): void {
  loadPerson()
  loadTimeline()
}

function onQuickChange(value: string): void {
  if (quick.value === value) return
  quick.value = value
  loadTimeline()
}

function openFilters(): void {
  if (selectMode.value) return
  filterSheet.value = true
}

function applyFilters(next: RecordFilters): void {
  filters.value = { ...next }
  quick.value = 'all'
  loadTimeline()
}

function resetFilters(): void {
  filters.value = emptyFilters()
  quick.value = 'all'
  loadTimeline()
}

function clearAllFilters(): void {
  filters.value = emptyFilters()
  quick.value = 'all'
  loadTimeline()
}

function onSummaryPending(): void {
  if (selectMode.value || quick.value === 'pending') return
  quick.value = 'pending'
  loadTimeline()
}

function enterSelect(): void {
  if (!pendingCount.value) {
    toast('没有待还的记录')
    return
  }
  previousQuick = quick.value
  previousFilters = { ...filters.value }
  filters.value = emptyFilters()
  selectMode.value = true
  selectedIds.value = []
  quick.value = 'pending'
  loadTimeline()
}

function cancelSelect(): void {
  selectMode.value = false
  selectedIds.value = []
  quick.value = previousQuick
  filters.value = { ...previousFilters }
  loadTimeline()
}

function toggleRecord(record: RecordItem): void {
  if (record.direction !== 'in' || record.status !== 'pending') return
  const index = selectedIds.value.indexOf(record.id)
  if (index >= 0) {
    selectedIds.value.splice(index, 1)
  } else {
    selectedIds.value.push(record.id)
  }
}

function selectAll(group: TimelineGroup): void {
  group.records.forEach((record) => {
    if (record.direction === 'in' && record.status === 'pending' && !selectedIds.value.includes(record.id)) {
      selectedIds.value.push(record.id)
    }
  })
}

function nextStep(): void {
  if (!selectedIds.value.length) {
    toast('先勾选要还的记录')
    return
  }
  const ids = selectedIds.value.join(',')
  selectMode.value = false
  selectedIds.value = []
  go(`/sub-record/form?mode=repay&originIds=${ids}`)
}

function addRecord(): void {
  go(`/sub-record/form?personId=${personId.value}`)
}

function openRecord(record: RecordItem): void {
  go(`/sub-record/detail?id=${record.id}`)
}

function openEdit(): void {
  if (!personId.value) return
  go(`/sub-person/edit?id=${personId.value}`)
}

function callPhone(): void {
  const phone = person.value?.phone ?? ''
  if (!phone) return
  uni.makePhoneCall({
    phoneNumber: phone,
    fail: () => toast('无法拨号'),
  })
}

onLoad((query) => {
  const id = Number(query?.id ?? 0)
  if (!id) {
    toast('亲友不存在')
    back()
    return
  }
  personId.value = id
  if (query?.status === 'pending') {
    quick.value = 'pending'
  }
})

onShow(() => {
  if (!personId.value) return
  loadAll()
})

onBackPress(() => {
  if (!selectMode.value) return false
  cancelSelect()
  return true
})
</script>

<template>
  <wd-config-provider :theme-vars="themeVars">
    <view class="screen screen--detail">
      <AppNav title="" :right="person ? '编辑' : ''" @right="openEdit" />

      <view class="page">
        <Skeleton v-if="personLoading && !person" type="timeline" />

        <view v-else-if="personError && !person" class="load-error">
          <text class="load-error__text">{{ personError }}</text>
          <view class="btn btn--secondary btn--small" @click="loadAll">重新加载</view>
        </view>

        <template v-else-if="person">
          <view class="profile">
            <PersonAvatar :id="person.id" :name="person.name" :size="56" />
            <view class="profile__main">
              <view class="profile__line">
                <text class="profile__name">{{ person.name }}</text>
                <text class="profile__relation">{{ person.relationLabel }}</text>
              </view>
              <text v-if="person.phone" class="profile__phone" @click="callPhone">
                {{ phoneLabel }} <text class="profile__call">☎</text>
              </text>
              <text v-if="person.remark" class="profile__remark">{{ person.remark }}</text>
            </view>
          </view>

          <SummaryStrip
            :in-cent="person.summary.inCent"
            :out-cent="person.summary.outCent"
            :net-cent="person.summary.netCent"
            :pending-count="person.summary.pendingCount"
            :pending-cent="person.summary.pendingCent"
            @pending="onSummaryPending"
          />

          <view class="tools" :class="{ 'tools--locked': selectMode }">
            <FilterBar :quick="selectMode ? 'pending' : quick" :count="activeFilterCount" @update:quick="onQuickChange" @open="openFilters" />
          </view>

          <view class="timeline">
            <Skeleton v-if="skeleton && !groups.length" type="timeline" />

            <view v-else-if="timelineError && !groups.length" class="load-error">
              <text class="load-error__text">{{ timelineError }}</text>
              <view class="btn btn--secondary btn--small" @click="loadTimeline">重新加载</view>
            </view>

            <template v-else>
              <EventGroup
                v-for="group in groups"
                :key="`${group.event.id}-${group.records[0]?.id ?? 0}`"
                :group="group"
                :selectable="selectMode"
                :selected-ids="selectedIds"
                @toggle="toggleRecord"
                @select-all="selectAll(group)"
                @record="openRecord"
              />

              <EmptyState
                v-if="!groups.length && !timelineLoading"
                :title="emptyTitle"
                :description="emptyDescription"
                :action-text="emptyActionText"
                @action="onEmptyAction"
              />

              <view v-if="timelineLoading && groups.length" class="loading-tip">正在加载…</view>
            </template>
          </view>
        </template>
      </view>

      <template v-if="person">
        <view v-if="selectMode" class="bottom-bar select-bar">
          <text class="select-bar__cancel" @click="cancelSelect">取消</text>
          <view class="select-bar__summary">
            <text class="select-bar__label">已选 {{ selectedIds.length }} 笔</text>
            <text class="select-bar__dot">·</text>
            <Money :cent="selectedCent" size="s" tone="pending" />
          </view>
          <view class="btn btn--primary select-bar__next" :class="{ 'btn--disabled': !selectedIds.length }" @click="nextStep">
            下一步
          </view>
        </view>
        <view v-else class="bottom-bar">
          <view class="btn btn--secondary bottom-bar__btn" @click="enterSelect">还礼</view>
          <view class="btn btn--primary bottom-bar__btn--main" @click="addRecord">
            <text class="bar-text">给{{ person.name }}记一笔</text>
          </view>
        </view>
      </template>

      <FilterSheet
        v-model:show="filterSheet"
        :value="filters"
        @apply="applyFilters"
        @reset="resetFilters"
      />
    </view>
  </wd-config-provider>
</template>

<style lang="scss" scoped>
.screen--detail {
  padding-bottom: calc(92px + env(safe-area-inset-bottom));
}

.profile {
  display: flex;
  align-items: flex-start;
  gap: 14px;
  padding: 4px 2px 16px;
}

.profile__main {
  flex: 1;
  min-width: 0;
  padding-top: 2px;
}

.profile__line {
  display: flex;
  align-items: center;
  gap: 8px;
}

.profile__name {
  font-size: 22px;
  line-height: 30px;
  font-weight: 600;
  color: var(--c-ink-1);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.profile__relation {
  flex: none;
  height: 20px;
  padding: 0 6px;
  border-radius: var(--r-xs);
  background: var(--c-fill);
  color: var(--c-ink-2);
  font-size: 12px;
  line-height: 20px;
}

.profile__phone {
  display: inline-block;
  margin-top: 4px;
  font-size: 13px;
  color: var(--c-ink-2);
}

.profile__call {
  color: var(--c-primary);
  font-size: 13px;
}

.profile__remark {
  display: block;
  margin-top: 5px;
  font-size: 13px;
  line-height: 19px;
  color: var(--c-ink-3);
}

.tools {
  margin-top: 10px;
}

.tools--locked {
  pointer-events: none;
}

.timeline {
  padding-top: 2px;
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

.bar-text {
  max-width: 100%;
  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
}

.select-bar {
  align-items: center;
}

.select-bar__cancel {
  flex: none;
  width: 44px;
  font-size: 15px;
  color: var(--c-ink-2);
  text-align: left;
}

.select-bar__summary {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: row;
  align-items: baseline;
  justify-content: center;
  gap: 4px;
}

.select-bar__label {
  font-size: 13px;
  color: var(--c-ink-2);
}

.select-bar__dot {
  font-size: 13px;
  color: var(--c-ink-3);
}

.select-bar__next {
  flex: none;
  width: 104px;
  height: 44px;
}
</style>
