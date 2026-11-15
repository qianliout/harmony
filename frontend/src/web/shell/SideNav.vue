<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'

import { fetchPendingNotes } from '@/api/record'
import { fetchPending } from '@/api/view'
import { useAuthStore } from '@/stores/auth'
import { useWebUiStore } from '@/stores/web'
import WebIcon from '@/web/components/WebIcon.vue'
import { goWeb } from '@/web/nav'

defineProps<{ nav: string }>()

const auth = useAuthStore()
const ui = useWebUiStore()
const pendingCount = ref(0)
const noteCount = ref(0)

const userLabel = computed(() => auth.user?.nickname || '本机用户')
const userSub = computed(() => (auth.isAdmin ? '管理员' : '记账人'))

interface NavItem {
  key: string
  label: string
  icon: string
  url: string
  badge?: string
}

interface NavGroup {
  title: string
  items: NavItem[]
}

const groups: NavGroup[] = [
  {
    title: '',
    items: [
      { key: 'home', label: '工作台', icon: 'home', url: '/pages/web/index' },
      { key: 'records', label: '记录', icon: 'list', url: '/sub-web/records' },
      { key: 'people', label: '亲友', icon: 'users', url: '/sub-web/people' },
      { key: 'events', label: '事项', icon: 'cal', url: '/sub-web/events' },
      { key: 'books', label: '礼簿', icon: 'book', url: '/sub-web/books' },
      { key: 'repay', label: '回礼清单', icon: 'return', url: '/sub-web/repay', badge: 'pending' },
      { key: 'stats', label: '统计', icon: 'chart', url: '/sub-web/stats' },
    ],
  },
  {
    title: '纸本录入',
    items: [
      { key: 'recognize', label: '识别礼簿', icon: 'scan', url: '/sub-web/recognize' },
      { key: 'notes', label: '整理纸本标注', icon: 'note', url: '/sub-web/notes', badge: 'notes' },
      { key: 'pages', label: '按页核对', icon: 'check', url: '/sub-web/pages' },
    ],
  },
  {
    title: '',
    items: [{ key: 'settings', label: '设置', icon: 'gear', url: '/sub-web/settings' }],
  },
]

function visibleItems(items: NavItem[]): NavItem[] {
  return items.filter((item) => item.key !== 'recognize' || auth.isAdmin)
}

function badgeValue(kind?: string): number {
  if (kind === 'pending') return pendingCount.value
  if (kind === 'notes') return noteCount.value
  return 0
}

async function loadBadges(): Promise<void> {
  try {
    const pending = await fetchPending('owed')
    pendingCount.value = pending.totalCount
  } catch {
    pendingCount.value = 0
  }
  try {
    noteCount.value = (await fetchPendingNotes()).length
  } catch {
    noteCount.value = 0
  }
}

onMounted(() => {
  void loadBadges()
})

watch(
  () => ui.revision,
  () => {
    void loadBadges()
  },
)
</script>

<template>
  <aside class="web-side">
    <div class="web-brand">
      <span class="web-brand__stamp">礼</span>
      <div>
        <b class="web-brand__name">人情账本</b>
        <small class="web-brand__sub">朱砂礼簿 · 网页版</small>
      </div>
    </div>
    <nav class="web-nav">
      <template v-for="group in groups" :key="group.title || group.items[0]?.key">
        <div v-if="group.title" class="web-nav__group">{{ group.title }}</div>
        <button
          v-for="item in visibleItems(group.items)"
          :key="item.key"
          type="button"
          class="web-nav__item"
          :class="{ 'web-nav__item--on': nav === item.key }"
          @click="goWeb(item.url)"
        >
          <WebIcon :name="item.icon" />
          {{ item.label }}
          <span v-if="item.badge && badgeValue(item.badge)" class="web-nav__badge">
            {{ badgeValue(item.badge) }}
          </span>
        </button>
      </template>
    </nav>
    <div class="web-side__me">
      <span class="web-avatar" style="background: var(--c-primary-light); color: var(--c-primary)">本</span>
      <div>
        <div class="web-side__me-name">{{ userLabel }}</div>
        <small class="web-side__me-sub">{{ userSub }} · 本机开发</small>
      </div>
    </div>
  </aside>
</template>
