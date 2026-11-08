import { http, unwrapItem, unwrapPage } from './http'
import type {
  EventParam,
  Page,
  PaperParseResult,
  RecordDetail,
  RecordItem,
  RepayItem,
} from './types'

export interface RecordListParams {
  bookCategory?: string
  eventType?: string
  bookId?: number
  originBookId?: number
  personId?: number
  eventId?: number
  itemKind?: string
  direction?: string
  status?: string
  from?: string
  to?: string
  keyword?: string
  page?: number
  pageSize?: number
}

export function fetchRecords(params: RecordListParams = {}): Promise<Page<RecordItem>> {
  return http
    .get<RecordItem[]>('/records', { page: 1, pageSize: 20, ...params })
    .then(unwrapPage)
}

export function fetchRecordDetail(id: number): Promise<RecordDetail> {
  return http.get<RecordDetail>('/records/detail', { id }).then(unwrapItem)
}

export interface RecordWriteParams {
  personId?: number
  personName?: string
  eventId: number
  bookId: number
  direction: string
  itemKind?: string
  amountCent: number
  giftName?: string
  giftValueCent?: number
  occurredOn: string
  datePrecision?: string
  status?: string
  remark?: string
  paperNote?: string
  paperPage?: string
  paperSeq?: number
  confirm?: boolean
}

export function createRecord(params: RecordWriteParams): Promise<RecordItem> {
  return http.post<RecordItem>('/records', params).then(unwrapItem)
}

export function updateRecord(id: number, params: RecordWriteParams): Promise<RecordItem> {
  return http.put<RecordItem>('/records', params, { id }).then(unwrapItem)
}

export function deleteRecord(id: number, confirm = false): Promise<void> {
  return http
    .delete<void>('/records', { id, ...(confirm ? { confirm: 'true' } : {}) })
    .then(() => undefined)
}

export interface RepayParams {
  originIds: number[]
  markRepaid?: boolean
  event?: EventParam | null
  occurredOn: string
  datePrecision: string
  paperNote?: string
  items: RepayItem[]
  remark?: string
}

export function repayRecords(params: RepayParams): Promise<RecordItem[]> {
  return http.post<{ records: RecordItem[]; repayCount: number }>('/records/repay', params).then((body) => body.data.records)
}

export function unlinkRepay(repayId: number, originId: number): Promise<void> {
  return http.post<void>('/records/unlink', { originId }, { id: repayId }).then(() => undefined)
}

export function markRecordRepaid(id: number, paperNote = ''): Promise<void> {
  return http.post<void>('/records/mark-repaid', { paperNote }, { id }).then(() => undefined)
}

export function parsePaperNote(note: string, personId = 0): Promise<PaperParseResult> {
  return http.post<PaperParseResult>('/paper-notes/parse', { note, personId }).then(unwrapItem)
}

export function fetchPendingNotes(bookId = 0, page = ''): Promise<RecordItem[]> {
  return http
    .get<RecordItem[]>('/paper-notes/pending', { bookId: bookId || undefined, page: page || undefined })
    .then((body) => unwrapPage(body).items)
}
