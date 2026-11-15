<script setup lang="ts">
import { onMounted, ref, watch } from 'vue'

import {
  fetchPendingNotes,
  markRecordRepaid,
  parsePaperNote,
  repayRecords,
} from '@/api/record'
import type { PaperDraft, RecordItem } from '@/api/types'
import EmptyState from '@/components/EmptyState.vue'
import Money from '@/components/Money.vue'
import PageHead from '@/web/components/PageHead.vue'
import PersonCell from '@/web/components/PersonCell.vue'
import { useAsync } from '@/web/composables/useAsync'
import { useWebUiStore } from '@/stores/web'
import { relations } from '@/styles/palette'
import { preciseDateLabel } from '@/utils/date'
import { formatCent } from '@/utils/money'

interface ParseState {
  loading: boolean
  done: boolean
  error: string
  onlyMark: boolean
  drafts: PaperDraft[]
}

const ui = useWebUiStore()

const res = useAsync(() => fetchPendingNotes(0), [] as RecordItem[])
const parses = ref<Record<number, ParseState>>({})
const parseTasks = new Map<number, Promise<ParseState>>()
const converting = ref(false)
const progress = ref({ done: 0, total: 0 })
const initialLoading = ref(true)

function draftCent(draft: PaperDraft): number {
  return (draft.items ?? []).reduce((sum, item) => sum + item.amountCent, 0)
}

/** 解析一条标注，结果写入预览缓存；同一行复用进行中的请求，失败也进入缓存。 */
function ensureParse(record: RecordItem): Promise<ParseState> {
  const cached = parses.value[record.id]
  if (cached?.done) return Promise.resolve(cached)
  const pending = parseTasks.get(record.id)
  if (pending) return pending

  parses.value = {
    ...parses.value,
    [record.id]: { loading: true, done: false, error: '', onlyMark: false, drafts: [] },
  }
  const task = (async (): Promise<ParseState> => {
    try {
      const result = await parsePaperNote(record.paperNote, record.personId)
      const state: ParseState = {
        loading: false,
        done: true,
        error: '',
        onlyMark: result.onlyMark,
        drafts: result.repays ?? [],
      }
      parses.value = { ...parses.value, [record.id]: state }
      return state
    } catch (err) {
      const state: ParseState = {
        loading: false,
        done: false,
        error: err instanceof Error ? err.message : '解析失败',
        onlyMark: false,
        drafts: [],
      }
      parses.value = { ...parses.value, [record.id]: state }
      return state
    }
  })()
  parseTasks.set(record.id, task)
  void task.then(() => {
    if (parseTasks.get(record.id) === task) parseTasks.delete(record.id)
  })
  return task
}

function previewText(record: RecordItem): string {
  const state = parses.value[record.id]
  if (!state || state.loading) return '解析中…'
  if (state.error) return state.error
  if (state.onlyMark) return '只有已回礼记号'
  const first = state.drafts[0]
  if (!first) return '未能解析出回礼'
  const parts: string[] = []
  if (state.drafts.length > 1) parts.push(`${state.drafts.length} 次`)
  parts.push(preciseDateLabel(first.occurredOn, first.datePrecision || 'day') || '时间待补')
  parts.push(first.event?.name || '事项待补')
  const cent = state.drafts.reduce((sum, draft) => sum + draftCent(draft), 0)
  parts.push(cent > 0 ? formatCent(cent) : '金额待补')
  return parts.join(' · ')
}

function previewTone(record: RecordItem): string {
  const state = parses.value[record.id]
  if (!state) return 'web-muted'
  if (state.error) return 'note-preview--err'
  if (state.onlyMark || (state.drafts.length && state.drafts.every((draft) => draftCent(draft) > 0))) {
    return 'note-preview--ok'
  }
  return 'note-preview--warn'
}

function dateLabel(record: RecordItem): string {
  return preciseDateLabel(record.occurredOn, record.datePrecision) || '未注明'
}

function relationLabel(code: string): string {
  return relations.find((item) => item.code === code)?.label ?? '其他'
}

async function loadAll(): Promise<void> {
  parses.value = {}
  await res.reload()
  for (const record of res.data.value) {
    void ensureParse(record)
  }
}

