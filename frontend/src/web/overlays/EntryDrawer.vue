<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'

import { fetchBooks } from '@/api/book'
import { createEvent, fetchEvents } from '@/api/event'
import { ApiError } from '@/api/http'
import { fetchPersonDetail } from '@/api/person'
import { createRecord, deleteRecord, fetchRecordDetail, updateRecord } from '@/api/record'
import type { BookItem, EventItem, RecordDetail } from '@/api/types'
import { useDictStore } from '@/stores/dict'
import { useContextStore } from '@/stores/context'
import { useWebUiStore } from '@/stores/web'
import { centToChinese } from '@/utils/chinese-amount'
import { todayString } from '@/utils/date'
import { centToYuanText, yuanToCent } from '@/utils/money'
import WebIcon from '@/web/components/WebIcon.vue'
import type { EntryContext } from '@/stores/web'

const props = defineProps<{ context: EntryContext }>()
const ui = useWebUiStore()
const dictStore = useDictStore()
const contextStore = useContextStore()

const books = ref<BookItem[]>([])
const events = ref<EventItem[]>([])
const loading = ref(true)
const saving = ref(false)
const error = ref('')
const conflict = ref('')
const recordId = ref(props.context.recordId ?? 0)
const newEventOpen = ref(false)
const newEvent = reactive({ name: '', type: 'other', date: todayString() })

const form = reactive({
  personId: 0,
  personName: '',
  eventId: 0,
  bookId: 0,
  direction: 'in',
  itemKind: '',
  amountText: '',
  giftName: '',
  giftValueText: '',
  date: todayString(),
  status: 'pending',
  remark: '',
})

const amountCent = computed(() => yuanToCent(form.amountText))
const giftValueCent = computed(() => yuanToCent(form.giftValueText))
const totalCent = computed(() => amountCent.value + giftValueCent.value)
const upper = computed(() => (amountCent.value > 0 ? centToChinese(amountCent.value) : ''))
const selectedBook = computed(() => books.value.find((book) => book.id === form.bookId) ?? null)

const statusOptions = computed(() => {
  if (form.direction === 'out') {
    return [
      { code: 'pending', label: '对方未回' },
      { code: 'repaid', label: '对方已回' },
      { code: 'none', label: '不用回' },
    ]
  }
  return [
    { code: 'pending', label: '未回礼' },
    { code: 'repaid', label: '已回礼' },
    { code: 'none', label: '不用回' },
  ]
})

const eventOptions = computed(() => events.value)

const title = computed(() => (recordId.value ? '编辑记录' : '记一笔'))

function inferItemKind(): string {
  if (form.giftName.trim() && amountCent.value === 0) return 'gift'
  return selectedBook.value?.category === 'huopao' ? 'huopao' : 'cash'
}

function applyRecord(detail: RecordDetail): void {
  form.personId = detail.personId
  form.personName = detail.personName
  form.eventId = detail.eventId
  form.bookId = detail.bookId
  form.direction = detail.direction
  form.itemKind = detail.itemKind
  form.amountText = detail.amountCent ? centToYuanText(detail.amountCent) : ''
  form.giftName = detail.giftName
  form.giftValueText = detail.giftValueCent ? centToYuanText(detail.giftValueCent) : ''
  form.date = detail.occurredOn || todayString()
  form.status = detail.status
  form.remark = detail.remark
}

async function load(): Promise<void> {
  loading.value = true
  error.value = ''
  try {
    const [bookList, eventPage] = await Promise.all([
      fetchBooks('normal'),
      fetchEvents({ page: 1, pageSize: 100 }),
      dictStore.ensure(),
      contextStore.load(),
    ])
    books.value = bookList
    events.value = eventPage.items
    if (recordId.value) {
      applyRecord(await fetchRecordDetail(recordId.value))
      return
    }
    if (props.context.personId) {
      const person = await fetchPersonDetail(props.context.personId)
      form.personId = person.id
      form.personName = person.name
    }
    const preferredBook =
      books.value.find((book) => book.id === props.context.bookId) ??
      books.value.find((book) => book.id === contextStore.settings.lastBookId) ??
      books.value[0]
    form.bookId = preferredBook?.id ?? 0
    form.eventId =
      props.context.eventId ||
      preferredBook?.defaultEventId ||
      contextStore.settings.lastEventId ||
      events.value[0]?.id ||
      0
    form.itemKind = preferredBook?.category === 'huopao' ? 'huopao' : 'cash'
    form.date = todayString()
  } catch (err) {
    error.value = err instanceof Error ? err.message : '加载失败'
  } finally {
    loading.value = false
  }
}

function chooseDirection(direction: string): void {
  form.direction = direction
  form.status = 'pending'
}

