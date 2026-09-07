import { useAuthStore } from '~/stores/auth'

export interface ApiResponse<T = any> {
  success: boolean
  data?: T
  error?: {
    code: string
    message: string
  }
  request_id?: string
  meta?: {
    page?: number
    per_page?: number
    total?: number
    unread_count?: number
  }
}

export function useApi() {
  const config = useRuntimeConfig()
  const baseUrl = (config.public.apiBaseUrl as string) || (config.public.apiUrl as string) || 'https://sms-api.ztechai.us'
  const authStore = useAuthStore()

  function getHeaders(customHeaders: Record<string, string> = {}): Record<string, string> {
    const headers: Record<string, string> = {
      'Accept': 'application/json',
      ...customHeaders,
    }
    if (authStore.token) {
      headers['Authorization'] = `Bearer ${authStore.token}`
    }
    return headers
  }

  async function get<T>(endpoint: string, headers: Record<string, string> = {}): Promise<ApiResponse<T>> {
    const url = `${baseUrl}${endpoint.startsWith('/') ? endpoint : `/${endpoint}`}`
    const response = await fetch(url, {
      method: 'GET',
      headers: getHeaders(headers),
    })
    return response.json()
  }

  async function post<T>(endpoint: string, body: any, headers: Record<string, string> = {}): Promise<ApiResponse<T>> {
    const url = `${baseUrl}${endpoint.startsWith('/') ? endpoint : `/${endpoint}`}`
    const response = await fetch(url, {
      method: 'POST',
      headers: getHeaders({
        'Content-Type': 'application/json',
        ...headers,
      }),
      body: JSON.stringify(body),
    })
    return response.json()
  }

  return {
    baseUrl,
    get,
    post,
  }
}
