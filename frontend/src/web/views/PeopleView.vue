<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'

import { ApiError } from '@/api/http'
import { createPerson, fetchPersons } from '@/api/person'
import type { Page, PersonBrief } from '@/api/types'
import { relations } from '@/styles/palette'
import { useWebUiStore } from '@/stores/web'
import { useAsync } from '@/web/composables/useAsync'
import Modal from '@/web/components/Modal.vue'
import PageHead from '@/web/components/PageHead.vue'
import Pager from '@/web/components/Pager.vue'
import PersonCell from '@/web/components/PersonCell.vue'
import Segmented from '@/web/components/Segmented.vue'
import WebIcon from '@/web/components/WebIcon.vue'
import { goWeb } from '@/web/nav'
import { relativeSince } from '@/utils/date'
import { formatCent } from '@/utils/money'

const ui = useWebUiStore()

const pageSize = 50
const keywordInput = ref('')
const keyword = ref('')
const relation = ref('')
const pageNo = ref(1)
let keywordTimer: ReturnType<typeof setTimeout> | undefined

const params = computed(() => ({
  keyword: keyword.value.trim(),
  relation: relation.value,
  page: pageNo.value,
  pageSize,
}))

const list = useAsync<Page<PersonBrief>>(() => fetchPersons(params.value), { items: [], total: 0 })

const persons = computed(() => list.data.value.items)
const pageCount = computed(() => Math.max(1, Math.ceil(list.data.value.total / pageSize)))
const meta = computed(
  () => `${list.data.value.total} 人；同一人只有一个条目，往来跨全部礼簿合计`,
)

const relationOptions = [
  { value: '', label: '全部' },
  ...relations.map((item) => ({ value: item.code, label: item.label })),
]

watch(keywordInput, (value) => {
  if (keywordTimer) clearTimeout(keywordTimer)
  keywordTimer = setTimeout(() => {
    keyword.value = value
    pageNo.value = 1
  }, 250)
})

watch(relation, () => {
  pageNo.value = 1
})

watch(
  params,
  () => {
    void list.reload()
  },
  { deep: true },
)

function openPerson(id: number): void {
  goWeb(`/sub-web/person?id=${id}`, 'push')
}

/** ——— 新建亲友 ——— */
const dialogOpen = ref(false)
const saving = ref(false)
const formError = ref('')
const conflict = ref('')
const form = ref({
  name: '',
  relation: 'other',
  phone: '',
  remark: '',
})

function openCreate(): void {
  form.value = { name: '', relation: 'other', phone: '', remark: '' }
  formError.value = ''
  conflict.value = ''
  dialogOpen.value = true
}

function closeDialog(): void {
  if (saving.value) return
  dialogOpen.value = false
  conflict.value = ''
  formError.value = ''
}

function conflictMessage(err: ApiError): string {
  const data = err.data as { message?: string } | undefined
  return data?.message || err.message || '后端需要确认后才能保存'
}

async function save(confirm = false): Promise<void> {
  if (saving.value) return
  const name = form.value.name.trim()
  if (!name) {
    formError.value = '请填写姓名'
    return
  }
  saving.value = true
  formError.value = ''
  try {
    await createPerson({
      name,
      relation: form.value.relation || 'other',
      phone: form.value.phone.trim(),
      remark: form.value.remark.trim(),
      confirm,
    })
    ui.toast('已新建亲友')
    dialogOpen.value = false
    conflict.value = ''
    await list.reload()
  } catch (err) {
    if (err instanceof ApiError && err.code === 409) {
      conflict.value = conflictMessage(err)
      return
    }
    formError.value = err instanceof Error ? err.message : '保存失败'
  } finally {
    saving.value = false
  }
}

onMounted(() => {
  void list.reload()
})

onBeforeUnmount(() => {
  if (keywordTimer) clearTimeout(keywordTimer)
})

watch(
  () => ui.revision,
  () => {
    void list.reload()
  },
)
</script>

