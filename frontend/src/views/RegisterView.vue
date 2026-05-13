<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/authStore'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'

const auth = useAuthStore()
const router = useRouter()

const email = ref('')
const password = ref('')
const confirmPassword = ref('')
const error = ref('')
const loading = ref(false)

async function handleSubmit() {
  error.value = ''
  if (password.value !== confirmPassword.value) { error.value = 'Passwords do not match'; return }
  if (password.value.length < 8) { error.value = 'Password must be at least 8 characters'; return }

  loading.value = true
  try {
    const resp = await auth.register(email.value, password.value)
    if (resp.success) router.push({ name: 'dashboard' })
  } catch (err: any) {
    error.value = err.response?.data?.error?.message || 'Registration failed. Please try again.'
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="flex min-h-[calc(100vh-3.5rem)] items-center justify-center px-4 py-12">
    <Card class="w-full max-w-sm">
      <CardHeader class="space-y-1">
        <CardTitle class="text-2xl">Create an account</CardTitle>
        <p class="text-sm text-muted-foreground">
          Already have an account?
          <router-link to="/login" class="font-medium text-primary hover:underline">Log in</router-link>
        </p>
      </CardHeader>
      <CardContent>
        <Alert v-if="error" variant="destructive" class="mb-4">
          <AlertDescription>{{ error }}</AlertDescription>
        </Alert>

        <form class="space-y-4" @submit.prevent="handleSubmit">
          <div class="space-y-1.5">
            <Label for="email">Email</Label>
            <Input id="email" v-model="email" type="email" placeholder="you@example.com" required />
          </div>
          <div class="space-y-1.5">
            <Label for="password">Password</Label>
            <Input id="password" v-model="password" type="password" placeholder="Min. 8 characters" required minlength="8" />
          </div>
          <div class="space-y-1.5">
            <Label for="confirmPassword">Confirm password</Label>
            <Input id="confirmPassword" v-model="confirmPassword" type="password" placeholder="Repeat your password" required />
          </div>
          <Button type="submit" class="w-full" :disabled="loading">
            {{ loading ? 'Creating account…' : 'Create account' }}
          </Button>
        </form>
      </CardContent>
    </Card>
  </div>
</template>
