<script setup lang="ts">
// H5 桌面登录页：全屏左右分栏，不挂 WebShell。
// 左边品牌、右边表单；支持小程序扫码与访问令牌，本机开发模式直接提示免登录。
import { computed, onMounted, onUnmounted, ref } from 'vue'

import { createWebTicket, pollWebTicket, type WebTicket } from '@/api/auth'
import { clearToken } from '@/api/http'
import { useAuthStore } from '@/stores/auth'
import Segmented from '@/web/components/Segmented.vue'
import WebIcon from '@/web/components/WebIcon.vue'

interface PageInstanceLike {
  options?: Record<string, string | undefined>
  $page?: { options?: Record<string, string | undefined> }
}

const MODES = [
  { value: 'qr', label: '小程序扫码' },
  { value: 'token', label: '访问令牌' },
]
const HOME_URL = '/pages/web/index'

const auth = useAuthStore()

const mode = ref<'qr' | 'token'>('qr')
const ticket = ref<WebTicket | null>(null)
const status = ref('waiting')
const error = ref('')
const loading = ref(false)
const tokenInput = ref('')
const remaining = ref(0)
const redirect = ref(HOME_URL)

let pollTimer: ReturnType<typeof setInterval> | null = null
let countdownTimer: ReturnType<typeof setInterval> | null = null

const devMode = computed(() => auth.client === 'dev')

const statusText = computed(() => {
  if (status.value === 'scanned') return '已扫码，请在手机上确认登录'
  if (status.value === 'confirmed') return '登录成功，正在进入账本…'
  if (status.value === 'canceled') return '已取消登录，可重新扫码'
  if (status.value === 'expired') return '二维码已过期，请刷新'
  return '打开微信，扫一扫小程序码'
})

const expireText = computed(() => {
  if (remaining.value <= 0) return ''
  const minutes = Math.floor(remaining.value / 60)
  const seconds = remaining.value % 60
  return `${minutes}:${String(seconds).padStart(2, '0')} 后过期`
})

const refreshLabel = computed(() =>
  status.value === 'expired' || status.value === 'canceled' ? '刷新二维码' : '换一张二维码',
)

/** 读取 redirect 查询参数；优先页面 options，H5 刷新时退回 hash 查询串。 */
function readPageQuery(): Record<string, string> {
  const result: Record<string, string> = {}
  const pages = getCurrentPages()
  const current = pages[pages.length - 1] as unknown as PageInstanceLike | undefined
  const options = current?.options ?? current?.$page?.options ?? {}
  Object.entries(options).forEach(([key, value]) => {
    if (value !== undefined && value !== null) result[key] = String(value)
  })
  if (Object.keys(result).length || typeof window === 'undefined') return result
  const hash = window.location.hash
  const queryIndex = hash.indexOf('?')
  const search = queryIndex >= 0 ? hash.slice(queryIndex + 1) : window.location.search
  new URLSearchParams(search).forEach((value, key) => {
    result[key] = value
  })
  return result
}

function safeRedirect(): string {
  const value = redirect.value
  if (!value.startsWith('/') || value.startsWith('/sub-web/login')) return HOME_URL
  return value
}

function stopTimers(): void {
  if (pollTimer) clearInterval(pollTimer)
  if (countdownTimer) clearInterval(countdownTimer)
  pollTimer = null
  countdownTimer = null
}

function enter(): void {
  uni.reLaunch({ url: safeRedirect() })
}

async function finishLogin(token: string): Promise<void> {
  loading.value = true
  error.value = ''
  try {
    await auth.loginWithToken(token)
    stopTimers()
    uni.reLaunch({ url: safeRedirect() })
  } catch (err) {
    clearToken()
    error.value = err instanceof Error ? err.message : '登录状态确认失败，请重试'
    loading.value = false
  }
}

