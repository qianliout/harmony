import { http, unwrapItem, unwrapPage } from './http'
import type { DictItem, Page } from './types'

export function fetchDicts(type?: string): Promise<DictItem[]> {
  return http.get<DictItem[]>('/dicts', { type }).then((body) => unwrapPage(body).items)
}

export function createDict(params: { type: string; label: string; color?: string }): Promise<DictItem> {
  return http.post<DictItem>('/dicts', params).then(unwrapItem)
}

export function updateDict(
  id: number,
  params: { label: string; color?: string; sort?: number },
): Promise<DictItem> {
  return http.put<DictItem>('/dicts', params, { id }).then(unwrapItem)
}

export function deleteDict(id: number): Promise<void> {
  return http.delete<void>('/dicts', { id }).then(() => undefined)
}

export type { Page }
