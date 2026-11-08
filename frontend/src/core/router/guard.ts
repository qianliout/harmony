// 全局路由守卫：页面标题跟随路由 meta.title。
import type { Router } from 'vue-router'

const APP_TITLE = '人情往来账本'

/** 首页路径，也是未匹配路由的兜底目标。 */
export const HOME_PATH = '/'

export function setupRouterGuard(router: Router): void {
  router.afterEach((to) => {
    const title = to.meta.title as string | undefined
    document.title = title ? `${title} - ${APP_TITLE}` : APP_TITLE
  })
}
