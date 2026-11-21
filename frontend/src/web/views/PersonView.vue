<script setup lang="ts">
// 亲友详情：这个人的全部往来，收送都显示。
import { onMounted, ref, watch } from 'vue'

import { deletePerson, fetchPersonDetail } from '@/api/person'
import type { PersonItem, RecordItem } from '@/api/types'
import PersonAvatar from '@/components/PersonAvatar.vue'
import { useWebUiStore } from '@/stores/web'
import { formatCent } from '@/utils/money'
import PageHead from '@/web/components/PageHead.vue'
import Pager from '@/web/components/Pager.vue'
import RecordTable from '@/web/components/RecordTable.vue'
import Segmented from '@/web/components/Segmented.vue'
import StatCard from '@/web/components/StatCard.vue'
import WebIcon from '@/web/components/WebIcon.vue'
import { useAsync } from '@/web/composables/useAsync'
import { useRecordList } from '@/web/composables/useRecordList'
import { backWeb, readPageId } from '@/web/nav'
import PersonDialog from '@/web/overlays/PersonDialog.vue'

const ui = useWebUiStore()
const personId = ref(0)
const dialogOpen = ref(false)

const detail = useAsync<PersonItem | null>(() => fetchPersonDetail(personId.value), null)
const list = useRecordList(() => ({ personId: personId.value }))

const directionOptions = [
  { value: '', label: '全部' },
  { value: 'in', label: '收' },
  { value: 'out', label: '送' },
]

function onSaved(): void {
  dialogOpen.value = false
  ui.toast('亲友已保存')
  ui.markChanged()
}

async function remove(): Promise<void> {
  const person = detail.data.value
  if (!person || !window.confirm(`删除亲友「${person.displayName}」？`)) return
  try {
    await deletePerson(person.id)
    ui.toast('亲友已删除')
    backWeb()
  } catch (err) {
    ui.toast(err instanceof Error ? err.message : '删除失败')
  }
}

function editRecord(record: RecordItem): void {
  ui.openEntry({ recordId: record.id })
}

watch(
  () => ui.revision,
  () => void detail.reload(),
)

onMounted(() => {
  personId.value = readPageId()
  // 记录列表随 personId 变化自动加载。
  void detail.reload()
})
</script>

<template>
  <div v-if="detail.error.value" class="web-alert web-alert--err">
    <WebIcon name="alert" />
    <span>{{ detail.error.value }}</span>
  </div>
  <template v-else-if="detail.data.value">
    <PageHead :title="detail.data.value.displayName" :meta="`往来 ${detail.data.value.recordCount} 笔`">
      <template #tools>
        <button class="web-btn" type="button" @click="dialogOpen = true">
          <WebIcon name="edit" />
          编辑
        </button>
        <button class="web-btn web-btn--danger" type="button" @click="remove">
          <WebIcon name="trash" />
          删除
        </button>
        <button class="web-btn web-btn--primary" type="button" @click="ui.openEntry({ personId })">
          <WebIcon name="plus" />
          记一笔
        </button>
      </template>
    </PageHead>

    <div class="web-stats">
      <div class="web-card web-stat person-avatar">
        <PersonAvatar :id="detail.data.value.id" :name="detail.data.value.name" :size="48" />
      </div>
      <StatCard :value="formatCent(detail.data.value.inCent)" label="总收" tone="in" />
      <StatCard :value="formatCent(detail.data.value.outCent)" label="总送" tone="out" />
      <StatCard :value="formatCent(detail.data.value.inCent - detail.data.value.outCent)" label="收送差额" />
    </div>

    <div class="web-card">
      <div class="web-filters">
        <span class="web-filters__lbl">方向</span>
        <Segmented v-model="list.filter.direction" :options="directionOptions" variant="dir" />
      </div>
      <RecordTable
        v-if="list.items.value.length"
        :records="list.items.value"
        :columns="['date', 'place', 'amount', 'note', 'ops']"
        @row="ui.openRecord($event.id)"
        @edit="editRecord"
        @remove="list.remove"
      />
      <div v-else class="web-empty">{{ list.loading.value ? '加载中…' : '还没有往来记录' }}</div>
      <div class="web-sumbar">
        <span>{{ list.total.value }} 笔</span>
        <span>收 <b class="web-amt web-amt--in">{{ formatCent(list.summary.value.inCent) }}</b></span>
        <span>送 <b class="web-amt web-amt--out">{{ formatCent(list.summary.value.outCent) }}</b></span>
        <Pager :page="list.page.value" :pages="list.pageCount.value" @change="list.page.value = $event" />
      </div>
    </div>
  </template>
  <div v-else class="web-empty">加载中…</div>

  <PersonDialog v-if="dialogOpen" :person="detail.data.value" @close="dialogOpen = false" @saved="onSaved" />
</template>

<style lang="scss" scoped>
.person-avatar {
  display: flex;
  align-items: center;
  justify-content: center;
}
</style>
