import { ref } from 'vue'

import { ApiError } from '@/api/http'
import type { ConfirmPayload } from '@/api/types'

interface ConfirmState {
  show: boolean
  payload: ConfirmPayload | null
}

export const confirmState = ref<ConfirmState>({ show: false, payload: null })
let resolver: ((value: boolean) => void) | null = null

export function openConfirmSheet(payload: ConfirmPayload): Promise<boolean> {
  confirmState.value = { show: true, payload }
  return new Promise<boolean>((resolve) => {
    resolver = resolve
  })
}

export function resolveConfirm(value: boolean): void {
  confirmState.value.show = false
  resolver?.(value)
  resolver = null
}

/**
 * 请求层统一处理 409 + need_confirm：业务页面只写
 * `await submitWithConfirm(confirm => api.createRecord({ ...form, confirm }))`。
 */
export async function submitWithConfirm<T>(send: (confirm: boolean) => Promise<T>): Promise<T | null> {
  try {
    return await send(false)
  } catch (error) {
    if (!(error instanceof ApiError) || error.code !== 409) {
      throw error
    }
    const reason = (error.custom as Record<string, unknown> | null)?.reason
    if (reason !== 'need_confirm') {
      throw error
    }
    const ok = await openConfirmSheet(error.data as ConfirmPayload)
    if (!ok) {
      return null
    }
    return await send(true)
  }
}
