<script setup lang="ts">
import { computed } from 'vue'

import type { RecordItem } from '@/api/types'
import DirTag from '@/components/DirTag.vue'
import EventSeal from '@/components/EventSeal.vue'
import Money from '@/components/Money.vue'
import PersonAvatar from '@/components/PersonAvatar.vue'
import StatusPill from '@/components/StatusPill.vue'
import { relations } from '@/styles/palette'
import { preciseDateLabel } from '@/utils/date'
import { formatCent } from '@/utils/money'

import WebIcon from './WebIcon.vue'

const props = withDefaults(
  defineProps<{
    records: RecordItem[]
    columns?: string[]
    selected?: number[]
    selectable?: boolean
    clickable?: boolean
  }>(),
  {
    columns: () => ['date', 'person', 'event', 'book', 'cat', 'amount', 'status'],
    selected: () => [],
    selectable: false,
    clickable: true,
  },
)

const emit = defineEmits<{
  row: [record: RecordItem]
  person: [id: number]
  toggle: [id: number]
}>()

const visible = computed(() => new Set(props.columns))
const selectedSet = computed(() => new Set(props.selected))

function relationLabel(code: string): string {
  return relations.find((item) => item.code === code)?.label ?? '其他'
}

function isPending(record: RecordItem): boolean {
  return record.direction === 'in' && record.status === 'pending' && !record.isRepay
}

function onRow(record: RecordItem): void {
  if (props.clickable) emit('row', record)
}
</script>

<template>
  <div class="web-table-wrap">
    <table class="web-table">
      <thead>
        <tr>
          <th v-if="selectable" class="web-table__r" />
          <th v-if="visible.has('date')">日期</th>
          <th v-if="visible.has('person')">亲友</th>
          <th v-if="visible.has('event')">事项</th>
          <th v-if="visible.has('book')">礼簿</th>
          <th v-if="visible.has('cat')">名目</th>
          <th v-if="visible.has('amount')" class="web-table__r">金额</th>
          <th v-if="visible.has('status')">状态</th>
          <th v-if="visible.has('page')">纸本</th>
          <th v-if="visible.has('anno')">纸本标注</th>
          <th v-if="visible.has('ops')" />
        </tr>
      </thead>
      <tbody>
        <tr
          v-for="record in records"
          :key="record.id"
          class="web-table__row"
          @click="onRow(record)"
        >
          <td v-if="selectable" class="web-table__r" @click.stop>
            <input
              v-if="isPending(record) || record.direction === 'out'"
              class="web-check"
              type="checkbox"
              :checked="selectedSet.has(record.id)"
              aria-label="选中记录"
              @change="emit('toggle', record.id)"
            />
          </td>
          <td v-if="visible.has('date')" class="web-sm">
            {{ preciseDateLabel(record.occurredOn, record.datePrecision) || '未注明' }}
          </td>
          <td v-if="visible.has('person')" :class="{ 'web-table__pend': isPending(record) }">
            <div class="web-cell2" @click.stop="emit('person', record.personId)">
              <PersonAvatar :id="record.personId" :name="record.personName" :size="28" />
              <span class="web-cell2__t">
                {{ record.personName }}
                <small>{{ relationLabel(record.personRelation) }}</small>
              </span>
            </div>
          </td>
          <td v-if="visible.has('event')">
            <span class="web-cell2">
              <EventSeal :type="record.eventType || 'other'" size="s" />
              <span class="web-cell2__t">{{ record.eventName }}</span>
            </span>
          </td>
          <td v-if="visible.has('book')" class="web-sm">{{ record.bookName }}</td>
          <td v-if="visible.has('cat')" class="web-sm web-muted">{{ record.itemKindLabel }}</td>
          <td v-if="visible.has('amount')" class="web-table__r">
            <div style="display: flex; align-items: center; justify-content: flex-end; gap: 8px">
              <DirTag :direction="record.direction" />
              <span v-if="record.amountCent || !record.giftName" class="web-amt" :class="`web-amt--${record.direction}`">
                <Money :cent="record.totalCent" :direction="record.direction" size="s" />
              </span>
              <span v-else class="web-cell2" style="justify-content: flex-end">
                <WebIcon name="gift" :size="15" />
                <span class="web-cell2__t" style="text-align: right">
                  {{ record.giftName }}
                  <small v-if="record.giftValueCent">估 {{ formatCent(record.giftValueCent) }}</small>
                </span>
              </span>
            </div>
          </td>
          <td v-if="visible.has('status')">
            <StatusPill
              :status="record.status"
              :direction="record.direction"
              :is-repay="record.isRepay"
              :label="record.statusLabel"
            />
          </td>
          <td v-if="visible.has('page')" class="web-sm web-muted">{{ record.paperPage || '—' }}</td>
          <td v-if="visible.has('anno')" class="web-sm">
            <span v-if="record.paperNote">{{ record.paperNote }}</span>
            <span v-else class="web-muted">—</span>
          </td>
          <td v-if="visible.has('ops')" class="web-table__r" @click.stop>
            <slot name="ops" :record="record" />
          </td>
        </tr>
      </tbody>
    </table>
  </div>
</template>
