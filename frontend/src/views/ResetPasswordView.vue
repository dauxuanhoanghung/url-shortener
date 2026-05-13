<script setup lang="ts">
import { computed, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { authService } from '../services/authService'

const route = useRoute()
const router = useRouter()

const token = computed(() => (typeof route.query.token === 'string' ? route.query.token : ''))
const password = ref('')
const confirmPassword = ref('')
const error = ref('')
const loading = ref(false)
const done = ref(false)

async function handleSubmit() {
  error.value = ''

  if (!token.value) {
    error.value = 'This reset link is missing a token.'
    return
  }
  if (password.value.length < 8) {
    error.value = 'Password must be at least 8 characters.'
    return
  }
  if (password.value !== confirmPassword.value) {
    error.value = 'Passwords do not match.'
    return
  }

  loading.value = true
  try {
    const resp = await authService.resetPassword(token.value, password.value)
    if (resp.success) {
      done.value = true
      setTimeout(() => router.push({ name: 'login' }), 2000)
    } else {
      error.value = resp.error?.message ?? 'Could not reset password.'
    }
  } catch (err: any) {
    const code = err.response?.data?.error?.code
    error.value =
      code === 'TOKEN_INVALID'
        ? 'This reset link is invalid or has expired.'
        : err.response?.data?.error?.message ?? 'Could not reset password.'
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="auth-page">
    <div class="auth-card">
      <h1>Choose a new password</h1>

      <div v-if="done" class="alert alert-success">
        Password updated. Redirecting you to login…
      </div>

      <template v-else>
        <div v-if="error" class="alert alert-error">{{ error }}</div>

        <form @submit.prevent="handleSubmit">
          <div class="form-group">
            <label for="password">New password</label>
            <input
              id="password"
              v-model="password"
              type="password"
              placeholder="Min. 8 characters"
              required
              minlength="8"
            />
          </div>
          <div class="form-group">
            <label for="confirmPassword">Confirm password</label>
            <input
              id="confirmPassword"
              v-model="confirmPassword"
              type="password"
              required
            />
          </div>
          <button type="submit" class="btn btn-primary btn-block" :disabled="loading">
            {{ loading ? 'Updating…' : 'Update password' }}
          </button>
        </form>
      </template>
    </div>
  </div>
</template>

<style scoped>
.auth-page {
  min-height: calc(100vh - 64px);
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 2rem;
}
.auth-card {
  width: 100%;
  max-width: 420px;
  padding: 2.5rem;
  border-radius: 12px;
  border: 1px solid #e5e7eb;
  background: #fff;
}
.auth-card h1 {
  font-size: 1.75rem;
  font-weight: 700;
  color: #111827;
  margin-bottom: 1rem;
}
.form-group {
  margin-bottom: 1rem;
}
.form-group label {
  display: block;
  font-weight: 500;
  color: #374151;
  margin-bottom: 0.35rem;
  font-size: 0.9rem;
}
.form-group input {
  width: 100%;
  padding: 0.65rem 0.75rem;
  border: 1px solid #d1d5db;
  border-radius: 6px;
  font-size: 1rem;
  outline: none;
  box-sizing: border-box;
}
.form-group input:focus {
  border-color: #4f46e5;
  box-shadow: 0 0 0 3px #4f46e520;
}
.alert {
  padding: 0.75rem 1rem;
  border-radius: 6px;
  margin-bottom: 1rem;
  font-size: 0.9rem;
}
.alert-success {
  background: #ecfdf5;
  color: #065f46;
  border: 1px solid #a7f3d0;
}
.alert-error {
  background: #fef2f2;
  color: #991b1b;
  border: 1px solid #fecaca;
}
.btn {
  padding: 0.65rem 1.5rem;
  border-radius: 8px;
  font-weight: 600;
  font-size: 1rem;
  cursor: pointer;
  border: none;
}
.btn-primary {
  background: #4f46e5;
  color: #fff;
}
.btn-primary:hover {
  background: #4338ca;
}
.btn-primary:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}
.btn-block {
  display: block;
  width: 100%;
  margin-top: 0.5rem;
}
</style>
