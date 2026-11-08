<script setup lang="ts">
import { onLoad } from '@dcloudio/uni-app'
import { ref } from 'vue'

import { cancelWebTicket, confirmWebTicket, scanWebTicket } from '@/api/auth'
import { useAuthStore } from '@/stores/auth'
import { toast } from '@/utils/nav'

const auth = useAuthStore()
const ticket = ref('')
const state = ref<'loading' | 'ready' | 'expired' | 'working' | 'done'>('loading')
const message = ref('')

async function scan(): Promise<void> {
  if (!ticket.value) {
    state.value = 'expired'
    message.value = '二维码参数缺失，请在电脑上重新生成'
    return
  }
  try {
    if (!auth.isLoggedIn) {
      await auth.wxLogin()
    }
    await scanWebTicket(ticket.value)
    state.value = 'ready'
  } catch (err) {
    state.value = 'expired'
    message.value = err instanceof Error ? err.message : '二维码已过期'
  }
}

async function confirm(): Promise<void> {
  state.value = 'working'
  try {
    await confirmWebTicket(ticket.value)
    state.value = 'done'
    toast('已确认登录', 'success')
    setTimeout(() => {
      uni.reLaunch({ url: '/pages/index/index' })
    }, 800)
  } catch (err) {
    state.value = 'ready'
    toast(err instanceof Error ? err.message : '确认失败')
  }
}

async function cancel(): Promise<void> {
  try {
    await cancelWebTicket(ticket.value)
  } catch {
    // 已过期时也退出。
  }
  uni.reLaunch({ url: '/pages/index/index' })
}

onLoad((query) => {
  ticket.value = String(query?.scene ?? query?.ticket ?? '')
  scan()
})
</script>

<template>
  <view class="screen confirm">
    <view class="confirm__card">
      <view class="confirm__seal">屏</view>
      <text class="confirm__title">在电脑浏览器上登录</text>
      <text class="confirm__sub">人情往来账本</text>

      <template v-if="state === 'ready' || state === 'working'">
        <text class="confirm__text">确认是你在电脑上发起的登录吗？</text>
        <view class="confirm__actions">
          <view class="btn btn--secondary confirm__btn" @click="cancel">取消</view>
          <view
            class="btn btn--primary confirm__btn"
            :class="{ 'btn--disabled': state === 'working' }"
            @click="confirm"
          >
            {{ state === 'working' ? '确认中…' : '确认登录' }}
          </view>
        </view>
      </template>

      <template v-else-if="state === 'done'">
        <text class="confirm__text confirm__text--ok">已确认，网页即将自动登录</text>
      </template>

      <template v-else-if="state === 'expired'">
        <text class="confirm__text confirm__text--warn">{{ message || '二维码已过期，请在电脑上刷新' }}</text>
        <view class="btn btn--secondary confirm__btn" @click="cancel">返回首页</view>
      </template>

      <text v-else class="confirm__text">正在确认二维码…</text>
    </view>
  </view>
</template>

<style lang="scss" scoped>
.confirm {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 32px var(--gutter);
}

.confirm__card {
  display: flex;
  flex-direction: column;
  align-items: center;
  width: 100%;
  padding: 32px 20px 24px;
  border-radius: var(--r-l);
  background: var(--c-surface);
  box-shadow: var(--shadow-card);
}

.confirm__seal {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 64px;
  height: 64px;
  border: 1px solid var(--c-primary);
  border-radius: 16px;
  background: var(--c-primary-light);
  color: var(--c-primary);
  font-family: $font-serif;
  font-size: 30px;
}

.confirm__title {
  margin-top: 16px;
  font-family: $font-serif;
  font-size: 22px;
  font-weight: 600;
}

.confirm__sub {
  margin-top: 4px;
  font-size: 13px;
  color: var(--c-ink-3);
}

.confirm__text {
  margin-top: 20px;
  font-size: 14px;
  line-height: 22px;
  color: var(--c-ink-2);
  text-align: center;
}

.confirm__text--ok {
  color: var(--c-repaid);
}

.confirm__text--warn {
  color: var(--c-pending);
}

.confirm__actions {
  display: flex;
  gap: 10px;
  width: 100%;
  margin-top: 24px;
}

.confirm__btn {
  flex: 1;
  height: 44px;
}
</style>
