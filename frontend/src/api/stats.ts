import { http, unwrapItem, unwrapPage } from './http'
import type { YearStats } from './types'

export function fetchYears(): Promise<number[]> {
  return http.get<number[]>('/stats/years').then((body) => unwrapPage(body).items)
}

export function fetchYearStats(year: number): Promise<YearStats> {
  return http.get<YearStats>('/stats/year', { year }).then(unwrapItem)
}
