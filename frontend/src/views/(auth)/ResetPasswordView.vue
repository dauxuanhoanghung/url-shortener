<script setup lang="ts">
import { computed, ref } from "vue";
import { useRoute, useRouter } from "vue-router";

import PasswordInput from "@/components/PasswordInput.vue";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Label } from "@/components/ui/label";
import { authService } from "@/services/authService";

const route = useRoute();
const router = useRouter();

const token = computed(() => (typeof route.query.token === "string" ? route.query.token : ""));
const password = ref("");
const confirmPassword = ref("");
const error = ref("");
const loading = ref(false);
const done = ref(false);

async function handleSubmit() {
  error.value = "";
  if (!token.value) {
    error.value = "This reset link is missing a token.";
    return;
  }
  if (password.value.length < 8) {
    error.value = "Password must be at least 8 characters.";
    return;
  }
  if (password.value !== confirmPassword.value) {
    error.value = "Passwords do not match.";
    return;
  }

  loading.value = true;
  try {
    const resp = await authService.resetPassword(token.value, password.value);
    if (resp.success) {
      done.value = true;
      setTimeout(() => router.push({ name: "login" }), 2000);
    } else {
      error.value = resp.error?.message ?? "Could not reset password.";
    }
  } catch (err: any) {
    const code = err.response?.data?.error?.code;
    error.value =
      code === "TOKEN_INVALID"
        ? "This reset link is invalid or has expired."
        : (err.response?.data?.error?.message ?? "Could not reset password.");
  } finally {
    loading.value = false;
  }
}
</script>

<template>
  <CardHeader class="space-y-1">
    <CardTitle class="text-2xl">Choose a new password</CardTitle>
  </CardHeader>
  <CardContent>
    <Alert v-if="done" class="mb-4 border-emerald-200 bg-emerald-50 text-emerald-800">
      <AlertDescription>Password updated. Redirecting you to login…</AlertDescription>
    </Alert>

    <template v-else>
      <Alert v-if="error" variant="destructive" class="mb-4">
        <AlertDescription>{{ error }}</AlertDescription>
      </Alert>
      <form class="space-y-4" @submit.prevent="handleSubmit">
        <div class="space-y-1.5">
          <Label for="password">New password</Label>
          <PasswordInput
            id="password"
            v-model="password"
            placeholder="Min. 8 characters"
            autocomplete="new-password"
          />
        </div>
        <div class="space-y-1.5">
          <Label for="confirmPassword">Confirm password</Label>
          <PasswordInput
            id="confirmPassword"
            v-model="confirmPassword"
            autocomplete="new-password"
          />
        </div>
        <Button type="submit" class="w-full" :disabled="loading">
          {{ loading ? "Updating…" : "Update password" }}
        </Button>
      </form>
    </template>
  </CardContent>
</template>
