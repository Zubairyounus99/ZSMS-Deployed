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
}

export const useAuthStore = defineStore('auth', {
  state: (): AuthState => ({
    token: null,
    user: null,
    loading: false,
  }),

  getters: {
    isAuthenticated: (state) => !!state.token,
    userRole: (state) => state.user?.role || 'user',
  },

  actions: {
    initAuth() {
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
      if (!this.token) return null
      const config = useRuntimeConfig()
      const apiEndpoint = (config.public.apiBaseUrl as string) || (config.public.apiUrl as string) || 'https://sms-api.ztechai.us'
      const url = `${apiEndpoint}/api/v1/me`

      const res = await fetch(url, {
        method: 'GET',
        headers: {
          'Accept': 'application/json',
          'Authorization': `Bearer ${this.token}`,
        },
      })
      const data = await res.json()
      if (res.ok && data.success) {
        this.user = data.data
        return data.data
      }
      throw new Error('Failed to retrieve user profile')
    },

    logout() {
      this.token = null
      this.user = null
      if (import.meta.client) {
        localStorage.removeItem('zsms_auth_token')
      }
      navigateTo('/auth/login')
    },
  },
})
