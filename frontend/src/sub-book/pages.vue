<script setup lang="ts">
import { onLoad, onShow } from '@dcloudio/uni-app'
import { computed, ref } from 'vue'

import { fetchBookDetail, fetchPaperPageDetail, fetchPaperPages, setPageCheck } from '@/api/book'
import type { BookItem, PaperPage, PaperPageDetail, RecordItem } from '@/api/types'
import AppNav from '@/components/AppNav.vue'
import EmptyState from '@/components/EmptyState.vue'
import Money from '@/components/Money.vue'
import RecordRow from '@/components/RecordRow.vue'
import Skeleton from '@/components/Skeleton.vue'
import { centToYuanText, yuanToCent } from '@/utils/money'
import { go, toast } from '@/utils/nav'

const bookId = ref(0)
const book = ref<BookItem | null>(null)
const pages = ref<PaperPage[]>([])
const expanded = ref('')
const details = ref<Record<string, PaperPageDetail>>({})
const detailLoading = ref('')
const detailError = ref('')
const declaredText = ref('')
const saving = ref(false)
const loading = ref(true)
const showSkeleton = ref(false)
const error = ref('')

let skeletonTimer: ReturnType<typeof setTimeout> | null = null
let loadingData = false

const stats = computed(() => {
  const total = pages.value.length
  const consistent = pages.value.filter((page) => page.consistent).length
  return { total, consistent, pending: total - consistent }
})

const currentDetail = computed(() => details.value[expanded.value] || null)

function patchPage(data: PaperPage): void {
  pages.value = pages.value.map((page) =>
    page.page === data.page
      ? {
          ...page,
          declaredCent: data.declaredCent,
          recordCent: data.recordCent,
          recordCount: data.recordCount,
          hasDeclared: data.hasDeclared,
          consistent: data.consistent,
          diffCent: data.diffCent,
        }
      : page,
  )
}

async function load(): Promise<void> {
  if (!bookId.value || loadingData) return
  loadingData = true
  const firstLoad = !book.value
  if (firstLoad) loading.value = true
  error.value = ''
  if (firstLoad) {
    if (skeletonTimer) clearTimeout(skeletonTimer)
    skeletonTimer = setTimeout(() => {
      showSkeleton.value = true
    }, 300)
  }
  try {
    const [detail, pageItems] = await Promise.all([fetchBookDetail(bookId.value), fetchPaperPages(bookId.value)])
    book.value = detail
    pages.value = pageItems
    expanded.value = ''
    details.value = {}
  } catch (err) {
    error.value = err instanceof Error ? err.message : '加载失败'
  } finally {
    if (skeletonTimer) clearTimeout(skeletonTimer)
    skeletonTimer = null
    showSkeleton.value = false
    loading.value = false
    loadingData = false
  }
}

function retry(): void {
  load()
}

async function loadDetail(page: PaperPage): Promise<void> {
  detailLoading.value = page.page
  detailError.value = ''
  try {
    const detail = await fetchPaperPageDetail(bookId.value, page.page)
    details.value = { ...details.value, [page.page]: detail }
    patchPage(detail)
    if (expanded.value === page.page) {
      declaredText.value = detail.hasDeclared ? centToYuanText(detail.declaredCent) : ''
    }
  } catch (err) {
    if (expanded.value === page.page) {
      detailError.value = err instanceof Error ? err.message : '这一页加载失败'
    }
  } finally {
    if (detailLoading.value === page.page) {
      detailLoading.value = ''
    }
  }
}

function togglePage(page: PaperPage): void {
  if (expanded.value === page.page) {
    expanded.value = ''
    return
  }
  expanded.value = page.page
  detailError.value = ''
  const cached = details.value[page.page]
  if (cached) {
    declaredText.value = cached.hasDeclared ? centToYuanText(cached.declaredCent) : ''
    return
  }
  declaredText.value = page.hasDeclared ? centToYuanText(page.declaredCent) : ''
  loadDetail(page)
}

function retryDetail(): void {
  const page = pages.value.find((item) => item.page === expanded.value)
  if (page) loadDetail(page)
}

function onDeclaredInput(event: Event): void {
  declaredText.value = (event as unknown as { detail: { value: string } }).detail.value
}

