<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import { authService } from '../services/authService'
import { useAuthStore } from '../stores/authStore'

const route = useRoute()
const auth = useAuthStore()

type Status = 'verifying' | 'success' | 'invalid' | 'error'
const status = ref<Status>('verifying')
const errorMessage = ref('')

onMounted(async () => {
  const token = typeof route.query.token === 'string' ? route.query.token : ''
  if (!token) {
    status.value = 'invalid'
    return
  }
  try {
    const resp = await authService.verifyEmail(token)
    if (resp.success) {
      status.value = 'success'
      auth.markEmailVerified()
    } else {
      status.value = resp.error?.code === 'TOKEN_INVALID' ? 'invalid' : 'error'
      errorMessage.value = resp.error?.message ?? ''
    }
  } catch (err: any) {
    const code = err.response?.data?.error?.code
    status.value = code === 'TOKEN_INVALID' ? 'invalid' : 'error'
    errorMessage.value =
      err.response?.data?.error?.message ?? 'Something went wrong verifying your email.'
  }
})
</script>

<template>
  <div class="auth-page">
    <div class="auth-card">
      <h1>Email verification</h1>

      <div v-if="status === 'verifying'" class="state">
        <p>Verifying your email…</p>
      </div>

      <div v-else-if="status === 'success'" class="state">
        <div class="alert alert-success">Your email is verified. You can continue using your account.</div>
        <router-link class="btn btn-primary btn-block" to="/dashboard">Go to dashboard</router-link>
      </div>

      <div v-else-if="status === 'invalid'" class="state">
        <div class="alert alert-error">
          This verification link is invalid or has expired. Request a new one from your dashboard.
        </div>
        <router-link class="btn btn-primary btn-block" to="/dashboard">Back to dashboard</router-link>
      </div>

      <div v-else class="state">
        <div class="alert alert-error">{{ errorMessage || 'Something went wrong.' }}</div>
        <router-link class="btn btn-primary btn-block" to="/dashboard">Back to dashboard</router-link>
      </div>
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
.state p {
  color: #6b7280;
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
  text-decoration: none;
  border: none;
  transition: background 0.15s;
  text-align: center;
  display: inline-block;
}
.btn-primary {
  background: #4f46e5;
  color: #fff;
}
.btn-primary:hover {
  background: #4338ca;
}
.btn-block {
  display: block;
  width: 100%;
  margin-top: 0.5rem;
  box-sizing: border-box;
}
</style>
