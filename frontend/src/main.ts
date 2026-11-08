// 应用入口：uni-app 要求导出 createApp，H5 与小程序共用；注册 Pinia。
import { createPinia } from 'pinia'
import { createSSRApp } from 'vue'

import App from './App.vue'

export function createApp() {
  const app = createSSRApp(App)
  app.use(createPinia())
  return { app }
}
