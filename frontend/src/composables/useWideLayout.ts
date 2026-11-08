// 识别页与识别历史在大屏上解除 430px 手机画布，改用宽屏布局。
// 整个文件只在 H5 编译；小程序产物中不出现 layout-wide 与 matchMedia。
// #ifdef H5
import { onHide, onShow, onUnload } from '@dcloudio/uni-app'
import { ref } from 'vue'

/** 宽屏阈值：960 宽放不下核对表七列，统一用 1024。 */
export const WIDE_MIN_WIDTH = 1024

/**
 * 页面显示时给 <html> 加 layout-wide（App.vue 据此把 uni-app 放宽到 1280），
 * 同时监听 1024px 断点返回 isWide，供页面切换单栏/双栏 DOM。
 *
 * onHide 与 onUnload 都要清理：uni-app H5 的页面栈会保留上一页，跳到记录详情时只触发 onHide，
 * 不清理会让其余手机画布页面继续保持宽屏。
 */
export function useWideLayout() {
  const isWide = ref(false)
  let media: MediaQueryList | null = null

  function sync(): void {
    if (media) {
      isWide.value = media.matches
    }
  }

  function leave(): void {
    if (typeof document !== 'undefined') {
      document.documentElement.classList.remove('layout-wide')
    }
    if (media) {
      media.removeEventListener('change', sync)
      media = null
    }
    isWide.value = false
  }

  onShow(() => {
    if (typeof window === 'undefined') return
    document.documentElement.classList.add('layout-wide')
    if (media) {
      media.removeEventListener('change', sync)
    }
    media = window.matchMedia(`(min-width: ${WIDE_MIN_WIDTH}px)`)
    sync()
    media.addEventListener('change', sync)
  })

  onHide(leave)
  onUnload(leave)

  return { isWide }
}
// #endif
