<script setup lang="ts">
import { onMounted, reactive, ref } from "vue";

import { Alert, AlertDescription } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { usePlanStore } from "@/stores/planStore";
import { adminService } from "@/services/adminService";
import type { Plan } from "@/types";

const planStore = usePlanStore();
const saving = ref<Record<string, boolean>>({});
const error = ref("");
const success = ref("");

// Local editable copy of features keyed by plan code.
const drafts = reactive<Record<string, Record<string, boolean>>>({});

onMounted(async () => {
  // Force a fresh load — the cached copy may not reflect recent admin edits.
  planStore.plans = [];
  await planStore.fetchAll();
  for (const p of planStore.plans) {
    drafts[p.code] = { ...p.features };
  }
});

// Union of feature keys across all plans, so toggling a key on one plan
// surfaces it consistently even if not yet set elsewhere.
const featureKeys = (): string[] => {
  const keys = new Set<string>();
  for (const p of planStore.plans) {
    Object.keys(p.features).forEach((k) => keys.add(k));
  }
  return Array.from(keys).sort();
};

async function save(plan: Plan) {
  saving.value[plan.code] = true;
  error.value = "";
  success.value = "";
  try {
    const resp = await adminService.updatePlanFeatures(plan.code, drafts[plan.code]);
    if (resp.success && resp.data) {
      // Update the cached plan in the shared store so other views see the change.
      const idx = planStore.plans.findIndex((p) => p.code === plan.code);
      if (idx >= 0) planStore.plans[idx] = resp.data;
      success.value = `Updated features for ${plan.name}.`;
    } else {
      error.value = resp.error?.message ?? "Failed to update plan";
    }
  } catch (err: any) {
    error.value = err.response?.data?.error?.message ?? "Failed to update plan";
  } finally {
    saving.value[plan.code] = false;
  }
}
</script>

<template>
  <section>
    <h2 class="mb-4 text-lg font-semibold">Plan feature flags</h2>
    <p class="text-muted-foreground mb-4 text-sm">
      Toggle boolean feature flags per plan. Pricing, URL limits, and rate limits are migration-only
      and not editable here. See
      <code>docs/25-admin-accounts.md §4</code>.
    </p>

    <Alert v-if="error" variant="destructive" class="mb-4">
      <AlertDescription>{{ error }}</AlertDescription>
    </Alert>
    <Alert v-if="success" class="mb-4 border-emerald-500 bg-emerald-50">
      <AlertDescription class="text-emerald-700">{{ success }}</AlertDescription>
    </Alert>

    <div class="grid gap-4 md:grid-cols-3">
      <Card v-for="plan in planStore.plans" :key="plan.code">
        <CardHeader>
          <CardTitle class="flex items-center justify-between">
            <span>{{ plan.name }}</span>
            <span class="text-muted-foreground text-sm font-normal">
              {{ plan.code }}
            </span>
          </CardTitle>
        </CardHeader>
        <CardContent>
          <ul class="space-y-2">
            <li
              v-for="key in featureKeys()"
              :key="key"
              class="flex items-center justify-between text-sm"
            >
              <label :for="`${plan.code}-${key}`" class="cursor-pointer">
                {{ key }}
              </label>
              <input
                :id="`${plan.code}-${key}`"
                type="checkbox"
                v-model="drafts[plan.code][key]"
                class="h-4 w-4 cursor-pointer"
              />
            </li>
          </ul>
          <Button class="mt-4 w-full" size="sm" :disabled="saving[plan.code]" @click="save(plan)">
            {{ saving[plan.code] ? "Saving…" : "Save" }}
          </Button>
        </CardContent>
      </Card>
    </div>
  </section>
</template>
