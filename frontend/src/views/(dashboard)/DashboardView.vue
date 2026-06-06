<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useAuthStore } from '@/stores/authStore'
import { useUrlStore } from '@/stores/urlStore'
import { useSSE } from '@/composables/useSSE'
import { authService } from '@/services/authService'
import UrlForm from '@/components/UrlForm.vue'
import UrlList from '@/components/UrlList.vue'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Alert, AlertDescription } from '@/components/ui/alert'
import type { SSEUrlDeletedEvent, SSEMetadataUpdatedEvent } from '@/types'

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
      resendMessage.value = err.response?.data?.error?.message ?? 'Could not resend verification email.'
    }
  }
}

onMounted(() => {
  store.fetchAll()
  const token = localStorage.getItem('access_token')
  if (token) {
    const sse = useSSE({ token })
    sse.on<SSEUrlDeletedEvent>('url_deleted', store.handleUrlDeleted)
    sse.on<SSEMetadataUpdatedEvent>('metadata_updated', store.handleMetadataUpdated)
    // sse.close() is called automatically on unmount via onUnmounted inside useSSE
  }
})
</script>

<template>
  <div class="mx-auto max-w-4xl px-4 py-8 md:px-6">

    <!-- Header -->
    <div class="mb-6 flex items-start justify-between gap-4">
      <div>
        <h1 class="text-2xl font-bold text-foreground">Your URLs</h1>
        <p class="mt-0.5 text-sm text-muted-foreground">
          {{ auth.user?.email }}
          <Badge variant="secondary" class="ml-1.5 capitalize">{{ auth.user?.plan_code }}</Badge>
          &middot;
          {{ store.total }} URL{{ store.total !== 1 ? 's' : '' }}
        </p>
      </div>
    </div>

    <!-- Email verification banner -->
    <Alert
      v-if="auth.user && !auth.user.email_verified"
      class="mb-6 border-amber-200 bg-amber-50 text-amber-900"
    >
      <AlertDescription class="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
        <span>
          <strong>Please verify your email.</strong>
          After 7 days, creating URLs will be blocked until you verify.
        </span>
        <div class="flex shrink-0 items-center gap-2">
          <span v-if="resendStatus === 'sent'" class="text-sm">Sent. Check your inbox.</span>
          <Button
            v-else
            size="sm"
            variant="outline"
            class="border-amber-300 bg-white hover:bg-amber-50"
            :disabled="resendStatus === 'sending'"
            @click="handleResend"
          >
            {{ resendStatus === 'sending' ? 'Sending…' : 'Resend verification' }}
          </Button>
        </div>
      </AlertDescription>
      <p v-if="resendStatus === 'error'" class="mt-1 text-xs text-destructive">{{ resendMessage }}</p>
    </Alert>

    <!-- Dead-link notifications (pushed via SSE) -->
    <div v-if="store.deletedNotifications.length" class="mb-4 flex flex-col gap-2">
      <Alert
        v-for="n in store.deletedNotifications"
        :key="n.id"
        class="border-red-200 bg-red-50 text-red-900"
      >
        <AlertDescription class="flex items-center justify-between gap-4">
          <span class="text-sm">
            <strong>Link removed:</strong> {{ n.message }}
          </span>
          <Button
            size="sm"
            variant="ghost"
            class="h-6 shrink-0 px-2 text-xs text-red-700 hover:bg-red-100"
            @click="store.dismissNotification(n.id)"
          >
            Dismiss
          </Button>
        </AlertDescription>
      </Alert>
    </div>

    <!-- URL form -->
    <UrlForm />

    <!-- Store error -->
    <Alert v-if="store.error" variant="destructive" class="mb-4">
      <AlertDescription>{{ store.error }}</AlertDescription>
    </Alert>

    <!-- URL list -->
    <UrlList />

  </div>
</template>
