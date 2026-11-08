import { http, unwrapItem, unwrapPage } from './http'
import type { Page, SearchRecordHit, SearchResult } from './types'

export function searchAll(query: string, limit = 5): Promise<SearchResult> {
  return http.get<SearchResult>('/search', { q: query, limit }).then(unwrapItem)
}

export function searchRecords(query: string, page = 1, pageSize = 20): Promise<Page<SearchRecordHit>> {
  return http.get<SearchRecordHit[]>('/search/records', { q: query, page, pageSize }).then(unwrapPage)
}
