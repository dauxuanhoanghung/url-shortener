<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useAuthStore } from '../stores/authStore'
import { useUrlStore } from '../stores/urlStore'
import { authService } from '../services/authService'
import UrlForm from '../components/UrlForm.vue'
import UrlList from '../components/UrlList.vue'

const auth = useAuthStore()
const store = useUrlStore()

const resendStatus = ref<'idle' | 'sending' | 'sent' | 'error'>('idle')
const resendMessage = ref('')

async function handleResend() {
  resendStatus.value = 'sending'
  resendMessage.value = ''
  try {
    const resp = await authService.resendVerification()
    if (resp.success) {
      resendStatus.value = 'sent'
    } else {
      resendStatus.value = 'error'
      resendMessage.value = resp.error?.message ?? 'Could not resend verification email.'
    }
  } catch (err: any) {
    const code = err.response?.data?.error?.code
    if (code === 'ALREADY_VERIFIED') {
      auth.markEmailVerified()
      resendStatus.value = 'idle'
    } else {
      resendStatus.value = 'error'
      resendMessage.value =
        err.response?.data?.error?.message ?? 'Could not resend verification email.'
    }
  }
}

onMounted(() => {
  store.fetchAll()
})
</script>

<template>
  <div class="dashboard">
    <div class="dashboard-header">
      <div>
        <h1>Your URLs</h1>
        <p>
          <strong>{{ auth.user?.email }}</strong> &middot;
          <span class="plan-badge">{{ auth.user?.plan_type }} plan</span>
          &middot; {{ store.total }} URL<span v-if="store.total !== 1">s</span>
        </p>
      </div>
    </div>

    <div v-if="auth.user && !auth.user.email_verified" class="alert alert-warning">
      <div>
        <strong>Please verify your email.</strong>
        After a 7-day grace period, creating URLs will be blocked until you verify.
      </div>
      <div class="banner-actions">
        <button
          v-if="resendStatus !== 'sent'"
          type="button"
          class="btn btn-secondary"
          :disabled="resendStatus === 'sending'"
          @click="handleResend"
        >
          {{ resendStatus === 'sending' ? 'Sending…' : 'Resend verification email' }}
        </button>
        <span v-else class="banner-note">Sent. Check your inbox.</span>
      </div>
      <div v-if="resendStatus === 'error'" class="banner-note error">{{ resendMessage }}</div>
    </div>

    <UrlForm />

    <div v-if="store.error" class="alert alert-error">{{ store.error }}</div>

    <UrlList />
  </div>
</template>

<style scoped>
.dashboard {
  max-width: 960px;
  margin: 0 auto;
  padding: 2rem 1.5rem;
}

.dashboard-header {
  margin-bottom: 2rem;
}

.dashboard-header h1 {
  font-size: 2rem;
  font-weight: 700;
  color: #111827;
  margin-bottom: 0.25rem;
}

.dashboard-header p {
  color: #6b7280;
}

.plan-badge {
  display: inline-block;
  padding: 0.15rem 0.6rem;
  border-radius: 999px;
  background: #eef2ff;
  color: #4f46e5;
  font-size: 0.8rem;
  font-weight: 600;
  text-transform: capitalize;
}

.alert {
  padding: 0.75rem 1rem;
  border-radius: 6px;
  margin-bottom: 1rem;
  font-size: 0.9rem;
}

.alert-error {
  background: #fef2f2;
  color: #991b1b;
  border: 1px solid #fecaca;
}

.alert-warning {
  background: #fffbeb;
  color: #92400e;
  border: 1px solid #fcd34d;
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.banner-actions {
  display: flex;
  align-items: center;
  gap: 0.75rem;
}

.banner-note {
  color: #92400e;
  font-size: 0.85rem;
}

.banner-note.error {
  color: #991b1b;
}

.btn {
  padding: 0.45rem 1rem;
  border-radius: 6px;
  font-weight: 600;
  font-size: 0.9rem;
  cursor: pointer;
  border: none;
}

.btn-secondary {
  background: #fff;
  color: #92400e;
  border: 1px solid #fcd34d;
}

.btn-secondary:hover {
  background: #fef3c7;
}

.btn-secondary:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}
</style>
