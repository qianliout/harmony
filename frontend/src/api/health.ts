import { http, unwrapItem } from './http'
import type { HealthStatus } from './types'

export async function fetchHealth(): Promise<HealthStatus> {
  const body = await http.get<HealthStatus>('/health')
  return unwrapItem(body)
}
