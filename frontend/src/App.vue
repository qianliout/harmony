<script setup lang="ts">
// 应用根组件：只承载全局样式；页面由 pages.json 声明，H5 与小程序共用。
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
/* #endif */
</style>
