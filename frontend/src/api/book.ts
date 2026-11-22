import { http, unwrapItem, unwrapPage } from './http'
import type { BookItem, Page } from './types'

export interface BookListParams {
  keyword?: string
  category?: string
  page?: number
  pageSize?: number
}

export function fetchBooks(params: BookListParams = {}): Promise<Page<BookItem>> {
  return http.get<BookItem[]>('/books', { page: 1, pageSize: 100, ...params }).then(unwrapPage)
}

export function fetchBookDetail(id: number): Promise<BookItem> {
  return http.get<BookItem>('/books/detail', { id }).then(unwrapItem)
}

export interface BookWriteParams {
  name: string
  category: string
  description: string
  occurredOn: string
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
