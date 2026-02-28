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
