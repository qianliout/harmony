import { http, unwrapItem, unwrapPage } from './http'
import type { BookItem, MergeLog, MergePreview, Page, PaperPage, PaperPageDetail } from './types'

export function fetchBooks(status: 'normal' | 'merged' = 'normal'): Promise<BookItem[]> {
  return http.get<BookItem[]>('/books', { status }).then((body) => unwrapPage(body).items)
}

export function fetchBookDetail(id: number): Promise<BookItem> {
  return http.get<BookItem>('/books/detail', { id }).then(unwrapItem)
}

export interface BookWriteParams {
  name: string
  category: string
  defaultEventId?: number
  description?: string
}

export function createBook(params: BookWriteParams): Promise<BookItem> {
  return http.post<BookItem>('/books', params).then(unwrapItem)
}

export function updateBook(id: number, params: BookWriteParams): Promise<BookItem> {
  return http.put<BookItem>('/books', params, { id }).then(unwrapItem)
}

export function deleteBook(id: number): Promise<void> {
  return http.delete<void>('/books', { id }).then(() => undefined)
}

export function previewMerge(sourceIds: number[], targetId: number): Promise<MergePreview> {
  return http.post<MergePreview>('/books/merge/preview', { sourceIds, targetId }).then(unwrapItem)
}

export function mergeBooks(sourceIds: number[], targetId: number): Promise<MergeLog> {
  return http.post<MergeLog>('/books/merge', { sourceIds, targetId }).then(unwrapItem)
}

export function fetchMergeLogs(): Promise<MergeLog[]> {
  return http.get<MergeLog[]>('/books/merge-logs').then((body) => unwrapPage(body).items)
}

export function fetchPaperPages(bookId: number): Promise<PaperPage[]> {
  return http.get<PaperPage[]>('/books/pages', { bookId }).then((body) => unwrapPage(body).items)
}

export function fetchPaperPageDetail(bookId: number, page: string): Promise<PaperPageDetail> {
  return http.get<PaperPageDetail>('/books/pages', { bookId, page }).then(unwrapItem)
}

export function setPageCheck(bookId: number, page: string, declaredCent: number): Promise<void> {
  return http.put<void>('/books/pages', { declaredCent }, { bookId, page }).then(() => undefined)
}

export type { Page }
