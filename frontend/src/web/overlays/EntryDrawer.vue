<script setup lang="ts">
// 记一笔 / 编辑记录：方向、亲友、礼簿或事情、金额、日期、备注六项。
// 收必须选礼簿；送不挂礼簿，只填事情。亲友可从候选里选，也可直接输入新姓名（同名不同人用备注区分）。
import { computed, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue'

import { fetchBooks } from '@/api/book'
import { ApiError } from '@/api/http'
import { fetchPersonDetail, fetchPersons } from '@/api/person'
import { createRecord, deleteRecord, fetchRecordDetail, updateRecord } from '@/api/record'
import type { BookItem, Direction, PersonItem } from '@/api/types'
import type { EntryContext } from '@/stores/web'
import { useWebUiStore } from '@/stores/web'
import { centToChinese } from '@/utils/chinese-amount'
import { todayString } from '@/utils/date'
import { centToYuanText, yuanToCent } from '@/utils/money'
import DateField from '@/web/components/DateField.vue'
import WebIcon from '@/web/components/WebIcon.vue'

/** 同名歧义时后端在 data 里返回的候选亲友。 */
interface PersonCandidate {
  id: number
  name: string
  remark: string
}

const props = defineProps<{ context: EntryContext }>()
const ui = useWebUiStore()

const recordId = props.context.recordId ?? 0
const books = ref<BookItem[]>([])
const loading = ref(true)
const saving = ref(false)
const error = ref('')
const suggestions = ref<PersonItem[]>([])
const candidates = ref<PersonCandidate[]>([])
let suggestTimer: ReturnType<typeof setTimeout> | undefined

const form = reactive({
  direction: 'in' as Direction,
  personId: 0,
  personLabel: '',
  personQuery: '',
  personRemark: '',
  bookId: 0,
  matter: '',
  amountText: '',
  date: todayString(),
  note: '',
})

const title = computed(() => (recordId ? '编辑记录' : '记一笔'))
const amountCent = computed(() => yuanToCent(form.amountText))
const upper = computed(() => (amountCent.value > 0 ? centToChinese(amountCent.value) : ''))
const isNewPerson = computed(() => !form.personId && form.personQuery.trim() !== '')

/** 记一笔选了礼簿时，日期随礼簿带出；编辑已有记录不覆盖记录自己的日期。 */
function applyBookDate(): void {
  if (recordId || form.direction !== 'in') return
  const book = books.value.find((item) => item.id === form.bookId)
  if (book?.occurredOn) form.date = book.occurredOn
}

function choosePerson(person: { id: number; name: string; remark: string; displayName?: string }): void {
  form.personId = person.id
  form.personLabel = person.displayName || (person.remark ? `${person.name} · ${person.remark}` : person.name)
  form.personQuery = form.personLabel
  form.personRemark = ''
  suggestions.value = []
  candidates.value = []
}

function onPersonInput(): void {
  form.personId = 0
  candidates.value = []
  if (suggestTimer) clearTimeout(suggestTimer)
  const keyword = form.personQuery.trim()
  if (!keyword) {
    suggestions.value = []
    return
  }
  suggestTimer = setTimeout(() => {
    fetchPersons({ keyword, pageSize: 8 })
      .then((page) => {
        suggestions.value = page.items
      })
      .catch(() => {
        suggestions.value = []
      })
  }, 200)
}

async function load(): Promise<void> {
  loading.value = true
  error.value = ''
  try {
    books.value = (await fetchBooks()).items
    if (recordId) {
      const record = await fetchRecordDetail(recordId)
      form.direction = record.direction
      choosePerson({ id: record.personId, name: record.personName, remark: record.personRemark })
      form.bookId = record.bookId
      form.matter = record.matter
      form.amountText = record.amountCent ? centToYuanText(record.amountCent) : ''
      form.date = record.occurredOn
      form.note = record.note
      return
    }
    if (props.context.personId) {
      choosePerson(await fetchPersonDetail(props.context.personId))
    }
    form.bookId = props.context.bookId || books.value[0]?.id || 0
  } catch (err) {
    error.value = err instanceof Error ? err.message : '加载失败'
  } finally {
    loading.value = false
  }
}

function validate(): string {
  if (!form.personId && !form.personQuery.trim()) return '请选择或填写亲友'
  if (recordId && !form.personId) return '请从候选里选择亲友'
  if (form.direction === 'in' && !form.bookId) return '收礼需要选择礼簿'
  if (!form.date) return '请选择日期'
  return ''
}

function resetForNext(): void {
  form.personId = 0
  form.personLabel = ''
  form.personQuery = ''
  form.personRemark = ''
  form.amountText = ''
  form.note = ''
}

async function save(again: boolean): Promise<void> {
  error.value = validate()
  if (error.value) return
  saving.value = true
  const bookId = form.direction === 'in' ? form.bookId : 0
  const matter = form.direction === 'out' ? form.matter.trim() : ''
  try {
    if (recordId) {
      await updateRecord(recordId, {
        personId: form.personId,
        bookId,
        direction: form.direction,
        amountCent: amountCent.value,
        matter,
        note: form.note.trim(),
        occurredOn: form.date,
      })
      ui.toast('已保存修改')
    } else {
      await createRecord({
        personId: form.personId || undefined,
        personName: form.personId ? undefined : form.personQuery.trim(),
        personRemark: form.personId ? undefined : form.personRemark.trim(),
        bookId,
        direction: form.direction,
        amountCent: amountCent.value,
        matter,
        note: form.note.trim(),
        occurredOn: form.date,
      })
      ui.toast(`已记下：${form.personQuery.trim()} ${form.direction === 'in' ? '收' : '送'}`)
    }
    ui.markChanged()
    if (again && !recordId) {
      resetForNext()
      return
    }
    ui.close()
  } catch (err) {
    const ambiguous = err instanceof ApiError && err.custom?.reason === 'person_ambiguous' && Array.isArray(err.data)
    if (ambiguous) {
      candidates.value = (err as ApiError).data as PersonCandidate[]
    }
    error.value = err instanceof Error ? err.message : '保存失败'
  } finally {
    saving.value = false
  }
}

async function remove(): Promise<void> {
  if (!recordId) return
  if (!window.confirm('删除这条记录？此操作不能撤销。')) return
  try {
    await deleteRecord(recordId)
    ui.markChanged()
    ui.toast('已删除')
    ui.close()
  } catch (err) {
    error.value = err instanceof Error ? err.message : '删除失败'
  }
}

watch(
  () => form.direction,
  (direction) => {
    if (direction !== 'in') return
    if (!form.bookId) form.bookId = books.value[0]?.id || 0
    applyBookDate()
  },
)

watch(() => form.bookId, applyBookDate)

onMounted(() => {
  void load()
})

onBeforeUnmount(() => {
  if (suggestTimer) clearTimeout(suggestTimer)
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
        <div v-if="candidates.length" class="web-alert web-alert--warn entry-candidates">
          <span>选择是哪一位：</span>
          <button
            v-for="item in candidates"
            :key="item.id"
            class="web-btn web-btn--sm"
            type="button"
            @click="choosePerson(item)"
          >
            {{ item.remark ? `${item.name} · ${item.remark}` : item.name }}
          </button>
        </div>

        <div class="web-field">
          <label>方向</label>
          <div class="web-seg web-seg--dir">
            <button type="button" data-v="in" :class="{ 'web-seg__on': form.direction === 'in' }" @click="form.direction = 'in'">
              收（别人来我家）
            </button>
            <button type="button" data-v="out" :class="{ 'web-seg__on': form.direction === 'out' }" @click="form.direction = 'out'">
              送（我去别人家）
            </button>
          </div>
        </div>

        <div class="web-field entry-person">
          <label for="web-entry-person">亲友</label>
          <input
            id="web-entry-person"
            v-model="form.personQuery"
            class="web-input"
            placeholder="姓名或拼音首字母"
            autocomplete="off"
            @input="onPersonInput"
          />
          <div v-if="suggestions.length" class="web-card entry-suggest">
            <button
              v-for="person in suggestions"
              :key="person.id"
              type="button"
              class="entry-suggest__item"
              @click="choosePerson(person)"
            >
              {{ person.displayName }}
              <small class="web-muted">往来 {{ person.recordCount }} 笔</small>
            </button>
          </div>
          <span v-if="form.personId" class="web-field__hint">已选：{{ form.personLabel }}</span>
          <span v-else-if="isNewPerson" class="web-field__hint">没有选中已有亲友，保存时按姓名匹配，找不到就新建。</span>
        </div>

        <div v-if="isNewPerson && !recordId" class="web-field">
          <label for="web-entry-remark">亲友备注</label>
          <input
            id="web-entry-remark"
            v-model="form.personRemark"
            class="web-input"
            maxlength="32"
            placeholder="选填；同名的不同人用备注区分，如 村东"
          />
        </div>

        <div v-if="form.direction === 'in'" class="web-field">
          <label for="web-entry-book">礼簿</label>
          <select id="web-entry-book" v-model.number="form.bookId" class="web-select">
            <option v-if="!books.length" :value="0">还没有礼簿，先去新建</option>
            <option v-for="item in books" :key="item.id" :value="item.id">{{ item.name }}</option>
          </select>
        </div>
        <div v-else class="web-field">
          <label for="web-entry-matter">事情</label>
          <input id="web-entry-matter" v-model="form.matter" class="web-input" maxlength="128" placeholder="如：张三升学酒" />
        </div>

        <div class="web-field-row">
          <div class="web-field">
            <label for="web-entry-amount">金额（元）</label>
            <input id="web-entry-amount" v-model="form.amountText" class="web-input" inputmode="decimal" placeholder="0" />
            <div class="web-upper">{{ upper }}</div>
          </div>
          <div class="web-field">
            <label>日期</label>
            <DateField v-model="form.date" />
            <span v-if="form.direction === 'in'" class="web-field__hint">随礼簿日期自动带出，可修改</span>
          </div>
        </div>

        <div class="web-field">
          <label for="web-entry-note">备注</label>
          <textarea
            id="web-entry-note"
            v-model="form.note"
            class="web-textarea"
            rows="2"
            maxlength="256"
            placeholder="礼品、火炮等补充信息"
          />
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
      <button v-if="!recordId" class="web-btn" type="button" :disabled="saving" @click="save(true)">保存并再记一笔</button>
      <button class="web-btn web-btn--primary" type="button" :disabled="saving" @click="save(false)">
        {{ saving ? '保存中…' : '保存' }}
      </button>
    </div>
  </aside>
</template>

<style lang="scss" scoped>
.entry-person {
  position: relative;
}

.entry-suggest {
  position: absolute;
  top: 100%;
  left: 0;
  right: 0;
  z-index: 2;
  display: flex;
  flex-direction: column;
  padding: var(--sp-1) 0;
}

.entry-suggest__item {
  display: flex;
  justify-content: space-between;
  padding: var(--sp-2) var(--sp-3);
  border: 0;
  background: none;
  text-align: left;
  cursor: pointer;

  &:hover {
    background: var(--c-primary-light);
  }
}

.entry-candidates {
  flex-wrap: wrap;
  gap: var(--sp-2);
}
</style>
