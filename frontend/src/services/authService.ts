import api from './api'
import type { ApiResponse, AuthResponse } from '../types'

export const authService = {
  async register(email: string, password: string): Promise<ApiResponse<AuthResponse>> {
    const { data } = await api.post<ApiResponse<AuthResponse>>('/auth/register', {
      email,
      password,
    })
    return data
  },

  async login(email: string, password: string): Promise<ApiResponse<AuthResponse>> {
    const { data } = await api.post<ApiResponse<AuthResponse>>('/auth/login', {
      email,
      password,
    })
    return data
  },

  async verifyEmail(token: string): Promise<ApiResponse<void>> {
    const { data } = await api.post<ApiResponse<void>>('/auth/verify-email', { token })
    return data
  },

  async resendVerification(): Promise<ApiResponse<void>> {
    const { data } = await api.post<ApiResponse<void>>('/auth/resend-verification')
    return data
  },

  async forgotPassword(email: string): Promise<ApiResponse<void>> {
    const { data } = await api.post<ApiResponse<void>>('/auth/forgot-password', { email })
    return data
  },

  async resetPassword(token: string, newPassword: string): Promise<ApiResponse<void>> {
    const { data } = await api.post<ApiResponse<void>>('/auth/reset-password', {
      token,
      new_password: newPassword,
    })
    return data
  },
}
