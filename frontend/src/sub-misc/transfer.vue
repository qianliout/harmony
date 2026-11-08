<script setup lang="ts">
import { computed, ref } from 'vue'

import { fetchBooks } from '@/api/book'
import { fetchEvents } from '@/api/event'
import { fetchPersons } from '@/api/person'
import { commitImport, exportRecords, previewImportFile } from '@/api/transfer'
import type { ExportParams } from '@/api/transfer'
import type {
  BookItem,
  EventItem,
  ImportCommitResult,
  ImportPreview,
  ImportPreviewRow,
  ImportRow,
  PersonBrief,
} from '@/api/types'
import AppNav from '@/components/AppNav.vue'
import DirTag from '@/components/DirTag.vue'
import Money from '@/components/Money.vue'
import SectionCard from '@/components/SectionCard.vue'
import { yuanToCent } from '@/utils/money'
import { toast } from '@/utils/nav'

type RangeKind = 'all' | 'book' | 'event' | 'person'
type FormatKind = 'csv' | 'json'

interface PickOption {
  id: number
  label: string
  sub: string
}

const RANGES: { value: RangeKind; label: string }[] = [
  { value: 'all', label: '全部记录' },
  { value: 'book', label: '按礼簿' },
  { value: 'event', label: '按事项' },
  { value: 'person', label: '按亲友' },
]

const tab = ref<'export' | 'import'>('export')

const range = ref<RangeKind>('all')
const format = ref<FormatKind>('csv')
const targetId = ref(0)
const targetLabel = ref('')
const rangeSheet = ref(false)
const targetSheet = ref(false)
const options = ref<PickOption[]>([])
const optionsLoading = ref(false)

const fileName = ref('')
const importError = ref('')
const preview = ref<ImportPreview | null>(null)
const previewLoading = ref(false)
const saving = ref(false)
const commitResult = ref<ImportCommitResult | null>(null)
let optionSeq = 0

const rangeLabel = computed(() => RANGES.find((item) => item.value === range.value)?.label ?? '全部记录')
const targetPrefix = computed(() => {
  if (range.value === 'book') return '选择礼簿'
  if (range.value === 'event') return '选择事项'
  return '选择亲友'
})
const rangeSummary = computed(() => {
  if (range.value === 'all') return rangeLabel.value
  return targetLabel.value || `请选择${targetPrefix.value.replace('选择', '')}`
})

const okRows = computed<ImportPreviewRow[]>(() => preview.value?.rows.filter((item) => item.ok) ?? [])
const badRows = computed<ImportPreviewRow[]>(() => preview.value?.rows.filter((item) => !item.ok) ?? [])

async function loadAllEvents(): Promise<EventItem[]> {
  const all: EventItem[] = []
  let page = 1
  while (page <= 20) {
    const result = await fetchEvents({ page, pageSize: 100 })
    all.push(...result.items)
    if (all.length >= result.total || result.items.length === 0) break
    page += 1
  }
  return all
}

async function loadAllPersons(): Promise<PersonBrief[]> {
  const all: PersonBrief[] = []
  let page = 1
  while (page <= 20) {
    const result = await fetchPersons({ page, pageSize: 100 })
    all.push(...result.items)
    if (all.length >= result.total || result.items.length === 0) break
    page += 1
  }
  return all
}

async function openTarget(): Promise<void> {
  if (range.value === 'all') return
  const seq = ++optionSeq
  targetSheet.value = true
  options.value = []
  optionsLoading.value = true
  try {
    let loaded: PickOption[] = []
    if (range.value === 'book') {
      const books = await fetchBooks()
      loaded = books.map((book: BookItem) => ({
        id: book.id,
        label: book.name,
        sub: `${book.categoryLabel} · ${book.recordCount} 笔`,
      }))
    } else if (range.value === 'event') {
      const events = await loadAllEvents()
      loaded = events.map((event) => ({
        id: event.id,
        label: event.name,
        sub: `${event.typeLabel} · ${event.eventDate || '日期未定'}`,
      }))
    } else {
      const persons = await loadAllPersons()
      loaded = persons.map((person) => ({
        id: person.id,
        label: person.name,
        sub: person.relationLabel,
      }))
    }
    if (seq === optionSeq) {
      options.value = loaded
    }
  } catch (err) {
    if (seq !== optionSeq) return
    toast(err instanceof Error ? err.message : '选项加载失败')
  } finally {
    if (seq === optionSeq) {
      optionsLoading.value = false
    }
  }
}

function chooseRange(value: RangeKind): void {
  rangeSheet.value = false
  range.value = value
  targetId.value = 0
  targetLabel.value = ''
  if (value !== 'all') {
    void openTarget()
  }
}

function chooseTarget(option: PickOption): void {
  targetId.value = option.id
  targetLabel.value = option.label
  targetSheet.value = false
}

