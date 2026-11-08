<script setup lang="ts">
// 主布局骨架：按手机视口宽度居中的单列页面，桌面浏览器中显示为 480px 宽的应用画布。
defineOptions({ name: 'AppLayout' })
</script>

<template>
  <div class="app-shell">
    <header class="app-shell__header">
      <p class="app-shell__eyebrow">HARMONY</p>
      <h1 class="app-shell__title">人情往来账本</h1>
      <p class="app-shell__caption">本机 Web 端</p>
    </header>
    <main class="app-shell__main">
      <router-view />
    </main>
  </div>
</template>

<style scoped>
.app-shell {
  width: 100%;
  max-width: 480px;
  min-height: 100vh;
  margin: 0 auto;
  background: var(--el-bg-color);
}

.app-shell__header {
  padding: 28px 20px 20px;
  border-bottom: 1px solid var(--el-border-color-light);
}

.app-shell__eyebrow {
  margin: 0 0 6px;
  font-size: 11px;
  line-height: 16px;
  font-weight: 500;
  letter-spacing: 0.12em;
  color: var(--el-text-color-secondary);
}

.app-shell__title {
  margin: 0;
  font-size: 24px;
  line-height: 32px;
  font-weight: 600;
  color: var(--el-text-color-primary);
}

.app-shell__caption {
  margin: 6px 0 0;
  font-size: 13px;
  line-height: 20px;
  color: var(--el-text-color-secondary);
}

.app-shell__main {
  padding: 24px 20px 40px;
}

@media (min-width: 520px) {
  .app-shell {
    border-inline: 1px solid var(--el-border-color-light);
  }
}
</style>
