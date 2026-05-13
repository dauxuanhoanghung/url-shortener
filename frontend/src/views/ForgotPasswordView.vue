<script setup lang="ts">
import { ref } from 'vue'
import { authService } from '../services/authService'

const email = ref('')
const submitted = ref(false)
const loading = ref(false)

async function handleSubmit() {
  loading.value = true
  try {
    await authService.forgotPassword(email.value)
  } catch {
    // Backend returns 200 regardless to prevent enumeration; any thrown
    // error here is a transport issue, not a user-facing failure we
    // want to distinguish.
  } finally {
    loading.value = false
    submitted.value = true
  }
}
</script>

<template>
  <div class="auth-page">
    <div class="auth-card">
      <h1>Reset your password</h1>

      <div v-if="submitted" class="alert alert-success">
        If an account exists for <strong>{{ email }}</strong>, we've sent a reset link. Check your inbox.
      </div>

      <template v-else>
        <p class="auth-subtitle">
          Enter your email and we'll send you a link to reset your password.
        </p>
        <form @submit.prevent="handleSubmit">
          <div class="form-group">
            <label for="email">Email</label>
            <input
              id="email"
              v-model="email"
              type="email"
              placeholder="you@example.com"
              required
            />
          </div>
          <button type="submit" class="btn btn-primary btn-block" :disabled="loading">
            {{ loading ? 'Sending…' : 'Send reset link' }}
          </button>
        </form>
      </template>

      <p class="auth-subtitle">
        <router-link to="/login">Back to login</router-link>
      </p>
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
  margin-bottom: 0.25rem;
}
.auth-subtitle {
  color: #6b7280;
  margin: 0.5rem 0 1.5rem;
  font-size: 0.95rem;
}
.auth-subtitle a {
  color: #4f46e5;
  text-decoration: none;
  font-weight: 600;
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
.btn {
  padding: 0.65rem 1.5rem;
  border-radius: 8px;
  font-weight: 600;
  font-size: 1rem;
  cursor: pointer;
  border: none;
  transition: background 0.15s;
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
