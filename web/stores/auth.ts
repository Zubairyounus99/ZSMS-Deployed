import { defineStore } from 'pinia'

export interface User {
  id: string
  email: string
  role: string
  created_at?: string
}

export interface AuthState {
  token: string | null
  user: User | null
  loading: boolean
  authDisabled: boolean
}

export const useAuthStore = defineStore('auth', {
  state: (): AuthState => ({
    token: null,
    user: null,
    loading: false,
    authDisabled: false,
  }),

  getters: {
    isAuthenticated: (state) => state.authDisabled || !!state.token,
    userRole: (state) => state.user?.role || (state.authDisabled ? 'admin' : 'user'),
  },

  actions: {
    async checkAuthConfig(): Promise<boolean> {
      const config = useRuntimeConfig()
      let isDisabled = !!config.public.authDisabled
      const apiEndpoint = (config.public.apiBaseUrl as string) || (config.public.apiUrl as string) || 'https://sms-api.ztechai.us'
      try {
        const res = await fetch(`${apiEndpoint}/api/v1/auth/config`)
        if (res.ok) {
          const json = await res.json()
          if (json?.success && typeof json?.data?.auth_disabled === 'boolean') {
            isDisabled = isDisabled || json.data.auth_disabled
          }
        }
      } catch {
        // Fallback to runtime config if network check fails
      }
      this.authDisabled = isDisabled
      return isDisabled
    },

    async initAuth() {
      await this.checkAuthConfig()
      if (this.authDisabled) {
        if (!this.token) {
          this.token = 'bypass_testing_token'
          this.user = {
            id: '00000000-0000-0000-0000-000000000001',
            email: 'admin@ztechai.us',
            role: 'admin',
          }
        }
        this.fetchMe().catch(() => {})
        return
      }

      if (import.meta.client) {
        const savedToken = localStorage.getItem('zsms_auth_token')
        if (savedToken) {
          this.token = savedToken
          this.fetchMe().catch(() => {
            this.logout()
          })
        }
      }
    },

    setSession(token: string, user?: User) {
      this.token = token
      if (user) {
        this.user = user
      }
      if (import.meta.client) {
        localStorage.setItem('zsms_auth_token', token)
      }
    },

    async login(email: string, password: string) {
      this.loading = true
      const config = useRuntimeConfig()
      const apiEndpoint = (config.public.apiBaseUrl as string) || (config.public.apiUrl as string) || 'https://sms-api.ztechai.us'
      const url = `${apiEndpoint}/api/v1/auth/login`

      try {
        const res = await fetch(url, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json', 'Accept': 'application/json' },
          body: JSON.stringify({ email, password }),
        })
        const data = await res.json()

        if (!res.ok || !data.success) {
          throw new Error(data.error?.message || 'Invalid email or password')
        }

        this.setSession(data.data.token, data.data.user)
        return data.data
      } finally {
        this.loading = false
      }
    },

    async register(email: string, password: string) {
      this.loading = true
      const config = useRuntimeConfig()
      const apiEndpoint = (config.public.apiBaseUrl as string) || (config.public.apiUrl as string) || 'https://sms-api.ztechai.us'
      const url = `${apiEndpoint}/api/v1/auth/register`

      try {
        const res = await fetch(url, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json', 'Accept': 'application/json' },
          body: JSON.stringify({ email, password }),
        })
        const data = await res.json()

        if (!res.ok || !data.success) {
          throw new Error(data.error?.message || 'Registration failed')
        }

        this.setSession(data.data.token, data.data.user)
        return data.data
      } finally {
        this.loading = false
      }
    },

    async fetchMe() {
      if (!this.token && !this.authDisabled) return null
      const config = useRuntimeConfig()
      const apiEndpoint = (config.public.apiBaseUrl as string) || (config.public.apiUrl as string) || 'https://sms-api.ztechai.us'
      const url = `${apiEndpoint}/api/v1/me`

      const headers: Record<string, string> = {
        'Accept': 'application/json',
      }
      if (this.token) {
        headers['Authorization'] = `Bearer ${this.token}`
      }

      try {
        const res = await fetch(url, {
          method: 'GET',
          headers,
        })
        const data = await res.json()
        if (res.ok && data.success && data.data?.user) {
          this.user = data.data.user
          return data.data.user
        }
      } catch (e) {
        if (!this.authDisabled) {
          throw e
        }
      }
      return this.user
    },

    logout() {
      this.token = null
      this.user = null
      if (import.meta.client) {
        localStorage.removeItem('zsms_auth_token')
      }
      if (!this.authDisabled) {
        navigateTo('/auth/login')
      }
    },
  },
})
