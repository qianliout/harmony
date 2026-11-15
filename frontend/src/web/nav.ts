/** H5 桌面导航：一级页面 redirectTo，详情页 navigateTo，保持浏览器后退可用。 */
export function goWeb(url: string, mode: 'redirect' | 'push' = 'redirect'): void {
  if (mode === 'push') {
    uni.navigateTo({ url })
    return
  }
  uni.redirectTo({
    url,
    fail: () => {
      uni.reLaunch({ url })
    },
  })
}

export function currentWebPath(): string {
  const pages = getCurrentPages()
  const current = pages[pages.length - 1]
  return current ? `/${current.route}` : ''
}

export function backWeb(delta = 1): void {
  const pages = getCurrentPages()
  if (pages.length > delta) {
    uni.navigateBack({ delta })
    return
  }
  goWeb('/pages/web/index')
}
