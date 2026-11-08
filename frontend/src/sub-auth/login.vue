<script setup lang="ts">
import { onLoad, onUnload } from '@dcloudio/uni-app'
import { computed, ref } from 'vue'

import { clearToken, getToken, setToken } from '@/api/http'
import { createWebTicket, pollWebTicket, type WebTicket } from '@/api/auth'
import { useAuthStore } from '@/stores/auth'
import { toast } from '@/utils/nav'

const auth = useAuthStore()
const ticket = ref<WebTicket | null>(null)
const status = ref('waiting')
const error = ref('')
const loading = ref(false)
const tokenInput = ref('')
const showTokenInput = ref(false)
const remaining = ref(0)
let pollTimer: ReturnType<typeof setInterval> | null = null
let countdownTimer: ReturnType<typeof setInterval> | null = null
let redirect = '/pages/index/index'

const statusText = computed(() => {
  if (status.value === 'scanned') return '已扫码，请在手机上确认'
  if (status.value === 'confirmed') return '登录成功，正在进入账本…'
  if (status.value === 'canceled') return '已取消登录'
  if (status.value === 'expired') return '二维码已过期，请刷新'
  return '打开微信，扫一扫小程序码'
})

const expireText = computed(() => {
  if (remaining.value <= 0) return ''
  const minutes = Math.floor(remaining.value / 60)
  const seconds = remaining.value % 60
  return `${minutes}:${String(seconds).padStart(2, '0')} 后过期`
})

function stopTimers(): void {
  if (pollTimer) clearInterval(pollTimer)
  if (countdownTimer) clearInterval(countdownTimer)
  pollTimer = null
  countdownTimer = null
}

function finishLogin(token: string): void {
  setToken(token)
  auth
    .loadMe()
    .then(() => {
      uni.reLaunch({ url: redirect })
    })
    .catch(() => {
      clearToken()
      error.value = '登录状态确认失败，请重试'
    })
}

function startPolling(): void {
  if (!ticket.value) return
  stopTimers()
  const current = ticket.value
  remaining.value = Math.max(0, Math.floor((current.expiresAt - Date.now()) / 1000))
  countdownTimer = setInterval(() => {
    remaining.value = Math.max(0, Math.floor((current.expiresAt - Date.now()) / 1000))
    if (remaining.value === 0) {
      status.value = 'expired'
      stopTimers()
    }
  }, 1000)
  pollTimer = setInterval(async () => {
    try {
      const result = await pollWebTicket(current.ticket)
      status.value = result.status
      if (result.status === 'confirmed' && result.token) {
        stopTimers()
        finishLogin(result.token)
      }
      if (result.status === 'canceled' || result.status === 'expired') {
        stopTimers()
      }
    } catch (err) {
      stopTimers()
      error.value = err instanceof Error ? err.message : '获取登录状态失败'
    }
  }, 2000)
}

async function refresh(): Promise<void> {
  stopTimers()
  error.value = ''
  status.value = 'waiting'
  loading.value = true
  try {
    ticket.value = await createWebTicket()
    startPolling()
  } catch (err) {
    error.value = err instanceof Error ? err.message : '生成登录二维码失败'
  } finally {
    loading.value = false
  }
}

async function useToken(): Promise<void> {
  const value = tokenInput.value.trim()
  if (!value) {
    toast('粘贴访问令牌')
    return
  }
  loading.value = true
  error.value = ''
  try {
    await auth.loginWithToken(value)
    uni.reLaunch({ url: redirect })
  } catch (err) {
    clearToken()
    error.value = err instanceof Error ? err.message : '令牌无效'
    toast(error.value)
  } finally {
    loading.value = false
  }
}

onLoad(async (query) => {
  if (query?.redirect) {
    let value = String(query.redirect)
    try {
      value = decodeURIComponent(value)
      value = decodeURIComponent(value)
    } catch {
      // 保留原值，交给路由兜底。
    }
    if (value.startsWith('/')) {
      redirect = value
    }
  }
  if (getToken()) {
    try {
      await auth.loadMe()
      if (auth.isLoggedIn) {
        uni.reLaunch({ url: redirect })
        return
      }
    } catch {
      clearToken()
    }
  }
  await refresh()
})

onUnload(stopTimers)
</script>

