import { http, unwrapItem } from '@/core/request'
import type { HealthStatus } from '@/modules/dashboard/types'

export async function fetchHealth(): Promise<HealthStatus> {
  const body = await http.get<HealthStatus>('/health')
  return unwrapItem(body)
}