async function saveDeclared(): Promise<void> {
  if (!expanded.value || saving.value) return
  if (!declaredText.value.trim()) {
    toast('请填写纸本合计')
    return
  }
  saving.value = true
  try {
    const cent = yuanToCent(declaredText.value)
    await setPageCheck(bookId.value, expanded.value, cent)
    const detail = await fetchPaperPageDetail(bookId.value, expanded.value)
    details.value = { ...details.value, [expanded.value]: detail }
    patchPage(detail)
    declaredText.value = detail.hasDeclared ? centToYuanText(detail.declaredCent) : ''
    toast('已保存', 'success')
  } catch (err) {
    toast(err instanceof Error ? err.message : '保存失败')
  } finally {
    saving.value = false
  }
}

function openRecord(record: RecordItem): void {
  go(`/sub-record/detail?id=${record.id}`)
}

function continueEntry(page: PaperPage): void {
  const detail = details.value[page.page]
  const maxSeq = detail?.records.reduce((max, record) => Math.max(max, record.paperSeq || 0), 0) ?? 0
  const seq = Math.max(maxSeq, detail?.records.length ?? 0) + 1
  go(
    `/sub-record/continuous?bookId=${bookId.value}&page=${encodeURIComponent(page.page)}&seq=${seq}`,
  )
}

function goContinuous(): void {
  go(`/sub-record/continuous?bookId=${bookId.value}`)
}

function statusLabel(page: PaperPage): string {
  if (page.consistent) return '✓ 一致'
  if (page.hasDeclared) return '⚠ 待查'
  return '未填合计'
}

function statusClass(page: PaperPage): string {
  if (page.consistent) return 'status--ok'
  if (page.hasDeclared) return 'status--pending'
  return 'status--none'
}

onLoad((query) => {
  bookId.value = Number(query?.bookId ?? query?.id ?? 0)
})

onShow(() => {
  if (!bookId.value) return
  if (loading.value && loadingData) return
  load()
})
</script>

<template>
  <view class="screen">
    <AppNav title="按页核对" />

    <view class="page">
      <Skeleton v-if="showSkeleton" />

      <view v-else-if="error && !book" class="state">
        <EmptyState title="没能打开按页核对" :description="error" action-text="重试" @action="retry" />
      </view>

      <template v-else-if="!loading">
        <view class="head">
          <text class="head__book">{{ book?.name || '礼簿' }}</text>
          <view class="head__stats">
            <text class="head__stat">共 {{ stats.total }} 页</text>
            <text class="head__dot">·</text>
            <text class="head__stat head__stat--ok">{{ stats.consistent }} 页一致</text>
            <text class="head__dot">·</text>
            <text class="head__stat head__stat--pending">{{ stats.pending }} 页待查</text>
          </view>
        </view>

        <view v-if="!pages.length" class="state">
          <EmptyState
            title="还没有页码记录"
            description="连续录入时填写页码，这里就能按页核对"
            action-text="去连续录入"
            @action="goContinuous"
          />
        </view>

        <view v-for="page in pages" :key="page.page" class="page-card card">
          <view class="page-head" @click="togglePage(page)">
            <view class="page-head__main">
              <view class="page-head__line">
                <text class="page-head__no">第 {{ page.page }} 页</text>
                <text class="page-head__count">{{ page.recordCount }} 笔</text>
                <text class="status" :class="statusClass(page)">{{ statusLabel(page) }}</text>
              </view>
              <view class="page-head__amounts">
                <text class="page-head__label">录入</text>
                <Money :cent="page.recordCent" direction="none" size="s" />
                <text class="page-head__label">/</text>
                <template v-if="page.hasDeclared">
                  <text class="page-head__label">纸本</text>
                  <Money :cent="page.declaredCent" direction="none" size="s" />
                </template>
                <text v-else class="page-head__label">未填合计</text>
                <template v-if="page.hasDeclared && !page.consistent">
                  <text class="page-head__diff">差</text>
                  <Money :cent="page.diffCent" direction="none" size="s" sign />
                </template>
              </view>
            </view>
            <text class="page-head__arrow">{{ expanded === page.page ? '▾' : '›' }}</text>
          </view>

          <view v-if="expanded === page.page" class="page-detail">
            <view v-if="detailLoading === page.page" class="loading-tip">加载中…</view>

            <view v-else-if="detailError" class="detail-error" @click="retryDetail">
              {{ detailError }}，点击重试
            </view>

            <template v-else-if="currentDetail && currentDetail.page === page.page">
              <view class="check-row">
                <text class="check-row__label">纸本合计</text>
                <view class="check-row__field">
                  <text class="check-row__currency">¥</text>
                  <input
                    class="check-row__input"
                    type="digit"
                    :value="declaredText"
                    placeholder="页脚写的合计"
                    @input="onDeclaredInput"
                  />
                </view>
                <view class="btn btn--small btn--primary check-row__save" :class="{ 'btn--disabled': saving }" @click="saveDeclared">
                  {{ saving ? '保存中…' : '保存' }}
                </view>
              </view>

              <view v-if="currentDetail.records.length" class="records">
                <RecordRow
                  v-for="record in currentDetail.records"
                  :key="record.id"
                  :record="record"
                  mode="book"
                  @tap="openRecord"
                />
              </view>
              <text v-else class="empty-tip">这一页还没有记录</text>

              <view class="continue" @click="continueEntry(page)">在本页继续录入</view>
            </template>
          </view>
        </view>
      </template>
    </view>
  </view>
