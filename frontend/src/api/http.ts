// 统一请求封装：基于 uni.request，H5 与小程序共用。
// 传输层恒为 HTTP 200，业务码在 body.code，成功为 200；其余业务码与网络错误统一转成 ApiError 抛出，
// 由调用方决定提示方式（页面内错误态或轻提示）。

import type { ApiResponse, Page } from './types'

export const CODE_OK = 200

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

/** 导出、上传等需要自行拼 URL 的场景使用。 */
export function apiBaseURL(): string {
  return BASE_URL
}

export function queryString(query?: object): string {
  return toQuery(query)
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

function send<T>(method: Method, url: string, data?: object, query?: object): Promise<ApiResponse<T>> {
  return new Promise((resolve, reject) => {
    uni.request({
      url: BASE_URL + url + toQuery(query),
      method,
      data: data as UniApp.RequestOptions['data'],
      timeout: 15_000,
      header: { 'X-Client-Platform': PLATFORM },
      success: (res) => {
        const body = res.data as ApiResponse<T> | undefined
        if (!body || typeof body.code !== 'number') {
          const message = res.statusCode === 200 ? '服务返回格式异常' : `服务暂时不可用（HTTP ${res.statusCode}）`
          reject(new ApiError(res.statusCode, message))
          return
        }
        if (body.code !== CODE_OK) {
          reject(new ApiError(body.code, body.msg || body.err, body.data, body.custom))
          return
        }
        resolve(body)
      },
      fail: () => reject(new ApiError(-1, '网络不太好，请重试')),
    })
  })
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
