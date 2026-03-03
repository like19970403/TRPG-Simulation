import type { ImageUploadResponse } from './types'
import { useAuthStore } from '../stores/auth-store'
import { ApiClientError } from './client'
import { API } from '../lib/constants'

/** Upload an image file. Returns the server URL for the uploaded image. */
export async function uploadImage(file: File): Promise<ImageUploadResponse> {
  const token = useAuthStore.getState().accessToken

  const form = new FormData()
  form.append('file', file)

  const headers: HeadersInit = {}
  if (token) {
    headers['Authorization'] = `Bearer ${token}`
  }

  const res = await fetch(API.IMAGE_UPLOAD, {
    method: 'POST',
    headers,
    credentials: 'include',
    body: form,
  })

  if (!res.ok) {
    const body = await res.json().catch(() => ({
      error: 'UNKNOWN',
      message: 'Upload failed',
    }))
    throw new ApiClientError(res.status, body)
  }

  return res.json()
}
