export function go(url: string): void {
  uni.navigateTo({
    url,
    fail: () => {
      uni.switchTab({ url })
    },
  })
}

export function back(delta = 1): void {
  const pages = getCurrentPages()
  if (pages.length > delta) {
    uni.navigateBack({ delta })
    return
  }
  uni.switchTab({ url: '/pages/index/index' })
}

export function toast(title: string, icon: 'none' | 'success' = 'none'): void {
  uni.showToast({ title, icon, duration: icon === 'success' ? 1200 : 1800 })
  // 小程序保存成功时轻震动；H5 不调用。
  // #ifndef H5
  if (icon === 'success') {
    uni.vibrateShort({ type: 'light' })
  }
  // #endif
}

export function confirmModal(content: string, title = '确认'): Promise<boolean> {
  return new Promise((resolve) => {
    uni.showModal({
      title,
      content,
      success: (res) => resolve(Boolean(res.confirm)),
      fail: () => resolve(false),
    })
  })
}
