<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'

import { ApiError } from '@/api/http'
import { deleteRecord, fetchRecordDetail, markRecordRepaid, updateRecord } from '@/api/record'
import type { RecordDetail, RecordItem } from '@/api/types'
import DirTag from '@/components/DirTag.vue'
import EventSeal from '@/components/EventSeal.vue'
import Money from '@/components/Money.vue'
import PersonAvatar from '@/components/PersonAvatar.vue'
import StatusPill from '@/components/StatusPill.vue'
import { useWebUiStore } from '@/stores/web'
import { relations } from '@/styles/palette'
import { centToChinese } from '@/utils/chinese-amount'
import { preciseDateLabel } from '@/utils/date'
import WebIcon from '@/web/components/WebIcon.vue'

const props = defineProps<{ id: number }>()
const ui = useWebUiStore()
const detail = ref<RecordDetail | null>(null)
const loading = ref(true)
const error = ref('')
const busy = ref(false)

function relationLabel(code: string): string {
  return relations.find((item) => item.code === code)?.label ?? '其他'
}

const isOwe = computed(
  () => detail.value?.direction === 'in' && detail.value.status === 'pending',
)
const canMarkOwed = computed(
  () => detail.value?.direction === 'out' && detail.value.status === 'pending' && !detail.value.isRepay,
)

function money(record: RecordItem): string {
  if (!record.amountCent && record.giftName) {
    return record.giftName
  }
  return `¥${(record.amountCent / 100).toFixed(record.amountCent % 100 ? 2 : 0)}`
}

async function load(): Promise<void> {
  loading.value = true
  error.value = ''
  try {
    detail.value = await fetchRecordDetail(props.id)
  } catch (err) {
    error.value = err instanceof Error ? err.message : '加载失败'
  } finally {
    loading.value = false
  }
}

async function markNone(): Promise<void> {
  if (!detail.value) return
  busy.value = true
  try {
    await updateRecord(detail.value.id, {
      personId: detail.value.personId,
      eventId: detail.value.eventId,
      bookId: detail.value.bookId,
      direction: detail.value.direction,
      itemKind: detail.value.itemKind,
      amountCent: detail.value.amountCent,
      giftName: detail.value.giftName,
      giftValueCent: detail.value.giftValueCent,
      occurredOn: detail.value.occurredOn,
      datePrecision: detail.value.datePrecision,
      status: 'none',
      remark: detail.value.remark,
      paperNote: detail.value.paperNote,
      paperPage: detail.value.paperPage,
    })
    ui.markChanged()
    ui.toast('已标为不用回')
    ui.close()
  } catch (err) {
    error.value = err instanceof Error ? err.message : '操作失败'
  } finally {
    busy.value = false
  }
}

async function markRepaid(): Promise<void> {
  if (!detail.value) return
  busy.value = true
  try {
    await markRecordRepaid(detail.value.id)
    ui.markChanged()
    ui.toast('已标为对方已回')
    ui.close()
  } catch (err) {
    error.value = err instanceof Error ? err.message : '操作失败'
  } finally {
    busy.value = false
  }
}

async function remove(): Promise<void> {
  if (!detail.value) return
  if (!window.confirm('删除这条记录？此操作不能撤销。')) return
  busy.value = true
  try {
    await deleteRecord(detail.value.id)
    ui.markChanged()
    ui.toast('已删除')
    ui.close()
  } catch (err) {
    if (err instanceof ApiError && err.code === 409) {
      const reason = (err.custom as Record<string, unknown> | null)?.reason
      if (reason === 'unlink_on_delete') {
        const ok = window.confirm(`${err.message}。确认删除并解除关联？`)
        if (ok) {
          await deleteRecord(detail.value.id, true)
          ui.markChanged()
          ui.toast('已删除')
          ui.close()
          return
        }
      }
    }
    error.value = err instanceof Error ? err.message : '删除失败'
  } finally {
    busy.value = false
  }
}

function openRecord(record: RecordItem): void {
  ui.openRecord(record.id)
}

onMounted(() => {
  void load()
})
</script>

