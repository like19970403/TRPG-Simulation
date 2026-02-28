import type { RegisterRequest, RegisterResponse, LoginRequest, TokenResponse } from './types'
import { apiClient } from './client'
import { API } from '../lib/constants'

export function register(data: RegisterRequest): Promise<RegisterResponse> {
  return apiClient<RegisterResponse>(API.REGISTER, {
    method: 'POST',
    body: JSON.stringify(data),
  })
}

export function login(data: LoginRequest): Promise<TokenResponse> {
  return apiClient<TokenResponse>(API.LOGIN, {
    method: 'POST',
    body: JSON.stringify(data),
  })
}

export function refresh(): Promise<TokenResponse> {
  return apiClient<TokenResponse>(API.REFRESH, {
    method: 'POST',
  })
}

export function logout(): Promise<void> {
  return apiClient<void>(API.LOGOUT, {
    method: 'POST',
  })
}
