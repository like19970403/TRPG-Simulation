export interface RegisterRequest {
  username: string
  email: string
  password: string
}

export interface RegisterResponse {
  id: string
  username: string
  email: string
  createdAt: string
}

export interface LoginRequest {
  email: string
  password: string
}

export interface TokenResponse {
  accessToken: string
  expiresIn: number
  tokenType: string
}

export interface ErrorDetail {
  field: string
  reason: string
}

export interface ApiError {
  error: string
  message: string
  details?: ErrorDetail[]
}

export interface User {
  id: string
  username: string
}

// --- Scenario types ---

export type ScenarioStatus = 'draft' | 'published' | 'archived'

export interface ScenarioResponse {
  id: string
  authorId: string
  title: string
  description: string
  version: number
  status: ScenarioStatus
  content: Record<string, unknown>
  createdAt: string
  updatedAt: string
}

export interface ScenarioListResponse {
  scenarios: ScenarioResponse[]
  total: number
  limit: number
  offset: number
}

export interface CreateScenarioRequest {
  title: string
  description: string
  content: Record<string, unknown>
}

export interface UpdateScenarioRequest {
  title: string
  description: string
  content: Record<string, unknown>
}
