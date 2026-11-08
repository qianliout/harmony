// 路由汇总：布局骨架 + 各模块路由自动收集 + 全局守卫。
import { createRouter, createWebHistory } from 'vue-router'

import AppLayout from '@/core/layout/index.vue'
import { HOME_PATH, setupRouterGuard } from '@/core/router/guard'
import { collectModuleRoutes } from '@/core/router/module'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: '/',
      component: AppLayout,
      children: collectModuleRoutes(),
    },
    // 未匹配路由兜底回到首页
    { path: '/:pathMatch(.*)*', redirect: HOME_PATH },
  ],
})

setupRouterGuard(router)

export default router
