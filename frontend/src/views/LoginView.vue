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
const error = ref('')
const loading = ref(false)

async function handleSubmit() {
  error.value = ''
  loading.value = true
  try {
    const resp = await auth.login(email.value, password.value)
    if (resp.success) router.push({ name: 'dashboard' })
  } catch (err: any) {
    error.value = err.response?.data?.error?.message || 'Login failed. Please try again.'
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="flex min-h-[calc(100vh-3.5rem)] items-center justify-center px-4 py-12">
    <Card class="w-full max-w-sm">
      <CardHeader class="space-y-1">
        <CardTitle class="text-2xl">Log in</CardTitle>
        <p class="text-sm text-muted-foreground">
          Don't have an account?
          <router-link to="/register" class="font-medium text-primary hover:underline">Sign up</router-link>
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
            <Input id="password" v-model="password" type="password" placeholder="Your password" required />
          </div>
          <Button type="submit" class="w-full" :disabled="loading">
            {{ loading ? 'Logging in…' : 'Log in' }}
          </Button>
        </form>

        <p class="mt-4 text-center text-sm text-muted-foreground">
          <router-link to="/forgot-password" class="text-primary hover:underline">Forgot your password?</router-link>
        </p>
      </CardContent>
    </Card>
  </div>
</template>
