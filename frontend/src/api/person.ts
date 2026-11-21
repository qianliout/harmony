import { http, unwrapItem, unwrapPage } from './http'
import type { Page, PersonItem } from './types'

export interface PersonListParams {
  keyword?: string
  page?: number
  pageSize?: number
}

export function fetchPersons(params: PersonListParams = {}): Promise<Page<PersonItem>> {
  return http.get<PersonItem[]>('/persons', { page: 1, pageSize: 50, ...params }).then(unwrapPage)
}

export function fetchPersonDetail(id: number): Promise<PersonItem> {
  return http.get<PersonItem>('/persons/detail', { id }).then(unwrapItem)
}

export interface PersonWriteParams {
  name: string
  remark: string
}

export function createPerson(params: PersonWriteParams): Promise<PersonItem> {
  return http.post<PersonItem>('/persons', params).then(unwrapItem)
}

export function updatePerson(id: number, params: PersonWriteParams): Promise<PersonItem> {
  return http.put<PersonItem>('/persons', params, { id }).then(unwrapItem)
}

export function deletePerson(id: number): Promise<void> {
  return http.delete<void>('/persons', { id }).then(() => undefined)
}