function doExport(): void {
  if (range.value !== 'all' && !targetId.value) {
    toast(targetPrefix.value)
    return
  }
  const params: ExportParams = { format: format.value }
  if (range.value === 'book') params.bookId = targetId.value
  if (range.value === 'event') params.eventId = targetId.value
  if (range.value === 'person') params.personId = targetId.value
  exportRecords(params)
  toast(format.value === 'csv' ? 'CSV 已开始下载' : 'JSON 已开始下载', 'success')
}

function pickFile(): void {
  // #ifdef H5
  const input = document.createElement('input')
  input.type = 'file'
  input.accept = '.csv,text/csv'
  input.style.display = 'none'
  input.onchange = () => {
    const file = input.files?.[0]
    if (file) {
      void uploadFile(file)
    }
    input.remove()
  }
  document.body.appendChild(input)
  input.click()
  // #endif
}

// #ifdef H5
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
    toast(importError.value)
  } finally {
    previewLoading.value = false
  }
}
// #endif

function downloadTemplate(): void {
  // #ifdef H5
  const header = '日期,姓名,关系,事项,事项类型,事项日期,本子,名目,方向,现金,礼品,礼品估价,状态,备注'
  const sample = [
    '2026-01-01,张三,亲戚,李家升学宴,升学,2026-06-20,人情簿,现金,收,500,,,待还,',
    '2026-02-14,李四,朋友,王家婚礼,红事,2026-02-10,人情簿,火炮,送,200,床品,300,无需还,自备',
  ]
  const content = `\uFEFF${[header, ...sample].join('\r\n')}\r\n`
  const blob = new Blob([content], { type: 'text/csv;charset=utf-8' })
  const url = URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = url
  link.download = '人情往来导入模板.csv'
  document.body.appendChild(link)
  link.click()
  link.remove()
  setTimeout(() => URL.revokeObjectURL(url), 1000)
  // #endif
  // #ifndef H5
  toast('请用浏览器打开本页下载模板')
  // #endif
}

function resetImport(): void {
  preview.value = null
  fileName.value = ''
  importError.value = ''
  commitResult.value = null
}

async function doCommit(): Promise<void> {
  const rows: ImportRow[] = okRows.value.map((item) => item.row)
  if (!rows.length || saving.value) return
  saving.value = true
  try {
    const result = await commitImport(rows)
    commitResult.value = result
    preview.value = null
    fileName.value = ''
    toast(`已导入 ${result.count} 条`, 'success')
  } catch (err) {
    toast(err instanceof Error ? err.message : '导入失败')
  } finally {
    saving.value = false
  }
}

function rowCent(row: ImportRow): number {
  const amount = yuanToCent(row.amount)
  return amount || yuanToCent(row.giftValue)
}

function rowDirection(row: ImportRow): 'in' | 'out' {
  return row.direction.includes('收') ? 'in' : 'out'
}

function rowTitle(row: ImportRow): string {
  if (row.giftName) return row.giftName
  if (row.amount) return `${row.amount} 元现金`
  return '金额待确认'
}
</script>

