import { http, unwrapItem, unwrapPage } from './http'
import type { Page, SearchRecordHit, SearchResult } from './types'

export function searchAll(query: string, limit = 5): Promise<SearchResult> {
  return http.get<SearchResult>('/search', { q: query, limit }).then(unwrapItem)
}

export interface SearchRecordParams {
  q: string
  bookCategory?: string
  eventType?: string
  personId?: number
  from?: string
  to?: string
  page?: number
  pageSize?: number
}

export function searchRecords(params: SearchRecordParams): Promise<Page<SearchRecordHit>> {
  const { q, page = 1, pageSize = 20, ...filters } = params
  return http
    .get<SearchRecordHit[]>('/search/records', { q, page, pageSize, ...filters })
    .then(unwrapPage)
}
