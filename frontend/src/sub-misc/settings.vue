<script setup lang="ts">
import { onShow } from '@dcloudio/uni-app'
import { computed, ref } from 'vue'

import { createDict, deleteDict, updateDict } from '@/api/dict'
import type { DictItem } from '@/api/types'
import AppNav from '@/components/AppNav.vue'
import Money from '@/components/Money.vue'
import SectionCard from '@/components/SectionCard.vue'
import Skeleton from '@/components/Skeleton.vue'
import { useDictStore } from '@/stores/dict'
import { useContextStore } from '@/stores/context'
import { paletteNames, paletteOf } from '@/styles/palette'
import { centToYuanText, yuanToCent } from '@/utils/money'
import { confirmModal, toast } from '@/utils/nav'

const MAX_QUICK_AMOUNTS = 6

const context = useContextStore()
const dict = useDictStore()

const loading = ref(false)
const loadError = ref('')
const savingDict = ref(false)
const sheet = ref(false)
const editingId = ref(0)
const editType = ref('book_category')
const editLabel = ref('')
const editColor = ref('stone')
const editBuiltin = ref(false)
let suppressQuickTap = false

const quickAmounts = computed(() => context.settings.quickAmounts ?? [])
const showOwing = computed(() => Boolean(context.settings.showOwing))
const sheetTitle = computed(
  () => `${editingId.value ? '编辑' : '新增'}${editType.value === 'book_category' ? '类别' : '名目'}`,
)

async function refresh(): Promise<void> {
  loading.value = !dict.loaded
  loadError.value = ''
  try {
    await Promise.all([context.load(), dict.ensure(true)])
  } catch (err) {
    loadError.value = err instanceof Error ? err.message : '设置加载失败'
    toast(loadError.value)
  } finally {
    loading.value = false
  }
}

function editQuickAmount(index: number): void {
  const current = index >= 0 ? quickAmounts.value[index] ?? 0 : 0
  uni.showModal({
    title: index >= 0 ? '修改快捷金额' : '新增快捷金额',
    editable: true,
    placeholderText: '金额（元）',
    content: current ? centToYuanText(current) : '',
    success: (res) => {
      if (!res.confirm) return
      void saveQuickAmount(index, String(res.content ?? ''))
    },
  })
}

async function saveQuickAmount(index: number, text: string): Promise<void> {
  const cent = yuanToCent(text.trim())
  if (cent <= 0) {
    toast('请输入大于 0 的金额')
    return
  }
  const list = [...quickAmounts.value]
  if (index >= 0) {
    list[index] = cent
  } else {
    if (list.length >= MAX_QUICK_AMOUNTS) {
      toast('最多 6 个快捷金额')
      return
    }
    list.push(cent)
  }
  await context.save({ quickAmounts: list })
  toast('已保存', 'success')
}

function onQuickTap(index: number): void {
  if (suppressQuickTap) {
    suppressQuickTap = false
    return
  }
  editQuickAmount(index)
}

function removeQuickAmount(index: number): void {
  suppressQuickTap = true
  setTimeout(() => {
    suppressQuickTap = false
  }, 600)
  if (quickAmounts.value.length <= 1) {
    toast('至少保留一个快捷金额')
    return
  }
  void confirmModal('删除后可以用「＋ 新增」再加回来。', '删除快捷金额').then(async (ok) => {
    if (!ok) return
    const list = [...quickAmounts.value]
    list.splice(index, 1)
    await context.save({ quickAmounts: list })
    toast('已删除')
  })
}

async function toggleOwing(): Promise<void> {
  await context.save({ showOwing: !showOwing.value })
}

function openCreate(type: string): void {
  editingId.value = 0
  editType.value = type
  editLabel.value = ''
  editColor.value = 'stone'
  editBuiltin.value = false
  sheet.value = true
}

function openEdit(item: DictItem): void {
  editingId.value = item.id
  editType.value = item.type
  editLabel.value = item.label
  editColor.value = item.color || 'stone'
  editBuiltin.value = item.builtin
  sheet.value = true
}

async function saveDict(): Promise<void> {
  const label = editLabel.value.trim()
  if (!label) {
    toast('请填写名称')
    return
  }
  if (savingDict.value) return
  savingDict.value = true
  try {
    if (editingId.value) {
      await updateDict(editingId.value, { label, color: editColor.value })
    } else {
      await createDict({ type: editType.value, label, color: editColor.value })
    }
    await dict.ensure(true)
    sheet.value = false
    toast('已保存', 'success')
  } catch (err) {
    toast(err instanceof Error ? err.message : '保存失败')
  } finally {
    savingDict.value = false
  }
}