<template>
  <view class="screen">
    <AppNav title="导入导出" />

    <view class="page">
      <view class="segmented tabs">
        <view
          class="segmented__item"
          :class="{ 'segmented__item--on': tab === 'export' }"
          @click="tab = 'export'"
        >
          导出
        </view>
        <view
          class="segmented__item"
          :class="{ 'segmented__item--on': tab === 'import' }"
          @click="tab = 'import'"
        >
          导入
        </view>
      </view>

      <template v-if="tab === 'export'">
        <SectionCard title="导出范围">
          <view class="field" @click="rangeSheet = true">
            <text class="field__label">范围</text>
            <text class="field__value">{{ rangeSummary }}</text>
            <text class="field__arrow">›</text>
          </view>
          <view v-if="range !== 'all'" class="field" @click="openTarget">
            <text class="field__label">{{ targetPrefix }}</text>
            <text class="field__value" :class="{ 'field__value--empty': !targetLabel }">
              {{ targetLabel || '请选择' }}
            </text>
            <text class="field__arrow">›</text>
          </view>
        </SectionCard>

        <SectionCard title="导出格式">
          <view class="formats">
            <view class="format" :class="{ 'format--on': format === 'csv' }" @click="format = 'csv'">
              <text class="format__title">CSV</text>
              <text class="caption">表格通用，Excel 可直接打开</text>
            </view>
            <view class="format" :class="{ 'format--on': format === 'json' }" @click="format = 'json'">
              <text class="format__title">JSON</text>
              <text class="caption">字段完整，适合备份留存</text>
            </view>
          </view>
        </SectionCard>

        <view class="actions">
          <view class="btn btn--primary" @click="doExport">
            导出 {{ format === 'csv' ? 'CSV' : 'JSON' }}
          </view>
        </view>

        <text class="caption note">
          H5 会在浏览器里直接下载文件；小程序会先下载，再用文档预览打开，可以继续发送给朋友。
        </text>
      </template>

      <template v-else>
        <SectionCard title="导入记录">
          <view class="import">
            <text class="import__desc">
              选择从本应用导出的 CSV 文件。日期写成 YYYY-MM-DD，方向填「收」或「送」，金额单位是元。
            </text>

            <!-- #ifdef H5 -->
            <view class="btn btn--secondary import__btn" @click="pickFile">选择 CSV 文件</view>
            <!-- #endif -->

            <!-- #ifndef H5 -->
            <view class="mini-tip">
              小程序端请从聊天记录里选择 CSV 文件；本版本可先在浏览器打开 H5 页面完成导入。
            </view>
            <!-- #endif -->

            <view class="import__template" @click="downloadTemplate">下载导入模板</view>
            <text v-if="fileName" class="caption">已选择：{{ fileName }}</text>
            <text v-if="importError" class="import__error">{{ importError }}</text>
          </view>
        </SectionCard>

        <text v-if="previewLoading" class="loading-tip">正在读取文件…</text>

        <view v-if="commitResult" class="card done">
          <text class="done__title">导入完成</text>
          <text class="done__line">已写入 {{ commitResult.count }} 条记录</text>
          <text class="caption">
            新增亲友 {{ commitResult.newPersons }} 人 · 新增事项 {{ commitResult.newEvents }} 个 ·
            新增礼簿 {{ commitResult.newBooks }} 本
          </text>
          <view class="btn btn--secondary btn--small done__btn" @click="resetImport">再导入一个文件</view>
        </view>

        <template v-if="preview">
          <view class="card preview">
            <text class="preview__title">预览</text>
            <text class="preview__line">
              共 {{ preview.total }} 行 · 可导入 {{ preview.okCount }} 行 · 有问题
              {{ preview.errorCount }} 行
            </text>
          </view>

          <SectionCard :title="`可导入 ${preview.okCount} 行`">
            <view v-if="!okRows.length" class="empty-tip">没有可导入的行</view>
            <view v-for="item in okRows" :key="item.line" class="prow">
              <text class="prow__line">第 {{ item.line }} 行</text>
              <view class="prow__main">
                <view class="prow__head">
                  <text class="prow__name">{{ item.row.personName }}</text>
                  <DirTag :direction="rowDirection(item.row)" />
                </view>
                <text class="caption">
                  {{ item.row.date }} · {{ item.row.eventName }} · {{ item.row.bookName }} ·
                  {{ item.row.itemKind }}
                </text>
              </view>
              <view class="prow__amount">
                <Money
                  v-if="rowCent(item.row) > 0"
                  :cent="rowCent(item.row)"
                  :direction="rowDirection(item.row)"
                  size="s"
                />
                <text v-else class="caption">{{ rowTitle(item.row) }}</text>
              </view>
            </view>
          </SectionCard>

          <SectionCard v-if="badRows.length" :title="`有错误 ${badRows.length} 行`">
            <view v-for="item in badRows" :key="item.line" class="prow prow--bad">
              <text class="prow__line">第 {{ item.line }} 行</text>
              <view class="prow__main">
                <text class="prow__name">{{ item.row.personName || '姓名未填' }}</text>
                <text class="prow__error">{{ item.errors.join('；') }}</text>
              </view>
            </view>
          </SectionCard>

          <view class="actions">
            <view
              class="btn btn--primary"
              :class="{ 'btn--disabled': !okRows.length || saving }"
              @click="doCommit"
            >
              {{ saving ? '正在导入…' : `导入 ${okRows.length} 条` }}
            </view>
            <view class="btn btn--secondary" @click="resetImport">重新选择</view>
          </view>
          <text class="caption note">导入只会新增记录，不会覆盖已有数据。</text>
        </template>
      </template>
    </view>

    <wd-popup
      v-model="rangeSheet"
      position="bottom"
      :safe-area-inset-bottom="true"
      custom-style="border-radius: 20px 20px 0 0;"
    >
      <view class="sheet">
        <text class="sheet__title">导出范围</text>
        <view
          v-for="item in RANGES"
          :key="item.value"
          class="sheet__item"
          :class="{ 'sheet__item--on': range === item.value }"
          @click="chooseRange(item.value)"
        >
          <text>{{ item.label }}</text>
          <text v-if="range === item.value" class="sheet__check">✓</text>
        </view>
      </view>
    </wd-popup>

    <wd-popup
      v-model="targetSheet"
      position="bottom"
      :safe-area-inset-bottom="true"
      custom-style="border-radius: 20px 20px 0 0;"
    >
      <view class="sheet">
        <text class="sheet__title">{{ targetPrefix }}</text>
        <text v-if="optionsLoading" class="loading-tip">正在加载…</text>
        <text v-else-if="!options.length" class="empty-tip">还没有可选的{{ targetPrefix.replace('选择', '') }}</text>
        <scroll-view v-else scroll-y class="sheet__list">
          <view
            v-for="option in options"
            :key="option.id"
            class="sheet__item"
            :class="{ 'sheet__item--on': option.id === targetId }"
            @click="chooseTarget(option)"
          >
            <view class="sheet__main">
              <text>{{ option.label }}</text>
              <text class="caption">{{ option.sub }}</text>
            </view>
            <text v-if="option.id === targetId" class="sheet__check">✓</text>
          </view>
        </scroll-view>
      </view>
    </wd-popup>
  </view>
