<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'

import { fetchSessions, deleteSession, type AuthSession } from '@/api/auth'
import { fetchBooks } from '@/api/book'
import { createDict, deleteDict, fetchDicts } from '@/api/dict'
import { fetchEvents } from '@/api/event'
import { fetchPersons } from '@/api/person'
import { commitImport, exportRecords, previewImportFile } from '@/api/transfer'
import type { ExportParams } from '@/api/transfer'
import type {
  DictItem,
  ImportCommitResult,
  ImportPreview,
  ImportPreviewRow,
  ImportRow,
} from '@/api/types'
import { useAuthStore } from '@/stores/auth'
import { useWebUiStore } from '@/stores/web'
import { paletteNames, paletteOf } from '@/styles/palette'
import { formatTimestamp } from '@/utils/format'
import { confirmModal } from '@/utils/nav'
import Modal from '@/web/components/Modal.vue'
import PageHead from '@/web/components/PageHead.vue'
import SectionHead from '@/web/components/SectionHead.vue'
import Segmented from '@/web/components/Segmented.vue'
import { goWeb } from '@/web/nav'

type Scope = 'all' | 'book' | 'event' | 'person'
type DictType = 'book_category' | 'item_kind'

interface PickOption {
  id: number
  label: string
  sub: string
}

const ui = useWebUiStore()
const auth = useAuthStore()

const mode = ref('export')
const format = ref('csv')
const scope = ref('all')
const targetId = ref(0)
const options = ref<PickOption[]>([])
const optionsLoading = ref(false)
const optionsError = ref('')

const fileName = ref('')
const importError = ref('')
const preview = ref<ImportPreview | null>(null)
const previewLoading = ref(false)
const savingImport = ref(false)
const commitResult = ref<ImportCommitResult | null>(null)

const dicts = ref<DictItem[]>([])
const dictLoading = ref(true)
const dictError = ref('')
const dictModal = ref(false)
const dictType = ref<DictType>('book_category')
const dictLabel = ref('')
const dictColor = ref('stone')
const savingDict = ref(false)

const sessions = ref<AuthSession[]>([])
const sessionsLoading = ref(true)
const sessionsError = ref('')
const deletingId = ref(0)

const modeOptions = [
  { value: 'export', label: '导出' },
  { value: 'import', label: '导入' },
]

const formatOptions = [
  { value: 'csv', label: 'CSV' },
  { value: 'json', label: 'JSON' },
]

const scopeLabels: { value: Scope; label: string }[] = [
  { value: 'all', label: '全部记录' },
  { value: 'book', label: '按礼簿' },
  { value: 'event', label: '按事项' },
  { value: 'person', label: '按亲友' },
]

const categoryDicts = computed(() =>
  dicts.value.filter((item) => item.type === 'book_category').sort((a, b) => a.sort - b.sort),
)
const itemDicts = computed(() =>
  dicts.value.filter((item) => item.type === 'item_kind').sort((a, b) => a.sort - b.sort),
)

const targetLabel = computed(() => {
  if (scope.value === 'all') return '全部记录'
  if (scope.value === 'book') return '礼簿'
  if (scope.value === 'event') return '事项'
  return '亲友'
})

const okRows = computed<ImportPreviewRow[]>(
  () => preview.value?.rows.filter((item) => item.ok) ?? [],
)
const badRows = computed<ImportPreviewRow[]>(
  () => preview.value?.rows.filter((item) => !item.ok) ?? [],
)

const userLabel = computed(() => auth.user?.nickname || '本机用户')
const userSub = computed(() => (auth.isAdmin ? '管理员' : '记账人'))
const dictTitle = computed(() =>
  dictType.value === 'book_category' ? '新增礼簿类别' : '新增名目',
)

function clientLabel(client: string): string {
  return client === 'mp' ? '微信小程序' : '网页'
}

async function loadAllEvents(): Promise<PickOption[]> {
  const all: PickOption[] = []
  let page = 1
  while (page <= 20) {
    const result = await fetchEvents({ page, pageSize: 100 })
    all.push(
      ...result.items.map((event) => ({
        id: event.id,
        label: event.name,
        sub: `${event.typeLabel} · ${event.eventDate || '日期未定'}`,
      })),
    )
    if (all.length >= result.total || result.items.length === 0) break
    page += 1
  }
  return all
}

async function loadAllPersons(): Promise<PickOption[]> {
  const all: PickOption[] = []
  let page = 1
  while (page <= 20) {
    const result = await fetchPersons({ page, pageSize: 100 })
    all.push(
      ...result.items.map((person) => ({
        id: person.id,
        label: person.name,
        sub: person.relationLabel,
      })),
    )
    if (all.length >= result.total || result.items.length === 0) break
    page += 1
  }
  return all
}

