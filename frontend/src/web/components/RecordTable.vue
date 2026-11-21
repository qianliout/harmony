<script setup lang="ts">
// 记录表：总账、礼簿详情、亲友详情共用；收显示礼簿，送显示事情。
import { computed } from 'vue'

import type { RecordItem } from '@/api/types'
import DirTag from '@/components/DirTag.vue'
import Money from '@/components/Money.vue'
import PersonCell from '@/web/components/PersonCell.vue'
import { formatDate } from '@/utils/date'

const props = withDefaults(
  defineProps<{
    records: RecordItem[]
    columns?: string[]
  }>(),
  {
    columns: () => ['date', 'person', 'place', 'amount', 'note', 'ops'],
  },
)

const emit = defineEmits<{
  row: [record: RecordItem]
  person: [id: number]
  edit: [record: RecordItem]
  remove: [record: RecordItem]
}>()

const visible = computed(() => new Set(props.columns))
</script>

<template>
  <div class="web-table-wrap">
    <table class="web-table">
      <thead>
        <tr>
          <th v-if="visible.has('date')">日期</th>
          <th v-if="visible.has('person')">亲友</th>
          <th v-if="visible.has('place')">礼簿 / 事情</th>
          <th v-if="visible.has('amount')" class="web-table__r">金额</th>
          <th v-if="visible.has('note')">备注</th>
          <th v-if="visible.has('ops')" class="web-table__r">操作</th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="record in records" :key="record.id" class="web-table__row" @click="emit('row', record)">
          <td v-if="visible.has('date')" class="web-sm">{{ formatDate(record.occurredOn) }}</td>
          <td v-if="visible.has('person')" @click.stop>
            <PersonCell
              :id="record.personId"
              :name="record.personName"
              :sub="record.personRemark"
              @open="emit('person', record.personId)"
            />
          </td>
          <td v-if="visible.has('place')" class="web-sm">
            <span v-if="record.direction === 'in'">{{ record.bookName }}</span>
            <span v-else>{{ record.matter || '—' }}</span>
          </td>
          <td v-if="visible.has('amount')" class="web-table__r">
            <div class="record-amount">
              <DirTag :direction="record.direction" />
              <span class="web-amt" :class="`web-amt--${record.direction}`">
                <Money :cent="record.amountCent" :direction="record.direction" size="s" />
              </span>
            </div>
          </td>
          <td v-if="visible.has('note')" class="web-sm">
            <span v-if="record.note">{{ record.note }}</span>
            <span v-else class="web-muted">—</span>
          </td>
          <td v-if="visible.has('ops')" class="web-table__r" @click.stop>
            <button class="web-btn web-btn--text web-btn--sm" type="button" @click="emit('edit', record)">编辑</button>
            <button class="web-btn web-btn--text web-btn--sm" type="button" @click="emit('remove', record)">删除</button>
          </td>
        </tr>
      </tbody>
    </table>
  </div>
</template>

<style lang="scss" scoped>
.record-amount {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: var(--sp-2);
}
</style>