async function removeDict(): Promise<void> {
  if (!editingId.value || editBuiltin.value) return
  const ok = await confirmModal('删除后，已经用它记过的记录仍会保留名称，但新记录不能再选它。', '删除这一项？')
  if (!ok) return
  try {
    await deleteDict(editingId.value)
    await dict.ensure(true)
    sheet.value = false
    toast('已删除', 'success')
  } catch (err) {
    toast(err instanceof Error ? err.message : '删除失败')
  }
}

onShow(() => {
  void refresh()
})
</script>

<template>
  <view class="screen">
    <AppNav title="偏好设置" />

    <view class="page">
      <Skeleton v-if="loading && !dict.loaded" type="list" />

      <view v-else-if="loadError && !dict.loaded" class="state">
        <text class="empty-tip">{{ loadError }}</text>
        <view class="btn btn--secondary btn--small" @click="refresh">重新加载</view>
      </view>

      <template v-else>
        <SectionCard title="快捷金额">
          <view class="quick-wrap">
            <view class="quick-grid">
              <view
                v-for="(amount, index) in quickAmounts"
                :key="`${amount}-${index}`"
                class="quick"
                @click="onQuickTap(index)"
                @longpress="removeQuickAmount(index)"
              >
                <Money :cent="amount" size="s" />
              </view>
              <view class="quick quick--add" @click="editQuickAmount(-1)">＋ 新增</view>
            </view>
            <text class="caption">点击修改金额，长按删除</text>
          </view>
        </SectionCard>

        <SectionCard title="首页显示">
          <view class="switch-row">
            <view class="switch-row__main">
              <text class="switch-row__title">该回我的</text>
              <text class="caption">关闭后，该回我的金额不再突出显示</text>
            </view>
            <view class="toggle" :class="{ 'toggle--on': showOwing }" @click="toggleOwing">
              <view class="toggle__knob" />
            </view>
          </view>
        </SectionCard>

        <SectionCard title="礼簿类别" link-text="新增" @link="openCreate('book_category')">
          <view v-if="!dict.categories.length" class="empty-tip">还没有类别，点右上角新增</view>
          <view
            v-for="item in dict.categories"
            :key="item.id"
            class="dict-row"
            @click="openEdit(item)"
          >
            <view class="dict-row__dot" :style="{ background: paletteOf(item.color).main }" />
            <text class="dict-row__name">{{ item.label }}</text>
            <text v-if="item.builtin" class="dict-row__builtin">内置</text>
            <text class="dict-row__arrow">›</text>
          </view>
        </SectionCard>

        <SectionCard title="名目" link-text="新增" @link="openCreate('item_kind')">
          <view v-if="!dict.itemKinds.length" class="empty-tip">还没有名目，点右上角新增</view>
          <view
            v-for="item in dict.itemKinds"
            :key="item.id"
            class="dict-row"
            @click="openEdit(item)"
          >
            <view class="dict-row__dot" :style="{ background: paletteOf(item.color).main }" />
            <text class="dict-row__name">{{ item.label }}</text>
            <text v-if="item.builtin" class="dict-row__builtin">内置</text>
            <text class="dict-row__arrow">›</text>
          </view>
        </SectionCard>

        <SectionCard title="关于">
          <view class="about">
            <text class="about__name">人情往来账本</text>
            <text class="caption">朱砂礼簿 · 版本 0.1.0</text>
            <view class="about__row">
              <text class="about__label">数据存储</text>
              <text class="about__value">自建服务</text>
            </view>
            <view class="about__row">
              <text class="about__label">备份方式</text>
              <text class="about__value">导入导出页可随时取回</text>
            </view>
          </view>
        </SectionCard>
      </template>
    </view>

    <wd-popup
      v-model="sheet"
      position="bottom"
      :safe-area-inset-bottom="true"
      custom-style="max-height: 90vh; border-radius: 20px 20px 0 0;"
    >
      <view class="sheet">
        <text class="sheet__title">{{ sheetTitle }}</text>

        <view class="field sheet__field">
          <text class="field__label">名称</text>
          <input v-model="editLabel" class="field__input" placeholder="例如 现金、火炮" maxlength="20" />
        </view>

        <text class="sheet__label">颜色</text>
        <view class="swatches">
          <view
            v-for="name in paletteNames"
            :key="name"
            class="swatch"
            :class="{ 'swatch--on': editColor === name }"
            :style="{ background: paletteOf(name).main }"
            @click="editColor = name"
          />
        </view>

        <text v-if="editBuiltin" class="caption sheet__hint">内置项可以改名和换色，但不能删除。</text>

        <view class="sheet__actions">
          <view class="btn btn--secondary" @click="sheet = false">取消</view>
          <view class="btn btn--primary" :class="{ 'btn--disabled': savingDict }" @click="saveDict">
            保存
          </view>
        </view>

        <view v-if="editingId && !editBuiltin" class="sheet__delete" @click="removeDict">
          删除这个{{ editType === 'book_category' ? '类别' : '名目' }}
        </view>
      </view>
    </wd-popup>
  </view>
