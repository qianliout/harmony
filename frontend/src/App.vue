<script setup lang="ts">
// 应用根组件：承载全局样式与启动登录；页面由 pages.json 声明，H5 与小程序共用。
import { onLaunch } from '@dcloudio/uni-app'

import { getToken } from '@/api/http'
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
  if (!getToken()) {
    uni.reLaunch({ url: '/sub-auth/login' })
    return
  }
  auth.loadMe().catch(() => {
    // 401 已由请求层跳到登录页。
  })
  // #endif
})
</script>

<style lang="scss">
@import './styles/tokens.scss';
@import './styles/base.scss';

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
/* 桌面浏览器中按手机宽度居中：内容区固定 430 宽，两侧加深底色。 */
@media (min-width: 521px) {
  body {
    background: #ede6dc;
  }

  /* uni-app 根节点带 transform 后成为 fixed 元素的包含块，导航栏、tabBar 等随之限制在 430 宽内，无需单独处理。 */
  uni-app {
    position: relative;
    display: block;
    max-width: 430px;
    min-height: 100vh;
    margin: 0 auto;
    transform: translateZ(0);
    box-shadow: var(--shadow-float);
  }
}

/* 识别页与识别历史在宽屏下临时解除手机画布限制。 */
@media (min-width: 1024px) {
  html.layout-wide uni-app {
    max-width: 1280px;
  }
}
/* #endif */
</style>
