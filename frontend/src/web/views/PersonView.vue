<script setup lang="ts">
import { onLoad } from '@dcloudio/uni-app'
import { computed, onMounted, ref, watch } from 'vue'

import { ApiError } from '@/api/http'
import { fetchPersonDetail, updatePerson } from '@/api/person'
import type { PersonDetail, RecordItem, Timeline } from '@/api/types'
import { fetchPersonTimeline } from '@/api/view'
import DirTag from '@/components/DirTag.vue'
import Money from '@/components/Money.vue'
import StatusPill from '@/components/StatusPill.vue'
import { relations } from '@/styles/palette'
import { useWebUiStore } from '@/stores/web'
import { useAsync } from '@/web/composables/useAsync'
import Modal from '@/web/components/Modal.vue'
import PageHead from '@/web/components/PageHead.vue'
import SectionHead from '@/web/components/SectionHead.vue'
import StatCard from '@/web/components/StatCard.vue'
import WebIcon from '@/web/components/WebIcon.vue'
import { backWeb } from '@/web/nav'
import { formatDate, preciseDateLabel } from '@/utils/date'
import { formatCent } from '@/utils/money'

interface ReadPage {
  options?: Record<string, string>
  $page?: { options?: Record<string, string> }
}

/** 优先读页面 options；H5 直接刷新时从 location 的 search/hash 兜底。 */
function readQueryId(): number {
  const pages = getCurrentPages()
  const current = pages[pages.length - 1] as unknown as ReadPage | undefined
  const options = current?.options ?? current?.$page?.options ?? {}
  if (options.id) {
    const id = Number(options.id)
    if (Number.isFinite(id) && id > 0) return id
  }
  if (typeof window !== 'undefined') {
    const raw = `${window.location.search}${window.location.hash}`
    const matched = raw.match(/[?&]id=(\d+)/)
    if (matched) return Number(matched[1])
  }
  return 0
}

const ui = useWebUiStore()
const personId = ref(readQueryId())

interface PersonPayload {
  detail: PersonDetail
  timeline: Timeline
}

const data = useAsync<PersonPayload | null>(async () => {
  if (!personId.value) throw new Error('缺少亲友参数')
  const [detail, timeline] = await Promise.all([
    fetchPersonDetail(personId.value),
    fetchPersonTimeline({ personId: personId.value }),
  ])
  return { detail, timeline }
}, null)

const detail = computed(() => data.data.value?.detail ?? null)
const groups = computed(() => data.data.value?.timeline.groups ?? [])
const summary = computed(() => detail.value?.summary ?? null)
const selected = ref<number[]>([])
const metaText = computed(() => {
  const person = detail.value
  if (!person) return ''
  return [person.relationLabel || person.relation, person.phone, person.remark].filter(Boolean).join(' · ')
})

onLoad((options) => {
  const id = Number(options?.id ?? 0)
  if (Number.isFinite(id) && id > 0) personId.value = id
})

onMounted(() => {
  if (!personId.value) personId.value = readQueryId()
})

watch(
  personId,
  (id) => {
    selected.value = []
    if (id > 0) {
      void data.reload()
    }
  },
  { immediate: true },
)

/** 刷新后清掉已失效或不再可回礼的勾选。 */
watch(
  () => data.data.value,
  (payload) => {
    if (!payload) return
    const allowed = new Set(
      payload.timeline.groups
        .flatMap((group) => group.records)
        .filter((record) => isSelectable(record))
        .map((record) => record.id),
    )
    selected.value = selected.value.filter((id) => allowed.has(id))
  },
)

/** 只有「收 + 未回礼」的记录可以勾选回礼。 */
function isSelectable(record: RecordItem): boolean {
  return record.direction === 'in' && record.status === 'pending' && !record.isRepay
}

function isSelected(id: number): boolean {
  return selected.value.includes(id)
}

function toggle(record: RecordItem): void {
  if (!isSelectable(record)) return
  const index = selected.value.indexOf(record.id)
  if (index >= 0) {
    selected.value.splice(index, 1)
    return
  }
  selected.value.push(record.id)
}

