<script setup lang="ts">
import WebIcon from '@/web/components/WebIcon.vue'
import { goWeb } from '@/web/nav'

defineProps<{ nav: string }>()

interface NavItem {
  key: string
  label: string
  icon: string
  url: string
}

const items: NavItem[] = [
  { key: 'ledger', label: '总账', icon: 'list', url: '/pages/web/index' },
  { key: 'books', label: '礼簿', icon: 'book', url: '/sub-web/books' },
  { key: 'people', label: '亲友', icon: 'users', url: '/sub-web/people' },
  { key: 'recognize', label: '识别礼簿', icon: 'scan', url: '/sub-web/recognize' },
]
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
      <button
        v-for="item in items"
        :key="item.key"
        type="button"
        class="web-nav__item"
        :class="{ 'web-nav__item--on': nav === item.key }"
        @click="goWeb(item.url)"
      >
        <WebIcon :name="item.icon" />
        {{ item.label }}
      </button>
    </nav>
    <div class="web-side__me">
      <span class="web-avatar web-side__avatar">本</span>
      <div>
        <div class="web-side__me-name">本机用户</div>
        <small class="web-side__me-sub">本机 Web 端</small>
      </div>
    </div>
  </aside>
</template>

<style lang="scss" scoped>
.web-side__avatar {
  background: var(--c-primary-light);
  color: var(--c-primary);
}
</style>
