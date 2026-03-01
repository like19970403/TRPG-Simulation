import type {
  CreateSessionRequest,
  JoinSessionRequest,
  SessionResponse,
  SessionListResponse,
  SessionPlayerListResponse,
} from './types'
import { apiClient } from './client'
import { API } from '../lib/constants'

export function createSession(
  data: CreateSessionRequest,
): Promise<SessionResponse> {
  return apiClient<SessionResponse>(API.SESSIONS, {
    method: 'POST',
    body: JSON.stringify(data),
  })
}

export function listSessions(
  limit = 20,
  offset = 0,
): Promise<SessionListResponse> {
  return apiClient<SessionListResponse>(
    `${API.SESSIONS}?limit=${limit}&offset=${offset}`,
  )
}

export function getSession(id: string): Promise<SessionResponse> {
  return apiClient<SessionResponse>(`${API.SESSIONS}/${id}`)
}

export function startSession(id: string): Promise<SessionResponse> {
  return apiClient<SessionResponse>(`${API.SESSIONS}/${id}/start`, {
    method: 'POST',
  })
}

export function pauseSession(id: string): Promise<SessionResponse> {
  return apiClient<SessionResponse>(`${API.SESSIONS}/${id}/pause`, {
    method: 'POST',
  })
}

export function resumeSession(id: string): Promise<SessionResponse> {
  return apiClient<SessionResponse>(`${API.SESSIONS}/${id}/resume`, {
    method: 'POST',
  })
}

export function endSession(id: string): Promise<SessionResponse> {
  return apiClient<SessionResponse>(`${API.SESSIONS}/${id}/end`, {
    method: 'POST',
  })
}

export function deleteSession(id: string): Promise<void> {
  return apiClient<void>(`${API.SESSIONS}/${id}`, {
    method: 'DELETE',
  })
}

export function joinSession(
  data: JoinSessionRequest,
): Promise<SessionResponse> {
  return apiClient<SessionResponse>(`${API.SESSIONS}/join`, {
    method: 'POST',
    body: JSON.stringify(data),
  })
}

export function listSessionPlayers(
  id: string,
): Promise<SessionPlayerListResponse> {
  return apiClient<SessionPlayerListResponse>(
    `${API.SESSIONS}/${id}/players`,
  )
}
