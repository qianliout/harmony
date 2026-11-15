<script setup lang="ts">
// 应用根组件：承载全局样式与启动登录；页面由 pages.json 声明，H5 与小程序共用。
import { onLaunch } from '@dcloudio/uni-app'

import { useAuthStore } from '@/stores/auth'

onLaunch(() => {
  const auth = useAuthStore()
  auth.configureHttp()

  // #ifdef MP-WEIXIN
  auth.wxLogin().catch(() => {
    uni.showToast({ title: '登录失败，下拉重试', icon: 'none' })
  })
  // #endif

  // #ifdef H5
  // 没有令牌也先请求：本机开发免登录时后端直接放行，否则 401 由请求层跳到登录页。
  auth.loadMe().catch(() => {
    // 401 已由请求层跳到登录页。
  })
  // #endif
})
</script>

<style lang="scss">
@import './styles/tokens.scss';
@import './styles/base.scss';
/* #ifdef H5 */
@import './web/styles/web.scss';
/* #endif */

page {
  min-height: 100%;
  background: var(--c-bg);
  color: var(--c-ink-1);
  font-family: $font-sans;
  font-size: 15px;
  line-height: 22px;
  -webkit-font-smoothing: antialiased;
}

/* #ifdef H5 */
/* H5 使用桌面外壳，根节点铺满视口；小程序仍按 430px 手机画布设计。 */
uni-app {
  display: block;
  width: 100%;
  min-height: 100vh;
}
/* #endif */
</style>
