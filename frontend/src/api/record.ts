import { http, unwrapItem, unwrapPage } from './http'
import type { Direction, Page, RecordItem, RecordSummary } from './types'

/** 总账、礼簿详情、亲友详情共用同一套筛选。 */
export interface RecordFilter {
  bookId?: number
  personId?: number
  direction?: Direction | ''
  from?: string
  to?: string
  keyword?: string
}

export interface RecordListParams extends RecordFilter {
  page?: number
  pageSize?: number
}

export function fetchRecords(params: RecordListParams = {}): Promise<Page<RecordItem>> {
  return http.get<RecordItem[]>('/records', { page: 1, pageSize: 20, ...params }).then(unwrapPage)
}

export function fetchRecordSummary(filter: RecordFilter = {}): Promise<RecordSummary> {
  return http.get<RecordSummary>('/records/summary', filter).then(unwrapItem)
}

export function fetchRecordDetail(id: number): Promise<RecordItem> {
  return http.get<RecordItem>('/records/detail', { id }).then(unwrapItem)
}

export interface RecordCreateParams {
  personId?: number
  personName?: string
  personRemark?: string
  bookId: number
  direction: Direction
  amountCent: number
  matter: string
  note: string
  occurredOn: string
}

export interface RecordUpdateParams {
  personId: number
  bookId: number
  direction: Direction
  amountCent: number
  matter: string
  note: string
  occurredOn: string
}

export function createRecord(params: RecordCreateParams): Promise<RecordItem> {
  return http.post<RecordItem>('/records', params).then(unwrapItem)
}

export function updateRecord(id: number, params: RecordUpdateParams): Promise<RecordItem> {
  return http.put<RecordItem>('/records', params, { id }).then(unwrapItem)
}

export function deleteRecord(id: number): Promise<void> {
  return http.delete<void>('/records', { id }).then(() => undefined)
}
