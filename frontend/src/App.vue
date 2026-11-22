<script setup lang="ts">
// 应用根组件：承载全局样式；页面由 pages.json 声明，H5 与小程序共用。
// #ifdef H5
import { onPageNotFound } from '@dcloudio/uni-app'

// 地址栏里的未知路由回到总账，避免白屏。
onPageNotFound(() => {
  uni.reLaunch({ url: '/pages/web/index' })
})
// #endif
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
