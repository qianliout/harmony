<script setup lang="ts">
// 识别核对表：确认、姓名（含序号与亲友匹配）、金额（大写在下）、备注四列。
import { centToChinese } from '@/utils/chinese-amount'
import { yuanToCent } from '@/utils/money'
import { LOW_CONFIDENCE, rowProblem, type ReviewRow } from '@/web/recognize'

import WebTable from './WebTable.vue'

defineProps<{ rows: ReviewRow[] }>()

function upperOf(row: ReviewRow): string {
  const cent = yuanToCent(row.amountText)
  return cent > 0 ? centToChinese(cent) : ''
}

function candidateLabel(name: string, remark: string): string {
  return remark ? `${name} · ${remark}` : name
}
</script>

<template>
  <WebTable class="rv-table">
    <thead>
      <tr>
        <th class="rv-check">确认</th>
        <th>姓名</th>
        <th class="rv-amount">金额（元）</th>
        <th>备注</th>
      </tr>
    </thead>
    <tbody>
      <tr
        v-for="row in rows"
        :key="row.seq"
        :class="{ 'rv-row--warn': row.checked && (rowProblem(row) || row.confidence < LOW_CONFIDENCE), 'rv-row--off': !row.checked }"
      >
        <td class="rv-check">
          <input v-model="row.checked" class="web-check" type="checkbox" :aria-label="`确认第 ${row.seq} 行`" />
        </td>
        <td>
          <div class="rv-name">
            <span class="web-muted rv-seq">{{ row.seq }}</span>
            <input v-model="row.name" class="web-input" maxlength="32" placeholder="姓名" @input="row.personChoice = 0" />
          </div>
          <select v-if="row.candidates.length" v-model.number="row.personChoice" class="web-select rv-sub">
            <option v-for="item in row.candidates" :key="item.personId" :value="item.personId">
              已有：{{ candidateLabel(item.personName, item.personRemark) }}
            </option>
            <option :value="0">新建亲友</option>
          </select>
          <input
            v-if="!row.personChoice"
            v-model="row.remark"
            class="web-input rv-sub"
            maxlength="32"
            placeholder="新亲友备注，同名时用来区分"
          />
          <small v-if="row.checked && rowProblem(row)" class="rv-flag">{{ rowProblem(row) }}</small>
          <small v-else-if="row.confidence < LOW_CONFIDENCE" class="rv-flag">字迹不清，请对照原图：{{ row.raw }}</small>
        </td>
        <td class="rv-amount">
          <input v-model="row.amountText" class="web-input" inputmode="decimal" placeholder="0" />
          <small class="web-upper">{{ upperOf(row) }}</small>
        </td>
        <td>
          <input v-model="row.note" class="web-input" maxlength="256" placeholder="礼品、火炮等" />
        </td>
      </tr>
    </tbody>
  </WebTable>
</template>

<style lang="scss" scoped>
.rv-check {
  width: 56px;
  text-align: center;
}

.rv-amount {
  width: 160px;
}

.rv-name {
  display: flex;
  align-items: center;
  gap: var(--sp-2);
}

.rv-seq {
  min-width: 20px;
}

.rv-sub {
  margin-top: var(--sp-1);
  margin-left: 28px;
  width: calc(100% - 28px);
}

.rv-flag {
  display: block;
  margin-left: 28px;
  color: var(--c-pending);
}

.rv-row--warn td {
  background: var(--c-pending-light);
}

.rv-row--off td {
  opacity: 0.5;
}
</style>