</template>

<style lang="scss" scoped>
.state {
  padding-top: 20px;
}

.head {
  padding: 8px 2px 14px;
}

.head__book {
  display: block;
  font-size: 19px;
  line-height: 26px;
  font-weight: 600;
  color: var(--c-ink-1);
}

.head__stats {
  display: flex;
  align-items: baseline;
  gap: 4px;
  margin-top: 4px;
}

.head__stat {
  font-size: 13px;
  color: var(--c-ink-3);
}

.head__stat--ok {
  color: var(--c-repaid);
}

.head__stat--pending {
  color: var(--c-pending);
}

.head__dot {
  font-size: 12px;
  color: var(--c-ink-4);
}

.page-card {
  margin-bottom: 10px;
  overflow: hidden;
}

.page-head {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 13px 16px;
}

.page-head__main {
  flex: 1;
  min-width: 0;
}

.page-head__line {
  display: flex;
  align-items: center;
  gap: 8px;
}

.page-head__no {
  font-size: 15px;
  line-height: 22px;
  font-weight: 600;
  color: var(--c-ink-1);
}

.page-head__count {
  font-size: 12px;
  color: var(--c-ink-3);
}

.status {
  margin-left: auto;
  flex: none;
  height: 20px;
  padding: 0 7px;
  border-radius: var(--r-full);
  font-size: 11px;
  line-height: 20px;
}

.status--ok {
  background: var(--c-repaid-light);
  color: var(--c-repaid);
}

.status--pending {
  background: var(--c-pending-light);
  color: var(--c-pending);
}

.status--none {
  background: var(--c-fill);
  color: var(--c-ink-3);
}

.page-head__amounts {
  display: flex;
  align-items: baseline;
  flex-wrap: wrap;
  gap: 3px;
  margin-top: 4px;
}

.page-head__label {
  font-size: 12px;
  color: var(--c-ink-3);
}

.page-head__diff {
  margin-left: 2px;
  font-size: 12px;
  color: var(--c-pending);
}

.page-head__arrow {
  flex: none;
  font-size: 18px;
  color: var(--c-ink-4);
}

.page-detail {
  box-shadow: inset 0 1px 0 var(--c-divider);
}

.check-row {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 12px 16px;
  background: var(--c-fill);
}

.check-row__label {
  flex: none;
  font-size: 13px;
  color: var(--c-ink-2);
}

.check-row__field {
  flex: 1;
  display: flex;
  align-items: center;
  gap: 2px;
  height: 36px;
  padding: 0 10px;
  border-radius: var(--r-s);
  background: var(--c-surface);
}

.check-row__currency {
  font-family: $font-num;
  font-size: 15px;
  color: var(--c-ink-2);
}

.check-row__input {
  flex: 1;
  min-width: 0;
  font-family: $font-num;
  font-size: 16px;
  color: var(--c-ink-1);
  text-align: right;
}

.check-row__save {
  flex: none;
}

.records {
  background: var(--c-surface);
}

.detail-error {
  padding: 20px 16px;
  text-align: center;
  font-size: 13px;
  color: var(--c-danger);
}

.continue {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 46px;
  margin: 12px 16px 16px;
  border-radius: var(--r-s);
  background: var(--c-primary-light);
  color: var(--c-primary);
  font-size: 14px;
  font-weight: 500;
}
</style>
