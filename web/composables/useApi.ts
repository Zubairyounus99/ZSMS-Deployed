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
  const rawBase = (config.public.apiBaseUrl as string) || (config.public.apiUrl as string) || 'https://sms-api.ztechai.us'
  const baseUrl = rawBase.replace(/\/+$/, '')
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
    const cleanEndpoint = endpoint.startsWith('/') ? endpoint : `/${endpoint}`
    const url = `${baseUrl}${cleanEndpoint}`
    try {
      const response = await fetch(url, {
        method: 'GET',
        headers: getHeaders(headers),
        credentials: 'include',
      })
      const text = await response.text()
      try {
        return JSON.parse(text)
      } catch {
        return {
          success: response.ok,
          error: {
            code: `HTTP_${response.status}`,
            message: `Server returned HTTP ${response.status}`,
          },
        }
      }
    } catch (err: any) {
      return {
        success: false,
        error: {
          code: 'NETWORK_ERROR',
          message: err.message || 'Network request failed',
        },
      }
    }
  }

  async function post<T>(endpoint: string, body: any, headers: Record<string, string> = {}): Promise<ApiResponse<T>> {
    const cleanEndpoint = endpoint.startsWith('/') ? endpoint : `/${endpoint}`
    const url = `${baseUrl}${cleanEndpoint}`
    try {
      const response = await fetch(url, {
        method: 'POST',
        headers: getHeaders({
          'Content-Type': 'application/json',
          ...headers,
        }),
        body: JSON.stringify(body),
        credentials: 'include',
      })
      const text = await response.text()
      try {
        return JSON.parse(text)
      } catch {
        return {
          success: response.ok,
          error: {
            code: `HTTP_${response.status}`,
            message: `Server returned HTTP ${response.status}`,
          },
        }
      }
    } catch (err: any) {
      return {
        success: false,
        error: {
          code: 'NETWORK_ERROR',
          message: err.message || 'Network request failed',
        },
      }
    }
  }

  return {
    baseUrl,
    get,
    post,
  }
}
