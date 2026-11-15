<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'

import { markRecordRepaid } from '@/api/record'
import type { PendingList, PendingPerson, RecordItem } from '@/api/types'
import { fetchPending } from '@/api/view'
import EmptyState from '@/components/EmptyState.vue'
import Money from '@/components/Money.vue'
import PageHead from '@/web/components/PageHead.vue'
import PersonCell from '@/web/components/PersonCell.vue'
import Segmented from '@/web/components/Segmented.vue'
import StatCard from '@/web/components/StatCard.vue'
import { useAsync } from '@/web/composables/useAsync'
import { useWebUiStore } from '@/stores/web'
import { preciseDateLabel } from '@/utils/date'
import { confirmModal } from '@/utils/nav'
import { formatCent } from '@/utils/money'

const ui = useWebUiStore()

const emptyList = (): PendingList => ({ persons: [], personCount: 0, totalCount: 0, totalCent: 0 })

const kind = ref('owed')
const expandedId = ref(0)
const busy = ref(false)
const initialLoading = ref(true)

const res = useAsync(() => fetchPending(kind.value === 'owing' ? 'owing' : 'owed'), emptyList())

const tabs = [
  { value: 'owed', label: '我该回的' },
  { value: 'owing', label: '该回我的' },
]

const rows = computed(() => res.data.value.persons)

const earliest = computed(() => {
  let value = ''
  for (const row of rows.value) {
    if (row.earliest && (!value || row.earliest < value)) value = row.earliest
  }
  return value
})

const sumTone = computed<'pending' | 'out'>(() => (kind.value === 'owed' ? 'pending' : 'out'))

function dateLabel(value: string): string {
  return preciseDateLabel(value, 'day') || '未注明'
}

function toggle(row: PendingPerson): void {
  expandedId.value = expandedId.value === row.person.id ? 0 : row.person.id
}

/** 我该回的：把这个人所有未回记录交给回礼抽屉。 */
function repayPerson(row: PendingPerson): void {
  const ids = row.records.map((record) => record.id)
  if (!ids.length) {
    ui.toast('没有可回礼的记录')
    return
  }
  ui.openRepay(ids)
}

/** 该回我的：逐笔标记为已回礼。 */
async function markOne(record: RecordItem): Promise<void> {
  if (busy.value) return
  busy.value = true
  try {
    await markRecordRepaid(record.id, record.paperNote)
    ui.toast('已标记为已回礼')
    ui.markChanged()
  } catch (err) {
    ui.toast(err instanceof Error ? err.message : '标记失败')
  } finally {
    busy.value = false
  }
}

/** 该回我的：整人标记为已回礼，中途失败也保留已完成的进度。 */
async function markPerson(row: PendingPerson): Promise<void> {
  if (busy.value) return
  const ok = await confirmModal(
    `把 ${row.person.name} 的 ${row.count} 笔未回礼都标记为已回礼？`,
    '整人标记',
  )
  if (!ok) return
  busy.value = true
  let done = 0
  try {
    for (const record of row.records) {
      await markRecordRepaid(record.id, record.paperNote)
      done += 1
    }
    ui.toast(`已标记 ${done} 笔为已回礼`)
  } catch (err) {
    const message = err instanceof Error ? err.message : '操作失败'
    ui.toast(done ? `已标记 ${done} 笔，后续失败：${message}` : message)
  } finally {
    busy.value = false
    if (done) ui.markChanged()
  }
}

watch(kind, () => {
  expandedId.value = 0
  void res.reload()
})

// 回礼抽屉或别处写库后统一刷新。
watch(
  () => ui.revision,
  () => {
    void res.reload()
  },
)

onMounted(async () => {
  try {
    await res.reload()
  } finally {
    initialLoading.value = false
  }
})
</script>

