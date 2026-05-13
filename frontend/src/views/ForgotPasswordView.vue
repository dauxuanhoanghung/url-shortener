<script setup lang="ts">
import { ref } from 'vue'
import { authService } from '@/services/authService'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'

const email = ref('')
const submitted = ref(false)
const loading = ref(false)

async function handleSubmit() {
  loading.value = true
  try {
    await authService.forgotPassword(email.value)
  } catch {
    // backend always returns 200 — transport errors are silently swallowed
  } finally {
    loading.value = false
    submitted.value = true
  }
}
</script>

<template>
  <div class="flex min-h-[calc(100vh-3.5rem)] items-center justify-center px-4 py-12">
    <Card class="w-full max-w-sm">
      <CardHeader class="space-y-1">
        <CardTitle class="text-2xl">Reset your password</CardTitle>
      </CardHeader>
      <CardContent>
        <Alert v-if="submitted" class="mb-4 border-emerald-200 bg-emerald-50 text-emerald-800">
          <AlertDescription>
            If an account exists for <strong>{{ email }}</strong>, we've sent a reset link.
            Check your inbox.
          </AlertDescription>
        </Alert>

        <template v-else>
          <p class="mb-4 text-sm text-muted-foreground">
            Enter your email and we'll send you a link to reset your password.
          </p>
          <form class="space-y-4" @submit.prevent="handleSubmit">
            <div class="space-y-1.5">
              <Label for="email">Email</Label>
              <Input id="email" v-model="email" type="email" placeholder="you@example.com" required />
            </div>
            <Button type="submit" class="w-full" :disabled="loading">
              {{ loading ? 'Sending…' : 'Send reset link' }}
            </Button>
          </form>
        </template>

        <p class="mt-4 text-center text-sm text-muted-foreground">
          <router-link to="/login" class="text-primary hover:underline">Back to login</router-link>
        </p>
      </CardContent>
    </Card>
  </div>
</template>
