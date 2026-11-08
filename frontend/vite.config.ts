import uni from '@dcloudio/vite-plugin-uni'
import { defineConfig, loadEnv } from 'vite'

// H5 开发服务器把 /api 代理到本机后端（REST 前缀 /api/v1），目标地址可用 VITE_PROXY_TARGET 覆盖。
// 小程序不走代理，直连 VITE_MP_API_BASE，见 src/api/http.ts。
export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd())

  return {
    plugins: [uni()],
    server: {
      port: 5173,
      proxy: {
        '/api': {
          target: env.VITE_PROXY_TARGET || 'http://127.0.0.1:8080',
          changeOrigin: true,
        },
        '/uploads': {
          target: env.VITE_PROXY_TARGET || 'http://127.0.0.1:8080',
          changeOrigin: true,
        },
      },
    },
  }
})