<template>
  <PageHead title="亲友" :meta="meta">
    <template #tools>
      <button class="web-btn web-btn--primary" type="button" @click="openCreate">
        <WebIcon name="plus" :size="16" />
        新建亲友
      </button>
    </template>
  </PageHead>

  <div class="web-card">
    <div class="web-filters">
      <input
        v-model="keywordInput"
        class="web-input"
        style="width: 240px"
        type="search"
        placeholder="姓名或拼音首字母，如 zjg"
        aria-label="搜索亲友"
      />
      <span class="web-filters__gap" />
      <Segmented v-model="relation" :options="relationOptions" />
      <div class="web-filters__right">
        <button
          v-if="keywordInput"
          class="web-btn web-btn--text web-btn--sm"
          type="button"
          @click="keywordInput = ''"
        >
          清除
        </button>
      </div>
    </div>

    <div v-if="list.error.value" class="web-alert web-alert--err people-alert">
      <WebIcon name="alert" :size="16" />
      <span>{{ list.error.value }}</span>
      <button class="web-btn web-btn--sm" type="button" @click="list.reload()">重试</button>
    </div>

    <div v-else class="web-table-wrap">
      <table class="web-table">
        <thead>
          <tr>
            <th>姓名</th>
            <th>关系</th>
            <th class="web-table__r">总收</th>
            <th class="web-table__r">总送</th>
            <th class="web-table__r">净差额</th>
            <th class="web-table__r">未回礼</th>
            <th>最近往来</th>
          </tr>
        </thead>
        <tbody>
          <tr
            v-for="person in persons"
            :key="person.id"
            class="web-table__row"
            @click="openPerson(person.id)"
          >
            <td
              :class="{ 'web-table__pend': person.pendingCount > 0 }"
              @click.stop="openPerson(person.id)"
            >
              <PersonCell
                :id="person.id"
                :name="person.name"
                :sub="person.remark"
                :size="28"
              />
            </td>
            <td class="web-sm">{{ person.relationLabel || person.relation }}</td>
            <td class="web-table__r">
              <span class="web-amt web-amt--in">{{ formatCent(person.inCent) }}</span>
            </td>
            <td class="web-table__r">
              <span class="web-amt web-amt--out">{{ formatCent(person.outCent) }}</span>
            </td>
            <td class="web-table__r web-amt">
              {{ formatCent(person.inCent - person.outCent, { sign: true }) }}
            </td>
            <td class="web-table__r">
              <span v-if="person.pendingCount" class="web-amt web-amt--pend">
                {{ person.pendingCount }} 笔
              </span>
              <span v-else class="web-muted">—</span>
            </td>
            <td class="web-sm web-muted">{{ relativeSince(person.lastDate) || '—' }}</td>
          </tr>
          <tr v-if="!persons.length">
            <td colspan="7">
              <div class="web-empty">
                {{ list.loading.value ? '加载中…' : '没有找到亲友，换个关键字试试' }}
              </div>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <div class="web-sumbar">
      <span>{{ list.data.value.total }} 人</span>
      <Pager :page="pageNo" :pages="pageCount" @change="pageNo = $event" />
    </div>
  </div>

  <div v-if="dialogOpen" class="web-mask" @click="closeDialog" />
  <Modal v-if="dialogOpen" title="新建亲友" @close="closeDialog">
    <div class="web-field">
      <label for="person-name">姓名</label>
      <input
        id="person-name"
        v-model="form.name"
        class="web-input"
        maxlength="32"
        placeholder="怎么称呼"
      />
    </div>
    <div class="web-field">
      <label for="person-relation">关系</label>
      <select id="person-relation" v-model="form.relation" class="web-select">
        <option v-for="item in relations" :key="item.code" :value="item.code">{{ item.label }}</option>
      </select>
    </div>
    <div class="web-field">
      <label for="person-phone">电话</label>
      <input
        id="person-phone"
        v-model="form.phone"
        class="web-input"
        maxlength="30"
        placeholder="选填，方便联系"
      />
    </div>
    <div class="web-field">
      <label for="person-remark">备注</label>
      <textarea
        id="person-remark"
        v-model="form.remark"
        class="web-textarea"
        rows="3"
        maxlength="200"
        placeholder="选填，例如同村、堂哥"
      />
    </div>

    <div v-if="conflict" class="web-alert web-alert--warn">{{ conflict }}</div>
    <div v-if="formError" class="web-alert web-alert--err">{{ formError }}</div>

    <template #footer>
      <button class="web-btn" type="button" :disabled="saving" @click="closeDialog">取消</button>
      <button
        v-if="conflict"
        class="web-btn web-btn--primary"
        type="button"
        :disabled="saving"
        @click="save(true)"
      >
        仍要保存
      </button>
      <button
        v-else
        class="web-btn web-btn--primary"
        type="button"
        :disabled="saving"
        @click="save(false)"
      >
        {{ saving ? '保存中…' : '保存' }}
      </button>
    </template>
  </Modal>
</template>

<style lang="scss" scoped>
.people-alert {
  margin: var(--sp-4);
}

.people-alert .web-btn {
  margin-left: auto;
}
</style>
