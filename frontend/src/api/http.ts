// 统一请求封装：基于 uni.request，H5 与小程序共用。
// 传输层恒为 HTTP 200，业务码在 body.code，成功为 200；其余业务码与网络错误统一转成 ApiError 抛出。
// 令牌只存 storage；401 时由注册的处理器决定重登（小程序）或跳登录页（网页）。

import type { ApiResponse, Page } from './types'

export const CODE_OK = 200
const TOKEN_KEY = 'harmony_token'

// H5 走 Vite 代理的同源路径；小程序直连后端地址。平台差异只用条件编译表达，不做运行时判断。
function resolveTarget(): { baseURL: string; platform: string } {
  // #ifdef H5
  return { baseURL: '/api/v1', platform: 'h5' }
  // #endif
  // #ifndef H5
  return { baseURL: import.meta.env.VITE_MP_API_BASE || 'http://127.0.0.1:8080/api/v1', platform: 'mp-weixin' }
  // #endif
}

const { baseURL: BASE_URL, platform: PLATFORM } = resolveTarget()

export function apiBaseURL(): string {
  return BASE_URL
}

export function queryString(query?: object): string {
  return toQuery(query)
}

export function getToken(): string {
  const value = uni.getStorageSync(TOKEN_KEY)
  return typeof value === 'string' ? value : ''
}

export function setToken(token: string): void {
  uni.setStorageSync(TOKEN_KEY, token)
}

export function clearToken(): void {
  uni.removeStorageSync(TOKEN_KEY)
}

type UnauthorizedHandler = () => Promise<boolean> | boolean
let unauthorizedHandler: UnauthorizedHandler | null = null

/** 由 auth store 注册：返回 true 表示已重登，可以重放原请求。 */
export function setUnauthorizedHandler(handler: UnauthorizedHandler | null): void {
  unauthorizedHandler = handler
}

/** 业务错误：code 为后端业务码，网络失败时为 -1。 */
export class ApiError extends Error {
  constructor(
    public readonly code: number,
    message: string,
    public readonly data?: unknown,
    public readonly custom?: Record<string, unknown> | null,
  ) {
    super(message)
  }
}

type Method = 'GET' | 'POST' | 'PUT' | 'DELETE'

async function send<T>(
  method: Method,
  url: string,
  data?: object,
  query?: object,
  retried = false,
): Promise<ApiResponse<T>> {
  const token = getToken()
  const res = await new Promise<UniApp.RequestSuccessCallbackResult>((resolve, reject) => {
    uni.request({
      url: BASE_URL + url + toQuery(query),
      method,
      data: data as UniApp.RequestOptions['data'],
      timeout: 20_000,
      header: {
        'X-Client-Platform': PLATFORM,
        ...(token ? { Authorization: `Bearer ${token}` } : {}),
      },
      success: resolve,
      fail: () => reject(new ApiError(-1, '网络不太好，请重试')),
    })
  })

  const body = res.data as ApiResponse<T> | undefined
  if (!body || typeof body.code !== 'number') {
    const message = res.statusCode === 200 ? '服务返回格式异常' : `服务暂时不可用（HTTP ${res.statusCode}）`
    throw new ApiError(res.statusCode, message)
  }
  if (body.code === 401) {
    clearToken()
    if (!retried && unauthorizedHandler) {
      const recovered = await unauthorizedHandler()
      if (recovered) {
        return send<T>(method, url, data, query, true)
      }
    }
    throw new ApiError(body.code, body.msg || '请先登录', body.data, body.custom)
  }
  if (body.code !== CODE_OK) {
    throw new ApiError(body.code, body.msg || body.err, body.data, body.custom)
  }
  return body
}

function toQuery(query?: object): string {
  if (!query) {
    return ''
  }
  const pairs = Object.entries(query)
    .filter(([, v]) => v !== undefined && v !== null && v !== '')
    .map(([k, v]) => `${encodeURIComponent(k)}=${encodeURIComponent(String(v))}`)
  return pairs.length ? `?${pairs.join('&')}` : ''
}

export const http = {
  get<T>(url: string, query?: object) {
    return send<T>('GET', url, undefined, query)
  },
  post<T>(url: string, data?: object, query?: object) {
    return send<T>('POST', url, data, query)
  },
  put<T>(url: string, data?: object, query?: object) {
    return send<T>('PUT', url, data, query)
  },
  delete<T>(url: string, query?: object) {
    return send<T>('DELETE', url, undefined, query)
  },
}

/** 展开单条响应。 */
export function unwrapItem<T>(body: ApiResponse<T>): T {
  return body.data
}

/** 展开列表响应；后端约定列表 data 为数组，空列表为 []。 */
export function unwrapPage<T>(body: ApiResponse<T[]>): Page<T> {
  return { items: body.data ?? [], total: body.totalItems }
}
