<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import { authService } from '@/services/authService'
import { useAuthStore } from '@/stores/authStore'
import { Button } from '@/components/ui/button'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'

const route = useRoute()
const auth = useAuthStore()

type Status = 'verifying' | 'success' | 'invalid' | 'error'
const status = ref<Status>('verifying')
const errorMessage = ref('')

onMounted(async () => {
  const token = typeof route.query.token === 'string' ? route.query.token : ''
  if (!token) { status.value = 'invalid'; return }
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
    errorMessage.value = err.response?.data?.error?.message ?? 'Something went wrong verifying your email.'
  }
})
</script>

<template>
  <div class="flex min-h-[calc(100vh-3.5rem)] items-center justify-center px-4 py-12">
    <Card class="w-full max-w-sm">
      <CardHeader>
        <CardTitle class="text-2xl">Email verification</CardTitle>
      </CardHeader>
      <CardContent class="space-y-4">

        <p v-if="status === 'verifying'" class="text-sm text-muted-foreground">
          Verifying your email…
        </p>

        <template v-else-if="status === 'success'">
          <Alert class="border-emerald-200 bg-emerald-50 text-emerald-800">
            <AlertDescription>Your email is verified. You can continue using your account.</AlertDescription>
          </Alert>
          <Button class="w-full" as-child>
            <router-link to="/dashboard">Go to dashboard</router-link>
          </Button>
        </template>

        <template v-else-if="status === 'invalid'">
          <Alert variant="destructive">
            <AlertDescription>
              This verification link is invalid or has expired. Request a new one from your dashboard.
            </AlertDescription>
          </Alert>
          <Button variant="outline" class="w-full" as-child>
            <router-link to="/dashboard">Back to dashboard</router-link>
          </Button>
        </template>

        <template v-else>
          <Alert variant="destructive">
            <AlertDescription>{{ errorMessage || 'Something went wrong.' }}</AlertDescription>
          </Alert>
          <Button variant="outline" class="w-full" as-child>
            <router-link to="/dashboard">Back to dashboard</router-link>
          </Button>
        </template>

      </CardContent>
    </Card>
  </div>
</template>