</template>

<style lang="scss" scoped>
.tabs {
  margin-top: 4px;
}

.field__value--empty {
  color: var(--c-ink-3);
}

.formats {
  display: flex;
  gap: 10px;
  padding: 14px;
}

.format {
  flex: 1;
  padding: 12px;
  border-radius: var(--r-s);
  background: var(--c-bg);
  box-shadow: inset 0 0 0 1px var(--c-line);
}

.format--on {
  background: var(--c-primary-light);
  box-shadow: inset 0 0 0 1px var(--c-primary);
}

.format__title {
  display: block;
  margin-bottom: 4px;
  font-size: 15px;
  font-weight: 600;
  color: var(--c-ink-1);
}

.format--on .format__title {
  color: var(--c-primary);
}

.actions {
  display: flex;
  gap: 10px;
  margin-top: 18px;
}

.actions .btn {
  flex: 1;
}

.note {
  display: block;
  margin-top: 12px;
  padding: 0 2px;
}

.import {
  padding: 14px 16px 16px;
}

.import__desc {
  display: block;
  font-size: 13px;
  line-height: 20px;
  color: var(--c-ink-2);
}

.import__btn {
  margin-top: 14px;
}

.import__template {
  margin-top: 12px;
  text-align: center;
  font-size: 13px;
  color: var(--c-primary);
}

.import__error {
  display: block;
  margin-top: 10px;
  font-size: 13px;
  line-height: 18px;
  color: var(--c-danger);
}

.mini-tip {
  margin-top: 14px;
  padding: 10px 12px;
  border-radius: var(--r-s);
  background: var(--c-fill);
  font-size: 13px;
  line-height: 20px;
  color: var(--c-ink-2);
}

.done {
  margin-top: 16px;
  padding: 16px;
}

.done__title {
  display: block;
  font-size: 16px;
  font-weight: 600;
  color: var(--c-ink-1);
}

.done__line {
  display: block;
  margin: 6px 0 2px;
  font-size: 15px;
  color: var(--c-ink-1);
}

.done__btn {
  display: inline-flex;
  margin-top: 12px;
}

.preview {
  margin-top: 16px;
  padding: 14px 16px;
}

.preview__title {
  display: block;
  font-size: 15px;
  font-weight: 600;
  color: var(--c-ink-1);
}

.preview__line {
  display: block;
  margin-top: 4px;
  font-size: 13px;
  color: var(--c-ink-2);
}

.prow {
  display: flex;
  align-items: flex-start;
  gap: 10px;
  padding: 10px 14px;
}

.prow + .prow {
  box-shadow: inset 0 1px 0 var(--c-divider);
}

.prow__line {
  width: 52px;
  flex: none;
  font-size: 11px;
  line-height: 20px;
  color: var(--c-ink-3);
}

.prow__main {
  flex: 1;
  min-width: 0;
}

.prow__head {
  display: flex;
  align-items: center;
  gap: 6px;
}

.prow__name {
  font-size: 14px;
  font-weight: 500;
  color: var(--c-ink-1);
}

.prow__amount {
  flex: none;
  max-width: 130px;
  text-align: right;
}

.prow__error {
  display: block;
  margin-top: 2px;
  font-size: 12px;
  line-height: 18px;
  color: var(--c-danger);
}

.sheet {
  padding: 18px 16px calc(12px + env(safe-area-inset-bottom));
}

.sheet__title {
  display: block;
  font-size: 17px;
  font-weight: 600;
  color: var(--c-ink-1);
}

.sheet__list {
  max-height: 55vh;
  margin-top: 8px;
}

.sheet__item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  min-height: 52px;
  padding: 8px 4px;
  font-size: 15px;
  color: var(--c-ink-1);
}

.sheet__item + .sheet__item {
  box-shadow: inset 0 1px 0 var(--c-divider);
}

.sheet__item--on {
  color: var(--c-primary);
}

.sheet__main {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.sheet__check {
  flex: none;
  color: var(--c-primary);
  font-weight: 600;
}
</style>
