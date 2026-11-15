import { ref, type Ref } from 'vue'

export interface AsyncResource<T> {
  data: Ref<T>
  loading: Ref<boolean>
  error: Ref<string>
  reload: () => Promise<void>
}

/** 桌面页面的统一加载状态：首次加载、重试与错误提示都走这里。 */
export function useAsync<T>(loader: () => Promise<T>, initial: T): AsyncResource<T> {
  const data = ref(initial) as Ref<T>
  const loading = ref(false)
  const error = ref('')
  let seq = 0

  async function reload(): Promise<void> {
    const current = ++seq
    loading.value = true
    error.value = ''
    try {
      const result = await loader()
      if (current === seq) data.value = result
    } catch (err) {
      if (current === seq) {
        error.value = err instanceof Error ? err.message : '加载失败'
      }
    } finally {
      if (current === seq) loading.value = false
    }
  }

  return { data, loading, error, reload }
}