<template>
  <aside class="web-drawer" role="dialog" aria-modal="true">
    <div class="web-drawer__head">
      <h3>记录详情</h3>
      <button class="web-btn web-btn--text" type="button" aria-label="关闭" @click="ui.close()">
        <WebIcon name="x" />
      </button>
    </div>
    <div class="web-drawer__body">
      <div v-if="loading" class="web-empty">加载中…</div>
      <div v-else-if="error" class="web-alert web-alert--err">{{ error }}</div>
      <template v-else-if="detail">
        <div class="web-cell2">
          <PersonAvatar :id="detail.personId" :name="detail.personName" :size="44" />
          <div style="flex: 1">
            <b style="font-size: 17px">{{ detail.personName }}</b>
            <div class="web-sm web-muted">{{ relationLabel(detail.personRelation) }}</div>
          </div>
          <div style="text-align: right">
            <div><DirTag :direction="detail.direction" /> <Money :cent="detail.totalCent" :direction="detail.direction" size="l" /></div>
            <StatusPill
              :status="detail.status"
              :direction="detail.direction"
              :is-repay="detail.isRepay"
              :label="detail.statusLabel"
            />
          </div>
        </div>

        <dl class="web-dl">
          <dt>事项</dt>
          <dd>
            <span class="web-cell2"><EventSeal :type="detail.eventType || 'other'" size="s" />{{ detail.eventName }}</span>
          </dd>
          <dt>礼簿</dt>
          <dd>{{ detail.bookName }}<span v-if="detail.originBookName" class="web-muted">（原：{{ detail.originBookName }}）</span></dd>
          <dt>名目</dt>
          <dd>{{ detail.itemKindLabel }}<span v-if="detail.giftName"> · {{ detail.giftName }}</span></dd>
          <dt>大写</dt>
          <dd style="font-family: var(--font-serif)">{{ detail.amountCent ? centToChinese(detail.amountCent) : '—' }}</dd>
          <dt>日期</dt>
          <dd>{{ preciseDateLabel(detail.occurredOn, detail.datePrecision) }}</dd>
          <dt v-if="detail.paperPage">纸本</dt>
          <dd v-if="detail.paperPage">{{ detail.paperPage }}</dd>
          <dt v-if="detail.paperNote">纸本标注</dt>
          <dd v-if="detail.paperNote">
            <span style="font-family: var(--font-serif)">{{ detail.paperNote }}</span>
            <span class="web-pill" :class="detail.status === 'repaid' ? 'web-pill--repaid' : 'web-pill--pending'">
              {{ detail.status === 'repaid' ? '已整理' : '待整理' }}
            </span>
          </dd>
          <dt v-if="detail.remark">备注</dt>
          <dd v-if="detail.remark">{{ detail.remark }}</dd>
        </dl>

        <div v-if="detail.originRecords.length" class="web-field">
          <label>这笔回礼回的是</label>
          <div class="web-chain">
            <button
              v-for="item in detail.originRecords"
              :key="item.id"
              type="button"
              class="web-chain__item"
              @click="openRecord(item)"
            >
              <DirTag :direction="item.direction" />
              <span style="flex: 1" class="web-sm">{{ item.eventName }} · {{ item.bookName }} · {{ item.itemKindLabel }}</span>
              <span class="web-amt web-amt--in">{{ money(item) }}</span>
            </button>
          </div>
        </div>
        <div v-if="detail.repayRecords.length" class="web-field">
          <label>回礼记录</label>
          <div class="web-chain">
            <button
              v-for="item in detail.repayRecords"
              :key="item.id"
              type="button"
              class="web-chain__item"
              @click="openRecord(item)"
            >
              <DirTag :direction="item.direction" />
              <span style="flex: 1" class="web-sm">{{ item.eventName }} · {{ item.bookName }} · {{ item.itemKindLabel }}</span>
              <span class="web-amt web-amt--out">{{ money(item) }}</span>
            </button>
          </div>
        </div>
      </template>
    </div>
    <div v-if="detail" class="web-drawer__foot">
      <div class="web-drawer__foot-left">
        <button class="web-btn web-btn--danger" type="button" :disabled="busy" @click="remove">
          <WebIcon name="trash" />
          删除
        </button>
      </div>
      <button v-if="isOwe" class="web-btn" type="button" @click="markNone">标为不用回</button>
      <button
        v-if="isOwe && detail.paperNote && detail.status !== 'repaid'"
        class="web-btn"
        type="button"
        @click="ui.openPaperAnno(detail.id)"
      >
        补录纸本回礼
      </button>
      <button v-if="isOwe" class="web-btn" type="button" @click="ui.openRepay([detail.id])">
        <WebIcon name="return" />
        回礼
      </button>
      <button v-if="canMarkOwed" class="web-btn" type="button" @click="markRepaid">标为对方已回</button>
      <button class="web-btn web-btn--primary" type="button" @click="ui.openEntry({ recordId: detail.id })">
        <WebIcon name="edit" />
        编辑
      </button>
    </div>
  </aside>
</template>
