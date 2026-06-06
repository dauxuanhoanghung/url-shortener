import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { authService } from '../services/authService'
import type { User } from '../types'

export const useAuthStore = defineStore('auth', () => {
  const user = ref<User | null>(null)
  const accessToken = ref<string | null>(null)

  const isAuthenticated = computed(() => !!accessToken.value)
  const isAdmin = computed(() => user.value?.role === 'admin')

  function init() {
    const storedToken = localStorage.getItem('access_token')
    const storedUser = localStorage.getItem('user')
    if (storedToken && storedUser) {
      if (isJwtExpired(storedToken)) {
        localStorage.removeItem('access_token')
        localStorage.removeItem('refresh_token')
        localStorage.removeItem('user')
        return
      }
      accessToken.value = storedToken
      user.value = JSON.parse(storedUser)
    }
  }

  function isJwtExpired(token: string): boolean {
    try {
      const payload = token.split('.')[1]
      if (!payload) return true
      const decoded = JSON.parse(
        atob(payload.replace(/-/g, '+').replace(/_/g, '/')),
      )
      if (typeof decoded.exp !== 'number') return true
      return decoded.exp * 1000 <= Date.now()
    } catch {
      return true
    }
  }

  async function register(email: string, password: string) {
    const resp = await authService.register(email, password)
    if (resp.success && resp.data) {
      setAuth(resp.data.access_token, resp.data.refresh_token, resp.data.user)
    }
    return resp
  }

  async function login(email: string, password: string) {
    const resp = await authService.login(email, password)
    if (resp.success && resp.data) {
      setAuth(resp.data.access_token, resp.data.refresh_token, resp.data.user)
    }
    return resp
  }

  function logout() {
    user.value = null
    accessToken.value = null
    localStorage.removeItem('access_token')
    localStorage.removeItem('refresh_token')
    localStorage.removeItem('user')
  }

  function setAuth(token: string, refresh: string, userData: User) {
    accessToken.value = token
    user.value = userData
    localStorage.setItem('access_token', token)
    localStorage.setItem('refresh_token', refresh)
    localStorage.setItem('user', JSON.stringify(userData))
  }

  function markEmailVerified() {
    if (user.value) {
      user.value = { ...user.value, email_verified: true }
      localStorage.setItem('user', JSON.stringify(user.value))
    }
  }

  return {
    user,
    accessToken,
    isAuthenticated,
    isAdmin,
    init,
    register,
    login,
    logout,
    markEmailVerified,
  }
})
