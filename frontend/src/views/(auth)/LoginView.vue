<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useForm } from 'vee-validate'
import { toTypedSchema } from '@vee-validate/zod'
import { z } from 'zod'
import { useAuthStore } from '@/stores/authStore'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { CardContent, CardHeader, CardTitle } from '@/components/ui/card'

const auth = useAuthStore()
const router = useRouter()

const schema = toTypedSchema(
  z.object({
    email: z.string().min(1, 'Email is required').email('Enter a valid email'),
    password: z.string().min(1, 'Password is required'),
  }),
)

const { defineField, handleSubmit, errors, isSubmitting } = useForm({
  validationSchema: schema,
  initialValues: { email: '', password: '' },
})

const [email, emailAttrs] = defineField('email')
const [password, passwordAttrs] = defineField('password')

const serverError = ref('')

const onSubmit = handleSubmit(async (values) => {
  serverError.value = ''
  try {
    const resp = await auth.login(values.email, values.password)
    if (resp.success) {
      router.push({ name: 'dashboard' })
      return
    }
    serverError.value = resp.error?.message || 'Login failed. Please try again.'
  } catch (err: any) {
    serverError.value =
      err.response?.data?.error?.message || 'Login failed. Please try again.'
  }
})
</script>

<template>
  <CardHeader class="space-y-1">
        <CardTitle class="text-2xl">Log in</CardTitle>
        <p class="text-sm text-muted-foreground">
          Don't have an account?
          <router-link to="/register" class="font-medium text-primary hover:underline">Sign up</router-link>
        </p>
      </CardHeader>
      <CardContent>
        <Alert v-if="serverError" variant="destructive" class="mb-4">
          <AlertDescription>{{ serverError }}</AlertDescription>
        </Alert>

        <form class="space-y-4" novalidate @submit="onSubmit">
          <div class="space-y-1.5">
            <Label for="email">Email</Label>
            <Input
              id="email"
              v-model="email"
              v-bind="emailAttrs"
              type="email"
              placeholder="you@example.com"
              autocomplete="email"
            />
            <p v-if="errors.email" class="text-sm text-destructive">{{ errors.email }}</p>
          </div>
          <div class="space-y-1.5">
            <Label for="password">Password</Label>
            <Input
              id="password"
              v-model="password"
              v-bind="passwordAttrs"
              type="password"
              placeholder="Your password"
              autocomplete="current-password"
            />
            <p v-if="errors.password" class="text-sm text-destructive">{{ errors.password }}</p>
          </div>
          <Button type="submit" class="w-full" :disabled="isSubmitting">
            {{ isSubmitting ? 'Logging in…' : 'Log in' }}
          </Button>
        </form>

        <p class="mt-4 text-center text-sm text-muted-foreground">
          <router-link to="/forgot-password" class="text-primary hover:underline">Forgot your password?</router-link>
        </p>
      </CardContent>
</template>