/** 导出范围切换后按需加载目标选项，避免设置页一进来就拉全量数据。 */
async function loadOptions(): Promise<void> {
  if (scope.value === 'all') {
    options.value = []
    return
  }
  optionsLoading.value = true
  optionsError.value = ''
  try {
    if (scope.value === 'book') {
      const books = await fetchBooks('normal')
      options.value = books.map((book) => ({
        id: book.id,
        label: book.name,
        sub: `${book.categoryLabel} · ${book.recordCount} 笔`,
      }))
    } else if (scope.value === 'event') {
      options.value = await loadAllEvents()
    } else {
      options.value = await loadAllPersons()
    }
  } catch (err) {
    options.value = []
    optionsError.value = err instanceof Error ? err.message : '选项加载失败'
  } finally {
    optionsLoading.value = false
  }
}

function doExport(): void {
  if (scope.value !== 'all' && !targetId.value) {
    ui.toast(`请选择${targetLabel.value}`)
    return
  }
  const params: ExportParams = { format: format.value === 'json' ? 'json' : 'csv' }
  if (scope.value === 'book') params.bookId = targetId.value
  if (scope.value === 'event') params.eventId = targetId.value
  if (scope.value === 'person') params.personId = targetId.value
  exportRecords(params)
  ui.toast(format.value === 'json' ? 'JSON 已开始下载' : 'CSV 已开始下载')
}

function pickFile(): void {
  // #ifdef H5
  const input = document.createElement('input')
  input.type = 'file'
  input.accept = '.csv,text/csv'
  input.style.display = 'none'
  input.onchange = () => {
    const file = input.files?.[0]
    if (file) void uploadFile(file)
    input.remove()
  }
  document.body.appendChild(input)
  input.click()
  // #endif
}

async function uploadFile(file: File): Promise<void> {
  fileName.value = file.name
  importError.value = ''
  preview.value = null
  commitResult.value = null
  previewLoading.value = true
  try {
    preview.value = await previewImportFile(file)
  } catch (err) {
    importError.value = err instanceof Error ? err.message : '读取文件失败'
    ui.toast(importError.value)
  } finally {
    previewLoading.value = false
  }
}

function resetImport(): void {
  preview.value = null
  fileName.value = ''
  importError.value = ''
  commitResult.value = null
}

async function doCommit(): Promise<void> {
  if (!okRows.value.length || savingImport.value) return
  savingImport.value = true
  try {
    const rows: ImportRow[] = okRows.value.map((item) => item.row)
    const result = await commitImport(rows)
    commitResult.value = result
    preview.value = null
    fileName.value = ''
    ui.toast(`已导入 ${result.count} 条`)
    ui.markChanged()
  } catch (err) {
    ui.toast(err instanceof Error ? err.message : '导入失败')
  } finally {
    savingImport.value = false
  }
}

function rowTitle(row: ImportRow): string {
  return row.personName || row.remark || '未命名'
}

async function loadDicts(): Promise<void> {
  dictLoading.value = true
  dictError.value = ''
  try {
    dicts.value = await fetchDicts()
  } catch (err) {
    dicts.value = []
    dictError.value = err instanceof Error ? err.message : '名目加载失败'
  } finally {
    dictLoading.value = false
  }
}

function openCreate(type: DictType): void {
  dictType.value = type
  dictLabel.value = ''
  dictColor.value = 'stone'
  dictModal.value = true
}

async function saveDict(): Promise<void> {
  const label = dictLabel.value.trim()
  if (!label) {
    ui.toast('请填写名称')
    return
  }
  if (savingDict.value) return
  savingDict.value = true
  try {
    await createDict({ type: dictType.value, label, color: dictColor.value })
    dictModal.value = false
    ui.toast('已新增')
    await loadDicts()
  } catch (err) {
    ui.toast(err instanceof Error ? err.message : '新增失败')
  } finally {
    savingDict.value = false
  }
}

async function removeDict(item: DictItem): Promise<void> {
  if (item.builtin) return
  const ok = await confirmModal(`删除「${item.label}」？已使用该名称的记录不受影响。`, '删除名目')
  if (!ok) return
  try {
    await deleteDict(item.id)
    ui.toast('已删除')
    await loadDicts()
  } catch (err) {
    ui.toast(err instanceof Error ? err.message : '删除失败')
  }
}

