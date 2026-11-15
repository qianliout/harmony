<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'

import {
  fetchRecordDetail,
  markRecordRepaid,
  parsePaperNote,
  repayRecords,
} from '@/api/record'
import type { PaperDraft, RecordDetail } from '@/api/types'
import { useWebUiStore } from '@/stores/web'
import { centToYuanText, yuanToCent } from '@/utils/money'
import WebIcon from '@/web/components/WebIcon.vue'

interface EditableDraft extends PaperDraft {
  items: (PaperDraft['items'][number] & { amountText: string })[]
  eventName: string
}

const props = defineProps<{ id: number }>()
const ui = useWebUiStore()
const detail = ref<RecordDetail | null>(null)
const drafts = ref<EditableDraft[]>([])
const onlyMark = ref(false)
const loading = ref(true)
const saving = ref(false)
const error = ref('')

const title = computed(() => `补录纸本回礼 · ${detail.value?.personName ?? ''}`)

async function load(): Promise<void> {
  loading.value = true
  error.value = ''
  try {
    const record = await fetchRecordDetail(props.id)
    detail.value = record
    const result = await parsePaperNote(record.paperNote, record.personId)
    onlyMark.value = result.onlyMark
    drafts.value = result.repays.map((draft) => ({
      ...draft,
      items: draft.items.map((item) => ({
        ...item,
        amountText: centToYuanText(item.amountCent),
      })),
      eventName: draft.event?.name ?? '',
    }))
  } catch (err) {
    error.value = err instanceof Error ? err.message : '解析失败'
  } finally {
    loading.value = false
  }
}

async function save(): Promise<void> {
  error.value = ''
  if (!detail.value) return
  if (onlyMark.value) {
    saving.value = true
    try {
      await markRecordRepaid(detail.value.id, detail.value.paperNote)
      ui.markChanged()
      ui.toast('已标为已回礼')
      ui.close()
    } catch (err) {
      error.value = err instanceof Error ? err.message : '保存失败'
    } finally {
      saving.value = false
    }
    return
  }
  if (!drafts.value.length) {
    error.value = '没有解析出可转换的回礼，请检查标注原文'
    return
  }
  saving.value = true
  try {
    for (const draft of drafts.value) {
      await repayRecords({
        originIds: [detail.value.id],
        markRepaid: true,
        event: {
          id: draft.event?.id ?? 0,
          name: draft.eventName.trim() || draft.event?.name || '纸本未注明事项',
          type: draft.event?.type ?? 'other',
          date: draft.occurredOn,
          datePrecision: draft.datePrecision,
        },
        occurredOn: draft.occurredOn,
        datePrecision: draft.datePrecision,
        paperNote: detail.value.paperNote,
        items: draft.items.map((item) => ({
          bookId: item.bookId,
          itemKind: item.itemKind,
          amountCent: yuanToCent(item.amountText),
          giftName: item.giftName,
          giftValueCent: item.giftValueCent,
        })),
      })
    }
    ui.markChanged()
    ui.toast(`已生成 ${drafts.value.length} 条回礼并关联原收礼`)
    ui.close()
  } catch (err) {
    error.value = err instanceof Error ? err.message : '保存失败'
  } finally {
    saving.value = false
  }
}

onMounted(() => {
  void load()
})
</script>

<template>
  <aside class="web-drawer" role="dialog" aria-modal="true">
    <div class="web-drawer__head">
      <h3>{{ title }}</h3>
      <button class="web-btn web-btn--text" type="button" aria-label="关闭" @click="ui.close()">
        <WebIcon name="x" />
      </button>
    </div>
    <div class="web-drawer__body">
      <div v-if="loading" class="web-empty">解析中…</div>
      <div v-else-if="error" class="web-alert web-alert--err">{{ error }}</div>
      <template v-else-if="detail">
        <div class="web-field">
          <label>原收礼</label>
          <div class="web-chain__item">
            <span style="flex: 1" class="web-sm">
              {{ detail.eventName }} · {{ detail.bookName }} · {{ detail.paperPage }}
              <br />
              <span class="web-muted">{{ detail.occurredOn }}</span>
            </span>
            <span class="web-amt web-amt--in">¥{{ (detail.totalCent / 100).toFixed(2) }}</span>
          </div>
        </div>
        <div class="web-field">
          <label>标注原文（原样保留）</label>
          <div style="padding: 12px; border-radius: 10px; background: var(--c-paper); font-family: var(--font-serif); font-size: 16px">
            {{ detail.paperNote }}
          </div>
        </div>
        <div v-if="onlyMark" class="web-alert web-alert--warn">
          标注只有已还记号、没有金额，保存后只把原记录标为已回礼。
        </div>
        <div v-else class="web-field">
          <label>解析出的回礼（{{ drafts.length }} 条，可改）</label>
          <div class="web-table-wrap">
            <table class="web-table web-table--compact">
              <thead>
                <tr>
                  <th>时间</th>
                  <th>精度</th>
                  <th>对方事项</th>
                  <th>名目</th>
                  <th class="web-table__r">金额</th>
                </tr>
              </thead>
              <tbody>
                <template v-for="draft in drafts" :key="draft.raw + draft.occurredOn + draft.eventName">
                  <tr v-for="(item, index) in draft.items" :key="`${draft.raw}-${index}`">
                    <td>
                      <input
                        v-if="index === 0"
                        v-model="draft.occurredOn"
                        class="web-input"
                        type="date"
                        style="width: 140px"
                      />
                    </td>
                    <td>
                      <select
                        v-if="index === 0"
                        v-model="draft.datePrecision"
                        class="web-select"
                        style="width: 72px"
                      >
                        <option value="day">日</option>
                        <option value="month">月</option>
                        <option value="year">年</option>
                      </select>
                    </td>
                    <td>
                      <input
                        v-if="index === 0"
                        v-model="draft.eventName"
                        class="web-input"
                        style="width: 130px"
                      />
                    </td>
                    <td class="web-sm">{{ item.itemKind === 'huopao' ? '火炮' : item.itemKind === 'gift' ? '实物礼品' : '现金' }}</td>
                    <td class="web-table__r">
                      <input
                        v-model="item.amountText"
                        class="web-input"
                        inputmode="decimal"
                        style="width: 90px; text-align: right"
                      />
                    </td>
                  </tr>
                </template>
              </tbody>
            </table>
          </div>
          <span class="web-field__hint">时间只到月或年时按当月/当年第一天存，界面显示“约”。</span>
        </div>
      </template>
    </div>
    <div class="web-drawer__foot">
      <button class="web-btn" type="button" @click="ui.close()">取消</button>
      <button class="web-btn web-btn--primary" type="button" :disabled="saving" @click="save">
        {{ onlyMark ? '标为已回礼' : `生成 ${drafts.length} 条回礼` }}
      </button>
    </div>
  </aside>
</template>
