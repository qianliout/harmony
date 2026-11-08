// 统一请求封装：
// - 自动拼接 /api/v1 前缀（axios baseURL）；
// - 响应拦截器统一处理业务码与网络错误（传输层恒 200，业务码在 body.code），失败时提示并抛出；
// - 提供 http.get/post/put/delete 与 unwrapItem/unwrapPage 拆包助手，
//   让模块 API 层只关心类型化的业务请求，不接触 axios 细节。
import axios, { type AxiosRequestConfig } from 'axios'
import { ElMessage } from 'element-plus'

import type { ApiResponse, Page } from './types'

const service = axios.create({
  baseURL: '/api/v1',
  timeout: 15_000,
})

// 响应拦截器：业务码非 200 统一提示；网络错误提取 msg/err 提示。
service.interceptors.response.use(
  (response) => {
    const body = response.data as ApiResponse<unknown> | undefined
    if (body && typeof body.code === 'number' && body.code !== 200) {
      const message = body.msg || body.err
      ElMessage.error(message)
      return Promise.reject(new Error(message))
    }
    return response
  },
  (error: unknown) => {
    const message = resolveErrorMessage(error)
    ElMessage.error(message)
    return Promise.reject(new Error(message))
  },
)

// 类型化请求入口：返回解包前的统一响应体，由调用方用 unwrap* 拆包。
function request<T>(config: AxiosRequestConfig): Promise<ApiResponse<T>> {
  return service.request<ApiResponse<T>>(config).then((response) => response.data)
}

// 业务请求方法集。params 用于 query，data 用于 JSON body（与后端 REST 约定一致）。
export const http = {
  get<T>(url: string, params?: object): Promise<ApiResponse<T>> {
    return request<T>({ url, method: 'get', params })
  },
  post<T>(url: string, data?: Record<string, unknown>): Promise<ApiResponse<T>> {
    return request<T>({ url, method: 'post', data })
  },
  put<T>(url: string, data?: Record<string, unknown>, params?: object): Promise<ApiResponse<T>> {
    return request<T>({ url, method: 'put', data, params })
  },
  delete<T>(url: string, params?: object): Promise<ApiResponse<T>> {
    return request<T>({ url, method: 'delete', params })
  },
}

// ---- 拆包助手：纯函数，将统一响应体展开为业务类型 ----

/** 展开单条响应，返回 data。 */
export function unwrapItem<T>(body: ApiResponse<T>): T {
  if (body.data === undefined || body.data === null) {
    throw new Error('响应缺少 data 字段')
  }
  return body.data as T
}

/** 展开列表响应，返回分页结构；空列表返回空数组（后端约定返回 []）。 */
export function unwrapPage<T>(body: ApiResponse<T>): Page<T> {
  const data = body.data as T[] | undefined
  return {
    items: data ?? [],
    total: body.totalItems ?? 0,
    itemsPerPage: body.itemsPerPage ?? 0,
    startIndex: body.startIndex ?? 0,
  }
}

/** 从 axios 错误中提取可展示的信息：优先后端 msg/err，其次网络错误。 */
function resolveErrorMessage(error: unknown): string {
  if (axios.isAxiosError(error)) {
    const body = error.response?.data as ApiResponse<unknown> | undefined
    if (body?.msg || body?.err) {
      return body.msg || body.err
    }
    if (error.code === 'ECONNABORTED') {
      return '请求超时，请稍后重试'
    }
    return '网络请求失败，请检查服务是否可用'
  }
  return error instanceof Error ? error.message : '未知错误'
}
