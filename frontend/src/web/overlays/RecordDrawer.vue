<script setup lang="ts">
// 记录详情：查看一笔往来，可编辑或删除。
import { computed } from 'vue'

import { deleteRecord, fetchRecordDetail } from '@/api/record'
import type { RecordItem } from '@/api/types'
import DirTag from '@/components/DirTag.vue'
import Money from '@/components/Money.vue'
import { useWebUiStore } from '@/stores/web'
import { centToChinese } from '@/utils/chinese-amount'
import { formatFullDate } from '@/utils/date'
import WebIcon from '@/web/components/WebIcon.vue'
import { useAsync } from '@/web/composables/useAsync'
import { goWeb } from '@/web/nav'

const props = defineProps<{ id: number }>()
const ui = useWebUiStore()

const detail = useAsync<RecordItem | null>(() => fetchRecordDetail(props.id), null)
const record = computed(() => detail.data.value)

function openPerson(): void {
  if (!record.value) return
  ui.close()
  goWeb(`/sub-web/person?id=${record.value.personId}`, 'push')
}

function openBook(): void {
  if (!record.value) return
  ui.close()
  goWeb(`/sub-web/book?id=${record.value.bookId}`, 'push')
}

async function remove(): Promise<void> {
  if (!window.confirm('删除这条记录？此操作不能撤销。')) return
  try {
    await deleteRecord(props.id)
    ui.markChanged()
    ui.toast('已删除')
    ui.close()
  } catch (err) {
    ui.toast(err instanceof Error ? err.message : '删除失败')
  }
}

void detail.reload()
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
      <div v-if="detail.error.value" class="web-alert web-alert--err">{{ detail.error.value }}</div>
      <div v-else-if="!record" class="web-empty">加载中…</div>
      <template v-else>
        <div class="record-head">
          <DirTag :direction="record.direction" />
          <span class="web-amt" :class="`web-amt--${record.direction}`">
            <Money :cent="record.amountCent" :direction="record.direction" size="l" />
          </span>
        </div>
        <div class="web-upper">{{ centToChinese(record.amountCent) }}</div>
        <dl class="web-dl">
          <dt>亲友</dt>
          <dd>
            <button class="web-btn web-btn--text web-btn--sm" type="button" @click="openPerson">
              {{ record.personDisplayName }}
            </button>
          </dd>
          <template v-if="record.direction === 'in'">
            <dt>礼簿</dt>
            <dd>
              <button class="web-btn web-btn--text web-btn--sm" type="button" @click="openBook">
                {{ record.bookName }}
              </button>
            </dd>
          </template>
          <template v-else>
            <dt>事情</dt>
            <dd>{{ record.matter || '—' }}</dd>
          </template>
          <dt>日期</dt>
          <dd>{{ formatFullDate(record.occurredOn) }}</dd>
          <dt>备注</dt>
          <dd>{{ record.note || '—' }}</dd>
        </dl>
      </template>
    </div>
    <div class="web-drawer__foot">
      <div class="web-drawer__foot-left">
        <button class="web-btn web-btn--danger" type="button" @click="remove">
          <WebIcon name="trash" />
          删除
        </button>
      </div>
      <button class="web-btn" type="button" @click="ui.close()">关闭</button>
      <button class="web-btn web-btn--primary" type="button" @click="ui.openEntry({ recordId: id })">
        <WebIcon name="edit" />
        编辑
      </button>
    </div>
  </aside>
</template>

<style lang="scss" scoped>
.record-head {
  display: flex;
  align-items: center;
  gap: var(--sp-2);
}
</style>
