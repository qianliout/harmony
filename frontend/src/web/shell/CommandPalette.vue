<script setup lang="ts">
import { computed, nextTick, onMounted, ref, watch } from 'vue'

import { searchAll } from '@/api/search'
import type { SearchResult } from '@/api/types'
import EventSeal from '@/components/EventSeal.vue'
import PersonAvatar from '@/components/PersonAvatar.vue'
import { useWebUiStore } from '@/stores/web'
import { formatCent } from '@/utils/money'
import { preciseDateLabel } from '@/utils/date'
import WebIcon from '@/web/components/WebIcon.vue'
import { goWeb } from '@/web/nav'

interface PaletteItem {
  key: string
  group: string
  label: string
  sub: string
  icon?: string
  personId?: number
  eventId?: number
  bookId?: number
  recordId?: number
  amount?: number
  direction?: string
  sealType?: string
}

const ui = useWebUiStore()
const query = ref('')
const loading = ref(false)
const result = ref<SearchResult | null>(null)
const active = ref(0)
let timer: number | undefined

const tips = ['zjg', '500', '500-1000', '伍佰', '2011', '11年', 'p1-10', '火炮']

const items = computed<PaletteItem[]>(() => {
  const data = result.value
  if (!data) return []
  const out: PaletteItem[] = []
  data.persons.items.forEach((item) => {
    out.push({
      key: `person-${item.id}`,
      group: '亲友',
      label: item.name,
      sub: `${item.relationLabel} · ${item.recordCount} 笔`,
      personId: item.id,
    })
  })
  data.events.items.forEach((item) => {
    out.push({
      key: `event-${item.id}`,
      group: '事项',
      label: item.name,
      sub: `${item.typeLabel} · ${item.eventDate || '日期未定'}`,
      eventId: item.id,
      sealType: item.type,
    })
  })
  data.books.items.forEach((item) => {
    out.push({
      key: `book-${item.id}`,
      group: '礼簿',
      label: item.name,
      sub: `${item.categoryLabel} · ${item.recordCount} 笔`,
      bookId: item.id,
      icon: 'book',
    })
  })
  data.records.items.forEach((item) => {
    out.push({
      key: `record-${item.id}`,
      group: '记录',
      label: `${item.personName} · ${item.eventName}`,
      sub: `${item.bookName} · ${preciseDateLabel(item.occurredOn, item.datePrecision)}`,
      recordId: item.id,
      amount: item.totalCent,
      direction: item.direction,
    })
  })
  return out
})

const groups = computed(() => {
  const map = new Map<string, PaletteItem[]>()
  items.value.forEach((item) => {
    const list = map.get(item.group) ?? []
    list.push(item)
    map.set(item.group, list)
  })
  return Array.from(map.entries())
})

async function runSearch(text: string): Promise<void> {
  const keyword = text.trim()
  if (!keyword) {
    result.value = null
    loading.value = false
    return
  }
  loading.value = true
  try {
    result.value = await searchAll(keyword, 5)
    active.value = 0
  } catch {
    result.value = null
  } finally {
    loading.value = false
  }
}

function select(item: PaletteItem): void {
  if (item.recordId) {
    ui.openRecord(item.recordId)
    return
  }
  ui.close()
  if (item.personId) {
    goWeb(`/sub-web/person?id=${item.personId}`, 'push')
    return
  }
  if (item.eventId) {
    goWeb(`/sub-web/event?id=${item.eventId}`, 'push')
    return
  }
  if (item.bookId) {
    goWeb(`/sub-web/book?id=${item.bookId}`, 'push')
  }
}

function chooseTip(tip: string): void {
  query.value = tip
  void runSearch(tip)
}

function move(delta: number): void {
  if (!items.value.length) return
  active.value = (active.value + delta + items.value.length) % items.value.length
}

function onKeydown(event: KeyboardEvent): void {
  if (event.key === 'ArrowDown') {
    event.preventDefault()
    move(1)
    return
  }
  if (event.key === 'ArrowUp') {
    event.preventDefault()
    move(-1)
    return
  }
  if (event.key === 'Enter') {
    event.preventDefault()
    const current = items.value[active.value]
    if (current) select(current)
  }
}

watch(query, (value) => {
  clearTimeout(timer)
  timer = setTimeout(() => {
    void runSearch(value)
  }, 220)
})

onMounted(() => {
  nextTick(() => {
    const input = document.querySelector<HTMLInputElement>('.web-palette input')
    input?.focus()
  })
})
</script>

<template>
  <div class="web-palette" role="dialog" aria-modal="true" aria-label="全局搜索">
    <div class="web-palette__q">
      <WebIcon name="search" />
      <input
        v-model="query"
        placeholder="姓名或拼音首字母、事项、礼簿、金额 500、区间 500-1000、伍佰、2011、11年、p1-10"
        autocomplete="off"
        @keydown="onKeydown"
      />
      <kbd class="web-kbd">Esc</kbd>
    </div>
    <div v-if="loading" class="web-empty">正在搜索…</div>
    <div v-else-if="!query.trim()" class="web-palette__tips">
      试试：
      <button v-for="tip in tips" :key="tip" type="button" @click="chooseTip(tip)">{{ tip }}</button>
    </div>
    <div v-else-if="!items.length" class="web-empty">没有找到。金额、年份、页码也能搜</div>
    <div v-else class="web-palette__res">
      <template v-for="[group, list] in groups" :key="group">
        <h6>{{ group }}</h6>
        <button
          v-for="item in list"
          :key="item.key"
          type="button"
          class="web-palette__item"
          :class="{ 'web-palette__item--on': items[active]?.key === item.key }"
          @click="select(item)"
          @mouseenter="active = items.findIndex((entry) => entry.key === item.key)"
        >
          <template v-if="item.sealType">
            <EventSeal :type="item.sealType" size="s" />
          </template>
          <template v-else-if="item.personId">
            <PersonAvatar :id="item.personId" :name="item.label" :size="28" />
          </template>
          <template v-else-if="item.recordId">
            <span class="web-dir" :class="item.direction === 'out' ? 'web-dir--out' : 'web-dir--in'">
              {{ item.direction === 'out' ? '送' : '收' }}
            </span>
          </template>
          <WebIcon v-else :name="item.icon || 'book'" />
          <span class="web-palette__grow">
            {{ item.label }}
            <small>{{ item.sub }}</small>
          </span>
          <span v-if="item.amount" class="web-amt web-amt--in">{{ formatCent(item.amount) }}</span>
        </button>
      </template>
    </div>
  </div>
</template>