async function save(again = false): Promise<void> {
  error.value = ''
  conflict.value = ''
  const name = form.personName.trim()
  if (!name) {
    error.value = '请填写亲友姓名'
    return
  }
  if (!amountCent.value && !form.giftName.trim()) {
    error.value = '金额与礼品至少填一项'
    return
  }
  if (!form.bookId || !form.eventId) {
    error.value = '请选择事项和礼簿'
    return
  }
  saving.value = true
  const params = {
    personId: form.personId || undefined,
    personName: name,
    eventId: form.eventId,
    bookId: form.bookId,
    direction: form.direction,
    itemKind: form.itemKind || inferItemKind(),
    amountCent: amountCent.value,
    giftName: form.giftName.trim(),
    giftValueCent: giftValueCent.value,
    occurredOn: form.date,
    status: form.status,
    remark: form.remark.trim(),
    confirm: false,
  }
  try {
    if (recordId.value) {
      await updateRecord(recordId.value, params)
      ui.toast('已保存修改')
    } else {
      await createRecord(params)
      ui.toast(`已记下：${name} ${form.direction === 'in' ? '收' : '送'}`)
    }
    ui.markChanged()
    if (again && !recordId.value) {
      form.amountText = ''
      form.giftName = ''
      form.giftValueText = ''
      form.remark = ''
      form.status = 'pending'
      return
    }
    ui.close()
  } catch (err) {
    if (err instanceof ApiError && err.code === 409 && (err.custom as Record<string, unknown> | null)?.reason === 'need_confirm') {
      conflict.value = err.message || '同一人、事项、礼簿、名目下已有记录'
      return
    }
    error.value = err instanceof Error ? err.message : '保存失败'
  } finally {
    saving.value = false
  }
}

async function saveConfirmed(): Promise<void> {
  saving.value = true
  error.value = ''
  try {
    const params = {
      personId: form.personId || undefined,
      personName: form.personName.trim(),
      eventId: form.eventId,
      bookId: form.bookId,
      direction: form.direction,
      itemKind: form.itemKind || inferItemKind(),
      amountCent: amountCent.value,
      giftName: form.giftName.trim(),
      giftValueCent: giftValueCent.value,
      occurredOn: form.date,
      status: form.status,
      remark: form.remark.trim(),
      confirm: true,
    }
    if (recordId.value) await updateRecord(recordId.value, params)
    else await createRecord(params)
    ui.markChanged()
    ui.toast('已保存')
    ui.close()
  } catch (err) {
    error.value = err instanceof Error ? err.message : '保存失败'
  } finally {
    saving.value = false
  }
}

async function remove(): Promise<void> {
  if (!recordId.value) return
  if (!window.confirm('删除这条记录？此操作不能撤销。')) return
  try {
    await deleteRecord(recordId.value)
    ui.markChanged()
    ui.toast('已删除')
    ui.close()
  } catch (err) {
    error.value = err instanceof Error ? err.message : '删除失败'
  }
}