<template>
  <div>
    <PageHead title="回礼清单" :meta="`${res.data.value.personCount} 人 · ${res.data.value.totalCount} 笔`">
      <template #tools>
        <button class="web-btn web-btn--text" type="button" @click="res.reload()">刷新</button>
      </template>
    </PageHead>

    <div class="repay-tabs">
      <Segmented v-model="kind" :options="tabs" />
    </div>

    <div class="web-stats repay-stats">
      <StatCard
        :value="formatCent(res.data.value.totalCent, { compact: true })"
        :label="kind === 'owed' ? '待回合计' : '待收合计'"
        :tone="sumTone"
      />
      <StatCard :value="String(res.data.value.personCount)" label="涉及亲友" />
      <StatCard :value="String(res.data.value.totalCount)" label="待回笔数" />
      <StatCard :value="earliest ? dateLabel(earliest) : '—'" label="最早一笔" />
    </div>

    <div v-if="initialLoading" class="web-empty">
      <span class="web-muted">正在加载回礼清单…</span>
    </div>

    <div v-else-if="res.error.value && !rows.length" class="web-alert web-alert--err repay-error">
      <span>{{ res.error.value }}</span>
      <button class="web-btn web-btn--text web-btn--sm" type="button" @click="res.reload()">重试</button>
    </div>

    <div v-else-if="res.loading.value && !rows.length" class="web-empty">
      <span class="web-muted">正在加载回礼清单…</span>
    </div>

    <EmptyState
      v-else-if="!rows.length"
      :title="kind === 'owed' ? '人情都回完了' : '暂时没有该回我的'"
      :description="
        kind === 'owed'
          ? '有新的收礼时会出现在这里'
          : '送出去的礼对方还没回时会出现在这里'
      "
    />

    <div v-else class="web-card repay-card">
      <table class="web-table">
        <thead>
          <tr>
            <th>亲友</th>
            <th class="web-table__r">笔数</th>
            <th class="web-table__r">合计</th>
            <th>最早一笔</th>
            <th class="web-table__r">操作</th>
          </tr>
        </thead>
        <tbody>
          <template v-for="row in rows" :key="row.person.id">
            <tr class="web-table__row" @click="toggle(row)">
              <td>
                <PersonCell
                  :id="row.person.id"
                  :name="row.person.name"
                  :sub="row.person.relationLabel"
                />
              </td>
              <td class="web-table__r web-num">{{ row.count }}</td>
              <td class="web-table__r">
                <Money
                  :cent="row.totalCent"
                  :direction="kind === 'owed' ? 'none' : 'out'"
                  :tone="kind === 'owed' ? 'pending' : undefined"
                  size="s"
                />
              </td>
              <td class="web-sm">{{ dateLabel(row.earliest) }}</td>
              <td class="web-table__r" @click.stop>
                <button
                  v-if="kind === 'owed'"
                  class="web-btn web-btn--text web-btn--sm"
                  type="button"
                  @click="repayPerson(row)"
                >
                  去回礼
                </button>
                <button
                  v-else
                  class="web-btn web-btn--text web-btn--sm"
                  type="button"
                  :disabled="busy"
                  @click="markPerson(row)"
                >
                  整人已回
                </button>
              </td>
            </tr>
            <tr v-if="expandedId === row.person.id" class="repay-detail-row">
              <td colspan="5">
                <div class="repay-detail">
                  <table class="web-table web-table--compact">
                    <thead>
                      <tr>
                        <th>事项</th>
                        <th>礼簿</th>
                        <th>名目</th>
                        <th class="web-table__r">金额</th>
                        <th class="web-table__r">操作</th>
                      </tr>
                    </thead>
                    <tbody>
                      <tr v-for="record in row.records" :key="record.id">
                        <td>{{ record.eventName }}</td>
                        <td class="web-sm">{{ record.bookName }}</td>
                        <td class="web-sm web-muted">{{ record.itemKindLabel }}</td>
                        <td class="web-table__r">
                          <Money :cent="record.totalCent" :direction="record.direction" size="s" />
                        </td>
                        <td class="web-table__r">
                          <button
                            v-if="kind === 'owed'"
                            class="web-btn web-btn--text web-btn--sm"
                            type="button"
                            @click="ui.openRepay([record.id])"
                          >
                            回这一笔
                          </button>
                          <button
                            v-else
                            class="web-btn web-btn--text web-btn--sm"
                            type="button"
                            :disabled="busy"
                            @click="markOne(record)"
                          >
                            标记已回
                          </button>
                        </td>
                      </tr>
                    </tbody>
                  </table>
                </div>
              </td>
            </tr>
          </template>
        </tbody>
      </table>
    </div>
  </div>
</template>

<style lang="scss" scoped>
.repay-tabs {
  margin: var(--sp-4) 0;
}

.repay-stats {
  margin-top: var(--sp-4);
}

.repay-error {
  margin-top: var(--sp-4);
  align-items: center;
  justify-content: space-between;
}

.repay-card {
  margin-top: var(--sp-5);
  overflow: hidden;
}

.repay-detail-row td {
  padding: 0;
  background: var(--c-fill);
}

.repay-detail {
  padding: var(--sp-3) var(--sp-4);
}

.repay-detail .web-table th {
  font-size: 12px;
}
</style>
