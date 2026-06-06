<script setup lang="ts">
import { toTypedSchema } from "@vee-validate/zod";
import { useForm } from "vee-validate";
import { ref } from "vue";
import { useRouter } from "vue-router";
import { z } from "zod";

import PasswordInput from "@/components/PasswordInput.vue";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { useAuthStore } from "@/stores/authStore";

const auth = useAuthStore();
const router = useRouter();

const schema = toTypedSchema(
  z
    .object({
      email: z.string().min(1, "Email is required").email("Enter a valid email"),
      password: z.string().min(8, "Password must be at least 8 characters"),
      confirmPassword: z.string().min(1, "Please confirm your password"),
    })
    .refine((v) => v.password === v.confirmPassword, {
      path: ["confirmPassword"],
      message: "Passwords do not match",
    }),
);

const { defineField, handleSubmit, errors, isSubmitting } = useForm({
  validationSchema: schema,
  initialValues: { email: "", password: "", confirmPassword: "" },
});

const [email, emailAttrs] = defineField("email");
const [password, passwordAttrs] = defineField("password");
const [confirmPassword, confirmPasswordAttrs] = defineField("confirmPassword");

const serverError = ref("");

const onSubmit = handleSubmit(async (values) => {
  serverError.value = "";
  try {
    const resp = await auth.register(values.email, values.password);
    if (resp.success) {
      router.push({ name: "dashboard" });
      return;
    }
    serverError.value = resp.error?.message || "Registration failed. Please try again.";
  } catch (err: any) {
    serverError.value =
      err.response?.data?.error?.message || "Registration failed. Please try again.";
  }
});
</script>

<template>
  <CardHeader class="space-y-1">
    <CardTitle class="text-2xl">Create an account</CardTitle>
    <p class="text-muted-foreground text-sm">
      Already have an account?
      <router-link to="/login" class="text-primary font-medium hover:underline">Log in</router-link>
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
        <p v-if="errors.email" class="text-destructive text-sm">{{ errors.email }}</p>
      </div>
      <div class="space-y-1.5">
        <Label for="password">Password</Label>
        <PasswordInput
          id="password"
          v-model="password"
          v-bind="passwordAttrs"
          placeholder="Min. 8 characters"
          autocomplete="new-password"
        />
        <p v-if="errors.password" class="text-destructive text-sm">{{ errors.password }}</p>
      </div>
      <div class="space-y-1.5">
        <Label for="confirmPassword">Confirm password</Label>
        <PasswordInput
          id="confirmPassword"
          v-model="confirmPassword"
          v-bind="confirmPasswordAttrs"
          placeholder="Repeat your password"
          autocomplete="new-password"
        />
        <p v-if="errors.confirmPassword" class="text-destructive text-sm">
          {{ errors.confirmPassword }}
        </p>
      </div>
      <Button type="submit" class="w-full" :disabled="isSubmitting">
        {{ isSubmitting ? "Creating account…" : "Create account" }}
      </Button>
    </form>
  </CardContent>
</template>