function hostLabel(host: string): string {
  return host === 'self' ? '自家办' : '别人家办'
}

function openRecord(record: RecordItem): void {
  ui.openRecord(record.id)
}

function repay(): void {
  ui.openRepay([...selected.value])
}

function addRecord(): void {
  if (!personId.value) return
  ui.openEntry({ personId: personId.value })
}

/** ——— 编辑亲友 ——— */
const dialogOpen = ref(false)
const saving = ref(false)
const formError = ref('')
const conflict = ref('')
const form = ref({ name: '', relation: 'other', phone: '', remark: '' })

function openEdit(): void {
  const person = detail.value
  if (!person) return
  form.value = {
    name: person.name,
    relation: person.relation || 'other',
    phone: person.phone,
    remark: person.remark,
  }
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
  const payload = err.data as { message?: string } | undefined
  return payload?.message || err.message || '后端需要确认后才能保存'
}

async function save(confirm = false): Promise<void> {
  if (saving.value || !personId.value) return
  const name = form.value.name.trim()
  if (!name) {
    formError.value = '请填写姓名'
    return
  }
  saving.value = true
  formError.value = ''
  try {
    await updatePerson(personId.value, {
      name,
      relation: form.value.relation || 'other',
      phone: form.value.phone.trim(),
      remark: form.value.remark.trim(),
      confirm,
    })
    ui.toast('已保存')
    dialogOpen.value = false
    conflict.value = ''
    await data.reload()
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

watch(
  () => ui.revision,
  () => {
    void data.reload()
  },
)
</script>

<template>
  <PageHead :title="detail?.name || '亲友详情'" :meta="metaText">
    <template #tools>
      <button class="web-btn web-btn--text" type="button" @click="backWeb()">
        <WebIcon name="left" :size="16" />
        返回
      </button>
      <button class="web-btn" type="button" :disabled="!detail" @click="openEdit">
        <WebIcon name="edit" :size="16" />
        编辑
      </button>
      <button
        class="web-btn"
        type="button"
        :disabled="!selected.length"
        :title="selected.length ? '回礼已勾选的收礼记录' : '先勾选要回的收礼记录'"
        @click="repay"
      >
        <WebIcon name="return" :size="16" />
        回礼{{ selected.length ? ` · 已选 ${selected.length} 笔` : '' }}
      </button>
      <button class="web-btn web-btn--primary" type="button" :disabled="!detail" @click="addRecord">
        <WebIcon name="plus" :size="16" />
        给 {{ detail?.name || 'TA' }} 记一笔
      </button>
    </template>
  </PageHead>

  <div v-if="!personId" class="web-alert web-alert--err">
    <WebIcon name="alert" :size="16" />
    <span>缺少亲友参数，无法加载详情。</span>
    <button class="web-btn web-btn--sm" type="button" @click="backWeb()">返回亲友列表</button>
  </div>

  <div v-else-if="data.error.value" class="web-alert web-alert--err">
    <WebIcon name="alert" :size="16" />
    <span>{{ data.error.value }}</span>
    <button class="web-btn web-btn--sm" type="button" @click="data.reload()">重试</button>
  </div>

  <div v-else-if="!detail" class="web-card">
    <div class="web-empty">{{ data.loading.value ? '加载中…' : '还没有这位亲友的资料' }}</div>
  </div>

  <template v-else>
    <div class="web-stats">
      <StatCard
        :value="summary ? formatCent(summary.inCent) : '—'"
        label="总收"
        tone="in"
      />
      <StatCard
        :value="summary ? formatCent(summary.outCent) : '—'"
        label="总送"
        tone="out"
      />
      <StatCard
        :value="summary ? formatCent(summary.netCent, { sign: true }) : '—'"
        label="净差额"
      />
      <StatCard
        :value="summary ? `${summary.pendingCount} 笔 · ${formatCent(summary.pendingCent)}` : '—'"
        label="未回礼（勾选后点「回礼」）"
        tone="pending"
      />
    </div>

    <section v-for="group in groups" :key="group.event.id" class="web-section">
      <SectionHead
        :title="group.event.name"
        :sub="`${formatDate(group.event.eventDate)} · ${hostLabel(group.event.host)} · ${group.records.length} 笔`"
      >
        <template #tools>
          <span class="group-sum">
            <span v-if="group.inCent">
              收
              <b class="web-amt web-amt--in">{{ formatCent(group.inCent) }}</b>
            </span>
            <span v-if="group.outCent">
              送
              <b class="web-amt web-amt--out">{{ formatCent(group.outCent) }}</b>
            </span>
          </span>
        </template>
      </SectionHead>

      <div class="web-card web-table-wrap">
        <table class="web-table">
          <thead>
            <tr>
              <th class="check-col" />
              <th>日期</th>
              <th>礼簿</th>
              <th>名目</th>
              <th class="web-table__r">金额</th>
              <th>状态</th>
            </tr>
          </thead>
          <tbody>
            <tr
              v-for="record in group.records"
              :key="record.id"
              class="web-table__row"
              :class="{ 'web-table__row--on': isSelected(record.id) }"
              @click="openRecord(record)"
            >
              <td class="check-col" @click.stop>
                <input
                  v-if="isSelectable(record)"
                  class="web-check"
                  type="checkbox"
                  :checked="isSelected(record.id)"
                  aria-label="选中待回礼记录"
                  @change="toggle(record)"
                />
              </td>
              <td class="web-sm" :class="{ 'web-table__pend': isSelectable(record) }">
                {{ preciseDateLabel(record.occurredOn, record.datePrecision) || '未注明' }}
              </td>
              <td class="web-sm">{{ record.bookName }}</td>
              <td class="web-sm web-muted">{{ record.itemKindLabel }}</td>
              <td class="web-table__r">
                <span class="amount-cell">
                  <DirTag :direction="record.direction" />
                  <Money
                    v-if="record.amountCent || !record.giftName"
                    :cent="record.totalCent"
                    :direction="record.direction"
                    size="s"
                  />
                  <span v-else class="web-sm">
                    {{ record.giftName }}
                    <span v-if="record.giftValueCent" class="web-muted">
                      估 {{ formatCent(record.giftValueCent) }}
                    </span>
                  </span>
                </span>
              </td>
              <td>
                <StatusPill
                  :status="record.status"
                  :direction="record.direction"
                  :is-repay="record.isRepay"
                  :label="record.statusLabel"
                />
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </section>

    <div v-if="!groups.length" class="web-card">
      <div class="web-empty">{{ data.loading.value ? '加载中…' : '还没有往来记录' }}</div>
    </div>
  </template>

  <div v-if="dialogOpen" class="web-mask" @click="closeDialog" />
  <Modal v-if="dialogOpen" title="编辑亲友" @close="closeDialog">
    <div class="web-field">
      <label for="edit-person-name">姓名</label>
      <input id="edit-person-name" v-model="form.name" class="web-input" maxlength="32" />
    </div>
    <div class="web-field">
      <label for="edit-person-relation">关系</label>
      <select id="edit-person-relation" v-model="form.relation" class="web-select">
        <option v-for="item in relations" :key="item.code" :value="item.code">{{ item.label }}</option>
      </select>
    </div>
    <div class="web-field">
      <label for="edit-person-phone">电话</label>
      <input id="edit-person-phone" v-model="form.phone" class="web-input" maxlength="30" />
    </div>
    <div class="web-field">
      <label for="edit-person-remark">备注</label>
      <textarea id="edit-person-remark" v-model="form.remark" class="web-textarea" rows="3" maxlength="200" />
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
.check-col {
  width: 40px;
}

.amount-cell {
  display: inline-flex;
  align-items: center;
  justify-content: flex-end;
  gap: 6px;
}

.group-sum {
  display: inline-flex;
  align-items: center;
  gap: var(--sp-3);
  color: var(--c-ink-3);
  font-size: 13px;
}
</style>