async function loadSessions(): Promise<void> {
  sessionsLoading.value = true
  sessionsError.value = ''
  try {
    sessions.value = await fetchSessions()
  } catch (err) {
    sessions.value = []
    sessionsError.value = err instanceof Error ? err.message : '登录设备加载失败'
  } finally {
    sessionsLoading.value = false
  }
}

async function revoke(session: AuthSession): Promise<void> {
  if (deletingId.value) return
  const ok = await confirmModal(`让「${session.device || clientLabel(session.client)}」下线？`, '下线设备')
  if (!ok) return
  deletingId.value = session.id
  try {
    await deleteSession(session.id)
    sessions.value = sessions.value.filter((item) => item.id !== session.id)
    ui.toast('已下线')
  } catch (err) {
    ui.toast(err instanceof Error ? err.message : '操作失败')
  } finally {
    deletingId.value = 0
  }
}

async function logout(): Promise<void> {
  const ok = await confirmModal('退出后需要重新扫码登录。', '退出登录')
  if (!ok) return
  await auth.logout()
}

watch(scope, () => {
  targetId.value = 0
  options.value = []
  void loadOptions()
})

onMounted(() => {
  void auth.loadMe().catch(() => undefined)
  void loadDicts()
  void loadSessions()
})
</script>

<template>
  <div>
    <PageHead title="设置" :meta="`${userLabel} · ${userSub}`" />

    <section class="settings-section">
      <SectionHead title="礼簿整理" sub="整理纸本、标注与页码" />
      <div class="web-card web-list">
        <button class="web-list__row settings-link" type="button" @click="goWeb('/sub-web/books')">
          <span class="web-list__grow">
            礼簿管理
            <small>新建、合并与整理礼簿</small>
          </span>
          <span class="settings-link__arrow">›</span>
        </button>
        <button class="web-list__row settings-link" type="button" @click="goWeb('/sub-web/notes')">
          <span class="web-list__grow">
            整理纸本标注
            <small>把连续录入时抄下的标注转成回礼</small>
          </span>
          <span class="settings-link__arrow">›</span>
        </button>
        <button class="web-list__row settings-link" type="button" @click="goWeb('/sub-web/pages')">
          <span class="web-list__grow">
            按页核对
            <small>对照纸本合计与录入金额</small>
          </span>
          <span class="settings-link__arrow">›</span>
        </button>
      </div>
    </section>

    <section class="settings-section">
      <SectionHead title="导入导出" sub="备份、迁移与批量写入" />
      <div class="web-card settings-card">
        <Segmented v-model="mode" :options="modeOptions" />

        <template v-if="mode === 'export'">
          <div class="web-field settings-field">
            <label>格式</label>
            <Segmented v-model="format" :options="formatOptions" />
            <span class="web-field__hint">CSV 适合表格查看，JSON 字段完整适合备份。</span>
          </div>

          <div class="web-field settings-field">
            <label>范围</label>
            <select v-model="scope" class="web-select">
              <option v-for="item in scopeLabels" :key="item.value" :value="item.value">
                {{ item.label }}
              </option>
            </select>
          </div>

          <div v-if="scope !== 'all'" class="web-field settings-field">
            <label>{{ targetLabel }}</label>
            <select v-if="!optionsLoading && options.length" v-model.number="targetId" class="web-select">
              <option :value="0" disabled>请选择{{ targetLabel }}</option>
              <option v-for="item in options" :key="item.id" :value="item.id">
                {{ item.label }}（{{ item.sub }}）
              </option>
            </select>
            <span v-else-if="optionsLoading" class="web-field__hint">正在加载{{ targetLabel }}…</span>
            <span v-else-if="optionsError" class="web-alert web-alert--err">{{ optionsError }}</span>
            <span v-else class="web-field__hint">还没有可选的{{ targetLabel }}。</span>
          </div>

          <div class="settings-actions">
            <button
              class="web-btn web-btn--primary"
              type="button"
              :disabled="scope !== 'all' && !targetId"
              @click="doExport"
            >
              导出 {{ format.toUpperCase() }}
            </button>
          </div>
        </template>

        <template v-else>
          <div class="web-field settings-field">
            <label>导入文件</label>
            <span class="web-field__hint">
              选择从本应用导出的 CSV。日期写成 YYYY-MM-DD，方向填「收」或「送」，金额单位是元。
            </span>
            <div class="settings-actions">
              <button class="web-btn web-btn--text" type="button" @click="pickFile">选择 CSV 文件</button>
            </div>
            <span v-if="fileName" class="web-field__hint">已选择：{{ fileName }}</span>
            <span v-if="importError" class="web-alert web-alert--err">{{ importError }}</span>
          </div>

          <div v-if="previewLoading" class="web-empty">
            <span class="web-muted">正在读取文件…</span>
          </div>

          <div v-if="commitResult" class="web-alert web-alert--info settings-result">
            <div>
              <strong>导入完成</strong>
              <div>已写入 {{ commitResult.count }} 条记录。</div>
              <div>
                新增亲友 {{ commitResult.newPersons }} 人 · 新增事项
                {{ commitResult.newEvents }} 个 · 新增礼簿 {{ commitResult.newBooks }} 本
              </div>
            </div>
            <button class="web-btn web-btn--text web-btn--sm" type="button" @click="resetImport">
              再导入一个
            </button>
          </div>

          <template v-if="preview">
            <div class="settings-preview">
              <span>共 {{ preview.total }} 行</span>
              <span>可导入 {{ preview.okCount }} 行</span>
              <span :class="{ 'settings-preview__bad': preview.errorCount > 0 }">
                有问题 {{ preview.errorCount }} 行
              </span>
            </div>

            <div v-if="badRows.length" class="web-alert web-alert--err settings-errors">
              <div class="settings-errors__list">
                <div v-for="item in badRows.slice(0, 20)" :key="item.line">
                  第 {{ item.line }} 行 · {{ rowTitle(item.row) }}：{{ item.errors.join('；') }}
                </div>
                <div v-if="badRows.length > 20">……共 {{ badRows.length }} 行有错误</div>
              </div>
            </div>

            <div class="settings-actions">
              <button
                class="web-btn web-btn--primary"
                type="button"
                :disabled="!okRows.length || savingImport"
                @click="doCommit"
              >
                {{ savingImport ? '正在导入…' : `确认写入 ${okRows.length} 条` }}
              </button>
              <button class="web-btn web-btn--text" type="button" @click="resetImport">重新选择</button>
            </div>
          </template>
        </template>
      </div>
    </section>

    <section class="settings-section">
      <SectionHead title="名目与类别" sub="内置项不可删除" />
      <div v-if="dictError" class="web-alert web-alert--err settings-error">
        <span>{{ dictError }}</span>
        <button class="web-btn web-btn--text web-btn--sm" type="button" @click="loadDicts">重试</button>
      </div>

      <div v-else-if="dictLoading" class="web-card web-empty">
        <span class="web-muted">正在加载名目…</span>
      </div>

      <div v-else class="web-grid-2e">
        <div class="web-card settings-dict">
          <SectionHead title="礼簿类别">
            <template #tools>
              <button class="web-btn web-btn--text web-btn--sm" type="button" @click="openCreate('book_category')">
                ＋ 新增
              </button>
            </template>
          </SectionHead>
          <div v-if="!categoryDicts.length" class="web-empty">
            <span class="web-muted">还没有礼簿类别</span>
          </div>
          <div v-else class="web-list">
            <div v-for="item in categoryDicts" :key="item.id" class="web-list__row">
              <span class="dict-dot" :style="{ background: paletteOf(item.color).main }" />
              <span class="web-list__grow">{{ item.label }}</span>
              <span v-if="item.builtin" class="web-pill web-pill--neutral">内置</span>
              <button
                v-else
                class="web-btn web-btn--text web-btn--sm"
                type="button"
                @click="removeDict(item)"
              >
                删除
              </button>
            </div>
          </div>
        </div>

        <div class="web-card settings-dict">
          <SectionHead title="名目">
            <template #tools>
              <button class="web-btn web-btn--text web-btn--sm" type="button" @click="openCreate('item_kind')">
                ＋ 新增
              </button>
            </template>
          </SectionHead>
          <div v-if="!itemDicts.length" class="web-empty">
            <span class="web-muted">还没有名目</span>
          </div>
          <div v-else class="web-list">
            <div v-for="item in itemDicts" :key="item.id" class="web-list__row">
              <span class="dict-dot" :style="{ background: paletteOf(item.color).main }" />
              <span class="web-list__grow">{{ item.label }}</span>
              <span v-if="item.builtin" class="web-pill web-pill--neutral">内置</span>
              <button
                v-else
                class="web-btn web-btn--text web-btn--sm"
                type="button"
                @click="removeDict(item)"
              >
                删除
              </button>
            </div>
          </div>
        </div>
      </div>
    </section>

    <section class="settings-section">
      <SectionHead title="账号与登录设备" :sub="`${userLabel} · ${userSub}`" />
      <div class="web-card settings-card">
        <div v-if="sessionsError" class="web-alert web-alert--err settings-error">
          <span>{{ sessionsError }}</span>
          <button class="web-btn web-btn--text web-btn--sm" type="button" @click="loadSessions">
            重试
          </button>
        </div>

        <div v-else-if="sessionsLoading" class="web-empty">
          <span class="web-muted">正在加载登录设备…</span>
        </div>

        <div v-else-if="!sessions.length" class="web-empty">
          <span class="web-muted">没有可管理的登录设备</span>
        </div>

        <table v-else class="web-table">
          <thead>
            <tr>
              <th>设备</th>
              <th>类型</th>
              <th>最近使用</th>
              <th>登录时间</th>
              <th class="web-table__r">操作</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="session in sessions" :key="session.id">
              <td>{{ session.device || clientLabel(session.client) }}</td>
              <td class="web-sm">{{ clientLabel(session.client) }}</td>
              <td class="web-sm">{{ formatTimestamp(session.lastUsedAt) }}</td>
              <td class="web-sm">{{ formatTimestamp(session.createdAt) }}</td>
              <td class="web-table__r">
                <button
                  class="web-btn web-btn--text web-btn--sm"
                  type="button"
                  :disabled="deletingId === session.id"
                  @click="revoke(session)"
                >
                  {{ deletingId === session.id ? '处理中…' : '下线' }}
                </button>
              </td>
            </tr>
          </tbody>
        </table>

        <div class="settings-actions settings-actions--end">
          <button class="web-btn web-btn--danger" type="button" @click="logout">退出登录</button>
        </div>
      </div>
    </section>

    <div v-if="dictModal" class="web-mask" @click="dictModal = false" />
    <Modal v-if="dictModal" :title="dictTitle" @close="dictModal = false">
      <div class="settings-card settings-dict-form">
        <div class="web-field">
          <label>名称</label>
          <input v-model="dictLabel" class="web-input" type="text" placeholder="例如：乔迁" maxlength="20" />
        </div>
        <div class="web-field">
          <label>颜色</label>
          <div class="settings-colors">
            <button
              v-for="name in paletteNames"
              :key="name"
              class="settings-color"
              :class="{ 'settings-color--on': dictColor === name }"
              type="button"
              :style="{ background: paletteOf(name).main }"
              :aria-label="name"
              @click="dictColor = name"
            />
          </div>
        </div>
      </div>
      <template #footer>
        <button class="web-btn web-btn--text" type="button" @click="dictModal = false">取消</button>
        <button class="web-btn web-btn--primary" type="button" :disabled="savingDict" @click="saveDict">
          {{ savingDict ? '保存中…' : '保存' }}
        </button>
      </template>
    </Modal>
  </div>
