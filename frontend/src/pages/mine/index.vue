<script setup lang="ts">
import { onShow } from '@dcloudio/uni-app'
import { ref } from 'vue'

import { fetchPersons } from '@/api/person'
import { fetchRecords } from '@/api/record'
import AppNav from '@/components/AppNav.vue'
import ListCell from '@/components/ListCell.vue'
import PersonAvatar from '@/components/PersonAvatar.vue'
import { themeVars } from '@/styles/theme'
import { go, toast } from '@/utils/nav'

const VERSION = '0.1.0'

const recordTotal = ref(0)
const personTotal = ref(0)
const loading = ref(false)

function formatCount(value: number): string {
  return value.toLocaleString('en-US')
}

async function loadCounts(): Promise<void> {
  if (loading.value) return
  loading.value = true
  try {
    const [records, persons] = await Promise.all([
      fetchRecords({ pageSize: 1 }),
      fetchPersons({ pageSize: 1 }),
    ])
    recordTotal.value = records.total
    personTotal.value = persons.total
  } catch (err) {
    toast(err instanceof Error ? err.message : '加载失败')
  } finally {
    loading.value = false
  }
}

function showAbout(): void {
  uni.showModal({
    title: '人情往来账本',
    content: `版本 ${VERSION}\n记下每一份人情往来，数据保存在自己的服务里。`,
    showCancel: false,
    confirmText: '知道了',
  })
}

onShow(loadCounts)
</script>

<template>
  <wd-config-provider :theme-vars="themeVars">
    <view class="screen screen--tab">
      <AppNav :back="false" title="" />

      <view class="page">
        <view class="me">
          <PersonAvatar name="我" :id="0" :size="56" />
          <view class="me__main">
            <text class="me__name">本机用户</text>
            <text class="me__stat">
              记账 {{ formatCount(recordTotal) }} 笔 · 亲友 {{ formatCount(personTotal) }} 人
            </text>
          </view>
        </view>

        <view class="card group">
          <ListCell title="全部记录" @click="go('/sub-record/list')" />
          <ListCell title="统计" @click="go('/sub-misc/stats')" />
        </view>

        <view class="card group">
          <ListCell title="合并礼簿" @click="go('/sub-book/merge')" />
          <ListCell title="已合并的礼簿" @click="go('/sub-book/merged')" />
          <ListCell title="导入导出" @click="go('/sub-misc/transfer')" />
        </view>

        <view class="card group">
          <ListCell title="偏好设置" @click="go('/sub-misc/settings')" />
          <ListCell title="关于" @click="showAbout" />
        </view>

        <text class="version">版本 {{ VERSION }}</text>
      </view>
    </view>
  </wd-config-provider>
</template>

<style lang="scss" scoped>
.me {
  display: flex;
  align-items: center;
  gap: 14px;
  padding: 6px 2px 18px;
}

.me__main {
  flex: 1;
  min-width: 0;
}

.me__name {
  display: block;
  font-size: 20px;
  line-height: 28px;
  font-weight: 600;
  color: var(--c-ink-1);
}

.me__stat {
  display: block;
  margin-top: 3px;
  font-size: 13px;
  color: var(--c-ink-3);
}

.group {
  margin-top: 12px;
  overflow: hidden;
}

.version {
  display: block;
  margin-top: 28px;
  text-align: center;
  font-size: 12px;
  color: var(--c-ink-4);
}
</style>