async function addEvent(): Promise<void> {
  if (!newEvent.name.trim()) return
  try {
    const created = await createEvent({
      name: newEvent.name.trim(),
      type: newEvent.type,
      eventDate: newEvent.date,
      datePrecision: 'day',
      host: 'other',
    })
    events.value = [created, ...events.value]
    form.eventId = created.id
    newEventOpen.value = false
    newEvent.name = ''
    ui.toast('事项已新建')
  } catch (err) {
    error.value = err instanceof Error ? err.message : '新建事项失败'
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
    <form class="web-drawer__body" @submit.prevent="save(false)">
      <div v-if="loading" class="web-empty">加载中…</div>
      <template v-else>
        <div v-if="error" class="web-alert web-alert--err">{{ error }}</div>
        <div v-if="conflict" class="web-alert web-alert--warn">
          <span>{{ conflict }}。如果是补录或分次给，可以继续保存。</span>
          <button class="web-btn web-btn--sm" type="button" style="margin-left: auto" @click="saveConfirmed">
            仍然保存
          </button>
        </div>

        <div class="web-field">
          <label for="web-entry-person">亲友</label>
          <input
            id="web-entry-person"
            v-model="form.personName"
            class="web-input"
            placeholder="姓名或拼音首字母，没有会新建"
            autocomplete="off"
            :disabled="Boolean(recordId)"
          />
          <span class="web-field__hint">同名时会按后端软确认规则处理。</span>
        </div>

        <div class="web-field-row">
          <div class="web-field">
            <label for="web-entry-event">事项</label>
            <div style="display: flex; gap: 8px">
              <select id="web-entry-event" v-model.number="form.eventId" class="web-select" style="flex: 1">
                <option v-for="item in eventOptions" :key="item.id" :value="item.id">{{ item.name }}</option>
                <option :value="0">＋ 新建事项…</option>
              </select>
              <button class="web-btn web-btn--sm" type="button" @click="newEventOpen = true">新建</button>
            </div>
          </div>
          <div class="web-field">
            <label for="web-entry-book">礼簿</label>
            <select id="web-entry-book" v-model.number="form.bookId" class="web-select">
              <option v-for="item in books" :key="item.id" :value="item.id">{{ item.name }}</option>
            </select>
          </div>
        </div>

        <div v-if="newEventOpen || form.eventId === 0" class="web-card" style="padding: 12px">
          <div class="web-field-row">
            <div class="web-field">
              <label>新事项名称</label>
              <input v-model="newEvent.name" class="web-input" placeholder="如：孙女结婚" />
            </div>
            <div class="web-field">
              <label>类型</label>
              <select v-model="newEvent.type" class="web-select">
                <option value="wedding">红事</option>
                <option value="funeral">白事</option>
                <option value="fullmoon">满月</option>
                <option value="housewarming">乔迁</option>
                <option value="school">升学</option>
                <option value="newyear">过年</option>
                <option value="other">其他</option>
              </select>
            </div>
          </div>
          <div class="web-field" style="margin-top: 8px">
            <label>日期</label>
            <input v-model="newEvent.date" class="web-input" type="date" />
          </div>
          <button class="web-btn web-btn--sm" type="button" style="margin-top: 10px" @click="addEvent">
            新建并使用
          </button>
        </div>

        <div class="web-field-row">
          <div class="web-field">
            <label>方向</label>
            <div class="web-seg web-seg--dir">
              <button
                type="button"
                data-v="in"
                :class="{ 'web-seg__on': form.direction === 'in' }"
                @click="chooseDirection('in')"
              >
                收礼
              </button>
              <button
                type="button"
                data-v="out"
                :class="{ 'web-seg__on': form.direction === 'out' }"
                @click="chooseDirection('out')"
              >
                送礼
              </button>
            </div>
          </div>
          <div class="web-field">
            <label for="web-entry-kind">名目</label>
            <select id="web-entry-kind" v-model="form.itemKind" class="web-select">
              <option value="">按礼簿推断</option>
              <option v-for="item in dictStore.itemKinds" :key="item.id" :value="item.code">
                {{ item.label }}
              </option>
            </select>
          </div>
        </div>

        <div class="web-field">
          <label for="web-entry-amount">金额</label>
          <div class="web-amount">
            <input
              id="web-entry-amount"
              v-model="form.amountText"
              class="web-input"
              inputmode="decimal"
              placeholder="0"
            />
          </div>
          <div class="web-upper">{{ upper }}</div>
        </div>

        <div class="web-field-row">
          <div class="web-field">
            <label for="web-entry-gift">礼品（金额为 0 时必填）</label>
            <input id="web-entry-gift" v-model="form.giftName" class="web-input" placeholder="如：被面一床" />
          </div>
          <div class="web-field">
            <label for="web-entry-gv">礼品估价</label>
            <input id="web-entry-gv" v-model="form.giftValueText" class="web-input" inputmode="decimal" placeholder="选填，计入统计" />
          </div>
        </div>

        <div class="web-field-row">
          <div class="web-field">
            <label for="web-entry-date">日期</label>
            <input id="web-entry-date" v-model="form.date" class="web-input" type="date" />
          </div>
          <div class="web-field">
            <label for="web-entry-status">回礼状态</label>
            <select id="web-entry-status" v-model="form.status" class="web-select">
              <option v-for="item in statusOptions" :key="item.code" :value="item.code">{{ item.label }}</option>
            </select>
          </div>
        </div>

        <div class="web-field">
          <label for="web-entry-note">备注</label>
          <textarea id="web-entry-note" v-model="form.remark" class="web-textarea" rows="2" />
        </div>
        <div class="web-muted web-sm">
          合计 {{ totalCent ? `¥${(totalCent / 100).toFixed(2)}` : '¥0' }}（金额 + 礼品估价）
        </div>
      </template>
    </form>
    <div class="web-drawer__foot">
      <div v-if="recordId" class="web-drawer__foot-left">
        <button class="web-btn web-btn--danger" type="button" @click="remove">
          <WebIcon name="trash" />
          删除
        </button>
      </div>
      <button class="web-btn" type="button" @click="ui.close()">取消</button>
      <button v-if="!recordId" class="web-btn" type="button" :disabled="saving" @click="save(true)">
        保存并再记一笔
      </button>
      <button class="web-btn web-btn--primary" type="button" :disabled="saving" @click="save(false)">
        {{ saving ? '保存中…' : '保存' }}
      </button>
    </div>
  </aside>
</template>