</template>

<style lang="scss" scoped>
.settings-section {
  margin-top: var(--sp-6);
}

.settings-card {
  padding: var(--sp-4) var(--sp-5);
  display: flex;
  flex-direction: column;
  gap: var(--sp-4);
  overflow: hidden;
}

.settings-field {
  margin-top: var(--sp-2);
}

.settings-link {
  width: 100%;
  border: 0;
  background: none;
  text-align: left;
  font-size: 14px;
  color: var(--c-ink-1);
}

.settings-link__arrow {
  color: var(--c-ink-4);
  font-size: 18px;
}

.settings-actions {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: var(--sp-2);
}

.settings-actions--end {
  justify-content: flex-end;
}

.settings-result {
  align-items: center;
  justify-content: space-between;
}

.settings-result strong {
  font-weight: 600;
}

.settings-preview {
  display: flex;
  flex-wrap: wrap;
  gap: var(--sp-4);
  font-size: 13px;
  color: var(--c-ink-2);
}

.settings-preview__bad {
  color: var(--c-danger);
}

.settings-errors {
  max-height: 220px;
  overflow-y: auto;
}

.settings-errors__list {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.settings-error {
  align-items: center;
  justify-content: space-between;
}

.settings-dict {
  padding: var(--sp-4) var(--sp-5) var(--sp-2);
}

.dict-dot {
  width: 10px;
  height: 10px;
  border-radius: var(--r-full);
  flex: none;
}

.settings-dict-form {
  padding: 0;
  gap: var(--sp-4);
}

.settings-colors {
  display: flex;
  flex-wrap: wrap;
  gap: var(--sp-2);
}

.settings-color {
  width: 26px;
  height: 26px;
  border: 2px solid transparent;
  border-radius: var(--r-full);
  padding: 0;
  box-shadow: inset 0 0 0 2px var(--c-surface);
}

.settings-color--on {
  border-color: var(--c-ink-1);
}
</style>
