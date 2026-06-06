<script setup lang="ts">
import { useRouter } from "vue-router";

import { Button } from "@/components/ui/button";
import { Separator } from "@/components/ui/separator";
import { useAuthStore } from "@/stores/authStore";

const auth = useAuthStore();
const router = useRouter();

function handleLogout() {
  auth.logout();
  router.push({ name: "landing" });
}
</script>

<template>
  <header
    class="border-border bg-background/95 supports-backdrop-filter:bg-background/60 sticky top-0 z-50 w-full border-b backdrop-blur"
  >
    <div class="mx-auto flex h-14 max-w-7xl items-center gap-4 px-4 md:px-6">
      <router-link to="/" class="text-primary flex shrink-0 items-center text-lg font-bold">
        Snip.ly
      </router-link>

      <nav class="flex flex-1 items-center gap-1 text-sm">
        <Button variant="ghost" size="sm" as-child>
          <router-link to="/pricing">Pricing</router-link>
        </Button>
      </nav>

      <div class="flex items-center gap-2">
        <template v-if="auth.isAuthenticated">
          <Button variant="ghost" size="sm" as-child>
            <router-link to="/dashboard">Dashboard</router-link>
          </Button>
          <Button v-if="auth.isAdmin" variant="ghost" size="sm" as-child>
            <router-link to="/admin">Admin</router-link>
          </Button>
          <Separator orientation="vertical" class="h-5" />
          <span class="text-muted-foreground hidden max-w-40 truncate text-xs sm:block">
            {{ auth.user?.email }}
          </span>
          <Button variant="outline" size="sm" @click="handleLogout">Log out</Button>
        </template>
        <template v-else>
          <Button variant="ghost" size="sm" as-child>
            <router-link to="/login">Log in</router-link>
          </Button>
          <Button size="sm" as-child>
            <router-link to="/register">Sign up free</router-link>
          </Button>
        </template>
      </div>
    </div>
  </header>
</template>
