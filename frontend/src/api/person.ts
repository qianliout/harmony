import { http, unwrapItem, unwrapPage } from './http'
import type { Page, PersonBrief, PersonDetail } from './types'

export interface PersonListParams {
  keyword?: string
  relation?: string
  page?: number
  pageSize?: number
}

export function fetchPersons(params: PersonListParams = {}): Promise<Page<PersonBrief>> {
  return http
    .get<PersonBrief[]>('/persons', { page: 1, pageSize: 50, ...params })
    .then(unwrapPage)
}

export function fetchRecentPersons(limit = 8): Promise<PersonBrief[]> {
  return http.get<PersonBrief[]>('/persons/recent', { limit }).then((body) => unwrapPage(body).items)
}

export function fetchPersonDetail(id: number): Promise<PersonDetail> {
  return http.get<PersonDetail>('/persons/detail', { id }).then(unwrapItem)
}

export interface PersonWriteParams {
  name: string
  relation: string
  phone?: string
  remark?: string
  confirm?: boolean
}

export function createPerson(params: PersonWriteParams): Promise<PersonDetail> {
  return http.post<PersonDetail>('/persons', params).then(unwrapItem)
}

export function updatePerson(id: number, params: PersonWriteParams): Promise<PersonDetail> {
  return http.put<PersonDetail>('/persons', params, { id }).then(unwrapItem)
}

export function deletePerson(id: number): Promise<void> {
  return http.delete<void>('/persons', { id }).then(() => undefined)
}