function startPolling(): void {
  const current = ticket.value
  if (!current) return
  stopTimers()
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
        await finishLogin(result.token)
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
  ticket.value = null
  remaining.value = 0
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

async function submitToken(): Promise<void> {
  const value = tokenInput.value.trim()
  if (!value) {
    error.value = '请粘贴访问令牌'
    return
  }
  await finishLogin(value)
}

function setMode(value: string): void {
  mode.value = value === 'token' ? 'token' : 'qr'
  error.value = ''
  if (mode.value === 'token') {
    stopTimers()
    return
  }
  if (!ticket.value) {
    void refresh()
    return
  }
  startPolling()
}

onMounted(async () => {
  const query = readPageQuery()
  if (query.redirect) {
    let value = String(query.redirect)
    try {
      value = decodeURIComponent(value)
      value = decodeURIComponent(value)
    } catch {
      // 解码失败时保留原值，交给 safeRedirect 兜底。
    }
    if (value.startsWith('/')) redirect.value = value
  }

  // 已登录（或本机开发免登录）时不再要求扫码。
  try {
    await auth.loadMe()
  } catch {
    clearToken()
  }
  if (auth.isLoggedIn && !devMode.value) {
    uni.reLaunch({ url: safeRedirect() })
    return
  }
  await refresh()
})

onUnmounted(stopTimers)
</script>

<template>
  <div class="web-login">
    <div class="web-login__art">
      <span class="login-stamp">礼</span>
      <div class="login-art__copy">
        <h1>人情往来，<br />一本账记清。</h1>
        <p>把家里的纸质礼簿搬进电脑：按人、按事项聚合，谁送过、回没回，一眼看清。</p>
      </div>
      <span class="login-art__foot">网页版 · 与小程序共用同一份账</span>
      <span class="login-art__big">簿</span>
    </div>

    <div class="web-login__form">
      <div class="web-login__box">
        <div v-if="devMode" class="web-devbar">
          <WebIcon name="info" :size="16" />
          本机开发模式已免登录
          <a href="#" @click.prevent="enter">直接进入</a>
        </div>

        <div>
          <h2 class="login-title">登录</h2>
          <p class="login-sub">在手机小程序里确认后，电脑上继续记账。</p>
        </div>

        <Segmented :model-value="mode" :options="MODES" @update:model-value="setMode" />

        <template v-if="mode === 'qr'">
          <div class="login-qr">
            <img v-if="ticket?.qrImage" class="login-qr__img" :src="ticket.qrImage" alt="登录二维码" />
            <div v-else class="login-qr__empty">
              <span class="login-qr__title">{{ loading ? '正在生成二维码…' : '二维码暂不可用' }}</span>
              <span class="web-sm web-muted">也可切到访问令牌登录，或在设置里重新生成。</span>
            </div>
          </div>

          <div class="login-status" :class="{ 'login-status--muted': status === 'expired' || status === 'canceled' }">
            {{ statusText }}
          </div>
          <div v-if="expireText" class="login-countdown web-num">{{ expireText }}</div>

          <button class="web-btn login-action" type="button" :disabled="loading" @click="refresh">
            {{ refreshLabel }}
          </button>

          <p class="login-hint">
            二维码 2 分钟后过期；过期后点一下刷新。小程序上线前，网页只能用访问令牌登录。
          </p>
        </template>

        <template v-else>
          <div class="web-field">
            <label for="login-token">访问令牌</label>
            <textarea
              id="login-token"
              v-model="tokenInput"
              class="web-textarea"
              rows="3"
              placeholder="粘贴 harmony token issue 签发的令牌"
              spellcheck="false"
            />
            <span class="web-field__hint">令牌在服务器上用命令签发，默认 14 天有效，使用中会自动续期。</span>
          </div>

          <button class="web-btn web-btn--primary login-action" type="button" :disabled="loading" @click="submitToken">
            {{ loading ? '正在登录…' : '登录' }}
          </button>

          <p class="login-hint">令牌无效或过期时，错误信息会显示在这里，可重新签发后再试。</p>
        </template>

        <div v-if="error" class="web-alert web-alert--err">{{ error }}</div>

        <p class="login-foot">登录成功回到刚才访问的页面；没有来源时进入工作台。</p>
      </div>
    </div>
  </div>
</template>

<style lang="scss" scoped>
.login-stamp {
  display: grid;
  place-items: center;
  width: 48px;
  height: 48px;
  border: 1px solid var(--c-on-primary-2);
  border-radius: 10px;
  font: 600 26px/1 $font-serif;
}

.login-art__copy {
  position: relative;
  z-index: 1;
  max-width: 520px;
}

.login-art__copy h1 {
  white-space: nowrap;
}

.login-art__foot {
  position: relative;
  z-index: 1;
  color: var(--c-on-primary-2);
  font-size: 13px;
}

.login-art__big {
  position: absolute;
  right: 40px;
  bottom: 24px;
  color: var(--c-on-primary);
  font: 600 220px/1 $font-serif;
  opacity: 0.08;
  pointer-events: none;
  user-select: none;
}

.login-title {
  margin: 0;
  font-size: 22px;
  font-weight: 600;
}

.login-sub {
  margin: 6px 0 0;
  color: var(--c-ink-3);
  font-size: 13px;
}

.login-qr {
  display: grid;
  place-items: center;
  width: 260px;
  height: 260px;
  margin: 0 auto;
  overflow: hidden;
  border-radius: var(--r-m);
  background: var(--c-fill);
}

.login-qr__img {
  width: 100%;
  height: 100%;
  object-fit: contain;
}

.login-qr__empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 6px;
  padding: var(--sp-4);
  text-align: center;
}

.login-qr__title {
  font-size: 14px;
  font-weight: 500;
}

.login-status {
  margin-top: var(--sp-1);
  font-size: 15px;
  text-align: center;
}

.login-status--muted {
  color: var(--c-ink-3);
}

.login-countdown {
  color: var(--c-ink-3);
  font-size: 12px;
  text-align: center;
}

.login-action {
  width: 100%;
  height: 40px;
}

.login-hint,
.login-foot {
  margin: 0;
  color: var(--c-ink-3);
  font-size: 12px;
  line-height: 18px;
}

.login-foot {
  padding-top: var(--sp-3);
  border-top: 1px solid var(--c-divider);
}
</style>
