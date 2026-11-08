// 模块接入机制：每个业务模块的 index.ts 导出 { routes }，本文件通过
// import.meta.glob 自动收集 modules/ 下所有模块，布局与路由层只依赖
// 这里的汇总结果 —— 新增模块无需改动公共代码。
import type { RouteRecordRaw } from 'vue-router'

/** 模块入口约定：index.ts 导出 routes。 */
export interface ModuleManifest {
  routes: RouteRecordRaw[]
}

// eager 模式同步加载全部模块入口，模块数量有限，无性能问题。
const moduleGlob = import.meta.glob('@/modules/*/index.ts', { eager: true })

function collectManifests(): ModuleManifest[] {
  return Object.values(moduleGlob).map((mod) => mod as ModuleManifest)
}

/** 汇总所有模块路由，注册为布局的内容区子路由。 */
export function collectModuleRoutes(): RouteRecordRaw[] {
  return collectManifests().flatMap((mod) => mod.routes)
}
