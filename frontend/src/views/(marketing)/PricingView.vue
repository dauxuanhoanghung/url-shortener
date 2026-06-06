<script setup lang="ts">
import { computed, onMounted } from "vue";

import { Alert, AlertDescription } from "@/components/ui/alert";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardFooter, CardHeader, CardTitle } from "@/components/ui/card";
import { Separator } from "@/components/ui/separator";
import { useAuthStore } from "@/stores/authStore";
import { usePlanStore } from "@/stores/planStore";
import type { Plan } from "@/types";

const store = usePlanStore();
const auth = useAuthStore();

onMounted(() => store.fetchAll());

function formatPrice(cents: number): string {
  return cents === 0 ? "Free" : `$${(cents / 100).toFixed(0)}`;
}

function isCurrentPlan(plan: Plan): boolean {
  return !!auth.user && auth.user.plan_code === plan.code;
}

const featureLabels: Record<string, string> = {
  custom_alias: "Custom short code",
  custom_domain: "Custom domain",
  link_expiration: "Link expiration date",
  password_protected: "Password-protected links",
  geo_analytics: "Geographic analytics",
  bulk_import: "Bulk CSV import",
  api_access: "API access",
  webhooks: "Outbound webhooks",
  remove_branding: "Remove redirect branding",
};

const featureOrder = Object.keys(featureLabels);

function planFeatures(plan: Plan) {
  return featureOrder.map((key) => ({
    label: featureLabels[key],
    enabled: plan.features[key] ?? false,
  }));
}

const ctaLabel = computed(() => (plan: Plan) => {
  if (!auth.isAuthenticated) return plan.code === "free" ? "Sign up free" : "Get started";
  if (isCurrentPlan(plan)) return "Current plan";
  return "Upgrade";
});

const ctaTo = (_plan: Plan) => (!auth.isAuthenticated ? "/register" : "/dashboard");
</script>

<template>
  <div class="mx-auto max-w-5xl px-4 py-12 md:px-6">
    <div class="mb-12 text-center">
      <h1 class="text-foreground text-4xl font-extrabold tracking-tight">
        Simple, transparent pricing
      </h1>
      <p class="text-muted-foreground mt-3 text-lg">
        Start free. Upgrade when you need more power.
      </p>
    </div>

    <p v-if="store.loading" class="text-muted-foreground text-center text-sm">Loading plans…</p>

    <Alert v-else-if="store.error" variant="destructive" class="mb-6">
      <AlertDescription>{{ store.error }}</AlertDescription>
    </Alert>

    <div v-else class="grid gap-6 md:grid-cols-3">
      <Card
        v-for="plan in store.plans"
        :key="plan.code"
        class="relative flex flex-col"
        :class="{
          'border-primary shadow-md': plan.code === 'pro' && !isCurrentPlan(plan),
          'border-emerald-500 shadow-sm': isCurrentPlan(plan),
        }"
      >
        <div
          v-if="plan.code === 'pro' && !isCurrentPlan(plan)"
          class="absolute left-1/2 -translate-x-1/2"
        >
          <Badge class="px-3 text-xs">Most popular</Badge>
        </div>
        <div v-if="isCurrentPlan(plan)" class="absolute left-1/2 -translate-x-1/2">
          <Badge
            variant="outline"
            class="border-emerald-400 bg-emerald-50 px-3 text-xs text-emerald-700"
          >
            Your plan
          </Badge>
        </div>

        <CardHeader class="pb-4">
          <CardTitle class="text-xl">{{ plan.name }}</CardTitle>
          <div class="flex items-baseline gap-1 pt-1">
            <span class="text-5xl font-extrabold tracking-tight">{{
              formatPrice(plan.price_cents)
            }}</span>
            <span v-if="plan.price_cents > 0" class="text-muted-foreground">/month</span>
          </div>
        </CardHeader>

        <CardContent class="flex-1 space-y-4">
          <ul class="text-muted-foreground space-y-1 text-sm">
            <li>
              <span class="text-foreground font-semibold">{{
                plan.max_urls.toLocaleString()
              }}</span>
              URLs
            </li>
            <li>
              <span class="text-foreground font-semibold"
                >{{ plan.analytics_retention_days }}d</span
              >
              analytics retention
            </li>
            <li v-if="plan.max_domains > 0">
              <span class="text-foreground font-semibold">{{ plan.max_domains }}</span>
              custom domain{{ plan.max_domains > 1 ? "s" : "" }}
            </li>
            <li v-if="plan.api_rate_limit_per_min">
              <span class="text-foreground font-semibold">{{ plan.api_rate_limit_per_min }}</span>
              API req/min
            </li>
          </ul>

          <Separator />

          <ul class="space-y-2">
            <li
              v-for="feat in planFeatures(plan)"
              :key="feat.label"
              class="flex items-center gap-2 text-sm"
              :class="feat.enabled ? 'text-foreground' : 'text-muted-foreground'"
            >
              <span
                class="w-4 shrink-0 text-center font-bold"
                :class="feat.enabled ? 'text-primary' : 'text-muted-foreground/40'"
              >
                {{ feat.enabled ? "✓" : "–" }}
              </span>
              {{ feat.label }}
            </li>
          </ul>
        </CardContent>

        <CardFooter class="pt-4">
          <Button
            class="w-full"
            :variant="plan.code === 'pro' ? 'default' : 'outline'"
            :disabled="isCurrentPlan(plan)"
            as-child
          >
            <router-link :to="ctaTo(plan)">{{ ctaLabel(plan) }}</router-link>
          </Button>
        </CardFooter>
      </Card>
    </div>
  </div>
</template>
