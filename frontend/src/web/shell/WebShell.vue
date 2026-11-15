<script setup lang="ts">
import { onMounted, ref } from 'vue'

import { useWebUiStore } from '@/stores/web'
import CommandPalette from '@/web/shell/CommandPalette.vue'
import NotesDrawer from '@/web/shell/NotesDrawer.vue'
import SideNav from '@/web/shell/SideNav.vue'
import ToastHost from '@/web/shell/ToastHost.vue'
import TopBar from '@/web/shell/TopBar.vue'
import EntryDrawer from '@/web/overlays/EntryDrawer.vue'
import MergeDialog from '@/web/overlays/MergeDialog.vue'
import PaperAnnoDrawer from '@/web/overlays/PaperAnnoDrawer.vue'
import RecordDrawer from '@/web/overlays/RecordDrawer.vue'
import RepayDrawer from '@/web/overlays/RepayDrawer.vue'
import { useWebHotkeys } from '@/web/composables/useWebHotkeys'

withDefaults(defineProps<{ nav: string; title: string; crumbs?: string }>(), { crumbs: '' })

const ui = useWebUiStore()
const root = ref<HTMLElement | null>(null)

useWebHotkeys(
  {
    search: () => ui.openSearch(),
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
  <CommandPalette v-if="ui.overlay?.type === 'search'" />
  <NotesDrawer v-else-if="ui.overlay?.type === 'notes'" />
  <EntryDrawer v-else-if="ui.overlay?.type === 'entry'" :context="ui.overlay.context" />
  <RecordDrawer v-else-if="ui.overlay?.type === 'record'" :id="ui.overlay.id" />
  <RepayDrawer v-else-if="ui.overlay?.type === 'repay'" :origin-ids="ui.overlay.originIds" />
  <PaperAnnoDrawer v-else-if="ui.overlay?.type === 'paperAnno'" :id="ui.overlay.id" />
  <MergeDialog v-else-if="ui.overlay?.type === 'merge'" :source-id="ui.overlay.sourceId" />
  <ToastHost />
</template>
