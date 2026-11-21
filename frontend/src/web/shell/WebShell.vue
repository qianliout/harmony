<script setup lang="ts">
import { onMounted, ref } from 'vue'

import { useWebUiStore } from '@/stores/web'
import EntryDrawer from '@/web/overlays/EntryDrawer.vue'
import RecordDrawer from '@/web/overlays/RecordDrawer.vue'
import SideNav from '@/web/shell/SideNav.vue'
import ToastHost from '@/web/shell/ToastHost.vue'
import TopBar from '@/web/shell/TopBar.vue'
import { useWebHotkeys } from '@/web/composables/useWebHotkeys'

// 页面 query（如 id）会作为属性透传进来，外壳是多根节点，不继承。
defineOptions({ inheritAttrs: false })

withDefaults(defineProps<{ nav: string; title: string; crumbs?: string }>(), { crumbs: '' })

const ui = useWebUiStore()
const root = ref<HTMLElement | null>(null)

useWebHotkeys(
  {
    entry: () => ui.openEntry(),
    escape: () => ui.close(),
  },
  () => Boolean(root.value?.offsetParent),
)

onMounted(() => {
  ui.close()
})
</script>

<template>
  <div ref="root" class="web-shell">
    <SideNav :nav="nav" />
    <div class="web-main">
      <TopBar :title="title" :crumbs="crumbs" />
      <main class="web-content">
        <div class="web-page">
          <slot />
        </div>
      </main>
    </div>
  </div>

  <div v-if="ui.overlay" class="web-mask" @click="ui.close()" />
  <EntryDrawer v-if="ui.overlay?.type === 'entry'" :context="ui.overlay.context" />
  <RecordDrawer v-else-if="ui.overlay?.type === 'record'" :id="ui.overlay.id" />
  <ToastHost />
</template>
