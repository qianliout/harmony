import { http, unwrapItem } from './http'
import type { EventSummary, PendingList, Timeline } from './types'

export interface TimelineParams {
  personId: number
  bookId?: number
  eventId?: number
  itemKind?: string
  direction?: string
  status?: string
  from?: string
  to?: string
  keyword?: string
}

export function fetchPersonTimeline(params: TimelineParams): Promise<Timeline> {
  return http.get<Timeline>('/views/person-timeline', params).then(unwrapItem)
}

export function fetchEventSummary(eventId: number): Promise<EventSummary> {
  return http.get<EventSummary>('/views/event-summary', { eventId }).then(unwrapItem)
}

export function fetchPending(kind: 'owed' | 'owing'): Promise<PendingList> {
  const path = kind === 'owed' ? '/views/pending/owed' : '/views/pending/owing'
  return http.get<PendingList>(path).then(unwrapItem)
}
