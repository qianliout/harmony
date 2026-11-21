<script setup lang="ts">
// 亲友列表：按姓名、备注、拼音查找；显示每个人的收送合计。
import { computed, onBeforeUnmount, ref, watch } from 'vue'

import { deletePerson, fetchPersons } from '@/api/person'
import type { Page, PersonItem } from '@/api/types'
import { useWebUiStore } from '@/stores/web'
import { relativeSince } from '@/utils/date'
import { formatCent } from '@/utils/money'
import PageHead from '@/web/components/PageHead.vue'
import Pager from '@/web/components/Pager.vue'
import PersonCell from '@/web/components/PersonCell.vue'
import WebIcon from '@/web/components/WebIcon.vue'
import WebTable from '@/web/components/WebTable.vue'
import { useAsync } from '@/web/composables/useAsync'
import { goWeb } from '@/web/nav'
import PersonDialog from '@/web/overlays/PersonDialog.vue'

const PAGE_SIZE = 20
const ui = useWebUiStore()
const keywordInput = ref('')
const keyword = ref('')
const pageNo = ref(1)
const dialogOpen = ref(false)
const editing = ref<PersonItem | null>(null)
let keywordTimer: ReturnType<typeof setTimeout> | undefined

const list = useAsync<Page<PersonItem>>(
  () => fetchPersons({ keyword: keyword.value || undefined, page: pageNo.value, pageSize: PAGE_SIZE }),
  { items: [], total: 0 },
)
const persons = computed(() => list.data.value.items)
const pageCount = computed(() => Math.max(1, Math.ceil(list.data.value.total / PAGE_SIZE)))

function openPerson(id: number): void {
  goWeb(`/sub-web/person?id=${id}`, 'push')
}

function openCreate(): void {
  editing.value = null
  dialogOpen.value = true
}

function openEdit(person: PersonItem): void {
  editing.value = person
  dialogOpen.value = true
}

function onSaved(): void {
  ui.toast(editing.value ? '亲友已保存' : '亲友已新建')
  dialogOpen.value = false
  ui.markChanged()
}

async function remove(person: PersonItem): Promise<void> {
  if (!window.confirm(`删除亲友「${person.displayName}」？`)) return
  try {
    await deletePerson(person.id)
    ui.toast('亲友已删除')
    ui.markChanged()
  } catch (err) {
    ui.toast(err instanceof Error ? err.message : '删除失败')
  }
}

watch(keywordInput, (value) => {
  if (keywordTimer) clearTimeout(keywordTimer)
  keywordTimer = setTimeout(() => {
    keyword.value = value.trim()
    pageNo.value = 1
  }, 250)
})
watch([keyword, pageNo], () => void list.reload())
watch(
  () => ui.revision,
  () => void list.reload(),
)
onBeforeUnmount(() => {
  if (keywordTimer) clearTimeout(keywordTimer)
})

void list.reload()
</script>

<template>
  <PageHead title="亲友" :meta="`${list.data.value.total} 人`">
    <template #tools>
      <button class="web-btn web-btn--primary" type="button" @click="openCreate">
        <WebIcon name="plus" :size="16" />
        新建亲友
      </button>
    </template>
  </PageHead>

  <div class="web-card">
    <div class="web-filters">
      <input
        v-model="keywordInput"
        class="web-input people-keyword"
        type="search"
        placeholder="姓名、备注或拼音首字母，如 zs"
        aria-label="搜索亲友"
      />
    </div>

    <div v-if="list.error.value" class="web-alert web-alert--err people-alert">
      <WebIcon name="alert" :size="16" />
      <span>{{ list.error.value }}</span>
      <button class="web-btn web-btn--sm" type="button" @click="list.reload()">重试</button>
    </div>
    <WebTable v-else>
      <thead>
        <tr>
          <th>亲友</th>
          <th class="web-table__r">总收</th>
          <th class="web-table__r">总送</th>
          <th class="web-table__r">笔数</th>
          <th>最近往来</th>
          <th class="web-table__r">操作</th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="person in persons" :key="person.id" class="web-table__row" @click="openPerson(person.id)">
          <td @click.stop>
            <PersonCell :id="person.id" :name="person.name" :sub="person.remark" @open="openPerson(person.id)" />
          </td>
          <td class="web-table__r web-amt web-amt--in">{{ formatCent(person.inCent) }}</td>
          <td class="web-table__r web-amt web-amt--out">{{ formatCent(person.outCent) }}</td>
          <td class="web-table__r web-num">{{ person.recordCount }}</td>
          <td class="web-sm web-muted">{{ relativeSince(person.lastOn) || '—' }}</td>
          <td class="web-table__r" @click.stop>
            <button class="web-btn web-btn--text web-btn--sm" type="button" @click="openEdit(person)">编辑</button>
            <button class="web-btn web-btn--text web-btn--sm" type="button" @click="remove(person)">删除</button>
          </td>
        </tr>
      </tbody>
    </WebTable>
    <div v-if="!list.error.value && !persons.length" class="web-empty">
      {{ list.loading.value ? '加载中…' : '没有找到亲友，换个关键字试试' }}
    </div>

    <div class="web-sumbar">
      <span>{{ list.data.value.total }} 人</span>
      <Pager :page="pageNo" :pages="pageCount" @change="pageNo = $event" />
    </div>
  </div>

  <PersonDialog v-if="dialogOpen" :person="editing" @close="dialogOpen = false" @saved="onSaved" />
</template>

<style lang="scss" scoped>
.people-keyword {
  width: 260px;
}

.people-alert {
  margin: var(--sp-4);
}

.people-alert .web-btn {
  margin-left: auto;
}
</style>