<template>
  <view class="screen login">
    <view class="login__card">
      <view class="login__seal">礼</view>
      <text class="login__title">人情往来账本</text>
      <text class="login__sub">用微信扫一扫，在电脑上继续记账</text>

      <view class="login__qr">
        <image v-if="ticket?.qrImage" class="login__image" :src="ticket.qrImage" mode="aspectFit" />
        <view v-else class="login__fallback">
          <text class="login__fallback-title">小程序码暂不可用</text>
          <text class="login__fallback-hint">二维码生成中，或在小程序「我的 → 扫码登录网页版」继续</text>
        </view>
      </view>

      <text class="login__status" :class="{ 'login__status--muted': status === 'expired' || status === 'canceled' }">
        {{ statusText }}
      </text>
      <text v-if="expireText" class="login__countdown">{{ expireText }}</text>
      <text v-if="error" class="login__error">{{ error }}</text>

      <view class="login__actions">
        <view class="btn btn--secondary login__btn" :class="{ 'btn--disabled': loading }" @click="refresh">
          {{ status === 'expired' || status === 'canceled' ? '刷新二维码' : '换一张' }}
        </view>
      </view>

      <view class="login__token">
        <text class="login__token-toggle" @click="showTokenInput = !showTokenInput">
          {{ showTokenInput ? '收起访问令牌登录' : '使用访问令牌登录' }}
        </text>
        <view v-if="showTokenInput" class="login__token-form">
          <input v-model="tokenInput" class="input-box login__token-input" placeholder="粘贴命令行签发的令牌" />
          <view class="btn btn--primary login__token-btn" :class="{ 'btn--disabled': loading }" @click="useToken">
            登录
          </view>
        </view>
      </view>
    </view>
  </view>
</template>

<style lang="scss" scoped>
.login {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 32px var(--gutter);
}

.login__card {
  display: flex;
  flex-direction: column;
  align-items: center;
  width: 100%;
  max-width: 360px;
  padding: 32px 24px 24px;
  border-radius: var(--r-l);
  background: var(--c-surface);
  box-shadow: var(--shadow-card);
}

.login__seal {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 64px;
  height: 64px;
  border: 1px solid var(--c-primary);
  border-radius: 16px;
  color: var(--c-primary);
  background: var(--c-primary-light);
  font-family: $font-serif;
  font-size: 32px;
  position: relative;
}

.login__seal::after {
  content: '';
  position: absolute;
  inset: 2px;
  border: 1px solid currentColor;
  border-radius: inherit;
  opacity: 0.25;
}

.login__title {
  margin-top: 16px;
  font-family: $font-serif;
  font-size: 22px;
  line-height: 30px;
  font-weight: 600;
}

.login__sub {
  margin-top: 6px;
  font-size: 13px;
  color: var(--c-ink-3);
  text-align: center;
}

.login__qr {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 240px;
  height: 240px;
  margin-top: 20px;
  border-radius: var(--r-m);
  background: var(--c-fill);
  overflow: hidden;
}

.login__image {
  width: 100%;
  height: 100%;
}

.login__fallback {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
  padding: 16px;
  text-align: center;
}

.login__fallback-title {
  font-size: 14px;
  font-weight: 500;
}

.login__fallback-code {
  font-family: ui-monospace, Menlo, monospace;
  font-size: 12px;
  color: var(--c-primary);
  word-break: break-all;
}

.login__fallback-hint {
  font-size: 12px;
  color: var(--c-ink-3);
}

.login__status {
  margin-top: 18px;
  font-size: 15px;
  color: var(--c-ink-1);
}

.login__status--muted {
  color: var(--c-ink-3);
}

.login__countdown {
  margin-top: 4px;
  font-size: 12px;
  color: var(--c-ink-3);
}

.login__error {
  margin-top: 8px;
  font-size: 13px;
  color: var(--c-danger);
  text-align: center;
}

.login__actions {
  display: flex;
  gap: 10px;
  width: 100%;
  margin-top: 18px;
}

.login__btn {
  flex: 1;
  height: 44px;
}

.login__token {
  width: 100%;
  margin-top: 16px;
  border-top: 1px solid var(--c-divider);
  padding-top: 12px;
}

.login__token-toggle {
  display: block;
  text-align: center;
  font-size: 12px;
  color: var(--c-ink-3);
}

.login__token-form {
  display: flex;
  gap: 8px;
  margin-top: 10px;
}

.login__token-input {
  flex: 1;
  min-width: 0;
}

.login__token-btn {
  height: 44px;
  padding: 0 20px;
  font-size: 14px;
}
</style>
