import { http, unwrapItem } from './http'
import type { Settings } from './types'

export function fetchSettings(): Promise<Settings> {
  return http.get<Settings>('/settings').then(unwrapItem)
}

export function updateSettings(params: Partial<Settings>): Promise<Settings> {
  return http.put<Settings>('/settings', params).then(unwrapItem)
}
