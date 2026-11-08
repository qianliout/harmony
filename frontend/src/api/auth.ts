import { http, unwrapItem, unwrapPage } from './http'

export interface AuthUser {
  id: number
  nickname: string
  isAdmin: boolean
}

export interface LoginResult {
  token: string
  expiresAt: number
  user: AuthUser
}

export interface MeResult {
  user: AuthUser
  isAdmin: boolean
  client: string
}

export interface AuthSession {
  id: number
  client: string
  device: string
  createdAt: number
  lastUsedAt: number
  expiresAt: number
}

export interface WebTicket {
  ticket: string
  qrKind: 'wxacode' | 'text' | string
  qrImage: string
  expiresAt: number
}

export interface TicketStatus {
  status: 'waiting' | 'scanned' | 'confirmed' | 'canceled' | 'expired' | string
  token?: string
}

export function wxLogin(code: string): Promise<LoginResult> {
  return http.post<LoginResult>('/auth/wx-login', { code }).then(unwrapItem)
}

export function createWebTicket(): Promise<WebTicket> {
  return http.post<WebTicket>('/auth/web-ticket').then(unwrapItem)
}

export function pollWebTicket(ticket: string): Promise<TicketStatus> {
  return http.get<TicketStatus>('/auth/web-ticket', { ticket }).then(unwrapItem)
}

export function scanWebTicket(ticket: string): Promise<void> {
  return http.post<void>('/auth/web-ticket/scan', undefined, { ticket }).then(() => undefined)
}

export function confirmWebTicket(ticket: string): Promise<void> {
  return http.post<void>('/auth/web-ticket/confirm', undefined, { ticket }).then(() => undefined)
}

export function cancelWebTicket(ticket: string): Promise<void> {
  return http.post<void>('/auth/web-ticket/cancel', undefined, { ticket }).then(() => undefined)
}

export function fetchMe(): Promise<MeResult> {
  return http.get<MeResult>('/auth/me').then(unwrapItem)
}

export function fetchSessions(): Promise<AuthSession[]> {
  return http.get<AuthSession[]>('/auth/sessions').then((body) => unwrapPage(body).items)
}

export function deleteSession(id: number): Promise<void> {
  return http.delete<void>('/auth/sessions', { id }).then(() => undefined)
}

export function logout(): Promise<void> {
  return http.post<void>('/auth/logout').then(() => undefined)
}
