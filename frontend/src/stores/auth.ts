import { defineStore } from 'pinia'
import { computed, ref } from 'vue'

import { clearToken, getToken, setToken, setUnauthorizedHandler } from '@/api/http'
import {
  fetchMe,
  logout as logoutApi,
  wxLogin as wxLoginApi,
  type AuthUser,
  type LoginResult,
} from '@/api/auth'

const LOGIN_PATH = '/sub-auth/login'

export const useAuthStore = defineStore('auth', () => {
  const user = ref<AuthUser | null>(null)
  const client = ref('')
  const ready = ref(false)
  const isAdmin = computed(() => Boolean(user.value?.isAdmin))
  const isLoggedIn = computed(() => Boolean(user.value))

  function applyLogin(result: LoginResult): void {
    setToken(result.token)
    user.value = result.user
    client.value = ''
  }

  async function loadMe(): Promise<void> {
    if (!getToken()) {
      user.value = null
      ready.value = true
      return
    }
    const me = await fetchMe()
    user.value = me.user
    client.value = me.client
    ready.value = true
  }

  async function wxLogin(): Promise<void> {
    const login = await new Promise<UniApp.LoginRes>((resolve, reject) => {
      uni.login({
        provider: 'weixin',
        success: resolve,
        fail: reject,
      })
    })
    if (!login.code) {
      throw new Error('微信登录失败')
    }
    applyLogin(await wxLoginApi(login.code))
    await loadMe()
  }

  async function logout(): Promise<void> {
    try {
      await logoutApi()
    } catch {
      // 令牌已失效时也继续清理本地状态。
    }
    clearToken()
    user.value = null
    client.value = ''
    uni.reLaunch({ url: LOGIN_PATH })
  }

  function configureHttp(): void {
    setUnauthorizedHandler(async () => {
      // #ifdef MP-WEIXIN
      try {
        await wxLogin()
        return true
      } catch {
        return false
      }
      // #endif
      // #ifdef H5
      const pages = getCurrentPages()
      const current = pages[pages.length - 1]
      const redirect = current ? `/${current.route}` : '/pages/index/index'
      if (redirect.startsWith(LOGIN_PATH)) {
        return false
      }
      uni.reLaunch({ url: `${LOGIN_PATH}?redirect=${encodeURIComponent(redirect)}` })
      return false
      // #endif
    })
  }

  function loginWithToken(token: string): Promise<void> {
    setToken(token)
    return loadMe()
  }

  return {
    user,
    client,
    ready,
    isAdmin,
    isLoggedIn,
    applyLogin,
    loadMe,
    wxLogin,
    logout,
    configureHttp,
    loginWithToken,
  }
})
