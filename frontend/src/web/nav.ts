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

interface WebPageInstance {
  options?: Record<string, string>
  $page?: { options?: Record<string, string> }
}

/** 读取当前详情页的 id：H5 页面 query 挂在页面实例 options 上，直接刷新时从地址栏兜底。 */
export function readPageId(): number {
  const stack = getCurrentPages()
  const current = stack[stack.length - 1] as unknown as WebPageInstance | undefined
  const options = current?.options ?? current?.$page?.options ?? {}
  if (options.id) return Number(options.id)
  const matched = `${window.location.search}${window.location.hash}`.match(/[?&]id=(\d+)/)
  return matched ? Number(matched[1]) : 0
}
