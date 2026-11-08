// 与后端统一响应契约对齐：传输层恒为 HTTP 200，业务状态在 code，成功为 200。

/** 后端统一响应体。 */
export interface ApiResponse<T> {
  code: number
  msg: string
  err: string
  data: T
  totalItems: number
  itemsPerPage: number
  startIndex: number
  custom: Record<string, unknown> | null
}

/** 分页列表的展开结构。 */
export interface Page<T> {
  items: T[]
  total: number
}

/** 后端健康检查响应。 */
export interface HealthStatus {
  status: string
  database: string
  version: string
  time: number
}
