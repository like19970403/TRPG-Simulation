import type {
  CreateCharacterRequest,
  UpdateCharacterRequest,
  CharacterResponse,
  CharacterListResponse,
  AssignCharacterRequest,
} from './types'
import { apiClient } from './client'
import { API } from '../lib/constants'

export function createCharacter(
  data: CreateCharacterRequest,
): Promise<CharacterResponse> {
  return apiClient<CharacterResponse>(API.CHARACTERS, {
    method: 'POST',
    body: JSON.stringify(data),
  })
}

export function listCharacters(
  limit = 20,
  offset = 0,
): Promise<CharacterListResponse> {
  return apiClient<CharacterListResponse>(
    `${API.CHARACTERS}?limit=${limit}&offset=${offset}`,
  )
}

export function getCharacter(id: string): Promise<CharacterResponse> {
  return apiClient<CharacterResponse>(`${API.CHARACTERS}/${id}`)
}

export function updateCharacter(
  id: string,
  data: UpdateCharacterRequest,
): Promise<CharacterResponse> {
  return apiClient<CharacterResponse>(`${API.CHARACTERS}/${id}`, {
    method: 'PUT',
    body: JSON.stringify(data),
  })
}

export function deleteCharacter(id: string): Promise<void> {
  return apiClient<void>(`${API.CHARACTERS}/${id}`, {
    method: 'DELETE',
  })
}

export function assignCharacter(
  sessionId: string,
  data: AssignCharacterRequest,
): Promise<void> {
  return apiClient<void>(`${API.SESSIONS}/${sessionId}/characters`, {
    method: 'POST',
    body: JSON.stringify(data),
  })
}
