<script setup lang="ts">
// 页面容器：统一页面标题与内容区间距，业务页面直接组合。
defineOptions({ name: 'PageContainer' })

defineProps<{ title: string; caption?: string }>()
</script>

<template>
  <section class="page">
    <header class="page__header">
      <h2 class="page__title">{{ title }}</h2>
      <p v-if="caption" class="page__caption">{{ caption }}</p>
    </header>
    <div class="page__body">
      <slot />
    </div>
  </section>
</template>

<style scoped>
.page {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.page__title {
  margin: 0;
  font-size: 20px;
  line-height: 28px;
  font-weight: 600;
  color: var(--el-text-color-primary);
}

.page__caption {
  margin: 4px 0 0;
  font-size: 13px;
  line-height: 20px;
  color: var(--el-text-color-secondary);
}
</style>
