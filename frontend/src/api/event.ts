import { http, unwrapItem, unwrapPage } from './http'
import type { EventItem, Page } from './types'

export interface EventListParams {
  type?: string
  host?: string
  from?: string
  to?: string
  keyword?: string
  page?: number
  pageSize?: number
}

export function fetchEvents(params: EventListParams = {}): Promise<Page<EventItem>> {
  return http
    .get<EventItem[]>('/events', { page: 1, pageSize: 50, ...params })
    .then(unwrapPage)
}

export function fetchRecentEvents(limit = 6): Promise<EventItem[]> {
  return http.get<EventItem[]>('/events/recent', { limit }).then((body) => unwrapPage(body).items)
}

export function fetchEventDetail(id: number): Promise<EventItem> {
  return http.get<EventItem>('/events/detail', { id }).then(unwrapItem)
}

export interface EventWriteParams {
  name: string
  type: string
  eventDate?: string
  datePrecision?: string
  location?: string
  host: string
  hostPersonId?: number
  remark?: string
}

export function createEvent(params: EventWriteParams): Promise<EventItem> {
  return http.post<EventItem>('/events', params).then(unwrapItem)
}

export function updateEvent(id: number, params: EventWriteParams): Promise<EventItem> {
  return http.put<EventItem>('/events', params, { id }).then(unwrapItem)
}

export function deleteEvent(id: number): Promise<void> {
  return http.delete<void>('/events', { id }).then(() => undefined)
}
