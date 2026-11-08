<script setup lang="ts">
import { onShow } from '@dcloudio/uni-app'
import { computed, ref } from 'vue'

import { cancelWebTicket, confirmWebTicket, deleteSession, fetchSessions, scanWebTicket, type AuthSession } from '@/api/auth'
import { fetchPersons } from '@/api/person'
import { fetchRecords } from '@/api/record'
import AppNav from '@/components/AppNav.vue'
import ListCell from '@/components/ListCell.vue'
import PersonAvatar from '@/components/PersonAvatar.vue'
import { useAuthStore } from '@/stores/auth'
import { themeVars } from '@/styles/theme'
import { formatTimestamp } from '@/utils/format'
import { confirmModal, go, toast } from '@/utils/nav'

const VERSION = '0.1.0'

const recordTotal = ref(0)
const personTotal = ref(0)
const loading = ref(false)
const auth = useAuthStore()
const sessions = ref<AuthSession[]>([])
const deletingId = ref(0)

const displayName = computed(() => auth.user?.nickname || '本机用户')
const isAdmin = computed(() => Boolean(auth.user?.isAdmin))

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

async function loadAuth(): Promise<void> {
  try {
    await auth.loadMe()
    sessions.value = await fetchSessions()
  } catch {
    sessions.value = []
  }
}

function clientLabel(client: string): string {
  return client === 'mp' ? '微信小程序' : '网页'
}

async function revoke(session: AuthSession): Promise<void> {
  if (deletingId.value) return
  const ok = await confirmModal(`让「${session.device || clientLabel(session.client)}」下线？`)
  if (!ok) return
  deletingId.value = session.id
  try {
    await deleteSession(session.id)
    sessions.value = sessions.value.filter((item) => item.id !== session.id)
    toast('已下线', 'success')
  } catch (err) {
    toast(err instanceof Error ? err.message : '操作失败')
  } finally {
    deletingId.value = 0
  }
}

async function logout(): Promise<void> {
  const ok = await confirmModal('退出当前登录？')
  if (ok) {
    await auth.logout()
  }
}

function scanWebLogin(): void {
  uni.scanCode({
    success: async (result) => {
      const raw = String(result.result || '')
      const ticket = raw.startsWith('harmony-login:') ? raw.slice('harmony-login:'.length) : ''
      if (!ticket) {
        toast('这不是登录二维码')
        return
      }
      try {
        await scanWebTicket(ticket)
        const ok = await confirmModal('在电脑浏览器上登录人情往来账本？')
        if (ok) {
          await confirmWebTicket(ticket)
          toast('已确认登录', 'success')
        } else {
          await cancelWebTicket(ticket)
        }
      } catch (err) {
        toast(err instanceof Error ? err.message : '扫码登录失败')
      }
    },
  })
}

onShow(() => {
  loadCounts()
  loadAuth()
})
</script>

<template>
  <wd-config-provider :theme-vars="themeVars">
    <view class="screen screen--tab">
      <AppNav :back="false" title="" />

      <view class="page">
        <view class="me">
          <PersonAvatar name="我" :id="0" :size="56" />
          <view class="me__main">
            <view class="me__title">
              <text class="me__name">{{ displayName }}</text>
              <text v-if="isAdmin" class="me__badge">管理员</text>
            </view>
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

        <view v-if="sessions.length" class="card session">
          <text class="session__title">登录设备</text>
          <view v-for="session in sessions" :key="session.id" class="session__row">
            <view class="session__main">
              <text class="session__device">{{ session.device || clientLabel(session.client) }}</text>
              <text class="session__meta">{{ clientLabel(session.client) }} · 最近使用 {{ formatTimestamp(session.lastUsedAt) }}</text>
            </view>
            <text class="session__revoke" @click="revoke(session)">
              {{ deletingId === session.id ? '处理中…' : '下线' }}
            </text>
          </view>
        </view>

        <!-- #ifdef MP-WEIXIN -->
        <view class="card group">
          <ListCell title="扫码登录网页版" @click="scanWebLogin" />
        </view>
        <!-- #endif -->

        <view class="card group">
          <ListCell title="退出登录" :arrow="false" danger @click="logout" />
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

.me__title {
  display: flex;
  align-items: center;
  gap: 8px;
}

.me__badge {
  display: inline-flex;
  align-items: center;
  height: 20px;
  padding: 0 7px;
  border-radius: var(--r-xs);
  background: var(--c-primary-light);
  color: var(--c-primary);
  font-size: 11px;
  font-weight: 500;
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

.session {
  margin-top: 12px;
  padding: 14px 16px 6px;
}

.session__title {
  font-size: 13px;
  color: var(--c-ink-3);
}

.session__row {
  display: flex;
  align-items: center;
  gap: 10px;
  min-height: 52px;
  box-shadow: inset 0 1px 0 var(--c-divider);
}

.session__main {
  flex: 1;
  min-width: 0;
}

.session__device {
  display: block;
  font-size: 14px;
  color: var(--c-ink-1);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.session__meta {
  display: block;
  margin-top: 2px;
  font-size: 11px;
  color: var(--c-ink-3);
}

.session__revoke {
  flex: none;
  font-size: 12px;
  color: var(--c-danger);
}

.version {
  display: block;
  margin-top: 28px;
  text-align: center;
  font-size: 12px;
  color: var(--c-ink-4);
}
</style>