</template>

<style lang="scss" scoped>
.quick-wrap {
  padding: 14px 16px 16px;
}

.state {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 4px;
  padding-top: 24px;
}

.quick-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 10px;
  margin-bottom: 10px;
}

.quick {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 44px;
  border-radius: var(--r-s);
  background: var(--c-bg);
  box-shadow: inset 0 0 0 1px var(--c-line);
}

.quick--add {
  font-size: 14px;
  color: var(--c-primary);
  background: transparent;
  box-shadow: inset 0 0 0 1px var(--c-primary);
}

.switch-row {
  display: flex;
  align-items: center;
  gap: 12px;
  min-height: 60px;
  padding: 10px 16px;
}

.switch-row__main {
  flex: 1;
  min-width: 0;
}

.switch-row__title {
  display: block;
  font-size: 15px;
  color: var(--c-ink-1);
}

.toggle {
  position: relative;
  width: 48px;
  height: 28px;
  flex: none;
  border-radius: var(--r-full);
  background: var(--c-ink-4);
  transition: background 0.2s ease;
}

.toggle--on {
  background: var(--c-primary);
}

.toggle__knob {
  position: absolute;
  top: 3px;
  left: 3px;
  width: 22px;
  height: 22px;
  border-radius: var(--r-full);
  background: var(--c-surface);
  box-shadow: 0 1px 3px var(--c-line);
  transition: transform 0.2s ease;
}

.toggle--on .toggle__knob {
  transform: translateX(20px);
}

.dict-row {
  display: flex;
  align-items: center;
  gap: 10px;
  min-height: 52px;
  padding: 0 16px;
}

.dict-row + .dict-row {
  box-shadow: inset 0 1px 0 var(--c-divider);
}

.dict-row__dot {
  width: 10px;
  height: 10px;
  flex: none;
  border-radius: var(--r-full);
}

.dict-row__name {
  flex: 1;
  min-width: 0;
  font-size: 15px;
  color: var(--c-ink-1);
}

.dict-row__builtin {
  flex: none;
  height: 20px;
  padding: 0 6px;
  border-radius: var(--r-xs);
  background: var(--c-fill);
  color: var(--c-ink-3);
  font-size: 11px;
  line-height: 20px;
}

.dict-row__arrow {
  flex: none;
  font-size: 18px;
  color: var(--c-ink-4);
}

.about {
  padding: 16px;
}

.about__name {
  display: block;
  font-size: 18px;
  font-weight: 600;
  color: var(--c-ink-1);
}

.about__row {
  display: flex;
  justify-content: space-between;
  gap: 12px;
  margin-top: 10px;
  font-size: 13px;
}

.about__label {
  color: var(--c-ink-3);
}

.about__value {
  color: var(--c-ink-2);
  text-align: right;
}

.sheet {
  padding: 18px 16px calc(14px + env(safe-area-inset-bottom));
}

.sheet__title {
  display: block;
  font-size: 17px;
  font-weight: 600;
  color: var(--c-ink-1);
}

.sheet__field {
  margin-top: 14px;
  padding: 0 12px;
  border-radius: var(--r-s);
  background: var(--c-fill);
}

.sheet__label {
  display: block;
  margin: 18px 2px 8px;
  font-size: 13px;
  color: var(--c-ink-3);
}

.swatches {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
}

.swatch {
  width: 32px;
  height: 32px;
  border-radius: var(--r-full);
  box-shadow: inset 0 0 0 1px var(--c-surface);
}

.swatch--on {
  box-shadow:
    0 0 0 2px var(--c-surface),
    0 0 0 4px var(--c-ink-1);
}

.sheet__hint {
  display: block;
  margin-top: 12px;
}

.sheet__actions {
  display: flex;
  gap: 10px;
  margin-top: 22px;
}

.sheet__actions .btn {
  flex: 1;
}

.sheet__delete {
  margin-top: 14px;
  padding: 10px 0;
  text-align: center;
  font-size: 14px;
  color: var(--c-danger);
}
</style>
