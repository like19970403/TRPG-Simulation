import type {
  CreateScenarioRequest,
  UpdateScenarioRequest,
  ScenarioResponse,
  ScenarioListResponse,
} from './types'
import { apiClient } from './client'
import { API } from '../lib/constants'

export function createScenario(
  data: CreateScenarioRequest,
): Promise<ScenarioResponse> {
  return apiClient<ScenarioResponse>(API.SCENARIOS, {
    method: 'POST',
    body: JSON.stringify(data),
  })
}

export function listScenarios(
  limit = 20,
  offset = 0,
): Promise<ScenarioListResponse> {
  return apiClient<ScenarioListResponse>(
    `${API.SCENARIOS}?limit=${limit}&offset=${offset}`,
  )
}

export function getScenario(id: string): Promise<ScenarioResponse> {
  return apiClient<ScenarioResponse>(`${API.SCENARIOS}/${id}`)
}

export function updateScenario(
  id: string,
  data: UpdateScenarioRequest,
): Promise<ScenarioResponse> {
  return apiClient<ScenarioResponse>(`${API.SCENARIOS}/${id}`, {
    method: 'PUT',
    body: JSON.stringify(data),
  })
}

export function deleteScenario(id: string): Promise<void> {
  return apiClient<void>(`${API.SCENARIOS}/${id}`, {
    method: 'DELETE',
  })
}

export function publishScenario(id: string): Promise<ScenarioResponse> {
  return apiClient<ScenarioResponse>(`${API.SCENARIOS}/${id}/publish`, {
    method: 'POST',
  })
}

export function archiveScenario(id: string): Promise<ScenarioResponse> {
  return apiClient<ScenarioResponse>(`${API.SCENARIOS}/${id}/archive`, {
    method: 'POST',
  })
}
