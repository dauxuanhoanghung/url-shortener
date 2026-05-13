<script setup lang="ts">
import { onMounted } from 'vue'
import { useAuthStore } from '@/stores/authStore'
import { usePlanStore } from '@/stores/planStore'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'

const auth = useAuthStore()
const plans = usePlanStore()

onMounted(() => plans.fetchAll())

function formatPrice(cents: number): string {
  return cents === 0 ? '$0' : `$${cents / 100}`
}
</script>

<template>
  <div class="mx-auto max-w-5xl px-4 py-8 md:px-6">

    <!-- Hero -->
    <section class="py-16 text-center md:py-24">
      <Badge variant="secondary" class="mb-4">Now in beta</Badge>
      <h1 class="text-4xl font-extrabold tracking-tight text-foreground md:text-6xl">
        Shorten your links,<br class="hidden sm:block" /> expand your reach
      </h1>
      <p class="mx-auto mt-4 max-w-xl text-lg text-muted-foreground">
        Create short, memorable links in seconds. Track clicks and manage all
        your URLs from one dashboard.
      </p>
      <div class="mt-8 flex flex-wrap justify-center gap-3">
        <Button size="lg" as-child v-if="!auth.isAuthenticated">
          <router-link to="/register">Get started free</router-link>
        </Button>
        <Button size="lg" as-child v-else>
          <router-link to="/dashboard">Go to Dashboard</router-link>
        </Button>
        <Button size="lg" variant="outline" as-child>
          <router-link to="/pricing">See pricing</router-link>
        </Button>
      </div>
    </section>

    <!-- Features -->
    <section class="grid gap-4 pb-8 md:grid-cols-3">
      <Card>
        <CardHeader>
          <div class="mb-1 text-3xl">⚡</div>
          <CardTitle class="text-base">Lightning Fast</CardTitle>
        </CardHeader>
        <CardContent>
          <p class="text-sm text-muted-foreground">Redirects in under 100ms with Redis-powered caching.</p>
        </CardContent>
      </Card>
      <Card>
        <CardHeader>
          <div class="mb-1 text-3xl">🔒</div>
          <CardTitle class="text-base">Secure</CardTitle>
        </CardHeader>
        <CardContent>
          <p class="text-sm text-muted-foreground">URL validation, rate limiting, and JWT authentication.</p>
        </CardContent>
      </Card>
      <Card>
        <CardHeader>
          <div class="mb-1 text-3xl">📈</div>
          <CardTitle class="text-base">Scalable</CardTitle>
        </CardHeader>
        <CardContent>
          <p class="text-sm text-muted-foreground">Start free, upgrade when you grow. Plans up to 100,000 URLs.</p>
        </CardContent>
      </Card>
    </section>

    <!-- Plans preview -->
    <section class="py-12 text-center">
      <h2 class="text-3xl font-bold tracking-tight text-foreground">Simple pricing</h2>
      <p class="mt-2 text-sm text-muted-foreground">
        <router-link to="/pricing" class="text-primary underline-offset-4 hover:underline">
          See full feature comparison →
        </router-link>
      </p>

      <div v-if="plans.loading" class="mt-10 text-sm text-muted-foreground">Loading plans…</div>

      <div v-else class="mt-8 grid gap-4 sm:grid-cols-3">
        <Card
          v-for="plan in plans.plans"
          :key="plan.code"
          class="text-left"
          :class="plan.code === 'pro' ? 'border-primary shadow-sm' : ''"
        >
          <CardHeader class="pb-2">
            <div class="flex items-center justify-between">
              <CardTitle>{{ plan.name }}</CardTitle>
              <Badge v-if="plan.code === 'pro'" class="text-xs">Popular</Badge>
            </div>
            <div class="flex items-baseline gap-1 pt-1">
              <span class="text-3xl font-extrabold">{{ formatPrice(plan.price_cents) }}</span>
              <span v-if="plan.price_cents > 0" class="text-sm text-muted-foreground">/mo</span>
            </div>
          </CardHeader>
          <CardContent class="space-y-1.5">
            <p class="text-sm text-muted-foreground">
              Up to <strong class="text-foreground">{{ plan.max_urls.toLocaleString() }}</strong> URLs
            </p>
            <p v-if="plan.features.custom_alias" class="text-sm text-muted-foreground">✓ Custom short codes</p>
            <p v-if="plan.features.api_access" class="text-sm text-muted-foreground">✓ API access</p>
            <Button
              v-if="!auth.isAuthenticated"
              class="mt-3 w-full"
              :variant="plan.code === 'pro' ? 'default' : 'outline'"
              as-child
            >
              <router-link to="/register">
                {{ plan.code === 'free' ? 'Sign up free' : 'Get started' }}
              </router-link>
            </Button>
          </CardContent>
        </Card>
      </div>
    </section>

  </div>
</template>
