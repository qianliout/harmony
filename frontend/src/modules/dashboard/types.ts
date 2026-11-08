/** 后端健康检查响应。 */
export interface HealthStatus {
  status: string
  database: string
  version: string
  time: number
}
