<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'

import { fetchEvents } from '@/api/event'
import { fetchRecordDetail, repayRecords } from '@/api/record'
import type { EventItem, RecordDetail } from '@/api/types'
import DirTag from '@/components/DirTag.vue'
import Money from '@/components/Money.vue'
import { useDictStore } from '@/stores/dict'
import { useWebUiStore } from '@/stores/web'
import { centToChinese } from '@/utils/chinese-amount'
import { todayString } from '@/utils/date'
import { centToYuanText, yuanToCent } from '@/utils/money'
import WebIcon from '@/web/components/WebIcon.vue'

const props = defineProps<{ originIds: number[] }>()
const ui = useWebUiStore()
const dictStore = useDictStore()

const origins = ref<RecordDetail[]>([])
const events = ref<EventItem[]>([])
const loading = ref(true)
const saving = ref(false)
const error = ref('')
const eventId = ref(0)
const newEventName = ref('')
const amountText = ref('')
const itemKind = ref('cash')
const date = ref(todayString())

const personName = computed(() => origins.value[0]?.personName ?? '')
const totalCent = computed(() => origins.value.reduce((sum, item) => sum + item.totalCent, 0))
const amountCent = computed(() => yuanToCent(amountText.value))
const upper = computed(() => (amountCent.value ? centToChinese(amountCent.value) : ''))
const eventOptions = computed(() => events.value.filter((item) => item.host === 'other'))

async function load(): Promise<void> {
  loading.value = true
  error.value = ''
  try {
    const [details, eventPage] = await Promise.all([
      Promise.all(props.originIds.map((id) => fetchRecordDetail(id))),
      fetchEvents({ page: 1, pageSize: 100 }),
      dictStore.ensure(),
    ])
    origins.value = details
    events.value = eventPage.items
    amountText.value = centToYuanText(totalCent.value)
    eventId.value = eventOptions.value[0]?.id ?? 0
  } catch (err) {
    error.value = err instanceof Error ? err.message : '加载失败'
  } finally {
    loading.value = false
  }
}

async function save(): Promise<void> {
  error.value = ''
  if (!amountCent.value) {
    error.value = '请填写回礼金额'
    return
  }
  const event =
    eventId.value > 0
      ? { id: eventId.value, name: '', type: '', date: date.value, datePrecision: 'day' }
      : { id: 0, name: newEventName.value.trim() || `${personName.value}家的事`, type: 'other', date: date.value, datePrecision: 'day' }
  saving.value = true
  try {
    await repayRecords({
      originIds: props.originIds,
      markRepaid: true,
      event,
      occurredOn: date.value,
      datePrecision: 'day',
      items: [
        {
          bookId: 0,
          itemKind: itemKind.value,
          amountCent: amountCent.value,
          giftName: '',
          giftValueCent: 0,
        },
      ],
    })
    ui.markChanged()
    ui.toast(`已回礼 ${personName.value}，${props.originIds.length} 笔改为已回礼`)
    ui.close()
  } catch (err) {
    error.value = err instanceof Error ? err.message : '回礼失败'
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
      <h3>回礼给 {{ personName }}</h3>
      <button class="web-btn web-btn--text" type="button" aria-label="关闭" @click="ui.close()">
        <WebIcon name="x" />
      </button>
    </div>
    <div class="web-drawer__body">
      <div v-if="loading" class="web-empty">加载中…</div>
      <div v-else-if="error" class="web-alert web-alert--err">{{ error }}</div>
      <template v-else>
        <div class="web-field">
          <label>回的是这 {{ origins.length }} 笔（可跨礼簿、跨名目）</label>
          <div class="web-chain">
            <div v-for="item in origins" :key="item.id" class="web-chain__item">
              <DirTag :direction="item.direction" />
              <span style="flex: 1" class="web-sm">
                {{ item.eventName }} · {{ item.bookName }} · {{ item.itemKindLabel }}
                <br />
                <span class="web-muted">{{ item.occurredOn }}</span>
              </span>
              <Money :cent="item.totalCent" :direction="item.direction" size="s" />
            </div>
          </div>
          <span class="web-field__hint">当年收了 ¥{{ (totalCent / 100).toFixed(2) }}。保存后这几笔改为已回礼，原记录不改动。</span>
        </div>

        <div class="web-field">
          <label for="web-repay-event">对方的事项</label>
          <select id="web-repay-event" v-model.number="eventId" class="web-select">
            <option v-for="item in eventOptions" :key="item.id" :value="item.id">{{ item.name }}</option>
            <option :value="0">＋ 新建：{{ personName }}家的事…</option>
          </select>
          <input
            v-if="eventId === 0"
            v-model="newEventName"
            class="web-input"
            style="margin-top: 8px"
            :placeholder="`${personName}家的事`"
          />
        </div>

        <div class="web-field">
          <label for="web-repay-amount">回礼金额</label>
          <div class="web-amount">
            <input id="web-repay-amount" v-model="amountText" class="web-input" inputmode="decimal" />
          </div>
          <div class="web-upper">{{ upper }}</div>
        </div>

        <div class="web-field-row">
          <div class="web-field">
            <label for="web-repay-kind">名目</label>
            <select id="web-repay-kind" v-model="itemKind" class="web-select">
              <option v-for="item in dictStore.itemKinds" :key="item.id" :value="item.code">{{ item.label }}</option>
            </select>
          </div>
          <div class="web-field">
            <label for="web-repay-date">日期</label>
            <input id="web-repay-date" v-model="date" class="web-input" type="date" />
          </div>
        </div>

        <div class="web-alert web-alert--info">
          回礼会新增一条送礼记录，记在原记录所在礼簿，状态显示为“回礼”，可以跳回原收礼记录。
        </div>
      </template>
    </div>
    <div class="web-drawer__foot">
      <button class="web-btn" type="button" @click="ui.close()">取消</button>
      <button class="web-btn web-btn--primary" type="button" :disabled="saving" @click="save">保存回礼</button>
    </div>
  </aside>
</template>