/** 逐条按预填结果写入；遇到解析或写入失败立即停止并提示。 */
async function convertAll(): Promise<void> {
  if (converting.value || !res.data.value.length) return
  const list = [...res.data.value]
  converting.value = true
  progress.value = { done: 0, total: list.length }
  let wrote = false
  try {
    for (const record of list) {
      const state = await ensureParse(record)
      if (state.error) {
        throw new Error(`${record.personName} 解析失败：${state.error}`)
      }
      if (state.onlyMark) {
        await markRecordRepaid(record.id, record.paperNote)
        wrote = true
      } else {
        if (!state.drafts.length) {
          throw new Error(`${record.personName} 没有解析出回礼`)
        }
        for (const draft of state.drafts) {
          await repayRecords({
            originIds: [record.id],
            markRepaid: true,
            event: draft.event,
            occurredOn: draft.occurredOn,
            datePrecision: draft.datePrecision || 'day',
            paperNote: record.paperNote,
            items: draft.items,
          })
          wrote = true
        }
      }
      progress.value = { ...progress.value, done: progress.value.done + 1 }
    }
    ui.toast(`已按预填转换 ${progress.value.done} 条`)
  } catch (err) {
    const message = err instanceof Error ? err.message : '转换失败'
    ui.toast(`已转换 ${progress.value.done} 条后停止：${message}`)
  } finally {
    converting.value = false
    if (wrote) ui.markChanged()
  }
}

watch(
  () => ui.revision,
  () => {
    void loadAll()
  },
)

onMounted(async () => {
  try {
    await loadAll()
  } finally {
    initialLoading.value = false
  }
})
</script>

<template>
  <div>
    <PageHead title="整理纸本标注" :meta="`${res.data.value.length} 条待整理`">
      <template #tools>
        <button
          class="web-btn web-btn--primary"
          type="button"
          :disabled="converting || !res.data.value.length"
          @click="convertAll"
        >
          {{ converting ? `正在转换 ${progress.done} / ${progress.total}` : '全部按预填转换' }}
        </button>
        <button class="web-btn web-btn--text" type="button" @click="loadAll">刷新</button>
      </template>
    </PageHead>

    <div v-if="converting" class="web-alert web-alert--info notes-progress">
      正在逐条写入，已转换 {{ progress.done }} / {{ progress.total }} 条。
    </div>

    <div v-if="initialLoading" class="web-empty">
      <span class="web-muted">正在读取待整理的标注…</span>
    </div>

    <div v-else-if="res.error.value && !res.data.value.length" class="web-alert web-alert--err notes-error">
      <span>{{ res.error.value }}</span>
      <button class="web-btn web-btn--text web-btn--sm" type="button" @click="loadAll">重试</button>
    </div>

    <div v-else-if="res.loading.value && !res.data.value.length" class="web-empty">
      <span class="web-muted">正在读取待整理的标注…</span>
    </div>

    <EmptyState
      v-else-if="!res.data.value.length"
      title="没有待整理的标注"
      description="连续录入时抄下的手写标注会出现在这里"
    />

    <div v-else class="web-card notes-card">
      <table class="web-table">
        <thead>
          <tr>
            <th>亲友</th>
            <th>原收礼</th>
            <th>标注原文</th>
            <th>预填回礼</th>
            <th class="web-table__r">操作</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="record in res.data.value" :key="record.id">
            <td>
              <PersonCell
                :id="record.personId"
                :name="record.personName"
                :sub="relationLabel(record.personRelation)"
              />
            </td>
            <td>
              <div class="note-origin">
                <Money :cent="record.totalCent" direction="in" size="s" />
                <small class="web-muted">{{ dateLabel(record) }} · {{ record.itemKindLabel }}</small>
              </div>
            </td>
            <td>
              <span class="note-raw">{{ record.paperNote || '—' }}</span>
            </td>
            <td>
              <span class="note-preview" :class="previewTone(record)">{{ previewText(record) }}</span>
            </td>
            <td class="web-table__r">
              <button
                class="web-btn web-btn--text web-btn--sm"
                type="button"
                @click="ui.openPaperAnno(record.id)"
              >
                核对并转换
              </button>
            </td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>

<style lang="scss" scoped>
.notes-progress {
  margin-top: var(--sp-4);
}

.notes-error {
  margin-top: var(--sp-4);
  align-items: center;
  justify-content: space-between;
}

.notes-card {
  margin-top: var(--sp-5);
  overflow: hidden;
}

.note-origin {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.note-origin small {
  font-size: 12px;
  line-height: 17px;
}

.note-raw {
  display: block;
  max-width: 320px;
  font-family: $font-serif;
  font-size: 13px;
  line-height: 20px;
  color: var(--c-ink-1);
}

.note-preview {
  font-size: 12px;
  line-height: 18px;
}

.note-preview--ok {
  color: var(--c-repaid);
}

.note-preview--warn {
  color: var(--c-pending);
}

.note-preview--err {
  color: var(--c-danger);
}
</style>
