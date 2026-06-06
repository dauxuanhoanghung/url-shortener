<script setup lang="ts">
import { Badge } from "@/components/ui/badge";
import { Separator } from "@/components/ui/separator";
import { useAuthStore } from "@/stores/authStore";

const auth = useAuthStore();
</script>

<template>
  <div class="mx-auto max-w-7xl px-4 py-8 md:px-6">
    <header class="mb-6 flex items-center justify-between">
      <div>
        <div class="flex items-center gap-2">
          <h1 class="text-2xl font-bold tracking-tight">Admin Console</h1>
          <Badge variant="outline" class="border-amber-400 bg-amber-50 text-amber-700">
            {{ auth.user?.email }}
          </Badge>
        </div>
        <p class="mt-1 text-sm text-muted-foreground">
          Operator-only tooling. All write actions are recorded in the audit log.
        </p>
      </div>
    </header>

    <nav class="mb-6 flex gap-1 text-sm">
      <router-link
        v-for="tab in [
          { to: '/admin/users', label: 'Users' },
          { to: '/admin/plans', label: 'Plans' },
          { to: '/admin/audit', label: 'Audit log' },
        ]"
        :key="tab.to"
        :to="tab.to"
        class="rounded-md px-3 py-1.5 text-muted-foreground transition-colors hover:bg-accent hover:text-foreground"
        active-class="bg-accent text-foreground font-medium"
      >
        {{ tab.label }}
      </router-link>
    </nav>

    <Separator class="mb-6" />

    <router-view />
  </div>
</template>
